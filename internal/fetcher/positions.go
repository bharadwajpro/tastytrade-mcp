package fetcher

import (
	"context"
	"net/url"
)

type Position struct {
	Symbol            string `json:"symbol"`
	InstrumentType    string `json:"instrument-type"`     // "Equity Option" | "Equity" | ...
	UnderlyingSymbol  string `json:"underlying-symbol"`
	Quantity          string `json:"quantity"`            // string per API
	QuantityDirection string `json:"quantity-direction"`  // "Long" | "Short" | "Zero"
	Multiplier        int    `json:"multiplier"`          // 100 for equity options
}

func ListPositions(ctx context.Context, d Doer, account string) ([]Position, error) {
	var env struct {
		Data struct{ Items []Position `json:"items"` } `json:"data"`
	}
	path := "/accounts/" + url.PathEscape(account) + "/positions"
	if err := d.DoJSON(ctx, "GET", path, nil, &env); err != nil {
		return nil, err
	}
	return env.Data.Items, nil
}
