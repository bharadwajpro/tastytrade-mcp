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

// dxFieldSpec mirrors the acceptEventFields we send in FEED_SETUP. The order here is
// the wire order of COMPACT FEED_DATA values.
var dxFieldSpec = FieldSpec{
	"Quote": {"eventType", "eventSymbol", "bidPrice", "askPrice", "bidSize", "askSize"},
}

// MarketStreamer maintains a DxLink websocket, subscribing to Quote events and
// caching the latest bid/ask per symbol.
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

// Subscribe adds symbols to the desired Quote subscription set and nudges the
// streamer to (re)subscribe.
func (m *MarketStreamer) Subscribe(symbols []string) {
	keys := buildKeys(symbols, []string{"Quote"})
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
		return m.write(conn, map[string]any{
			"type": "FEED_SETUP", "channel": dxChannel,
			"acceptAggregationPeriod": 0.1, "acceptDataFormat": "COMPACT",
			"acceptEventFields": map[string]any{"Quote": dxFieldSpec["Quote"]},
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

// ingest decodes a COMPACT FEED_DATA payload ([eventType, [flat...]]) into quotes.
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
		if v, ok := toFloat(ev.Fields["bidPrice"]); ok {
			q.Bid = v
		}
		if v, ok := toFloat(ev.Fields["askPrice"]); ok {
			q.Ask = v
		}
		q.UpdatedAt = now
		m.quotes[sym] = q
	}
	m.status.UpdatedAt = now
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
