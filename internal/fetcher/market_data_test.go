package fetcher

import (
	"encoding/json"
	"testing"
)

func TestMarketQuoteUnmarshalOpenInterestNumber(t *testing.T) {
	const raw = `{
		"symbol": "./ESU6 EW1Q6 260807C7880",
		"instrument-type": "Future Option",
		"open-interest": 14
	}`
	var q MarketQuote
	if err := json.Unmarshal([]byte(raw), &q); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if q.OpenInterest.String() != "14" {
		t.Fatalf("open-interest: got %q", q.OpenInterest)
	}
}

func TestMarketQuoteUnmarshalOpenInterestString(t *testing.T) {
	const raw = `{"symbol":"SPY","open-interest":"100"}`
	var q MarketQuote
	if err := json.Unmarshal([]byte(raw), &q); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if q.OpenInterest.String() != "100" {
		t.Fatalf("open-interest: got %q", q.OpenInterest)
	}
}
