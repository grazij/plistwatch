package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// filterSet holds the domain patterns of one filter source, split the way
// filterDomains consumes them.
type filterSet struct {
	include []string
	exclude []string
}

// add parses one comma-separated filter list: the argument of --filter, or one
// line of the filters file. A pattern prefixed with "!" excludes its domains.
func (f *filterSet) add(s string) error {
	for _, v := range strings.Split(s, ",") {
		v = strings.ToLower(strings.TrimSpace(v))
		domain, found := strings.CutPrefix(v, "!")
		// Users might write "! com.apple.dock" so we trim again
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}
		if _, err := filepath.Match(domain, ""); err != nil {
			return fmt.Errorf("invalid filter pattern %q: %w", domain, err)
		}
		if found {
			f.exclude = append(f.exclude, domain)
		} else {
			f.include = append(f.include, domain)
		}
	}
	return nil
}

// addSet merges another source into f. The file and --filter accumulate rather
// than override, so an exclusion set once in the file stays in force.
func (f *filterSet) addSet(o filterSet) {
	f.include = append(f.include, o.include...)
	f.exclude = append(f.exclude, o.exclude...)
}

func (f filterSet) empty() bool {
	return len(f.include) == 0 && len(f.exclude) == 0
}

// filtersPath returns the persistent filters file:
// $XDG_CONFIG_HOME/plistwatch/filters when XDG_CONFIG_HOME is set, otherwise
// ~/.config/plistwatch/filters.
func filtersPath() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "plistwatch", "filters"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "plistwatch", "filters"), nil
}

// loadFilters reads the persistent filters file. Each line holds one filter, or
// a comma-separated list of them; "#" starts a comment that runs to the end of
// the line, and blank lines are ignored. A missing file is not an error: it
// just means no persistent filters.
func loadFilters(path string) (filterSet, error) {
	var f filterSet

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return f, nil
	}
	if err != nil {
		return f, err
	}

	for i, line := range strings.Split(string(data), "\n") {
		if comment := strings.IndexByte(line, '#'); comment >= 0 {
			line = line[:comment]
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		if err := f.add(line); err != nil {
			return filterSet{}, fmt.Errorf("%s:%d: %w", path, i+1, err)
		}
	}

	return f, nil
}

// resolveFilters picks the persistent filters file and reads it. An explicit
// --config path replaces the default one rather than adding to it, so
// filtersPath() is not consulted at all when one is given. A path named on the
// command line has to exist: a typo there would otherwise silently watch
// everything. A missing default file stays the "no persistent filters" case.
func resolveFilters(config string) (string, filterSet, error) {
	path := config
	if path == "" {
		var err error
		if path, err = filtersPath(); err != nil {
			return "", filterSet{}, err
		}
	} else if _, err := os.Stat(path); err != nil {
		return "", filterSet{}, fmt.Errorf("--config: %w", err)
	}

	f, err := loadFilters(path)
	return path, f, err
}

// filterBanner announces the filters in force, one block per source: a header
// naming the source, then the include and exclude patterns, then a blank
// comment. Exclusions are listed without their "!" prefix, since the Exclude
// heading already says what they are. Every line is a shell comment, so output
// stays pasteable alongside the `defaults` commands plistwatch prints. Empty
// lists, sources that contributed nothing, and an unfiltered run print nothing.
func filterBanner(path string, file filterSet, cli filterSet) string {
	var b strings.Builder
	for _, src := range []struct {
		name string
		set  filterSet
	}{
		{path, file},
		{"the command line", cli},
	} {
		if src.set.empty() {
			continue
		}
		fmt.Fprintf(&b, "# plistwatch: filters from %s:\n", src.name)
		if len(src.set.include) > 0 {
			fmt.Fprintf(&b, "#     Include: %s\n", strings.Join(src.set.include, ", "))
		}
		if len(src.set.exclude) > 0 {
			fmt.Fprintf(&b, "#     Exclude: %s\n", strings.Join(src.set.exclude, ", "))
		}
		b.WriteString("#\n")
	}
	return b.String()
}
