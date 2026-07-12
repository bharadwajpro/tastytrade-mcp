package stream

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDecodeCompactQuote(t *testing.T) {
	// One Quote event: 12 fields per dxFieldSpec["Quote"].
	flat := []any{
		"SPY", float64(1), float64(2), float64(3),
		float64(4), "Q", 500.1, 10.0,
		float64(5), "Q", 500.2, 12.0,
	}
	evs, err := DecodeCompact("Quote", flat, dxFieldSpec)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 1 {
		t.Fatalf("events=%d", len(evs))
	}
	if evs[0].Fields["eventSymbol"] != "SPY" {
		t.Fatalf("sym=%v", evs[0].Fields["eventSymbol"])
	}
	if evs[0].Fields["bidPrice"] != 500.1 || evs[0].Fields["askSize"] != 12.0 {
		t.Fatalf("fields=%v", evs[0].Fields)
	}
}

func TestDecodeCompactGreeks(t *testing.T) {
	flat := []any{
		".SPY260717C500", float64(1), float64(0), float64(2), float64(3), float64(4),
		1.23, 0.22, 0.45, 0.02, -0.05, 0.01, 0.12,
	}
	evs, err := DecodeCompact("Greeks", flat, dxFieldSpec)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 1 {
		t.Fatalf("events=%d", len(evs))
	}
	if evs[0].Fields["delta"] != 0.45 || evs[0].Fields["volatility"] != 0.22 || evs[0].Fields["rho"] != 0.01 {
		t.Fatalf("fields=%v", evs[0].Fields)
	}
}

func TestIngestMergesAllEventTypes(t *testing.T) {
	m := NewMarketStreamer(nil, 0)

	quoteFlat := []any{".A", 1.0, 2.0, 3.0, 4.0, "Q", 1.0, 2.0, 5.0, "Q", 1.1, 3.0}
	tradeFlat := []any{".A", 1.0, 0.0, "X", 10.5, 0.1, 100.0, 1000.0, 5000.0, "Plus", true}
	summaryFlat := []any{".A", 1.0, 9.0, 11.0, 8.0, 10.0, "REGULAR", 0.0, 9.5, "REGULAR", 900.0, 50.0}
	profileFlat := []any{".A", "Test", "UNDEFINED", "ACTIVE", "", 0.0, 0.0, 12.0, 7.0, 15.0, 5.0, 1.2, 0.0, 4.0, 0.5, 1.0, 1e9, 0.8}
	greeksFlat := []any{".A", 1.0, 0.0, 2.0, 3.0, 4.0, 1.05, 0.25, 0.4, 0.01, -0.02, 0.005, 0.1}

	mustIngest := func(et string, flat []any) {
		t.Helper()
		payload, err := json.Marshal([]any{et, flat})
		if err != nil {
			t.Fatal(err)
		}
		m.ingest(payload)
	}
	mustIngest("Quote", quoteFlat)
	mustIngest("Trade", tradeFlat)
	mustIngest("Summary", summaryFlat)
	mustIngest("Profile", profileFlat)
	mustIngest("Greeks", greeksFlat)

	q, found, _ := m.Quote(".A", m.status.UpdatedAt)
	if !found || !q.HasQuote || !q.HasTrade || !q.HasSummary || !q.HasProfile || !q.HasGreeks {
		t.Fatalf("flags: found=%v q=%v t=%v s=%v p=%v g=%v", found, q.HasQuote, q.HasTrade, q.HasSummary, q.HasProfile, q.HasGreeks)
	}
	if q.Bid != 1.0 || q.Ask != 1.1 || q.BidSize != 2.0 || q.AskSize != 3.0 {
		t.Fatalf("quote: %+v", q)
	}
	if q.Last != 10.5 || q.DayVolume != 1000.0 || !q.ExtendedTradingHours {
		t.Fatalf("trade: %+v", q)
	}
	if q.DayOpen != 9.0 || q.OpenInterest != 50.0 {
		t.Fatalf("summary: %+v", q)
	}
	if q.Description != "Test" || q.TradingStatus != "ACTIVE" || q.Beta != 1.2 {
		t.Fatalf("profile: %+v", q)
	}
	if q.Delta != 0.4 || q.Rho != 0.005 || q.TheoPrice != 1.05 || q.Volatility != 0.25 {
		t.Fatalf("greeks: %+v", q)
	}
}

func TestSubscribeRequestsAllEventTypes(t *testing.T) {
	m := NewMarketStreamer(nil, 0)
	m.Subscribe([]string{"SPY"})
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, et := range dxEventTypes {
		k := SubKey{Type: et, Symbol: "SPY"}
		if !m.desired[k] {
			t.Fatalf("missing desired sub %v", k)
		}
	}
	if len(m.desired) != len(dxEventTypes) {
		t.Fatalf("desired=%d want=%d", len(m.desired), len(dxEventTypes))
	}
}

func TestIsStaleUsesQuoteAt(t *testing.T) {
	now := time.Now()
	q := Quote{HasQuote: true, QuoteAt: now.Add(-10 * time.Second)}
	if !isStale(q, now, 5*time.Second) {
		t.Fatal("expected stale")
	}
	q.QuoteAt = now.Add(-1 * time.Second)
	if isStale(q, now, 5*time.Second) {
		t.Fatal("expected fresh")
	}
	if !isStale(Quote{}, now, 5*time.Second) {
		t.Fatal("no quote should be stale")
	}
}

func TestDxFieldSpecCoversAllEventTypes(t *testing.T) {
	for _, et := range dxEventTypes {
		if len(dxFieldSpec[et]) == 0 {
			t.Fatalf("missing field spec for %s", et)
		}
	}
}
