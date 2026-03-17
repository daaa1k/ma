class Ma < Formula
  desc "TODO: describe your app"
  homepage "https://github.com/daaa1k/ma"
  version "0.1.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/daaa1k/ma/releases/download/v#{version}/ma-macos-aarch64"
      sha256 "f9d4c0e66b81c71299c1289b22877c7829f596ae803a937c084dc815400390e0" # macos-aarch64

      def install
        bin.install "ma-macos-aarch64" => "ma"
      end
    else
      url "https://github.com/daaa1k/ma/releases/download/v#{version}/ma-macos-x86_64"
      sha256 "" # macos-x86_64

      def install
        bin.install "ma-macos-x86_64" => "ma"
      end
    end
  end

  on_linux do
    url "https://github.com/daaa1k/ma/releases/download/v#{version}/ma-linux-x86_64"
    sha256 "32a3f7fe256594abedeb8b772f32e2a82e30d5286c1dfbc09efc70e980447523" # linux-x86_64

    def install
      bin.install "ma-linux-x86_64" => "ma"
    end
  end

  test do
    system "#{bin}/ma", "--help"
  end
end
