package main

import (
	"bytes"
	"flag"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/grazij/plistwatch/go-plist"
)

// Version scheme: <upstream core>+grazij.<counter>, matching the sibling duti
// fork. Upstream tags no releases, so the core is the commit date of the newest
// catilac/plistwatch master commit this fork contains (currently cd0de73). The
// core moves by hand on an upstream sync; the counter moves on every fork
// release. The git tag is this string prefixed with "v".
const version = "2025.09.24+grazij.8"

func getDefaults() (bytes.Buffer, error) {
	var out bytes.Buffer
	cmd := exec.Command("defaults", "read")
	cmd.Env = os.Environ()
	cmd.Stdout = &out
	err := cmd.Run()
	return out, err
}

// filterDomains drops the domains the filters reject and returns the ones an
// exclusion removed, so that changes to them can still be announced without
// their values. Domains dropped for not matching an include pattern are not
// returned: under "-f com.apple.dock" that would be most of the system.
func filterDomains(m map[string]any, include, exclude []string) map[string]any {
	maps.DeleteFunc(m, func(k string, v any) bool {
		// Allow every domain by default
		if len(include) == 0 {
			return false
		}
		for _, pattern := range include {
			if matched, _ := filepath.Match(pattern, strings.ToLower(k)); matched {
				return false
			}
		}
		return true
	})

	excluded := make(map[string]any)
	maps.DeleteFunc(m, func(k string, v any) bool {
		for _, pattern := range exclude {
			if matched, _ := filepath.Match(pattern, strings.ToLower(k)); matched {
				excluded[k] = v
				return true
			}
		}
		return false
	})
	return excluded
}

// isTerminal reports whether f is a terminal, and so whether colour escapes
// have anything to render them.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func main() {
	var cliFilters filterSet

	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.BoolVar(&showVersion, "v", false, "shorthand for --version")

	var quiet bool
	flag.BoolVar(&quiet, "quiet", false, "print only the defaults commands: no filter banner, no excluded-domain notices")
	flag.BoolVar(&quiet, "q", false, "shorthand for --quiet")

	var configPath string
	flag.StringVar(&configPath, "config", "", "read the persistent filters from this `file` instead of the default one")

	flag.Func("filter", "a comma-separated list of `domains`. Prefix names with \"!\" to exclude them. Supports globbing.", cliFilters.add)
	flag.Func("f", "shorthand for --filter", cliFilters.add)
	flag.Parse()

	if showVersion {
		fmt.Println("plistwatch " + version)
		return
	}

	// Persistent filters live in a file and merge with --filter, so an
	// exclusion set once stays in force for one-off filtered runs.
	path, fileFilters, err := resolveFilters(configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	filters := fileFilters
	filters.addSet(cliFilters)

	// NO_COLOR (https://no-color.org) and a redirected stdout both mean the
	// escapes would be noise rather than colour.
	colorOutput = !quiet && isTerminal(os.Stdout) && os.Getenv("NO_COLOR") == ""

	if !quiet {
		fmt.Print(dim(filterBanner(path, fileFilters, cliFilters)))
	}

	var prev map[string]interface{}
	var curr map[string]interface{}
	var prevExcluded map[string]interface{}

	for {
		data, err := getDefaults()
		if _, err = plist.Unmarshal(data.Bytes(), &curr); err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}

		excluded := filterDomains(curr, filters.include, filters.exclude)

		if prev != nil {
			if err = Diff(prev, curr); err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}
		}
		if prevExcluded != nil && !quiet {
			for _, line := range excludedChanges(prevExcluded, excluded) {
				fmt.Print(dim(line + "\n"))
			}
		}

		prev = curr
		curr = nil
		prevExcluded = excluded

		time.Sleep(1 * time.Second)
	}
}
