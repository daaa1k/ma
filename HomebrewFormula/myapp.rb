class Myapp < Formula
  desc "TODO: describe your app"
  homepage "https://github.com/daaa1k/myapp"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/daaa1k/myapp/releases/download/v#{version}/myapp-macos-aarch64"
      sha256 "" # macos-aarch64

      def install
        bin.install "myapp-macos-aarch64" => "myapp"
      end
    else
      url "https://github.com/daaa1k/myapp/releases/download/v#{version}/myapp-macos-x86_64"
      sha256 "" # macos-x86_64

      def install
        bin.install "myapp-macos-x86_64" => "myapp"
      end
    end
  end

  on_linux do
    url "https://github.com/daaa1k/myapp/releases/download/v#{version}/myapp-linux-x86_64"
    sha256 "" # linux-x86_64

    def install
      bin.install "myapp-linux-x86_64" => "myapp"
    end
  end

  test do
    system "#{bin}/myapp", "--help"
  end
end
