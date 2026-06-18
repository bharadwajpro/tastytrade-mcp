package fetcher

import (
	"encoding/json"
	"strconv"
)

type Strike struct {
	Strike         float64
	CallSymbol     string
	CallStreamerSym string
	PutSymbol      string
	PutStreamerSym  string
}

type Expiration struct {
	Date             string
	DaysToExpiration int
	Strikes          []Strike
}

type OptionChain struct {
	Underlying  string
	Expirations []Expiration
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
						Strike string `json:"strike-price"`
						Call   string `json:"call"`
						CallSS string `json:"call-streamer-symbol"`
						Put    string `json:"put"`
						PutSS  string `json:"put-streamer-symbol"`
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
			sp, _ := strconv.ParseFloat(s.Strike, 64)
			exp.Strikes = append(exp.Strikes, Strike{
				Strike: sp, CallSymbol: s.Call, CallStreamerSym: s.CallSS,
				PutSymbol: s.Put, PutStreamerSym: s.PutSS,
			})
		}
		oc.Expirations = append(oc.Expirations, exp)
	}
	return oc, nil
}
