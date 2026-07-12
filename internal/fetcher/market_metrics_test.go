package fetcher

import (
	"encoding/json"
	"testing"
)

func TestMarketMetricUnmarshalLiquidityNumber(t *testing.T) {
	const raw = `{
		"symbol": "AAPL",
		"implied-volatility-index-rank": "0.63",
		"implied-volatility-percentile": "0.67",
		"beta": "1.08",
		"liquidity-rating": 4
	}`
	var m MarketMetric
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.Symbol != "AAPL" {
		t.Fatalf("symbol: got %q", m.Symbol)
	}
	if m.Liquidity.String() != "4" {
		t.Fatalf("liquidity: got %q", m.Liquidity)
	}
}

func TestMarketMetricUnmarshalLiquidityString(t *testing.T) {
	const raw = `{"symbol":"SPY","liquidity-rating":"3"}`
	var m MarketMetric
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.Liquidity.String() != "3" {
		t.Fatalf("liquidity: got %q", m.Liquidity)
	}
}
