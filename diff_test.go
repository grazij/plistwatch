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

// Excluded domains are still watched, but only enough to say that something
// moved: one comment line per domain per poll, with no keys and no values.
func TestExcludedChanges(t *testing.T) {
	dom := func(v interface{}) map[string]interface{} {
		return map[string]interface{}{"key": v}
	}

	tests := []struct {
		name string
		prev map[string]interface{}
		curr map[string]interface{}
		want []string
	}{
		{
			name: "unchanged domain is silent",
			prev: map[string]interface{}{"a": dom("x")},
			curr: map[string]interface{}{"a": dom("x")},
		},
		{
			name: "changed value",
			prev: map[string]interface{}{"a": dom("x")},
			curr: map[string]interface{}{"a": dom("y")},
			want: []string{`# defaults write "a"`},
		},
		{
			// `defaults read` renders every scalar as a string today, so
			// these two cases are about not depending on that: cmp()
			// would report both pairs equal.
			name: "changed integer value",
			prev: map[string]interface{}{"a": dom(uint64(1))},
			curr: map[string]interface{}{"a": dom(uint64(2))},
			want: []string{`# defaults write "a"`},
		},
		{
			name: "changed boolean value",
			prev: map[string]interface{}{"a": dom(true)},
			curr: map[string]interface{}{"a": dom(false)},
			want: []string{`# defaults write "a"`},
		},
		{
			name: "added key",
			prev: map[string]interface{}{"a": map[string]interface{}{}},
			curr: map[string]interface{}{"a": dom("x")},
			want: []string{`# defaults write "a"`},
		},
		{
			name: "new domain",
			curr: map[string]interface{}{"a": dom("x")},
			want: []string{`# defaults write "a"`},
		},
		{
			name: "removed domain",
			prev: map[string]interface{}{"a": dom("x")},
			want: []string{`# defaults delete "a"`},
		},
		{
			name: "one line per domain, sorted, however many keys moved",
			prev: map[string]interface{}{
				"b": map[string]interface{}{"k1": "x", "k2": "x"},
				"a": dom("x"),
			},
			curr: map[string]interface{}{
				"b": map[string]interface{}{"k1": "y", "k2": "y"},
				"a": dom("y"),
			},
			want: []string{`# defaults write "a"`, `# defaults write "b"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := excludedChanges(tt.prev, tt.curr)
			if len(got) != len(tt.want) {
				t.Fatalf("excludedChanges() = %q, want %q", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("excludedChanges() = %q, want %q", got, tt.want)
				}
			}
		})
	}
}

// Comment output is dimmed so it reads as secondary next to the runnable
// commands, but only when a terminal is there to render it: redirected output
// must stay pasteable.
func TestDim(t *testing.T) {
	t.Cleanup(func() { colorOutput = false })

	colorOutput = false
	if got := dim("# one\n# two\n"); got != "# one\n# two\n" {
		t.Errorf("dim() without color = %q, want it unchanged", got)
	}

	colorOutput = true
	want := "\x1b[90m# one\x1b[0m\n\x1b[90m# two\x1b[0m\n"
	if got := dim("# one\n# two\n"); got != want {
		t.Errorf("dim() = %q, want %q", got, want)
	}
	if got := dim(""); got != "" {
		t.Errorf("dim(\"\") = %q, want empty", got)
	}
}

// Runnable commands alternate between the terminal's own foreground and cyan as
// the domain changes. The color holds for consecutive lines of one domain and
// flips at the boundary; a returning domain may reuse either.
func TestDomainColorerLine(t *testing.T) {
	t.Cleanup(func() { colorOutput = false })
	colorOutput = true

	tests := []struct {
		name    string
		domains []string
		// alt[i] says whether line i is the cyan one.
		alt []bool
	}{
		{
			name:    "consecutive lines of one domain share a color",
			domains: []string{"a", "a", "a"},
			alt:     []bool{true, true, true},
		},
		{
			name:    "the color flips at a domain boundary",
			domains: []string{"a", "b", "b", "c"},
			alt:     []bool{true, false, false, true},
		},
		{
			// Only neighbors have to differ: "a" is cyan both times,
			// and need not have been.
			name:    "a returning domain may reuse either color",
			domains: []string{"a", "b", "a"},
			alt:     []bool{true, false, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c domainColorer
			for i, domain := range tt.domains {
				const cmd = `defaults write "x" "k" 'v'`
				want := cmd
				if tt.alt[i] {
					want = "\x1b[36m" + cmd + "\x1b[0m"
				}
				if got := c.line(domain, cmd); got != want {
					t.Errorf("line %d (domain %q) = %q, want %q", i, domain, got, want)
				}
			}
		})
	}
}

// The alternation shares the gate with the dimmed comments: with color off the
// commands are byte-identical, so redirected output stays pasteable.
func TestDomainColorerLineWithoutColor(t *testing.T) {
	t.Cleanup(func() { colorOutput = false })
	colorOutput = false

	var c domainColorer
	const cmd = `defaults write "x" "k" 'v'`
	for _, domain := range []string{"a", "a", "b", "a"} {
		if got := c.line(domain, cmd); got != cmd {
			t.Errorf("line(%q) without color = %q, want it unchanged", domain, got)
		}
	}
}
