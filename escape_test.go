package main

import (
	"testing"

	"github.com/grazij/plistwatch/go-plist"
)

// `defaults read` escapes quoted strings the ordinary OpenStep way: one
// backslash before a literal `"` or `\`, and `\n`/`\t`/`\Uxxxx` for control
// and non-ASCII characters. The vendored parser used to assume `defaults`
// emitted an extra backslash before everything but `\\`, which silently
// corrupted every value containing a quote and desynchronised the parser
// completely when a quote flipped the string parity for the rest of the file.
func TestParseDefaultsEscapes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"escaped quote", `"\"lastpass.com\""`, `"lastpass.com"`},
		{"embedded json", `"[{\"enabled\":true}]"`, `[{"enabled":true}]`},
		{"escaped backslash", `"/\\.(git|hg|svn)"`, `/\.(git|hg|svn)`},
		{"backslash then quote", `"a\\\"b"`, `a\"b`},
		{"newline", `"a\nb"`, "a\nb"},
		{"tab", `"a\tb"`, "a\tb"},
		{"unicode", `"caf\U00e9"`, "café"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got map[string]interface{}
			if _, err := plist.Unmarshal([]byte("{ k = "+tt.in+"; }"), &got); err != nil {
				t.Fatalf("Unmarshal(%s) failed: %v", tt.in, err)
			}
			if got["k"] != tt.want {
				t.Errorf("Unmarshal(%s) = %q, want %q", tt.in, got["k"], tt.want)
			}
		})
	}
}

// An escaped quote inside an array element used to leave the parser one quote
// out of phase, so parsing failed far away from the offending value.
func TestParseDefaultsEscapedQuoteKeepsArrayInSync(t *testing.T) {
	const in = `{ k = ( "a.com", "\"lastpass.com\"", "b.com" ); }`

	var got map[string]interface{}
	if _, err := plist.Unmarshal([]byte(in), &got); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	arr, ok := got["k"].([]interface{})
	if !ok {
		t.Fatalf("k is %T, want []interface{}", got["k"])
	}
	want := []string{"a.com", `"lastpass.com"`, "b.com"}
	if len(arr) != len(want) {
		t.Fatalf("got %d elements (%q), want %d", len(arr), arr, len(want))
	}
	for i, w := range want {
		if arr[i] != w {
			t.Errorf("element %d = %q, want %q", i, arr[i], w)
		}
	}
}
