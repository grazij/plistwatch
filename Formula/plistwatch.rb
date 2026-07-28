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
#   curl -sL https://github.com/grazij/plistwatch/archive/refs/tags/v2025.9.24.2.tar.gz | shasum -a 256
#
# To test edits to this file before publishing: current Homebrew rejects any
# formula outside a tap ("Homebrew requires formulae to be in a tap"), so both
# `brew style` and `brew install` on this path fail. Work through the tapped
# clone instead — note that is a separate checkout from ../homebrew-tap, and
# `brew update` will reset it:
#   cp Formula/plistwatch.rb "$(brew --repository grazij/tap)/Formula/"
#   brew style --formula grazij/tap/plistwatch
#   brew install --build-from-source grazij/tap/plistwatch
#   git -C "$(brew --repository grazij/tap)" checkout -- Formula/plistwatch.rb

class Plistwatch < Formula
  desc "Watch macOS defaults and print the commands that recreate each change"
  homepage "https://github.com/grazij/plistwatch"
  url "https://github.com/grazij/plistwatch/archive/refs/tags/v2025.9.24.2.tar.gz"
  sha256 "6ab5debdf688583d0635b7afe882b449e29775dba7f9e4143ad0f6a541891bbd"
  license "MIT"
  head "https://github.com/grazij/plistwatch.git", branch: "main"

  depends_on "go" => :build
  depends_on :macos

  def install
    # std_go_args already adds `-s -w` itself (and drops them for
    # --debug-symbols builds), so don't pass them again.
    system "go", "build", *std_go_args
  end

  test do
    assert_match "plistwatch", shell_output("#{bin}/plistwatch --version")
  end
end
