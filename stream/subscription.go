package stream

import "sort"

// SubKey uniquely identifies a subscription (one symbol + one event type).
type SubKey struct {
	Type   string `json:"type"`
	Symbol string `json:"symbol"`
}

// buildKeys expands symbols × types into a sorted, de-duplicated key slice.
func buildKeys(symbols, types []string) []SubKey {
	seen := map[SubKey]bool{}
	var out []SubKey
	for _, ty := range types {
		for _, s := range symbols {
			if s == "" || ty == "" {
				continue
			}
			k := SubKey{Type: ty, Symbol: s}
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].Symbol < out[j].Symbol
	})
	return out
}

// diffSubs returns keys to add (in desired, not in current).
func diffSubs(current map[SubKey]bool, desired []SubKey) (toAdd []SubKey) {
	for _, k := range desired {
		if !current[k] {
			toAdd = append(toAdd, k)
		}
	}
	return toAdd
}
