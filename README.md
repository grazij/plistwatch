# PlistWatch

## About
PlistWatch monitors real-time changes to plist files on your system.
It outputs a `defaults` command to recreate that change.

## Install
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
```

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
