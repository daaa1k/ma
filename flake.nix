{
  description = "ma — MCP config adapter and tool launcher";

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
      # Leave as "" to disable the ma-bin package for that platform.
      binaryHashes = {
        "x86_64-linux"   = "sha256-HNTQl4pe1R0tlUaJn1nVUIPj09ZErvKmjfoOShxs2/8=";
        "aarch64-darwin" = "sha256-9A2ygWwlAs7fwzulzg21XrXWc0lF/1kkv+HPGQHbe1c=";
      };

      # Map Nix system strings to GitHub Release artifact names.
      binaryArtifacts = {
        "x86_64-linux"   = "ma-linux-amd64";
        "aarch64-darwin" = "ma-darwin-arm64";
      };

      # Build a package wrapping the pre-built GitHub Release binary.
      #
      # On Linux (including WSL2 + NixOS), autoPatchelfHook rewrites the ELF
      # interpreter and RPATH so the binary works under the Nix store layout.
      mkBinaryPackage = pkgs:
        let
          system   = pkgs.stdenv.hostPlatform.system;
          artifact = binaryArtifacts.${system}
            or (throw "ma-bin: no pre-built binary for ${system}");
          hash     = binaryHashes.${system};
          src = pkgs.fetchurl {
            url = "https://github.com/daaa1k/ma/releases/download/v${version}/${artifact}";
            inherit hash;
          };
        in
        pkgs.stdenv.mkDerivation {
          pname = "ma-bin";
          inherit version src;

          dontUnpack = true;

          nativeBuildInputs = pkgs.lib.optionals pkgs.stdenv.isLinux [
            pkgs.autoPatchelfHook
          ];

          buildInputs = pkgs.lib.optionals pkgs.stdenv.isLinux [
            pkgs.glibc
          ];

          installPhase = ''
            install -Dm755 $src $out/bin/ma
          '';
        };

      # Home Manager module — system-agnostic, exported at the top level.
      #
      # Usage in a Home Manager configuration:
      #
      #   inputs.ma.url = "github:daaa1k/ma";
      #
      #   { inputs, ... }: {
      #     imports = [ inputs.ma.homeManagerModules.default ];
      #     programs.ma = {
      #       enable = true;
      #       # Use the pre-built binary instead of building from source:
      #       # package = inputs.ma.packages.${pkgs.system}.ma-bin;
      #     };
      #   }
      hmModule = { config, lib, pkgs, ... }:
        let
          cfg = config.programs.ma;
        in
        {
          options.programs.ma = {
            enable = lib.mkEnableOption "ma";

            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.system}.default;
              defaultText = lib.literalExpression "ma.packages.\${pkgs.system}.default";
              description = ''
                The ma package to install.

                Two variants are available:
                - `ma.packages.''${pkgs.system}.default` — built from source via buildGoModule (default)
                - `ma.packages.''${pkgs.system}.ma-bin` — pre-built binary from GitHub Releases
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

        # Use Go 1.26 to match go.mod requirement.
        go = pkgs.go_1_26;

        ma = (pkgs.buildGoModule.override { inherit go; }) {
          pname = "ma";
          inherit version;

          src = pkgs.lib.cleanSource ./.;

          # Hash of the Go vendor directory fetched by Nix.
          # vendor/ is not committed; Nix fetches modules via go.sum.
          # To update after changing go.mod / go.sum:
          #   1. Set vendorHash to pkgs.lib.fakeHash
          #   2. Run: nix build .#ma 2>&1 | grep 'got:'
          #   3. Replace the value below with the hash shown in 'got:'
          vendorHash = "sha256-f7cmgTwgkon4SbuR7RsKIALDH0l/5L0CFa4jF6xrGTM=";

          ldflags = [ "-s" "-w" "-X main.version=${version}" ];

          meta = with pkgs.lib; {
            description = "MCP config adapter and tool launcher for AI coding tools";
            homepage    = "https://github.com/daaa1k/ma";
            license     = licenses.mit;
            mainProgram = "ma";
          };
        };

        # Format check: `gofmt -l` on non-vendor Go files must produce no output.
        fmtCheck = pkgs.runCommandLocal "ma-fmt" { } ''
          src=${pkgs.lib.cleanSource ./.}
          unformatted=$(find "$src" -name '*.go' -not -path "*/vendor/*" \
            | xargs ${go}/bin/gofmt -l)
          if [ -n "$unformatted" ]; then
            echo "gofmt found unformatted files — run: gofmt -w ."
            echo "$unformatted"
            exit 1
          fi
          touch $out
        '';

        # Vet check: runs `go vet ./...` against the source.
        vetCheck = pkgs.runCommandLocal "ma-vet"
          { nativeBuildInputs = [ go ]; }
          ''
            cp -r ${pkgs.lib.cleanSource ./.} src
            chmod -R u+w src
            ln -sf ${ma.goModules} src/vendor
            cd src
            HOME=$TMPDIR CGO_ENABLED=0 GOFLAGS=-mod=vendor go vet ./...
            touch $out
          '';
      in
      {
        # --- packages ---------------------------------------------------
        packages = {
          default = ma;
          inherit ma;
        } // pkgs.lib.optionalAttrs (binaryArtifacts ? ${system} && binaryHashes.${system} != "") {
          ma-bin = mkBinaryPackage pkgs;
        };

        # --- checks (run by `nix flake check`) --------------------------
        checks = {
          inherit ma;
          ma-fmt = fmtCheck;
          ma-vet = vetCheck;
        };

        # --- devShell ---------------------------------------------------
        devShells.default = pkgs.mkShell {
          packages = [
            go
            pkgs.gopls
            pkgs.gotools
            pkgs.golangci-lint
          ];
        };
      }
    ) // {
      # --- Home Manager module (system-agnostic) ----------------------
      homeManagerModules.default = hmModule;
    };
}
