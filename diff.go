package main

import (
	"fmt"
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/grazij/plistwatch/go-plist"
)

// shellQuote wraps s in single quotes so it survives the shell as one argument.
// A single quote cannot appear inside a single-quoted string, so each one is
// replaced by: close the quote, emit an escaped quote, reopen the quote.
func shellQuote(s string) string {
	return "'" + strings.Replace(s, "'", `'\''`, -1) + "'"
}

// valueArg builds the value portion of a `defaults write` command for a value
// already marshaled to OpenStep text, given its `defaults read-type` name.
//
// The integer, float and date cases must stay in one case clause: Go does not
// fall through, so splitting them leaves the value empty and emits a `defaults
// write` with no value at all.
func valueArg(typ string, s string) string {
	switch typ {
	case "boolean":
		if s == "1" {
			return "-bool true"
		}
		return "-bool false"
	case "integer", "float", "date":
		return "-" + typ + " " + s
	// strings, arrays, dicts and data round-trip as quoted OpenStep text
	default:
		return shellQuote(s)
	}
}

// fallbackType reports the `defaults read-type` name for a parsed plist value.
// It is used only when `defaults read-type` itself fails; without it the value
// falls back to quoted text and a number would be rewritten as a string.
// Types the quoted-text branch already handles correctly report "".
func fallbackType(v interface{}) string {
	switch v.(type) {
	case bool:
		return "boolean"
	case uint64, int64, int:
		return "integer"
	case float32, float64:
		return "float"
	case time.Time:
		return "date"
	}
	return ""
}

// colorOutput colours what plistwatch prints: the dimmed comment lines and the
// alternating runnable commands. It is set at startup and left off unless a
// terminal is there to render the escapes.
var colorOutput bool

// dim wraps each line of s in the ANSI bright-black escape, so comments read as
// secondary next to the runnable `defaults` commands. Each line is closed
// separately, so a line that is piped or interleaved never leaks the colour.
func dim(s string) string {
	if !colorOutput || s == "" {
		return s
	}
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	for i, line := range lines {
		lines[i] = "\x1b[90m" + line + "\x1b[0m"
	}
	return strings.Join(lines, "\n") + "\n"
}

// domainColorer alternates the colour of the runnable `defaults` lines as the
// domain being written changes, so a burst reads as groups rather than one wall
// of text. Consecutive lines for one domain keep their colour; the colour flips
// at a domain boundary. A domain that comes back later may reuse either colour
// — only neighbours have to differ.
//
// The two colours are the terminal's own foreground (no escape at all) and ANSI
// cyan, both legible on a light and a dark background.
type domainColorer struct {
	domain string // the domain the current colour was chosen for
	alt    bool   // whether that colour is the cyan one
}

// lineColor carries the alternation across polls, so a domain that keeps
// changing over several seconds still reads as one group.
var lineColor domainColorer

// line returns one runnable `defaults` command, without its newline, in the
// colour that belongs to domain. The command itself is untouched: stripping or
// disabling the escapes leaves the same pasteable bytes.
func (c *domainColorer) line(domain string, s string) string {
	if domain != c.domain {
		c.domain = domain
		c.alt = !c.alt
	}
	if !colorOutput || !c.alt {
		return s
	}
	return "\x1b[36m" + s + "\x1b[0m"
}

// excludedChanges reports the excluded domains that moved between two polls, as
// one comment line per domain: enough to know that something changed there,
// without the keys or the values that got the domain excluded in the first
// place. Lines are sorted so a poll that touches several domains reads the same
// way every time.
//
// The comparison is reflect.DeepEqual rather than cmp(): cmp only has cases
// for the types `defaults read` happens to produce today, where every scalar
// arrives as a string, and silently reports anything else equal. DeepEqual
// needs no such list.
func excludedChanges(prev map[string]interface{}, curr map[string]interface{}) []string {
	var lines []string

	for domain, v := range curr {
		if old, ok := prev[domain]; !ok || !reflect.DeepEqual(old, v) {
			lines = append(lines, fmt.Sprintf("# defaults write \"%s\"", domain))
		}
	}
	for domain := range prev {
		if _, ok := curr[domain]; !ok {
			lines = append(lines, fmt.Sprintf("# defaults delete \"%s\"", domain))
		}
	}

	sort.Strings(lines)
	return lines
}

func Diff(d1 map[string]interface{}, d2 map[string]interface{}) error {
	// check for additions and changes of domains
	for domain, v2 := range d2 {
		if v1, ok := d1[domain]; ok {
			// compare v1 and v2
			prev := v1.(map[string]interface{})
			curr := v2.(map[string]interface{})

			// check for deleted keys
			for key, _ := range prev {
				if _, ok := curr[key]; !ok {
					fmt.Println(lineColor.line(domain, fmt.Sprintf("defaults delete \"%s\" \"%s\"", domain, key)))
				}
			}

			for key, currVal := range curr {
				prevVal, ok := prev[key]
				if !ok || !cmp(prevVal, currVal) {
					// add this key
					s, err := marshal(currVal)
					if err != nil {
						return err
					}

					out, err := exec.Command("defaults", "read-type", domain, key).Output()
					typ := ""
					if err == nil {
						typ = strings.TrimSpace(strings.Replace(string(out), "Type is ", "", -1))
					}
					if typ == "" {
						typ = fallbackType(currVal)
					}

					fmt.Println(lineColor.line(domain, fmt.Sprintf("defaults write \"%s\" \"%s\" %s", domain, key, valueArg(typ, *s))))
				}
			}
		} else {
			s, err := marshal(v2)
			if err != nil {
				return err
			}
			fmt.Println(lineColor.line(domain, fmt.Sprintf("defaults write \"%s\" %s", domain, shellQuote(*s))))
		}
	}

	// check for deletions
	for domain, _ := range d1 {
		if _, ok := d2[domain]; !ok {
			fmt.Println(lineColor.line(domain, fmt.Sprintf("defaults delete \"%s\"", domain)))
		}
	}

	return nil
}

func cmp(a interface{}, b interface{}) bool {
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		return false
	}

	switch valA := a.(type) {
	case string:
		return a.(string) == b.(string)
	case int:
		return a.(int) == b.(int)
	case []interface{}:
		valB := b.([]interface{})

		if len(valA) != len(valB) {
			return false
		}
		for i := range valA {
			if !cmp(valA[i], valB[i]) {
				return false
			}
		}
	case map[string]interface{}:
		valB := b.(map[string]interface{})
		if len(valA) != len(valB) {
			return false
		}

		for k := range valA {
			if !cmp(valA[k], valB[k]) {
				return false
			}
		}
	}

	return true
}

func marshal(v interface{}) (*string, error) {
	bytes, err := plist.Marshal(v, plist.OpenStepFormat)
	if err != nil {
		return nil, err
	}

	s := string(bytes)

	return &s, nil
}
