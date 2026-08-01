#!/bin/bash
#
# Increment the fork counter and print the new version.
# <upstream core>+<fork>.<counter>, e.g. 2025.09.24+grazij.3 -- the core is the
# commit date of the newest upstream commit this fork contains and moves by hand
# only. Refuses to guess if the version is not in that shape.
#
# The `version` const in main.go is the source of truth; the Makefile's
# `VERSION ?=` default has to agree with it and is rewritten in step.

set -euo pipefail

FORK="grazij"
SOURCE_FILE="main.go"
MAKEFILE="Makefile"

die() {
	printf 'bump-fork-version: %s\n' "$1" >&2
	exit 1
}

cd "$(dirname "$0")"

[ -f "$SOURCE_FILE" ] || die "no such file: $SOURCE_FILE"
[ -f "$MAKEFILE" ] || die "no such file: $MAKEFILE"

current=$(sed -n \
	's/^const version = "\(.*\)"[[:space:]]*$/\1/p' \
	"$SOURCE_FILE")
[ -n "$current" ] || die "no version const in $SOURCE_FILE"

make_current=$(sed -n \
	's/^VERSION[[:space:]]*?=[[:space:]]*\(.*[^[:space:]]\)[[:space:]]*$/\1/p' \
	"$MAKEFILE")
[ -n "$make_current" ] || die "no VERSION default in $MAKEFILE"
[ "$make_current" = "$current" ] || die \
	"$MAKEFILE says '$make_current' but $SOURCE_FILE says '$current' -- fix by hand first"

case "$current" in
	*"+$FORK."*) ;;
	*) die "version '$current' is not <core>+$FORK.<counter>" ;;
esac

core="${current%%+*}"
counter="${current##*"+$FORK."}"

case "$counter" in
	'' | *[!0-9]*) die "counter '$counter' in '$current' is not a number" ;;
esac

next="$core+$FORK.$((counter + 1))"

# rewrite in place via temp files, so a failed write cannot truncate a source
# file, and so the originals' permissions are preserved
tmp_source=$(mktemp "${TMPDIR:-/tmp}/main.go.XXXXXX")
tmp_make=$(mktemp "${TMPDIR:-/tmp}/Makefile.XXXXXX")
trap 'rm -f "$tmp_source" "$tmp_make"' EXIT

sed "s|^const version = \".*\"|const version = \"$next\"|" \
	"$SOURCE_FILE" >"$tmp_source"
grep -q "^const version = \"$next\"$" "$tmp_source" ||
	die "version const not rewritten in $SOURCE_FILE"

sed "s|^VERSION[[:space:]]*?=.*|VERSION ?= $next|" "$MAKEFILE" >"$tmp_make"
grep -q "^VERSION ?= $next$" "$tmp_make" ||
	die "VERSION default not rewritten in $MAKEFILE"

cat "$tmp_source" >"$SOURCE_FILE"
cat "$tmp_make" >"$MAKEFILE"

printf '%s\n' "$next"
