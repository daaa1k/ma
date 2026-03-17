{
  description = "myapp — TODO: describe your app";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      # Read version from VERSION file to keep it in sync automatically.
      version = builtins.replaceStrings [ "\n" ] [ "" ] (builtins.readFile ./VERSION);

      # Pre-built binary hashes for each supported platform.
      # Update these whenever a new version is released:
      #   nix store prefetch-file --hash-type sha256 --json <url>
      binaryHashes = {
        "x86_64-linux"   = ""; # TODO: fill in after first release
        "aarch64-darwin" = ""; # TODO: fill in after first release
      };

      # Map Nix system strings to GitHub Release artifact names.
      binaryArtifacts = {
        "x86_64-linux"   = "myapp-linux-x86_64";
        "aarch64-darwin" = "myapp-macos-aarch64";
      };

      # Build a package wrapping the pre-built GitHub Release binary.
      #
      # On Linux (including WSL2 + NixOS), autoPatchelfHook rewrites the ELF
      # interpreter and RPATH so the binary works under the Nix store layout.
      mkBinaryPackage = pkgs:
        let
          system   = pkgs.stdenv.hostPlatform.system;
          artifact = binaryArtifacts.${system}
            or (throw "myapp-bin: no pre-built binary for ${system}");
          hash     = binaryHashes.${system};
          src = pkgs.fetchurl {
            url = "https://github.com/daaa1k/myapp/releases/download/v${version}/${artifact}";
            inherit hash;
          };
        in
        pkgs.stdenv.mkDerivation {
          pname = "myapp-bin";
          inherit version src;

          dontUnpack = true;

          nativeBuildInputs = pkgs.lib.optionals pkgs.stdenv.isLinux [
            pkgs.autoPatchelfHook
          ];

          buildInputs = pkgs.lib.optionals pkgs.stdenv.isLinux [
            pkgs.glibc
          ];

          installPhase = ''
            install -Dm755 $src $out/bin/myapp
          '';
        };

      # Home Manager module — system-agnostic, exported at the top level.
      #
      # Usage in a Home Manager configuration:
      #
      #   inputs.myapp.url = "github:daaa1k/myapp";
      #
      #   { inputs, ... }: {
      #     imports = [ inputs.myapp.homeManagerModules.default ];
      #     programs.myapp = {
      #       enable = true;
      #       # Use the pre-built binary instead of building from source:
      #       # package = inputs.myapp.packages.${pkgs.system}.myapp-bin;
      #     };
      #   }
      hmModule = { config, lib, pkgs, ... }:
        let
          cfg = config.programs.myapp;
        in
        {
          options.programs.myapp = {
            enable = lib.mkEnableOption "myapp";

            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.system}.default;
              defaultText = lib.literalExpression "myapp.packages.\${pkgs.system}.default";
              description = ''
                The myapp package to install.

                Two variants are available:
                - `myapp.packages.''${pkgs.system}.default` — built from source via buildGoModule (default)
                - `myapp.packages.''${pkgs.system}.myapp-bin` — pre-built binary from GitHub Releases
                  (faster setup; no Go compilation required; supports x86_64-linux and aarch64-darwin)
              '';
            };
          };

          config = lib.mkIf cfg.enable {
            home.packages = [ cfg.package ];
          };
        };
    in
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        myapp = pkgs.buildGoModule {
          pname = "myapp";
          inherit version;

          src = pkgs.lib.cleanSource ./.;

          # Hash of the Go vendor directory produced by `go mod vendor`.
          # To update after changing go.mod / go.sum:
          #   1. Set vendorHash to pkgs.lib.fakeHash
          #   2. Run: nix build .#myapp 2>&1 | grep 'got:'
          #   3. Replace the value below with the hash shown in 'got:'
          vendorHash = null;

          ldflags = [ "-s" "-w" "-X main.version=${version}" ];

          meta = with pkgs.lib; {
            description = "TODO: describe your app";
            homepage    = "https://github.com/daaa1k/myapp";
            license     = licenses.mit;
            mainProgram = "myapp";
          };
        };

        # Format check: `gofmt -l` on non-vendor Go files must produce no output.
        fmtCheck = pkgs.runCommandLocal "myapp-fmt" { } ''
          src=${pkgs.lib.cleanSource ./.}
          unformatted=$(find "$src" -name '*.go' -not -path "*/vendor/*" \
            | xargs ${pkgs.go}/bin/gofmt -l)
          if [ -n "$unformatted" ]; then
            echo "gofmt found unformatted files — run: gofmt -w ."
            echo "$unformatted"
            exit 1
          fi
          touch $out
        '';

        # Vet check: runs `go vet ./...` against the source.
        vetCheck = pkgs.runCommandLocal "myapp-vet"
          { nativeBuildInputs = [ pkgs.go ]; }
          ''
            cp -r ${pkgs.lib.cleanSource ./.} src
            chmod -R u+w src
            ln -sf ${myapp.goModules} src/vendor
            cd src
            HOME=$TMPDIR CGO_ENABLED=0 GOFLAGS=-mod=vendor go vet ./...
            touch $out
          '';
      in
      {
        # --- packages ---------------------------------------------------
        packages = {
          default = myapp;
          inherit myapp;
        } // pkgs.lib.optionalAttrs (binaryArtifacts ? ${system} && binaryHashes.${system} != "") {
          myapp-bin = mkBinaryPackage pkgs;
        };

        # --- checks (run by `nix flake check`) --------------------------
        checks = {
          inherit myapp;
          myapp-fmt = fmtCheck;
          myapp-vet = vetCheck;
        };

        # --- devShell ---------------------------------------------------
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            golangci-lint
          ];
        };
      }
    ) // {
      # --- Home Manager module (system-agnostic) ----------------------
      homeManagerModules.default = hmModule;
    };
}
