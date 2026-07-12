package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// QuoteProvider returns a fresh DxLink api-quote-token and its websocket URL.
type QuoteProvider func(ctx context.Context) (token, dxlinkURL string, err error)

// MarketStreamStatus is a snapshot of the DxLink connection health.
type MarketStreamStatus struct {
	Enabled       bool      `json:"enabled"`
	Connected     bool      `json:"connected"`
	Authorized    bool      `json:"authorized"`
	FeedReady     bool      `json:"feed_ready"`
	Subscriptions int       `json:"subscriptions"`
	CachedQuotes  int       `json:"cached_quotes"`
	LastError     string    `json:"last_error,omitempty"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
}

const (
	dxChannel          = 1
	dxKeepalive        = 30 * time.Second
	dxSetupVersion     = "0.1-DXF-JS/0.3.0"
	dxKeepaliveTimeout = 60
)

// MarketStreamer maintains a DxLink websocket, subscribing to Quote/Trade/Summary/
// Profile/Greeks and caching a merged snapshot per symbol.
type MarketStreamer struct {
	provider       QuoteProvider
	dialer         *websocket.Dialer
	staleThreshold time.Duration

	mu      sync.Mutex
	quotes  map[string]Quote
	desired map[SubKey]bool
	current map[SubKey]bool
	status  MarketStreamStatus
	conn    *websocket.Conn
	ready   bool
	writeMu sync.Mutex
	wake    chan struct{}
}

// NewMarketStreamer constructs (but does not start) the streamer.
func NewMarketStreamer(provider QuoteProvider, staleThreshold time.Duration) *MarketStreamer {
	if staleThreshold <= 0 {
		staleThreshold = 5 * time.Second
	}
	return &MarketStreamer{
		provider:       provider,
		dialer:         websocket.DefaultDialer,
		staleThreshold: staleThreshold,
		quotes:         map[string]Quote{},
		desired:        map[SubKey]bool{},
		current:        map[SubKey]bool{},
		status:         MarketStreamStatus{Enabled: true},
		wake:           make(chan struct{}, 1),
	}
}

// Start launches the connection lifecycle in the background.
func (m *MarketStreamer) Start(ctx context.Context) {
	go m.loop(ctx)
}

// Subscribe adds symbols to the desired Quote/Trade/Summary/Profile/Greeks
// subscription set and nudges the streamer to (re)subscribe.
func (m *MarketStreamer) Subscribe(symbols []string) {
	keys := buildKeys(symbols, dxEventTypes)
	m.mu.Lock()
	for _, k := range keys {
		m.desired[k] = true
	}
	m.mu.Unlock()
	select {
	case m.wake <- struct{}{}:
	default:
	}
}

// Quote returns the cached quote for a symbol plus whether it is stale.
func (m *MarketStreamer) Quote(symbol string, now time.Time) (Quote, bool, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	q, ok := m.quotes[symbol]
	if !ok {
		return Quote{Symbol: symbol}, false, true
	}
	return q, true, isStale(q, now, m.staleThreshold)
}

// WaitForQuotes polls until every symbol has a Quote event (bid/ask) or the
// context is done. Trade/Summary/Profile/Greeks are best-effort in the same window.
func (m *MarketStreamer) WaitForQuotes(ctx context.Context, symbols []string) {
	if len(symbols) == 0 {
		return
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		if m.allQuotesFound(symbols) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (m *MarketStreamer) allQuotesFound(symbols []string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range symbols {
		if q, ok := m.quotes[s]; !ok || !q.HasQuote {
			return false
		}
	}
	return true
}

// Status returns a copy of the current status.
func (m *MarketStreamer) Status() MarketStreamStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.status
	st.CachedQuotes = len(m.quotes)
	st.Subscriptions = len(m.desired)
	return st
}

func (m *MarketStreamer) loop(ctx context.Context) {
	backoff := reconnectMin
	for ctx.Err() == nil {
		err := m.session(ctx)
		m.mu.Lock()
		m.ready, m.status.Connected, m.status.Authorized, m.status.FeedReady = false, false, false, false
		m.current = map[SubKey]bool{}
		if err != nil && ctx.Err() == nil {
			m.status.LastError = err.Error()
		}
		m.mu.Unlock()
		if ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff *= 2; backoff > reconnectMax {
			backoff = reconnectMax
		}
	}
}

func (m *MarketStreamer) session(ctx context.Context) error {
	token, dxURL, err := m.provider(ctx)
	if err != nil {
		return err
	}
	conn, _, err := m.dialer.DialContext(ctx, dxURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	m.mu.Lock()
	m.conn = conn
	m.status.Connected = true
	m.mu.Unlock()

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	if err := m.write(conn, map[string]any{
		"type": "SETUP", "channel": 0, "version": dxSetupVersion,
		"keepaliveTimeout": dxKeepaliveTimeout, "acceptKeepaliveTimeout": dxKeepaliveTimeout,
	}); err != nil {
		return err
	}

	kaCtx, stopKA := context.WithCancel(ctx)
	defer stopKA()
	go m.keepaliveLoop(kaCtx, conn)
	go m.resubscribeLoop(kaCtx, conn)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if err := m.handle(conn, token, msg); err != nil {
			return err
		}
	}
}

func (m *MarketStreamer) handle(conn *websocket.Conn, token string, msg []byte) error {
	var env struct {
		Type    string          `json:"type"`
		Channel int             `json:"channel"`
		State   string          `json:"state"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(msg, &env); err != nil {
		return nil // ignore unparseable frames
	}
	switch env.Type {
	case "AUTH_STATE":
		if env.State == "UNAUTHORIZED" {
			return m.write(conn, map[string]any{"type": "AUTH", "channel": 0, "token": token})
		}
		if env.State == "AUTHORIZED" {
			m.mu.Lock()
			m.status.Authorized = true
			m.mu.Unlock()
			return m.write(conn, map[string]any{
				"type": "CHANNEL_REQUEST", "channel": dxChannel, "service": "FEED",
				"parameters": map[string]any{"contract": "AUTO"},
			})
		}
	case "CHANNEL_OPENED":
		accept := make(map[string]any, len(dxFieldSpec))
		for et, fields := range dxFieldSpec {
			accept[et] = fields
		}
		return m.write(conn, map[string]any{
			"type": "FEED_SETUP", "channel": dxChannel,
			"acceptAggregationPeriod": 0.1, "acceptDataFormat": "COMPACT",
			"acceptEventFields": accept,
		})
	case "FEED_CONFIG":
		m.mu.Lock()
		m.ready, m.status.FeedReady = true, true
		m.current = map[SubKey]bool{}
		m.mu.Unlock()
		m.nudge()
	case "FEED_DATA":
		m.ingest(env.Data)
	}
	return nil
}

