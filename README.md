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
- **Persistent filters** — `$XDG_CONFIG_HOME/plistwatch/filters` (or
  `~/.config/plistwatch/filters`), with `#` comments; the filters in force are
  announced at startup as `#` comment lines. `--config <path>` reads an
  alternate file instead of the default one.
- **Excluded domains are still announced** — a change to an excluded domain
  prints `# defaults write "<domain>"`, with no keys and no values, so a
  silenced domain is not silently missed.
- **`--quiet`/`-q`** prints only the runnable commands, and comment lines are
  dimmed when stdout is a terminal.
- **`--version`/`-v` flag** and a fork version scheme (`<upstream core>+grazij.<counter>`).
- **Integer/float/date values are emitted correctly.** Upstream splits the
  `read-type` switch into separate `case` clauses, so those types produce a
  valueless `defaults write` that fails with "Rep argument is not a dictionary"
  (upstream issue #9).
- **Values are shell-quoted properly** — an embedded apostrophe no longer
  produces a command the shell rejects.
- **Type fallback when `defaults read-type` fails**, so numbers are not rewritten
  as strings.
- **Vendored `go-plist` updated** to upstream `ee69052` (2025-03-14), with
  quoted-string escapes read the way `defaults` actually writes them (a single
  backslash before `"` or `\`), and all local patches documented in
  `go-plist/PATCHES.diff`.
- **Unit tests** (`diff_test.go`, `escape_test.go`) for the value-formatting
  helpers and `defaults`-style string escapes.
- **Distribution** — Homebrew tap (`grazij/tap`), a `Makefile`, and
  `build-macos-universal.sh` for a fat Intel + Apple Silicon binary.

## Install

### Homebrew

```
brew tap grazij/tap
brew install grazij/tap/plistwatch
```

### go install

```
go install github.com/grazij/plistwatch@latest
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
  --config file
    	read the persistent filters from this file instead of the default one
  -f, --filter domains
    	a comma-separated list of domains. Prefix names with "!" to exclude them. Supports globbing.
  -q, --quiet
    	print only the `defaults` commands: no filter banner, no excluded-domain notices
  -v, --version
    	print version and exit
```

Invalid glob patterns (e.g. an unclosed `[`) are rejected at startup with an error.

Examples:
- Hide annoying settings domains
`plistwatch --filter "!com.apple.knowledge-agent,!ContextStoreAgent"`
- Only show changes to the dock
`plistwatch -f "com.apple.dock"`
- Hide every Apple domain
`plistwatch -f "!com.apple.*"`

### Persistent filters

Filters you always want are read from `$XDG_CONFIG_HOME/plistwatch/filters`, or
`~/.config/plistwatch/filters` when `XDG_CONFIG_HOME` is unset. The file is
optional; a missing one simply means no persistent filters.

```
# ~/.config/plistwatch/filters
!com.apple.knowledge-agent   # spotlight noise
!ContextStoreAgent
```

- One filter per line, or a comma-separated list on one line, in the same syntax
  as `--filter`.
- `#` starts a comment that runs to the end of the line. Blank lines are ignored.
- The file and `--filter` **merge**: an exclusion set in the file stays in force
  during a one-off filtered run.
- An invalid glob is rejected at startup with the file and line number.

`--config <path>` reads that file **instead of** the default one, so an
alternate filter set needs no editing or moving of `~/.config/plistwatch/filters`:

```
plistwatch --config ./dock-only.filters
```

It still merges with `--filter`, and the banner names it as the source. Unlike
the default file, a `--config` path that does not exist is an error at startup:
the path was named explicitly, so a typo must not quietly watch everything.

The filters in force are announced before watching starts, one line per source.
The lines are shell comments, so the output stays pasteable as a script:

```console
$ plistwatch -f "com.apple.dock"
# plistwatch: filters from /Users/me/.config/plistwatch/filters:
#     Exclude: com.apple.knowledge-agent, contextstoreagent
#
# plistwatch: filters from the command line:
#     Include: com.apple.dock
#
```

Exclusions are listed without their `!` prefix. An empty list, a source that
contributed nothing, and an unfiltered run print nothing. Patterns are matched
case-insensitively, and the banner echoes them in the lowercased form used for
matching.

### Excluded domains are still announced

An excluded domain is silenced, not ignored. When one changes, plistwatch prints
one comment line per poll naming only the domain:

```console
# defaults write "com.apple.spaces"
defaults write "com.apple.dock" "orientation" 'left'
```

There are no keys and no values, however many of its settings moved — so a
domain excluded for being noisy stays quiet, without a change there going
unnoticed. A domain that disappears entirely prints `# defaults delete
"<domain>"`.

Only `!` exclusions are announced. Domains dropped for not matching an include
pattern are not: under `-f "com.apple.dock"` that would be most of the system.

### Quiet mode and colour

`--quiet`/`-q` prints only the runnable `defaults` commands — no banner, no
excluded-domain notices, and no colour.

Comment lines are dimmed (ANSI bright black). The runnable commands alternate
between the terminal's own foreground colour and cyan as the domain changes, so
a burst reads as groups rather than one wall of text: consecutive lines for one
domain keep their colour — across polls too — and the colour flips at a domain
boundary. A domain that comes back later may reuse either colour; only
neighbours differ.

Colour is used only when stdout is a terminal. Redirected output, `--quiet`, and
`NO_COLOR` in the environment ([no-color.org](https://no-color.org)) leave every
line plain. The commands themselves are the same bytes either way: the escapes
wrap a line, they never change it, so the output stays pasteable.

## Vendored go-plist

The `go-plist/` directory is a patched vendored copy of
[howett.net/plist](https://gitlab.howett.net/go/plist), modified to parse the
not-quite-OpenStep output of `defaults read`. The local changes and the exact
upstream base commit are documented in `go-plist/PATCHES.diff`.
