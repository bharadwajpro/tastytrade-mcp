package fetcher

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// OptionSeries is one futures-option series (future + option root), with its expirations.
type OptionSeries struct {
	FutureSymbol     string       `json:"future_symbol"`
	OptionRootSymbol string       `json:"option_root_symbol"`
	ExpirationType   string       `json:"expiration_type,omitempty"`
	Expirations      []Expiration `json:"expirations"`
}

// Strike, Expiration, OptionChain are shared by equity and futures-option chains.
// For futures options, Series is populated and Expirations is the flattened union
// (same contracts) for helpers that walk a single expiration list.

type Strike struct {
	Strike          float64
	CallSymbol      string
	CallStreamerSym string
	PutSymbol       string
	PutStreamerSym  string
}

type Expiration struct {
	Date                 string
	DaysToExpiration     int
	OptionContractSymbol string `json:",omitempty"`
	Strikes              []Strike
}

type OptionChain struct {
	Underlying  string
	Expirations []Expiration
	Series      []OptionSeries `json:",omitempty"`
}

func ParseNestedChain(raw []byte) (*OptionChain, error) {
	var env struct {
		Data struct {
			Items []struct {
				Underlying  string `json:"underlying-symbol"`
				Expirations []struct {
					Date    string `json:"expiration-date"`
					DTE     int    `json:"days-to-expiration"`
					Strikes []struct {
						Strike json.RawMessage `json:"strike-price"`
						Call   string          `json:"call"`
						CallSS string          `json:"call-streamer-symbol"`
						Put    string          `json:"put"`
						PutSS  string          `json:"put-streamer-symbol"`
					} `json:"strikes"`
				} `json:"expirations"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if len(env.Data.Items) == 0 {
		return &OptionChain{}, nil
	}
	it := env.Data.Items[0]
	oc := &OptionChain{Underlying: it.Underlying}
	for _, e := range it.Expirations {
		exp := Expiration{Date: e.Date, DaysToExpiration: e.DTE}
		for _, s := range e.Strikes {
			sp, _ := parseStrike(s.Strike)
			exp.Strikes = append(exp.Strikes, Strike{
				Strike: sp, CallSymbol: s.Call, CallStreamerSym: s.CallSS,
				PutSymbol: s.Put, PutStreamerSym: s.PutSS,
			})
		}
		oc.Expirations = append(oc.Expirations, exp)
	}
	return oc, nil
}

// ParseNestedFutureOptionChain parses GET /futures-option-chains/{root}/nested.
// If futureFilter is non-empty (e.g. "/CLZ6"), only series for that future are kept.
func ParseNestedFutureOptionChain(raw []byte, requested, futureFilter string) (*OptionChain, error) {
	var env struct {
		Data struct {
			OptionChains []struct {
				Expirations []struct {
					Underlying           string `json:"underlying-symbol"`
					OptionRootSymbol     string `json:"option-root-symbol"`
					OptionContractSymbol string `json:"option-contract-symbol"`
					ExpirationType       string `json:"expiration-type"`
					Date                 string `json:"expiration-date"`
					DTE                  int    `json:"days-to-expiration"`
					Strikes              []struct {
						Strike json.RawMessage `json:"strike-price"`
						Call   string          `json:"call"`
						CallSS string          `json:"call-streamer-symbol"`
						Put    string          `json:"put"`
						PutSS  string          `json:"put-streamer-symbol"`
					} `json:"strikes"`
				} `json:"expirations"`
			} `json:"option-chains"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}

	type seriesKey struct {
		Future string
		Root   string
		Typ    string
	}
	grouped := map[seriesKey]*OptionSeries{}
	var order []seriesKey

	for _, ch := range env.Data.OptionChains {
		for _, e := range ch.Expirations {
			if futureFilter != "" && e.Underlying != futureFilter {
				continue
			}
			key := seriesKey{e.Underlying, e.OptionRootSymbol, e.ExpirationType}
			ser, ok := grouped[key]
			if !ok {
				ser = &OptionSeries{
					FutureSymbol:     e.Underlying,
					OptionRootSymbol: e.OptionRootSymbol,
					ExpirationType:   e.ExpirationType,
				}
				grouped[key] = ser
				order = append(order, key)
			}
			exp := Expiration{
				Date:                 e.Date,
				DaysToExpiration:     e.DTE,
				OptionContractSymbol: e.OptionContractSymbol,
			}
			for _, s := range e.Strikes {
				sp, _ := parseStrike(s.Strike)
				exp.Strikes = append(exp.Strikes, Strike{
					Strike: sp, CallSymbol: s.Call, CallStreamerSym: s.CallSS,
					PutSymbol: s.Put, PutStreamerSym: s.PutSS,
				})
			}
			ser.Expirations = append(ser.Expirations, exp)
		}
	}

	oc := &OptionChain{Underlying: requested}
	for _, key := range order {
		ser := grouped[key]
		sort.Slice(ser.Expirations, func(i, j int) bool {
			return ser.Expirations[i].Date < ser.Expirations[j].Date
		})
		oc.Series = append(oc.Series, *ser)
		oc.Expirations = append(oc.Expirations, ser.Expirations...)
	}
	if len(oc.Series) == 0 {
		if futureFilter != "" {
			return nil, fmt.Errorf("no futures-option series for %s", futureFilter)
		}
		return nil, fmt.Errorf("no futures-option series for %s", requested)
	}
	return oc, nil
}

func parseStrike(raw json.RawMessage) (float64, error) {
	if len(raw) == 0 {
		return 0, nil
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
}
