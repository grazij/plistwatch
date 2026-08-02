# Homebrew formula for plistwatch.
#
# To publish:
#   1. Tag a release (e.g. v2025.09.24+grazij.3) and push to GitHub.
#   2. Run `make formula VERSION=2025.09.24+grazij.3` from the project root — it
#      bumps `url`, `version` and `sha256` here, mirrors the file to
#      ../homebrew-tap, and pushes both repos. Users then run:
#        brew tap grazij/tap
#        brew install grazij/tap/plistwatch
#
# To compute SHA256 for the release tarball (a literal + in the path confuses
# GitHub's redirects; %2B does not):
#   curl -sL https://github.com/grazij/plistwatch/archive/refs/tags/v2025.09.24%2Bgrazij.3.tar.gz | shasum -a 256
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
  url "https://github.com/grazij/plistwatch/archive/refs/tags/v2025.09.24%2Bgrazij.4.tar.gz"
  version "2025.09.24+grazij.4"
  sha256 "0fbb0d6cb82662c5214037c09cb2c838f7efbe2d83efe998fe14854e62546dce"
  license "MIT"
  head "https://github.com/grazij/plistwatch.git", branch: "main"

  # Version.detect reads a "+grazij.N" tarball name as "1", so `version` above
  # is pinned by hand and livecheck needs an explicit regex.
  livecheck do
    url :stable
    strategy :github_latest
    regex(/v?(\d+(?:\.\d+)+\+grazij\.\d+)/i)
  end

  depends_on "go" => :build
  depends_on :macos

  def install
    # std_go_args already adds `-s -w` itself (and drops them for
    # --debug-symbols builds), so don't pass them again.
    system "go", "build", *std_go_args
  end

  test do
    assert_equal "plistwatch #{version}", shell_output("#{bin}/plistwatch --version").strip
  end
end
