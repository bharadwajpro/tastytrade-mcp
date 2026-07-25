package fetcher

import (
	"context"
	"net/url"
)

// Doer is the minimal HTTP dependency (implemented by ttclient.Client).
type Doer interface {
	DoJSON(ctx context.Context, method, path string, body, out any) error
}

type Equity struct {
	Symbol         string `json:"symbol"`
	InstrumentType string `json:"instrument-type"`
	Description    string `json:"description"`
	Active         bool   `json:"active"`
}

func GetEquity(ctx context.Context, d Doer, symbol string) (*Equity, error) {
	var env struct {
		Data Equity `json:"data"`
	}
	path := "/instruments/equities/" + url.PathEscape(symbol)
	if err := d.DoJSON(ctx, "GET", path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

type Future struct {
	Symbol         string `json:"symbol"`
	ProductCode    string `json:"product-code"`
	InstrumentType string `json:"instrument-type"`
	StreamerSymbol string `json:"streamer-symbol"`
}

func GetFuture(ctx context.Context, d Doer, symbol string) (*Future, error) {
	var env struct {
		Data Future `json:"data"`
	}
	// Future symbols start with "/" (e.g. /ESU6) — escape carefully.
	path := "/instruments/futures/" + url.PathEscape(symbol)
	if err := d.DoJSON(ctx, "GET", path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}
