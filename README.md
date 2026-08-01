# PlistWatch

## About
PlistWatch monitors real-time changes to plist files on your system.
It outputs a `defaults` command to recreate that change.

## Differences from upstream

This is a fork of [catilac/plistwatch](https://github.com/catilac/plistwatch).
It contains everything in upstream `master` (as of `cd0de73`, 2025-09-24) plus:

- **Domain filtering** — `--filter`/`-f` with globs, `!` exclusions and
  case-insensitive matching (from an upstream PR that was never merged there).
- **Invalid glob patterns are rejected at startup** instead of silently never matching.
- **`--version`/`-v` flag** and a fork version scheme (`<upstream-date>-grazij/<N>`).
- **Integer/float/date values are emitted correctly.** Upstream splits the
  `read-type` switch into separate `case` clauses, so those types produce a
  valueless `defaults write` that fails with "Rep argument is not a dictionary"
  (upstream issue #9).
- **Values are shell-quoted properly** — an embedded apostrophe no longer
  produces a command the shell rejects.
- **Type fallback when `defaults read-type` fails**, so numbers are not rewritten
  as strings.
- **Vendored `go-plist` updated** to upstream `ee69052` (2025-03-14), with a
  trailing-backslash parser panic fixed and all local patches documented in
  `go-plist/PATCHES.diff`.
- **Unit tests** (`diff_test.go`) for the value-formatting helpers.
- **Distribution** — Homebrew tap (`grazij/tap`), a `Makefile`, and
  `build-macos-universal.sh` for a fat Intel + Apple Silicon binary.

Note: the module path is still `github.com/catilac/plistwatch`, so the
`go install` command below installs **upstream**, not this fork. Use Homebrew or
build from a clone to get the fork.

## Install

### Homebrew

```
brew tap grazij/tap
brew install grazij/tap/plistwatch
```

### go install

```
go install  github.com/catilac/plistwatch@latest
```

### Universal binary (Intel + Apple Silicon)

To build a fat binary that runs natively on both architectures:
```
./build-macos-universal.sh
```
This produces `./plistwatch` containing x86_64 and arm64 slices (verify with
`lipo -info plistwatch`). Requires the Go toolchain and Xcode command line
tools (for `lipo`).

## Usage
Just run:
```
plistwatch 
```

Now make some changes, such as moving the Dock and moving it back by clicking the *Position of Screen* options. 
You should see the changes being reported. 
You may also see other events being reported.

And you should see output such as:
```
defaults write "com.apple.dock" "orientation" 'left'
defaults write "com.apple.dock" "wvous-br-corner" -integer 14
```

Each line is a complete, runnable command: re-running it reapplies the change
with its original type preserved.

The output can also be filtered:
```
Usage of plistwatch:
  -f, --filter domains
    	a comma-separated list of domains. Prefix names with "!" to exclude them. Supports globbing.
  -v, --version
    	print version and exit
```

Invalid glob patterns (e.g. an unclosed `[`) are rejected at startup with an error.

`plistwatch --version` prints the version and exits. The scheme is
`<upstream-date>-grazij/<N>`: the commit date of the newest
[catilac/plistwatch](https://github.com/catilac/plistwatch) commit this fork
contains, plus a fork release number.
Release git tags encode the same version with dots only (e.g. `v2025.9.24.1`),
which is the version Homebrew reports.

Examples:
- Hide annoying settings domains
`plistwatch --filter "!com.apple.knowledge-agent,!ContextStoreAgent"`
- Only show changes to the dock
`plistwatch -f "com.apple.dock"`
- Hide every Apple domain
`plistwatch -f "!com.apple.*"`

## Vendored go-plist

The `go-plist/` directory is a patched vendored copy of
[howett.net/plist](https://gitlab.howett.net/go/plist), modified to parse the
not-quite-OpenStep output of `defaults read`. The local changes and the exact
upstream base commit are documented in `go-plist/PATCHES.diff`.
