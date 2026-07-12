package stream

import "time"

// Quote is the merged DxLink snapshot for one symbol across Quote, Trade,
// Summary, Profile, and Greeks events (fields populate as each event arrives).
type Quote struct {
	Symbol string `json:"symbol"`

	// Quote
	EventTime       float64   `json:"event_time,omitempty"`
	Sequence        float64   `json:"sequence,omitempty"`
	TimeNanoPart    float64   `json:"time_nano_part,omitempty"`
	BidTime         float64   `json:"bid_time,omitempty"`
	BidExchangeCode string    `json:"bid_exchange_code,omitempty"`
	Bid             float64   `json:"bid"`
	BidSize         float64   `json:"bid_size"`
	AskTime         float64   `json:"ask_time,omitempty"`
	AskExchangeCode string    `json:"ask_exchange_code,omitempty"`
	Ask             float64   `json:"ask"`
	AskSize         float64   `json:"ask_size"`
	HasQuote        bool      `json:"has_quote"`
	QuoteAt         time.Time `json:"quote_updated_at,omitempty"`

	// Trade
	TradeEventTime       float64   `json:"trade_event_time,omitempty"`
	TradeTimeNanoPart    float64   `json:"trade_time_nano_part,omitempty"`
	TradeExchangeCode    string    `json:"trade_exchange_code,omitempty"`
	Last                 float64   `json:"last"`
	Change               float64   `json:"change"`
	LastSize             float64   `json:"last_size"`
	DayVolume            float64   `json:"day_volume"`
	DayTurnover          float64   `json:"day_turnover"`
	TickDirection        string    `json:"tick_direction,omitempty"`
	ExtendedTradingHours bool      `json:"extended_trading_hours"`
	HasTrade             bool      `json:"has_trade"`
	TradeAt              time.Time `json:"trade_updated_at,omitempty"`

	// Summary
	DayID              float64   `json:"day_id,omitempty"`
	DayOpen            float64   `json:"day_open"`
	DayHigh            float64   `json:"day_high"`
	DayLow             float64   `json:"day_low"`
	DayClose           float64   `json:"day_close"`
	DayClosePriceType  string    `json:"day_close_price_type,omitempty"`
	PrevDayID          float64   `json:"prev_day_id,omitempty"`
	PrevClose          float64   `json:"prev_close"`
	PrevClosePriceType string    `json:"prev_close_price_type,omitempty"`
	PrevDayVolume      float64   `json:"prev_day_volume"`
	OpenInterest       float64   `json:"open_interest"`
	HasSummary         bool      `json:"has_summary"`
	SummaryAt          time.Time `json:"summary_updated_at,omitempty"`

	// Profile
	Description          string    `json:"description,omitempty"`
	ShortSaleRestriction string    `json:"short_sale_restriction,omitempty"`
	TradingStatus        string    `json:"trading_status,omitempty"`
	StatusReason         string    `json:"status_reason,omitempty"`
	HaltStartTime        float64   `json:"halt_start_time,omitempty"`
	HaltEndTime          float64   `json:"halt_end_time,omitempty"`
	HighLimitPrice       float64   `json:"high_limit_price"`
	LowLimitPrice        float64   `json:"low_limit_price"`
	High52WeekPrice      float64   `json:"high_52_week_price"`
	Low52WeekPrice       float64   `json:"low_52_week_price"`
	Beta                 float64   `json:"beta"`
	EarningsPerShare     float64   `json:"earnings_per_share"`
	DividendFrequency    float64   `json:"dividend_frequency"`
	ExDividendAmount     float64   `json:"ex_dividend_amount"`
	ExDividendDayID      float64   `json:"ex_dividend_day_id,omitempty"`
	Shares               float64   `json:"shares"`
	FreeFloat            float64   `json:"free_float"`
	HasProfile           bool      `json:"has_profile"`
	ProfileAt            time.Time `json:"profile_updated_at,omitempty"`

	// Greeks
	GreeksEventTime  float64   `json:"greeks_event_time,omitempty"`
	GreeksEventFlags float64   `json:"greeks_event_flags,omitempty"`
	GreeksIndex      float64   `json:"greeks_index,omitempty"`
	GreeksTime       float64   `json:"greeks_time,omitempty"`
	GreeksSequence   float64   `json:"greeks_sequence,omitempty"`
	TheoPrice        float64   `json:"theo_price"`
	Volatility       float64   `json:"volatility"`
	Delta            float64   `json:"delta"`
	Gamma            float64   `json:"gamma"`
	Theta            float64   `json:"theta"`
	Rho              float64   `json:"rho"`
	Vega             float64   `json:"vega"`
	HasGreeks        bool      `json:"has_greeks"`
	GreeksAt         time.Time `json:"greeks_updated_at,omitempty"`

	// UpdatedAt is the most recent receive time across any event type.
	UpdatedAt time.Time `json:"updated_at"`
}

// isStale reports whether the Quote bid/ask is older than threshold at now.
// A symbol with no Quote event yet is always stale.
func isStale(q Quote, now time.Time, threshold time.Duration) bool {
	if !q.HasQuote || q.QuoteAt.IsZero() {
		return true
	}
	return now.Sub(q.QuoteAt) > threshold
}
