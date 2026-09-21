#!/bin/bash
#
# Increment the fork counter and print the new version.
# <upstream core>+<fork>.<counter>, e.g. 2025.09.24+grazij.3 -- the core is the
# commit date of the newest upstream commit this fork contains and moves by hand
# only. Refuses to guess if the version is not in that shape.
#
# The `version` const in main.go is the only place the version lives. The
# Makefile used to carry a VERSION default for the `formula` target; both went
# away in 59fb9ba, and this script went on requiring it until 2026-09-21.

set -euo pipefail

FORK="grazij"
SOURCE_FILE="main.go"

die() {
	printf 'bump-fork-version: %s\n' "$1" >&2
	exit 1
}

cd "$(dirname "$0")/.."

[ -f "$SOURCE_FILE" ] || die "no such file: $SOURCE_FILE"

current=$(sed -n \
	's/^const version = "\(.*\)"[[:space:]]*$/\1/p' \
	"$SOURCE_FILE")
[ -n "$current" ] || die "no version const in $SOURCE_FILE"

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

# rewrite in place via a temp file, so a failed write cannot truncate the
# source file, and so its permissions are preserved
tmp_source=$(mktemp "${TMPDIR:-/tmp}/main.go.XXXXXX")
trap 'rm -f "$tmp_source"' EXIT

sed "s|^const version = \".*\"|const version = \"$next\"|" \
	"$SOURCE_FILE" >"$tmp_source"
grep -q "^const version = \"$next\"$" "$tmp_source" ||
	die "version const not rewritten in $SOURCE_FILE"

cat "$tmp_source" >"$SOURCE_FILE"

printf '%s\n' "$next"
