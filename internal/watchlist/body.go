package watchlist

import "fmt"

type Entry struct {
	Symbol         string `json:"symbol"`
	InstrumentType string `json:"instrument-type"` // "Equity" | "Equity Option" | ...
}

type Body struct {
	Name             string  `json:"name"`
	GroupName        string  `json:"group-name,omitempty"`
	WatchlistEntries []Entry `json:"watchlist-entries"`
}

func BuildBody(name, groupName string, entries []Entry) (Body, error) {
	if name == "" {
		return Body{}, fmt.Errorf("watchlist name required")
	}
	if len(entries) == 0 {
		return Body{}, fmt.Errorf("watchlist requires at least one entry")
	}
	for i, e := range entries {
		if e.Symbol == "" {
			return Body{}, fmt.Errorf("entry %d: symbol required", i)
		}
		if e.InstrumentType == "" {
			return Body{}, fmt.Errorf("entry %d: instrument-type required", i)
		}
	}
	return Body{Name: name, GroupName: groupName, WatchlistEntries: entries}, nil
}