// ingest decodes a COMPACT FEED_DATA payload ([eventType, [flat...]]) into the
// per-symbol cache, merging each event type independently.
func (m *MarketStreamer) ingest(data json.RawMessage) {
	var pair []json.RawMessage
	if err := json.Unmarshal(data, &pair); err != nil || len(pair) != 2 {
		return
	}
	var eventType string
	if err := json.Unmarshal(pair[0], &eventType); err != nil {
		return
	}
	var flat []any
	if err := json.Unmarshal(pair[1], &flat); err != nil {
		return
	}
	events, err := DecodeCompact(eventType, flat, dxFieldSpec)
	if err != nil {
		return
	}
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ev := range events {
		sym, _ := ev.Fields["eventSymbol"].(string)
		if sym == "" {
			continue
		}
		q := m.quotes[sym]
		q.Symbol = sym
		switch eventType {
		case "Quote":
			applyQuoteFields(&q, ev.Fields, now)
		case "Trade":
			applyTradeFields(&q, ev.Fields, now)
		case "Summary":
			applySummaryFields(&q, ev.Fields, now)
		case "Profile":
			applyProfileFields(&q, ev.Fields, now)
		case "Greeks":
			applyGreeksFields(&q, ev.Fields, now)
		}
		m.quotes[sym] = q
	}
	m.status.UpdatedAt = now
}

