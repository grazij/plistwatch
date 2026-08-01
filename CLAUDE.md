# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

PlistWatch is a small macOS-only Go CLI that polls `defaults read` every second, diffs the result against the previous snapshot, and prints the `defaults write`/`defaults delete` commands that would recreate each change. Module path: `github.com/grazij/plistwatch` (renamed from upstream's `github.com/catilac/plistwatch` so `go install` targets this fork; upstream syncs will conflict on `go.mod` and the two `go-plist` import lines).

## Commands

```sh
go build .                  # build the tool (go-plist builds as part of it)
go run .                    # run the watcher (must be run on macOS; shells out to `defaults`)
go vet .                    # lint
go test .                   # run the root-package unit tests (never `./...`)
./build-macos-universal.sh  # build ./plistwatch as a universal (x86_64 + arm64) binary via lipo
make build                  # same as go build .
make universal              # same as ./build-macos-universal.sh
make formula VERSION=X.Y.Z  # publish Formula/plistwatch.rb to the grazij/homebrew-tap repo (tag vX.Y.Z must be pushed)
make formula-verify         # end-to-end tap install sanity check
```

Do not use `./...`: the vendored `go-plist/cmd/` tools and one example test import `howett.net/plist` and third-party modules that are not in `go.mod`, so `go build ./...` and `go test ./go-plist` fail (test setup error) — this predates all local work and is not worth fixing in third-party code.

`diff_test.go` covers the pure helpers in `diff.go` (`valueArg`, `shellQuote`, `fallbackType`) and needs neither the `defaults` binary nor the host's real preferences, so `go test .` runs anywhere. Everything else — the poll loop and `Diff`'s use of `defaults read-type` — can only be exercised on macOS.

## Architecture

Two Go files make up the entire tool:

- `main.go` — flag parsing (`--filter`/`-f`, comma-separated domain globs, `!` prefix excludes, matched case-insensitively via `filepath.Match`; malformed glob patterns are rejected at parse time; `--version`/`-v` prints the version and exits; short/long forms are separate stdlib-flag registrations sharing one variable/closure), the 1-second poll loop, and `filterDomains` which prunes the top-level domain map before diffing. The `version` const uses the scheme `<upstream-date>-grazij/<N>` — commit date of the newest upstream (catilac/plistwatch master) commit this fork contains, plus a fork release number; bump the date on upstream syncs, increment `<N>` for fork-only releases.
- `diff.go` — `Diff(prev, curr)` walks the two-level structure (domain → keys) and prints `defaults` commands for added/changed/deleted domains and keys. It shells out to `defaults read-type` to decide the value flag (`-bool`, `-integer`, etc.) and uses `cmp` for deep equality of plist values. Values are serialized back to OpenStep format via the plist library's `plist.Marshal(v, plist.OpenStepFormat)`. Three pure helpers do the formatting:
  - `valueArg(typ, s)` — maps a `read-type` name plus marshaled value to the command's value argument. **`case "integer", "float", "date":` must stay one clause**: Go has no implicit fallthrough, and splitting it (as upstream did) leaves the value empty, emitting a valueless `defaults write` that fails with "Rep argument is not a dictionary" — upstream issue #9.
  - `shellQuote(s)` — single-quotes a value, escaping embedded `'` as `'\''`. Any value containing an apostrophe would otherwise produce a command the shell rejects.
  - `fallbackType(v)` — derives the type name from the parsed value's Go type (`uint64`→integer, `float64`→float, `bool`→boolean, `time.Time`→date) when `defaults read-type` fails; without it numbers get rewritten as strings. Returns `""` for types the quoted-OpenStep branch already handles.

  `defaults` parses quoted OpenStep text back to the right type, so strings, arrays, dicts and data need no type flag — only the scalar flags above are emitted explicitly.

`go-plist/` is a vendored, in-tree copy of `howett.net/plist` (imported as `github.com/grazij/plistwatch/go-plist`), used for both parsing `defaults read` output (which is OpenStep/GNUStep text format) and re-serializing values. It is third-party code (BSD-licensed, see `go-plist/LICENSE`); the leftover `howett.net/plist` entries in `go.sum` are unused.

### Vendoring provenance

The vendored copy is upstream commit `ee69052` (2025-03-14) plus local patches that make it parse the not-quite-OpenStep output of `defaults read` — see `go-plist/PATCHES.diff` for the full diff, its header for the patch summary, and the regeneration command. Don't edit `go-plist/` beyond those patches; if you must, regenerate `PATCHES.diff` afterwards.

Remotes: upstream lives at https://gitlab.howett.net/go/plist (private GitLab, pull via HTTPS); we keep a mirror at `git@github.com:grazij/go-plist.git` (branch `main`). A local clone with both remotes (`upstream` = GitLab, `origin` = GitHub) is expected at `../plist`.

To upgrade: fetch `upstream` in `../plist` and review changes since the base commit; copy the new tree over `go-plist/` (minus CI configs, `go.mod`, `go.sum`); re-apply `PATCHES.diff` and resolve conflicts; regenerate `PATCHES.diff` against the new base and update the base commit recorded here; `go build .`; push the new upstream state to `origin main`.

## Releasing (Homebrew)

`Formula/plistwatch.rb` is the Homebrew formula; it installs from the tagged GitHub
tarball and is mirrored into `git@github.com:grazij/homebrew-tap.git` (expected
cloned at `../homebrew-tap`) by `make formula`. Release tags use the `version`
const's scheme with dots only: `2025.09.24-grazij/1` ⇒ tag `v2025.9.24.1`; Homebrew
derives its version from the tag. Flow: bump the `version` const if needed, commit,
`git tag v<X> && git push origin main --tags`, then `make formula VERSION=<X>` and
sanity-check with `make formula-verify`. The formula's `url`/`sha256` lines are
rewritten by sed — keep their exact formatting. Also bump the Makefile's
`VERSION ?=` default so a bare `make formula` targets the new release.

Gotchas hit in practice:

- **`make formula` never pulls the tap.** If `../homebrew-tap` is behind its remote
  the push fails after the local commits are already made; `git -C ../homebrew-tap
  pull --rebase origin main` and push again.
- **Testing formula edits needs a tap.** Current Homebrew rejects any formula
  outside one, so `brew style --formula Formula/plistwatch.rb` and
  `brew install --build-from-source ./Formula/plistwatch.rb` both fail with
  "Homebrew requires formulae to be in a tap". Copy the file into
  `$(brew --repository grazij/tap)/Formula/` and lint/build via
  `grazij/tap/plistwatch`, then `git checkout` it there. That tapped clone is a
  *third* checkout — separate from both this repo and `../homebrew-tap` — and
  `brew update` resets it.
- **Don't pass `ldflags: "-s -w"` to `std_go_args`.** It already prepends `-s -w`,
  so passing them again yields `-ldflags=-s -w -s -w` and overrides the
  `--debug-symbols` opt-out. Plain `*std_go_args` is correct.
- **Homebrew's `==> go build` line censors args containing the Cellar path**, so it
  under-reports the real command. `~/Library/Logs/Homebrew/plistwatch/01.go.log`
  has the actual argv.
- Formula-only changes (anything not affecting `url`/`sha256`) need no new tag —
  users get them on their next `brew update`.

## Data model conventions

Both diff sides are `map[string]interface{}` where the top level maps domain names to `map[string]interface{}` of keys — `Diff` type-asserts this unconditionally, so any change feeding it non-map domain values will panic. Domain matching in filters lowercases both the pattern and the domain name.
