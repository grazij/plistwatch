package main

import (
	"fmt"
	"os/exec"
	"reflect"
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
					fmt.Printf("defaults delete \"%s\" \"%s\"\n", domain, key)
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

					fmt.Printf("defaults write \"%s\" \"%s\" %s\n", domain, key, valueArg(typ, *s))
				}
			}
		} else {
			s, err := marshal(v2)
			if err != nil {
				return err
			}
			fmt.Printf("defaults write \"%s\" %s\n", domain, shellQuote(*s))
		}
	}

	// check for deletions
	for domain, _ := range d1 {
		if _, ok := d2[domain]; !ok {
			fmt.Printf("defaults delete \"%s\"\n", domain)
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
