class Ma < Formula
  desc "TODO: describe your app"
  homepage "https://github.com/daaa1k/ma"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/daaa1k/ma/releases/download/v#{version}/ma-macos-aarch64"
      sha256 "" # macos-aarch64

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
    sha256 "" # linux-x86_64

    def install
      bin.install "ma-linux-x86_64" => "ma"
    end
  end

  test do
    system "#{bin}/ma", "--help"
  end
end
