package fetcher

import (
	"context"
	"net/url"
)

// MarketDataRequest groups symbols by instrument type.
type MarketDataRequest struct {
	Equities       []string
	EquityOptions  []string
	Futures        []string
	FutureOptions  []string
}

// buildByTypeQuery encodes the request as query params. Empty groups are omitted.
func buildByTypeQuery(r MarketDataRequest) string {
	q := url.Values{}
	addAll := func(key string, syms []string) {
		for _, s := range syms {
			if s != "" {
				q.Add(key, s)
			}
		}
	}
	addAll("equity", r.Equities)
	addAll("equity-option", r.EquityOptions)
	addAll("future", r.Futures)
	addAll("future-option", r.FutureOptions)
	return q.Encode()
}

type MarketQuote struct {
	Symbol string `json:"symbol"`
	Bid    string `json:"bid"`
	Ask    string `json:"ask"`
	Last   string `json:"last"`
}

func GetMarketData(ctx context.Context, d Doer, r MarketDataRequest) ([]MarketQuote, error) {
	var env struct {
		Data struct {
			Items []MarketQuote `json:"items"`
		} `json:"data"`
	}
	path := "/market-data/by-type"
	if qs := buildByTypeQuery(r); qs != "" {
		path += "?" + qs
	}
	if err := d.DoJSON(ctx, "GET", path, nil, &env); err != nil {
		return nil, err
	}
	return env.Data.Items, nil
}
