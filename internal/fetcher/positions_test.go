package fetcher

import (
	"encoding/json"
	"testing"
)

func TestPositionUnmarshalPnlFields(t *testing.T) {
	const raw = `{
		"symbol": "./GCV6 OGV6  260924C4375",
		"instrument-type": "Future Option",
		"underlying-symbol": "/GCV6",
		"streamer-symbol": "./OGV26C4375:XCEC",
		"quantity": 1,
		"multiplier": "100.0",
		"quantity-direction": "Long",
		"average-open-price": "74.5",
		"close-price": "75.4",
		"realized-day-gain": "0.0",
		"realized-today": "0.0"
	}`
	var p Position
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.StreamerSymbol != "./OGV26C4375:XCEC" {
		t.Fatalf("streamer-symbol: got %q", p.StreamerSymbol)
	}
	if p.AverageOpenPrice != "74.5" || p.ClosePrice != "75.4" {
		t.Fatalf("prices: avg=%q close=%q", p.AverageOpenPrice, p.ClosePrice)
	}
}
