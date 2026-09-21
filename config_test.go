package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func writeFilters(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "filters")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadFilters(t *testing.T) {
	tests := []struct {
		name        string
		contents    string
		wantInclude []string
		wantExclude []string
	}{
		{
			name:        "one domain per line",
			contents:    "com.apple.dock\n!com.apple.knowledge-agent\n",
			wantInclude: []string{"com.apple.dock"},
			wantExclude: []string{"com.apple.knowledge-agent"},
		},
		{
			name:        "full-line comment",
			contents:    "# noise from Spotlight\n!com.apple.knowledge-agent\n",
			wantExclude: []string{"com.apple.knowledge-agent"},
		},
		{
			name:        "trailing comment",
			contents:    "!com.apple.knowledge-agent # too chatty\n",
			wantExclude: []string{"com.apple.knowledge-agent"},
		},
		{
			name:        "blank lines and indentation ignored",
			contents:    "\n   \n\tcom.apple.dock  \n\n",
			wantInclude: []string{"com.apple.dock"},
		},
		{
			name:        "comma-separated line",
			contents:    "com.apple.dock, !com.apple.*\n",
			wantInclude: []string{"com.apple.dock"},
			wantExclude: []string{"com.apple.*"},
		},
		{
			name:        "case is normalised like --filter",
			contents:    "COM.Apple.Dock\n",
			wantInclude: []string{"com.apple.dock"},
		},
		{
			name:        "space after the bang",
			contents:    "! com.apple.dock\n",
			wantExclude: []string{"com.apple.dock"},
		},
		{
			name:     "comments only",
			contents: "# nothing enabled yet\n",
		},
		{
			name:        "no trailing newline",
			contents:    "com.apple.dock",
			wantInclude: []string{"com.apple.dock"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadFilters(writeFilters(t, tt.contents))
			if err != nil {
				t.Fatalf("loadFilters() error = %v", err)
			}
			assertStrings(t, "include", got.include, tt.wantInclude)
			assertStrings(t, "exclude", got.exclude, tt.wantExclude)
		})
	}
}

// A missing file means "no persistent filters", not an error: not every user
// has one.
func TestLoadFiltersMissingFile(t *testing.T) {
	got, err := loadFilters(filepath.Join(t.TempDir(), "filters"))
	if err != nil {
		t.Fatalf("loadFilters() error = %v", err)
	}
	if !got.empty() {
		t.Errorf("loadFilters() = %+v, want empty", got)
	}
}

// An unusable pattern is rejected at startup, the same way --filter rejects
// one, and the message names the offending line.
func TestLoadFiltersInvalidPattern(t *testing.T) {
	path := writeFilters(t, "com.apple.dock\n# a comment\ncom.apple.[\n")
	_, err := loadFilters(path)
	if err == nil {
		t.Fatal("loadFilters() error = nil, want an error")
	}
	if !strings.Contains(err.Error(), path+":3") {
		t.Errorf("loadFilters() error = %q, want it to name %s line 3", err, path)
	}
}

func TestFiltersPath(t *testing.T) {
	t.Run("XDG_CONFIG_HOME wins", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/xdg")
		t.Setenv("HOME", "/home/someone")
		got, err := filtersPath()
		if err != nil {
			t.Fatal(err)
		}
		if want := "/xdg/plistwatch/filters"; got != want {
			t.Errorf("filtersPath() = %q, want %q", got, want)
		}
	})

	t.Run("falls back to ~/.config", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "/home/someone")
		got, err := filtersPath()
		if err != nil {
			t.Fatal(err)
		}
		if want := "/home/someone/.config/plistwatch/filters"; got != want {
			t.Errorf("filtersPath() = %q, want %q", got, want)
		}
	})
}

// The banner is printed as shell comments so that output stays pasteable, and
// it names where each filter came from now that the file and --filter merge.
// Exclusions lose their "!" prefix: the Exclude heading already says what they
// are.
func TestFilterBanner(t *testing.T) {
	var file, cli filterSet
	if err := file.add("!com.apple.xpc.activity2,!tokenbucketratelimiter"); err != nil {
		t.Fatal(err)
	}
	if err := cli.add("com.apple.dock,!com.apple.spaces"); err != nil {
		t.Fatal(err)
	}

	got := filterBanner("/xdg/plistwatch/filters", file, cli)
	want := "# plistwatch: filters from /xdg/plistwatch/filters:\n" +
		"#     Exclude: com.apple.xpc.activity2, tokenbucketratelimiter\n" +
		"#\n" +
		"# plistwatch: filters from the command line:\n" +
		"#     Include: com.apple.dock\n" +
		"#     Exclude: com.apple.spaces\n" +
		"#\n"
	if got != want {
		t.Errorf("filterBanner() =\n%s\nwant\n%s", got, want)
	}

	// A source that contributed nothing gets no block at all.
	got = filterBanner("/xdg/plistwatch/filters", filterSet{}, cli)
	want = "# plistwatch: filters from the command line:\n" +
		"#     Include: com.apple.dock\n" +
		"#     Exclude: com.apple.spaces\n" +
		"#\n"
	if got != want {
		t.Errorf("filterBanner() with no file filters =\n%s\nwant\n%s", got, want)
	}

	if got := filterBanner("/xdg/plistwatch/filters", filterSet{}, filterSet{}); got != "" {
		t.Errorf("filterBanner() with no filters = %q, want empty", got)
	}
}

func assertStrings(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %q, want %q", name, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
}

// filterDomains hands back the domains an "!" pattern removed, so their changes
// can still be announced without their values. Domains dropped for failing to
// match an include pattern are not handed back: with "-f com.apple.dock" that
// would be most of the system.
func TestFilterDomains(t *testing.T) {
	all := func() map[string]any {
		return map[string]any{
			"com.apple.dock":   map[string]any{},
			"com.apple.spaces": map[string]any{},
			"org.example.app":  map[string]any{},
		}
	}

	tests := []struct {
		name         string
		include      []string
		exclude      []string
		wantKept     []string
		wantExcluded []string
	}{
		{
			name:     "no filters keeps everything",
			wantKept: []string{"com.apple.dock", "com.apple.spaces", "org.example.app"},
		},
		{
			name:         "exclusions are reported",
			exclude:      []string{"com.apple.spaces"},
			wantKept:     []string{"com.apple.dock", "org.example.app"},
			wantExcluded: []string{"com.apple.spaces"},
		},
		{
			name:     "domains that miss the include list are dropped silently",
			include:  []string{"com.apple.*"},
			wantKept: []string{"com.apple.dock", "com.apple.spaces"},
		},
		{
			name:         "include plus exclude",
			include:      []string{"com.apple.*"},
			exclude:      []string{"com.apple.spaces"},
			wantKept:     []string{"com.apple.dock"},
			wantExcluded: []string{"com.apple.spaces"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := all()
			excluded := filterDomains(m, tt.include, tt.exclude)
			assertStrings(t, "kept", sortedKeys(m), tt.wantKept)
			assertStrings(t, "excluded", sortedKeys(excluded), tt.wantExcluded)
		})
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