func applyQuoteFields(q *Quote, fields map[string]any, now time.Time) {
	setFloat(&q.EventTime, fields["eventTime"])
	setFloat(&q.Sequence, fields["sequence"])
	setFloat(&q.TimeNanoPart, fields["timeNanoPart"])
	setFloat(&q.BidTime, fields["bidTime"])
	setString(&q.BidExchangeCode, fields["bidExchangeCode"])
	setFloat(&q.Bid, fields["bidPrice"])
	setFloat(&q.BidSize, fields["bidSize"])
	setFloat(&q.AskTime, fields["askTime"])
	setString(&q.AskExchangeCode, fields["askExchangeCode"])
	setFloat(&q.Ask, fields["askPrice"])
	setFloat(&q.AskSize, fields["askSize"])
	q.HasQuote = true
	q.QuoteAt = now
	q.UpdatedAt = now
}

func applyTradeFields(q *Quote, fields map[string]any, now time.Time) {
	setFloat(&q.TradeEventTime, fields["eventTime"])
	setFloat(&q.TradeTimeNanoPart, fields["timeNanoPart"])
	setString(&q.TradeExchangeCode, fields["exchangeCode"])
	setFloat(&q.Last, fields["price"])
	setFloat(&q.Change, fields["change"])
	setFloat(&q.LastSize, fields["size"])
	setFloat(&q.DayVolume, fields["dayVolume"])
	setFloat(&q.DayTurnover, fields["dayTurnover"])
	setString(&q.TickDirection, fields["tickDirection"])
	setBool(&q.ExtendedTradingHours, fields["extendedTradingHours"])
	q.HasTrade = true
	q.TradeAt = now
	q.UpdatedAt = now
}

func applySummaryFields(q *Quote, fields map[string]any, now time.Time) {
	setFloat(&q.DayID, fields["dayId"])
	setFloat(&q.DayOpen, fields["dayOpenPrice"])
	setFloat(&q.DayHigh, fields["dayHighPrice"])
	setFloat(&q.DayLow, fields["dayLowPrice"])
	setFloat(&q.DayClose, fields["dayClosePrice"])
	setString(&q.DayClosePriceType, fields["dayClosePriceType"])
	setFloat(&q.PrevDayID, fields["prevDayId"])
	setFloat(&q.PrevClose, fields["prevDayClosePrice"])
	setString(&q.PrevClosePriceType, fields["prevDayClosePriceType"])
	setFloat(&q.PrevDayVolume, fields["prevDayVolume"])
	setFloat(&q.OpenInterest, fields["openInterest"])
	q.HasSummary = true
	q.SummaryAt = now
	q.UpdatedAt = now
}

func applyProfileFields(q *Quote, fields map[string]any, now time.Time) {
	setString(&q.Description, fields["description"])
	setString(&q.ShortSaleRestriction, fields["shortSaleRestriction"])
	setString(&q.TradingStatus, fields["tradingStatus"])
	setString(&q.StatusReason, fields["statusReason"])
	setFloat(&q.HaltStartTime, fields["haltStartTime"])
	setFloat(&q.HaltEndTime, fields["haltEndTime"])
	setFloat(&q.HighLimitPrice, fields["highLimitPrice"])
	setFloat(&q.LowLimitPrice, fields["lowLimitPrice"])
	setFloat(&q.High52WeekPrice, fields["high52WeekPrice"])
	setFloat(&q.Low52WeekPrice, fields["low52WeekPrice"])
	setFloat(&q.Beta, fields["beta"])
	setFloat(&q.EarningsPerShare, fields["earningsPerShare"])
	setFloat(&q.DividendFrequency, fields["dividendFrequency"])
	setFloat(&q.ExDividendAmount, fields["exDividendAmount"])
	setFloat(&q.ExDividendDayID, fields["exDividendDayId"])
	setFloat(&q.Shares, fields["shares"])
	setFloat(&q.FreeFloat, fields["freeFloat"])
	q.HasProfile = true
	q.ProfileAt = now
	q.UpdatedAt = now
}

