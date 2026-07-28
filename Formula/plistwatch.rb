# Homebrew formula for plistwatch.
#
# To publish:
#   1. Tag a release (e.g. v2025.9.24.1) and push to GitHub.
#   2. Run `make formula VERSION=2025.9.24.1` from the project root — it bumps
#      `url` and `sha256` here, mirrors the file to ../homebrew-tap, and
#      pushes both repos. Users then run:
#        brew tap grazij/tap
#        brew install grazij/tap/plistwatch
#
# To compute SHA256 for the release tarball:
#   curl -sL https://github.com/grazij/plistwatch/archive/refs/tags/v2025.9.24.1.tar.gz | shasum -a 256
#
# To install locally for testing without a tap:
#   brew install --build-from-source ./Formula/plistwatch.rb

class Plistwatch < Formula
  desc "Watch macOS defaults and print the commands that recreate each change"
  homepage "https://github.com/grazij/plistwatch"
  url "https://github.com/grazij/plistwatch/archive/refs/tags/v2025.9.24.1.tar.gz"
  sha256 "2feb541dcd9fa8f6a3a24c55162e1381c44a0d857b3aad055e9fcbcd3c4bfdb0"
  license "MIT"
  head "https://github.com/grazij/plistwatch.git", branch: "main"

  depends_on "go" => :build
  depends_on :macos

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w")
  end

  test do
    assert_match "plistwatch", shell_output("#{bin}/plistwatch --version")
  end
end
