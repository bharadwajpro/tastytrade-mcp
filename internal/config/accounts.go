package config

import "strings"

// parseAccounts merges single + csv; trims, drops blanks, de-dupes, preserves order.
// The single value (if present) is placed first.
func parseAccounts(single, csv string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	add(single)
	for _, p := range strings.Split(csv, ",") {
		add(p)
	}
	return out
}
