# Homebrew distribution for plistwatch

**Date:** 2026-07-28
**Status:** Historical. Implemented, then partly superseded on 2026-08-01: the
Makefile no longer touches the tap at all (no `TAP_DIR`, no tap commit/push, no
`formula-verify`). `make formula` only rewrites and pushes `Formula/plistwatch.rb`
here; copying it into `grazij/homebrew-tap` is manual, matching the `../duti` fork.

## Goal

Make plistwatch installable via `brew tap grazij/tap && brew install grazij/tap/plistwatch`,
using the same release workflow as pathset: a formula kept in-repo under `Formula/`,
published to the existing `grazij/homebrew-tap` GitHub repo by a Makefile target.

## Versioning

- Git tags use the date-based scheme `v<upstream-date>.<fork-release>` with dots only,
  e.g. `v2025.9.24.1` (first release). Homebrew parses the version from the tag.
- The internal `version` const in `main.go` (`2025.09.24-grazij/1`) is unchanged; it is
  the human-facing `--version` string. The two encode the same release.

## Components

### 1. `Formula/plistwatch.rb`

Source-build formula, modeled on pathset's:

- `desc`, `homepage "https://github.com/grazij/plistwatch"`, `license "MIT"`,
  `head` on branch `main`.
- `url` points at the tagged GitHub tarball; `sha256` computed at release time by
  `make formula`.
- `depends_on "go" => :build` and `depends_on :macos` (the tool shells out to
  `defaults`, macOS-only).
- `install`: `system "go", "build", *std_go_args(ldflags: "-s -w")` — native
  single-arch build on the user's machine (the universal binary script remains for
  non-brew distribution).
- `test`: assert `#{bin}/plistwatch --version` output contains `plistwatch`.
- Header comment documents the publish steps, as in pathset's formula.

### 2. `Makefile`

Minimal, modeled on pathset's but adapted for Go:

- `build` — `go build .` (never `./...`; see CLAUDE.md).
- `vet` — `go vet .`.
- `clean`, `install` (binary into `$(PREFIX)/bin`), `uninstall`.
- `universal` — delegates to the existing `./build-macos-universal.sh`.
- `formula` — same shell recipe as pathset's: compute sha256 of the tagged tarball,
  sed `url`/`sha256` into `Formula/plistwatch.rb`, commit + push here, copy into
  `$(TAP_DIR)/Formula/` (default `../homebrew-tap`), commit + push there.
  Variables: `VERSION ?= 2025.9.24.1`, `TAP_DIR ?= ../homebrew-tap`,
  `GITHUB_USER ?= grazij`, `GITHUB_REPO ?= plistwatch`.
- `formula-verify` — untap/retap/install/`--version`/uninstall sanity check.

No `man`/`help2man` machinery (plistwatch has no man page) and no `test` target
(the root package has no tests; `vet` covers linting).

### 3. Docs

- `README.md`: add a Homebrew install section (`brew tap grazij/tap`,
  `brew install grazij/tap/plistwatch`) and a short release-process note.
- `CLAUDE.md`: document the Makefile targets, the tag scheme and its relation to the
  `version` const, and the tap publish flow.

## Release flow (first release)

1. Clone `git@github.com:grazij/homebrew-tap.git` to `../homebrew-tap` (exists on
   GitHub, not yet cloned locally).
2. Commit the new files; tag `v2025.9.24.1`; push commits and tag to `origin`.
3. `make formula` — fills in url/sha256 and publishes to the tap.
4. `make formula-verify` — end-to-end install check.

## Error handling

- `make formula` fails fast (set -e) if the tap dir is missing or the tarball sha
  comes back empty (tag not pushed) — same guards as pathset.
- The formula itself has no runtime error handling beyond Homebrew's `test do` block.

## Out of scope

- homebrew-core submission, bottles/prebuilt binaries, man page, changes to the
  `--version` string or the poller itself.
