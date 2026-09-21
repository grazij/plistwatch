# PlistWatch

Watch plist changes on macOS in real time. Each change prints as a runnable
`defaults` command that reapplies it, with its original type preserved.

```
defaults write "com.apple.dock" "orientation" 'left'
defaults write "com.apple.dock" "wvous-br-corner" -integer 14
```

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

### Prebuilt release

Download a universal binary from
[Releases](https://github.com/grazij/plistwatch/releases).

The binary is not code-signed or notarized. Clear the macOS quarantine flag
before use:

```
xattr -dr com.apple.quarantine plistwatch
```

### From source

```
make install     # into ~/.local/bin; override with PREFIX=/usr/local
make universal   # fat x86_64 + arm64 binary at ./plistwatch
```

## Usage

Run `plistwatch`, then change something — move the Dock, toggle a setting — and
the commands appear.

| Flag | Meaning |
| --- | --- |
| `-f`, `--filter <domains>` | comma-separated domains; `!` excludes, globs supported |
| `--config <file>` | read persistent filters from this file instead of the default |
| `-q`, `--quiet` | print only the `defaults` commands |
| `-v`, `--version` | print version and exit |

## Filtering

Matching is case-insensitive, and an invalid glob is rejected at startup.

```
plistwatch -f "com.apple.dock"                                  # only the Dock
plistwatch -f "!com.apple.*"                                    # hide Apple domains
plistwatch --filter "!com.apple.knowledge-agent,!ContextStoreAgent"
```

Filters you always want go in `$XDG_CONFIG_HOME/plistwatch/filters`, or
`~/.config/plistwatch/filters`. One filter per line or a comma-separated list;
`#` starts a comment; blank lines are ignored. The file is optional.

```
# ~/.config/plistwatch/filters
!com.apple.knowledge-agent   # spotlight noise
!ContextStoreAgent
```

The file and `--filter` merge. `--config <path>` reads that file instead of the
default one; unlike the default, a `--config` path that does not exist is an
error, so a typo cannot quietly watch everything.

The filters in use are displayed at startup as `#` comments, so the output
stays pasteable as a script:

```console
$ plistwatch -f "com.apple.dock"
# plistwatch: filters from /Users/me/.config/plistwatch/filters:
#     Exclude: com.apple.knowledge-agent, contextstoreagent
#
# plistwatch: filters from the command line:
#     Include: com.apple.dock
#
```

### Excluded domains are still displayed

A `!`-excluded domain only prints its name and nothing else, 
so a verbose domain stays quiet without going unnoticed:

```console
# defaults write "com.apple.spaces"
defaults write "com.apple.dock" "orientation" 'left'
```

A domain that is deleted prints `# defaults delete "<domain>"`. Domains dropped
for not matching an include pattern are not announced.

### Quiet mode and color

`--quiet`/`-q` prints only the runnable commands: no banner, no notices, no
color.

Otherwise comment lines are dimmed and the commands alternate between the
terminal's foreground and cyan as the domain changes, so a burst reads as
groups. Color appears only when stdout is a terminal, and `NO_COLOR`
([no-color.org](https://no-color.org)) disables it. The escapes wrap lines
without changing them, so the output stays pasteable either way.

## Differences from upstream

A fork of [catilac/plistwatch](https://github.com/catilac/plistwatch) containing
all of upstream `master` (`cd0de73`, 2025-09-24) plus:

- **Domain filtering** — `--filter`/`-f` with globs and `!` exclusions, from an
  unmerged upstream PR; invalid globs are rejected at startup.
- **Persistent filters**, with `--config` to point elsewhere.
- **Excluded domains are still announced**, by name only.
- **`--quiet`/`-q`**, dimmed comments, and alternating command color.
- **`--version`/`-v`** and a fork version scheme (`<upstream core>+grazij.<n>`).
- **Integer, float and date values are emitted correctly.** Upstream produces a
  valueless `defaults write` for those types, which fails with "Rep argument is
  not a dictionary" (upstream issue #9).
- **Values are shell-quoted properly** — an embedded apostrophe no longer
  breaks the command.
- **Type fallback when `defaults read-type` fails**, so numbers are not
  rewritten as strings.
- **Vendored `go-plist` updated** to `ee69052` (2025-03-14), reading
  quoted-string escapes the way `defaults` writes them.
- **Unit tests** for value formatting and `defaults`-style escapes.
- **Distribution** — Homebrew tap, tagged releases with a universal binary, and
  a `Makefile`.

## Vendored go-plist

`go-plist/` is a patched copy of
[howett.net/plist](https://gitlab.howett.net/go/plist) that parses the
not-quite-OpenStep output of `defaults read`. The local changes and the upstream
base commit are in `go-plist/PATCHES.diff`.
