package fetcher

import (
	"context"
	"encoding/json"
	"net/url"
)

// MarketDataRequest groups symbols by instrument type.
type MarketDataRequest struct {
	Equities      []string
	EquityOptions []string
	Futures       []string
	FutureOptions []string
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

// MarketQuote is the full /market-data/by-type item observed live across Equity,
// Equity Option, Future, and Future Option. Numeric fields arrive as strings;
// omitempty keeps instrument-specific extras absent when unset.
type MarketQuote struct {
	Symbol           string `json:"symbol"`
	InstrumentType   string `json:"instrument-type"`
	UpdatedAt        string `json:"updated-at"`
	Bid              string `json:"bid"`
	BidSize          string `json:"bid-size"`
	Ask              string `json:"ask"`
	AskSize          string `json:"ask-size"`
	Mid              string `json:"mid"`
	Mark             string `json:"mark"`
	DxMark           string `json:"dx-mark,omitempty"`
	Last             string `json:"last"`
	LastMkt          string `json:"last-mkt"`
	Beta             string `json:"beta,omitempty"`
	DividendAmount   string `json:"dividend-amount,omitempty"`
	DividendFreq     string `json:"dividend-frequency,omitempty"`
	Open             string `json:"open"`
	DayHighPrice     string `json:"day-high-price"`
	DayLowPrice      string `json:"day-low-price"`
	Close            string `json:"close,omitempty"`
	ClosePriceType   string `json:"close-price-type,omitempty"`
	PrevClose        string `json:"prev-close"`
	PrevClosePrcType string `json:"prev-close-price-type"`
	SummaryDate      string `json:"summary-date"`
	PrevCloseDate    string `json:"prev-close-date"`
	LowLimitPrice    string `json:"low-limit-price,omitempty"`
	HighLimitPrice   string `json:"high-limit-price,omitempty"`
	IsTradingHalted  bool   `json:"is-trading-halted"`
	HaltStartTime    int64  `json:"halt-start-time"`
	HaltEndTime      int64  `json:"halt-end-time"`
	YearLowPrice     string `json:"year-low-price"`
	YearHighPrice    string `json:"year-high-price"`
	Volume           string `json:"volume,omitempty"`
	VolumeExt        string `json:"volume-ext,omitempty"`
	OpenInterest     json.Number `json:"open-interest,omitempty"` // API returns number for FO, string for some others
	TickSize         string `json:"tick-size,omitempty"`
	// Option / future-option greeks (absent on plain equities/futures)
	Delta      string `json:"delta,omitempty"`
	Gamma      string `json:"gamma,omitempty"`
	Theta      string `json:"theta,omitempty"`
	Vega       string `json:"vega,omitempty"`
	Rho        string `json:"rho,omitempty"`
	Volatility string `json:"volatility,omitempty"`
	TheoPrice  string `json:"theo-price,omitempty"`
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
