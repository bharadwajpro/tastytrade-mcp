package fetcher

import (
	"context"
	"encoding/json"
	"net/url"
	"time"
)

// GetOptionChain fetches the nested option chain for an underlying and parses it.
func GetOptionChain(ctx context.Context, d Doer, underlying string) (*OptionChain, error) {
	var raw json.RawMessage
	path := "/option-chains/" + url.PathEscape(underlying) + "/nested"
	if err := d.DoJSON(ctx, "GET", path, nil, &raw); err != nil {
		return nil, err
	}
	return ParseNestedChain(raw)
}

// MarketStatus reports whether the equities market is currently open.
type MarketStatus struct {
	IsOpen  bool    `json:"is_open"`
	Session Session `json:"session"`
}

// IsMarketOpen fetches the current equities session and computes open/closed at now.
// collection defaults to "equities" when empty.
func IsMarketOpen(ctx context.Context, d Doer, collection string, now time.Time) (*MarketStatus, error) {
	if collection == "" {
		collection = "equities"
	}
	s, err := GetCurrentSession(ctx, d, collection)
	if err != nil {
		return nil, err
	}
	return &MarketStatus{IsOpen: sessionOpen(*s, now), Session: *s}, nil
}

// SymbolMatch is one result from the symbol-search endpoint.
type SymbolMatch struct {
	Symbol         string `json:"symbol"`
	Description    string `json:"description"`
	InstrumentType string `json:"instrument-type"`
}

// SearchSymbols looks up tradeable symbols matching a query fragment.
func SearchSymbols(ctx context.Context, d Doer, query string) ([]SymbolMatch, error) {
	var env struct {
		Data struct {
			Items []SymbolMatch `json:"items"`
		} `json:"data"`
	}
	path := "/symbols/search/" + url.PathEscape(query)
	if err := d.DoJSON(ctx, "GET", path, nil, &env); err != nil {
		return nil, err
	}
	return env.Data.Items, nil
}

// QuoteToken is the DxLink api-quote-token plus its streamer URL (valid 24h).
type QuoteToken struct {
	Token     string `json:"token"`
	DxLinkURL string `json:"dxlink-url"`
	Level     string `json:"level"`
}

// GetQuoteToken fetches the customer's DxLink streaming token.
func GetQuoteToken(ctx context.Context, d Doer) (*QuoteToken, error) {
	var env struct {
		Data QuoteToken `json:"data"`
	}
	if err := d.DoJSON(ctx, "GET", "/api-quote-tokens", nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}
