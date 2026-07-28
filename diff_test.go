package main

import (
	"testing"
	"time"
)

// Regression tests for https://github.com/catilac/plistwatch/issues/9: integer-
// and float-typed preferences used to emit a bare `defaults write "dom" "key"`
// with no type flag and no value, because the switch in valueArg relied on
// C-style fallthrough that Go does not perform.
func TestValueArg(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		s    string
		want string
	}{
		{"boolean true", "boolean", "1", "-bool true"},
		{"boolean false", "boolean", "0", "-bool false"},

		// issue #9: com.barebones.bbedit ExtraSpaceInTextViews
		{"integer", "integer", "2", "-integer 2"},
		// hot corners: com.apple.dock wvous-br-corner
		{"integer hot corner", "integer", "14", "-integer 14"},
		{"integer negative", "integer", "-1", "-integer -1"},
		{"float", "float", "1.5", "-float 1.5"},
		{"date", "date", `"1970-01-01 00:00:00 +0000"`, `-date "1970-01-01 00:00:00 +0000"`},

		// strings, arrays, dicts and data are passed through as quoted
		// OpenStep text; `defaults` parses them back to the right type.
		{"string", "string", "plain", "'plain'"},
		{"string with space", "string", `"has space"`, `'"has space"'`},
		{"array", "array", "(a,1,)", "'(a,1,)'"},
		{"dictionary", "dictionary", "{k=v;}", "'{k=v;}'"},
		{"data", "data", "<dead>", "'<dead>'"},

		// A value containing a single quote must not break out of the quoting.
		{"string with single quote", "string", "don't", `'don'\''t'`},

		// Unknown or missing type falls back to quoting.
		{"unknown type", "wat", "x", "'x'"},
		{"empty type", "", "x", "'x'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := valueArg(tt.typ, tt.s); got != tt.want {
				t.Errorf("valueArg(%q, %q) = %q, want %q", tt.typ, tt.s, got, tt.want)
			}
		})
	}
}

// valueArg must never return an empty string: an empty value argument produces
// `defaults write "dom" "key"`, which fails with "Rep argument is not a
// dictionary" instead of applying the change.
func TestValueArgNeverEmpty(t *testing.T) {
	for _, typ := range []string{"boolean", "integer", "float", "date", "string", "array", "dictionary", "data", "", "bogus"} {
		if got := valueArg(typ, "1"); got == "" {
			t.Errorf("valueArg(%q, \"1\") returned an empty argument", typ)
		}
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{"plain", "plain", "'plain'"},
		{"empty", "", "''"},
		{"space", "has space", "'has space'"},
		{"double quote", `"dq"`, `'"dq"'`},
		{"single quote", "don't", `'don'\''t'`},
		{"leading single quote", "'x", `''\''x'`},
		{"only a single quote", "'", `''\'''`},
		{"two single quotes", "a'b'c", `'a'\''b'\''c'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shellQuote(tt.s); got != tt.want {
				t.Errorf("shellQuote(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

// When `defaults read-type` fails we still know the type from the parsed plist
// value, so integers must not silently degrade into strings.
func TestFallbackType(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		want string
	}{
		{"bool", true, "boolean"},
		{"uint64", uint64(14), "integer"},
		{"int64", int64(14), "integer"},
		{"int", 14, "integer"},
		{"float64", 1.5, "float"},
		{"time", time.Unix(0, 0).UTC(), "date"},

		// Types the default (quoted OpenStep) branch already handles correctly.
		{"string", "s", ""},
		{"array", []interface{}{"a"}, ""},
		{"dictionary", map[string]interface{}{"k": "v"}, ""},
		{"data", []byte{0xde}, ""},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fallbackType(tt.v); got != tt.want {
				t.Errorf("fallbackType(%T) = %q, want %q", tt.v, got, tt.want)
			}
		})
	}
}
