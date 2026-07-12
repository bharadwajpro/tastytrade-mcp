package fetcher

import (
	"context"
	"encoding/json"
	"net/url"
)

type Position struct {
	Symbol            string      `json:"symbol"`
	InstrumentType    string      `json:"instrument-type"` // "Equity Option" | "Equity" | ...
	UnderlyingSymbol  string      `json:"underlying-symbol"`
	StreamerSymbol    string      `json:"streamer-symbol,omitempty"`
	Quantity          json.Number `json:"quantity"`
	QuantityDirection string      `json:"quantity-direction"` // "Long" | "Short" | "Zero"
	Multiplier        json.Number `json:"multiplier"`
	AverageOpenPrice  string      `json:"average-open-price,omitempty"`
	ClosePrice        string      `json:"close-price,omitempty"`
	RealizedDayGain   string      `json:"realized-day-gain,omitempty"`
	RealizedToday     string      `json:"realized-today,omitempty"`
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
