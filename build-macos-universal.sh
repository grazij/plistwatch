#!/bin/bash
# Build a universal (x86_64 + arm64) macOS binary at ./plistwatch.
# Requires the Go toolchain and lipo (Xcode command line tools).
set -euo pipefail

cd "$(dirname "$0")"

tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/plistwatch-build.XXXXXX")"
trap 'rm -rf "$tmpdir"' EXIT

GOOS=darwin GOARCH=amd64 go build -o "$tmpdir/plistwatch_amd64" .
GOOS=darwin GOARCH=arm64 go build -o "$tmpdir/plistwatch_arm64" .
lipo -create -output plistwatch "$tmpdir/plistwatch_amd64" "$tmpdir/plistwatch_arm64"
lipo -info plistwatch