func applyGreeksFields(q *Quote, fields map[string]any, now time.Time) {
	setFloat(&q.GreeksEventTime, fields["eventTime"])
	setFloat(&q.GreeksEventFlags, fields["eventFlags"])
	setFloat(&q.GreeksIndex, fields["index"])
	setFloat(&q.GreeksTime, fields["time"])
	setFloat(&q.GreeksSequence, fields["sequence"])
	setFloat(&q.TheoPrice, fields["price"])
	setFloat(&q.Volatility, fields["volatility"])
	setFloat(&q.Delta, fields["delta"])
	setFloat(&q.Gamma, fields["gamma"])
	setFloat(&q.Theta, fields["theta"])
	setFloat(&q.Rho, fields["rho"])
	setFloat(&q.Vega, fields["vega"])
	q.HasGreeks = true
	q.GreeksAt = now
	q.UpdatedAt = now
}

func setFloat(dst *float64, v any) {
	if f, ok := toFloat(v); ok {
		*dst = f
	}
}

func setString(dst *string, v any) {
	switch t := v.(type) {
	case string:
		if t != "" && t != "NaN" {
			*dst = t
		}
	case float64:
		// dxFeed sometimes encodes char exchange codes as numbers
		if t != 0 {
			*dst = string(rune(int(t)))
		}
	}
}

func setBool(dst *bool, v any) {
	switch t := v.(type) {
	case bool:
		*dst = t
	case float64:
		*dst = t != 0
	case string:
		*dst = t == "true" || t == "True" || t == "1"
	}
}

// resubscribeLoop sends incremental FEED_SUBSCRIPTION adds whenever the desired set
// grows and the feed is ready.
func (m *MarketStreamer) resubscribeLoop(ctx context.Context, conn *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.wake:
			m.flushSubs(conn)
		}
	}
}

func (m *MarketStreamer) flushSubs(conn *websocket.Conn) {
	m.mu.Lock()
	if !m.ready {
		m.mu.Unlock()
		return
	}
	desired := make([]SubKey, 0, len(m.desired))
	for k := range m.desired {
		desired = append(desired, k)
	}
	toAdd := diffSubs(m.current, desired)
	for _, k := range toAdd {
		m.current[k] = true
	}
	m.mu.Unlock()
	if len(toAdd) == 0 {
		return
	}
	add := make([]map[string]string, 0, len(toAdd))
	for _, k := range toAdd {
		add = append(add, map[string]string{"type": k.Type, "symbol": k.Symbol})
	}
	_ = m.write(conn, map[string]any{"type": "FEED_SUBSCRIPTION", "channel": dxChannel, "add": add})
}

func (m *MarketStreamer) keepaliveLoop(ctx context.Context, conn *websocket.Conn) {
	t := time.NewTicker(dxKeepalive)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := m.write(conn, map[string]any{"type": "KEEPALIVE", "channel": 0}); err != nil {
				return
			}
		}
	}
}

func (m *MarketStreamer) nudge() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
}

func (m *MarketStreamer) write(conn *websocket.Conn, v any) error {
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	return conn.WriteJSON(v)
}

// toFloat coerces a COMPACT value (number or numeric string) to float64. "NaN" and
// non-numeric strings report ok=false.
func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		var f float64
		if _, err := fmt.Sscanf(t, "%g", &f); err == nil {
			return f, true
		}
	}
	return 0, false
}
