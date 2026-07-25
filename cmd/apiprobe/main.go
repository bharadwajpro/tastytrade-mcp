// Command apiprobe live-calls tastytrade REST + WebSocket surfaces and writes
// exhaustive field inventories under ./probe-out for mapping MCP responses.
//
// Usage (from repo root, with .env present):
//
//	go run ./cmd/apiprobe
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/bharadwajpro/tastytrade-mcp/config"
	"github.com/bharadwajpro/tastytrade-mcp/fetcher"
	"github.com/bharadwajpro/tastytrade-mcp/ttclient"
)

const outDir = "probe-out"

func main() {
	if err := run(); err != nil {
		log.Fatalf("apiprobe: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	client := ttclient.New(ttclient.Config{
		BaseURL:      cfg.APIBaseURL,
		ClientSecret: cfg.ClientSecret,
		RefreshToken: cfg.RefreshToken,
		UserAgent:    cfg.UserAgent,
		APIVersion:   cfg.APIVersion,
	}, nil)

	report := &Report{StartedAt: time.Now().UTC()}

	log.Printf("probing REST against %s", cfg.APIBaseURL)
	if err := probeREST(ctx, client, cfg, report); err != nil {
		report.Errors = append(report.Errors, "rest: "+err.Error())
	}

	log.Printf("probing DxLink market stream")
	if err := probeDxLink(ctx, client, report); err != nil {
		report.Errors = append(report.Errors, "dxlink: "+err.Error())
	}

	if len(cfg.AccountNumbers) > 0 {
		log.Printf("probing account streamer %s", cfg.AccountStreamerURL)
		if err := probeAccountStream(ctx, client, cfg, report); err != nil {
			report.Errors = append(report.Errors, "account-stream: "+err.Error())
		}
	} else {
		report.Notes = append(report.Notes, "skipped account streamer: no TT_ACCOUNT_NUMBER")
	}

	report.FinishedAt = time.Now().UTC()
	return writeReport(report)
}

// ---------------------------------------------------------------------------
// Report model
// ---------------------------------------------------------------------------

type Report struct {
	StartedAt  time.Time       `json:"started_at"`
	FinishedAt time.Time       `json:"finished_at"`
	REST       []EndpointProbe `json:"rest"`
	DxLink     *DxLinkProbe    `json:"dxlink,omitempty"`
	AccountWS  *AccountProbe   `json:"account_streamer,omitempty"`
	Notes      []string        `json:"notes,omitempty"`
	Errors     []string        `json:"errors,omitempty"`
}

type EndpointProbe struct {
	Name       string   `json:"name"`
	Method     string   `json:"method"`
	Path       string   `json:"path"`
	StatusOK   bool     `json:"status_ok"`
	Error      string   `json:"error,omitempty"`
	FieldPaths []string `json:"field_paths"`
	SampleFile string   `json:"sample_file,omitempty"`
}

type DxLinkProbe struct {
	URL            string              `json:"url"`
	Subscribed     []string            `json:"subscribed_symbols"`
	EventTypes     []string            `json:"event_types_requested"`
	AcceptFields   map[string][]string `json:"accept_event_fields"`
	EventsSeen     map[string]int      `json:"events_seen"`
	FieldsByEvent  map[string][]string `json:"fields_observed_by_event"`
	SampleFile     string              `json:"sample_file,omitempty"`
	FeedConfigRaw  json.RawMessage     `json:"feed_config_raw,omitempty"`
	Duration       string              `json:"listen_duration"`
	LastError      string              `json:"last_error,omitempty"`
}

type AccountProbe struct {
	URL              string         `json:"url"`
	Accounts         []string       `json:"accounts"`
	Connected        bool           `json:"connected"`
	SessionID        string         `json:"session_id,omitempty"`
	ControlActions   []string       `json:"control_actions_seen"`
	NotificationType map[string]int `json:"notification_types_seen"`
	FieldPathsByType map[string][]string `json:"field_paths_by_type"`
	SampleFile       string         `json:"sample_file,omitempty"`
	Duration         string         `json:"listen_duration"`
	LastError        string         `json:"last_error,omitempty"`
}

func writeReport(r *Report) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(outDir, "report.json")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return err
	}
	md := renderMarkdown(r)
	if err := os.WriteFile(filepath.Join(outDir, "FIELD_INVENTORY.md"), []byte(md), 0o644); err != nil {
		return err
	}
	log.Printf("wrote %s and FIELD_INVENTORY.md", path)
	return nil
}

// ---------------------------------------------------------------------------
// REST
// ---------------------------------------------------------------------------

func probeREST(ctx context.Context, client *ttclient.Client, cfg *config.Config, report *Report) error {
	acct := ""
	if len(cfg.AccountNumbers) > 0 {
		acct = cfg.AccountNumbers[0]
	}

	// Discover live future + option streamer symbols from chains.
	eqOptOCC, eqOptStreamer := discoverEquityOption(ctx, client)
	futSym, foOCC, foStreamer := discoverFutureAndFO(ctx, client)

	type call struct {
		name, method, path string
	}
	calls := []call{
		{"api_quote_tokens", "GET", "/api-quote-tokens"},
		{"instruments_equity_SPY", "GET", "/instruments/equities/SPY"},
		{"option_chain_nested_SPY", "GET", "/option-chains/SPY/nested"},
		{"market_metrics_SPY", "GET", "/market-metrics?symbols=SPY"},
		{"market_sessions_equities", "GET", "/market-time/equities/sessions/current"},
		{"symbols_search_SPY", "GET", "/symbols/search/SPY"},
		{"watchlists", "GET", "/watchlists"},
		{"quote_alerts", "GET", "/quote-alerts"},
		{"customers_me", "GET", "/customers/me"},
		{"customers_me_accounts", "GET", "/customers/me/accounts"},
	}
	if futSym != "" {
		calls = append(calls, call{"instruments_future", "GET", "/instruments/futures/" + url.PathEscape(futSym)})
		root := productRoot(futSym)
		calls = append(calls, call{"future_option_chain_nested", "GET", "/futures-option-chains/" + url.PathEscape(root) + "/nested"})
	}
	// Market data by-type covering equity / EO / future / FO.
	mdq := "/market-data/by-type?equity=SPY"
	if eqOptOCC != "" {
		mdq += "&equity-option=" + url.QueryEscape(eqOptOCC)
	}
	if futSym != "" {
		mdq += "&future=" + url.QueryEscape(futSym)
	}
	if foOCC != "" {
		mdq += "&future-option=" + url.QueryEscape(foOCC)
	}
	calls = append(calls, call{"market_data_by_type", "GET", mdq})

	if acct != "" {
		ap := "/accounts/" + url.PathEscape(acct)
		calls = append(calls,
			call{"account_balances", "GET", ap + "/balances"},
			call{"account_balances_USD", "GET", ap + "/balances/USD"},
			call{"account_positions", "GET", ap + "/positions"},
			call{"account_trading_status", "GET", ap + "/trading-status"},
			call{"account_transactions", "GET", ap + "/transactions?per-page=5"},
			call{"account_orders_live", "GET", ap + "/orders/live"},
			call{"account_orders", "GET", ap + "/orders?per-page=5"},
			call{"account_margin_requirements", "GET", "/margin/accounts/" + url.PathEscape(acct) + "/requirements"},
		)
	}

	_ = eqOptStreamer
	_ = foStreamer
	report.Notes = append(report.Notes,
		fmt.Sprintf("discovered equity-option OCC=%q streamer=%q", eqOptOCC, eqOptStreamer),
		fmt.Sprintf("discovered future=%q future-option OCC=%q streamer=%q", futSym, foOCC, foStreamer),
	)

	for _, c := range calls {
		p := EndpointProbe{Name: c.name, Method: c.method, Path: sanitizePath(c.path)}
		raw, err := rawGET(ctx, client, c.path)
		if err != nil {
			p.Error = err.Error()
			report.REST = append(report.REST, p)
			log.Printf("  FAIL %s: %v", c.name, err)
			continue
		}
		p.StatusOK = true
		p.FieldPaths = sortedKeys(collectPaths(raw, ""))
		sample := filepath.Join(outDir, "rest_"+c.name+".json")
		_ = os.WriteFile(sample, pretty(raw), 0o644)
		p.SampleFile = sample
		report.REST = append(report.REST, p)
		log.Printf("  OK   %s (%d fields)", c.name, len(p.FieldPaths))
	}
	return nil
}

func discoverEquityOption(ctx context.Context, client *ttclient.Client) (occ, streamer string) {
	oc, err := fetcher.GetOptionChain(ctx, client, "SPY")
	if err != nil || oc == nil || len(oc.Expirations) == 0 {
		return "", ""
	}
	for _, exp := range oc.Expirations {
		for _, s := range exp.Strikes {
			if s.CallSymbol != "" && s.CallStreamerSym != "" {
				return s.CallSymbol, s.CallStreamerSym
			}
		}
	}
	return "", ""
}

func discoverFutureAndFO(ctx context.Context, client *ttclient.Client) (fut, foOCC, foStreamer string) {
	// Prefer a dated equity-index future; Jul 2026 → /ESU6 (Sep).
	for _, c := range []string{"/ESU6", "/ES", "/NQU6", "/CL"} {
		oc, err := fetcher.GetOptionChain(ctx, client, c)
		if err != nil || oc == nil {
			continue
		}
		fut = c
		for _, exp := range oc.Expirations {
			for _, s := range exp.Strikes {
				if s.CallSymbol == "" {
					continue
				}
				foOCC, foStreamer = s.CallSymbol, s.CallStreamerSym
				if u := underlyingFromFO(s.CallSymbol); u != "" {
					fut = u
				} else if len(oc.Series) > 0 && oc.Series[0].FutureSymbol != "" {
					fut = oc.Series[0].FutureSymbol
				}
				return fut, foOCC, foStreamer
			}
		}
		if len(oc.Series) > 0 && oc.Series[0].FutureSymbol != "" {
			return oc.Series[0].FutureSymbol, "", ""
		}
		return fut, "", ""
	}
	return "/ESU6", "", ""
}

func underlyingFromFO(sym string) string {
	sym = strings.TrimSpace(sym)
	if i := strings.IndexAny(sym, " \t"); i > 0 {
		sym = sym[:i]
	}
	sym = strings.TrimPrefix(sym, "./")
	sym = strings.TrimPrefix(sym, "/")
	if len(sym) >= 4 && sym[0] >= 'A' && sym[0] <= 'Z' {
		return "/" + sym
	}
	return ""
}

func rawGET(ctx context.Context, client *ttclient.Client, path string) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := client.DoJSON(ctx, "GET", path, nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// ---------------------------------------------------------------------------
// DxLink
// ---------------------------------------------------------------------------

// Broad field lists from dxFeed event schemas (COMPACT acceptEventFields order).
var dxAcceptFields = map[string][]string{
	"Quote": {
		"eventSymbol", "eventTime", "sequence", "timeNanoPart",
		"bidTime", "bidExchangeCode", "bidPrice", "bidSize",
		"askTime", "askExchangeCode", "askPrice", "askSize",
	},
	"Trade": {
		"eventSymbol", "eventTime", "timeNanoPart", "exchangeCode",
		"price", "change", "size", "dayVolume", "dayTurnover",
		"tickDirection", "extendedTradingHours",
	},
	"Summary": {
		"eventSymbol", "dayId", "dayOpenPrice", "dayHighPrice", "dayLowPrice",
		"dayClosePrice", "dayClosePriceType", "prevDayId", "prevDayClosePrice",
		"prevDayClosePriceType", "prevDayVolume", "openInterest",
	},
	"Profile": {
		"eventSymbol", "description", "shortSaleRestriction", "tradingStatus",
		"statusReason", "haltStartTime", "haltEndTime", "highLimitPrice", "lowLimitPrice",
		"high52WeekPrice", "low52WeekPrice", "beta", "earningsPerShare",
		"dividendFrequency", "exDividendAmount", "exDividendDayId", "shares", "freeFloat",
	},
	"Greeks": {
		"eventSymbol", "eventTime", "eventFlags", "index", "time", "sequence",
		"price", "volatility", "delta", "gamma", "theta", "rho", "vega",
	},
}

func probeDxLink(ctx context.Context, client *ttclient.Client, report *Report) error {
	qt, err := fetcher.GetQuoteToken(ctx, client)
	if err != nil {
		return err
	}
	eqOptOCC, eqOptStreamer := discoverEquityOption(ctx, client)
	fut, foOCC, foStreamer := discoverFutureAndFO(ctx, client)
	_ = eqOptOCC
	_ = foOCC

	syms := []string{"SPY"}
	if eqOptStreamer != "" {
		syms = append(syms, eqOptStreamer)
	}
	if fut != "" {
		syms = append(syms, fut)
	}
	if foStreamer != "" {
		syms = append(syms, foStreamer)
	}

	eventTypes := []string{"Quote", "Trade", "Summary", "Profile", "Greeks"}
	probe := &DxLinkProbe{
		URL:           qt.DxLinkURL,
		Subscribed:    syms,
		EventTypes:    eventTypes,
		AcceptFields:  dxAcceptFields,
		EventsSeen:    map[string]int{},
		FieldsByEvent: map[string][]string{},
		Duration:      "12s",
	}
	report.DxLink = probe

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, qt.DxLinkURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	write := func(v any) error { return conn.WriteJSON(v) }
	if err := write(map[string]any{
		"type": "SETUP", "channel": 0, "version": "0.1-DXF-JS/0.3.0",
		"keepaliveTimeout": 60, "acceptKeepaliveTimeout": 60,
	}); err != nil {
		return err
	}

	var (
		mu      sync.Mutex
		samples []json.RawMessage
		ready   bool
	)
	deadline := time.Now().Add(12 * time.Second)
	_ = conn.SetReadDeadline(deadline)

	authorized := false
	for time.Now().Before(deadline) {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			probe.LastError = err.Error()
			break
		}
		var env struct {
			Type  string          `json:"type"`
			State string          `json:"state"`
			Data  json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(msg, &env); err != nil {
			continue
		}
		switch env.Type {
		case "AUTH_STATE":
			if env.State == "UNAUTHORIZED" {
				_ = write(map[string]any{"type": "AUTH", "channel": 0, "token": qt.Token})
			}
			if env.State == "AUTHORIZED" && !authorized {
				authorized = true
				_ = write(map[string]any{
					"type": "CHANNEL_REQUEST", "channel": 1, "service": "FEED",
					"parameters": map[string]any{"contract": "AUTO"},
				})
			}
		case "CHANNEL_OPENED":
			_ = write(map[string]any{
				"type": "FEED_SETUP", "channel": 1,
				"acceptAggregationPeriod": 0.1, "acceptDataFormat": "COMPACT",
				"acceptEventFields": dxAcceptFields,
			})
		case "FEED_CONFIG":
			probe.FeedConfigRaw = append(json.RawMessage(nil), msg...)
			ready = true
			add := make([]map[string]string, 0, len(syms)*len(eventTypes))
			for _, s := range syms {
				for _, t := range eventTypes {
					add = append(add, map[string]string{"type": t, "symbol": s})
				}
			}
			_ = write(map[string]any{"type": "FEED_SUBSCRIPTION", "channel": 1, "add": add})
		case "FEED_DATA":
			if !ready {
				continue
			}
			mu.Lock()
			if len(samples) < 200 {
				samples = append(samples, append(json.RawMessage(nil), msg...))
			}
			ingestDxFields(env.Data, probe)
			mu.Unlock()
		case "ERROR":
			probe.LastError = string(msg)
		}
	}

	samplePath := filepath.Join(outDir, "dxlink_samples.json")
	_ = os.WriteFile(samplePath, pretty(samples), 0o644)
	probe.SampleFile = samplePath
	for et, fields := range probe.FieldsByEvent {
		sort.Strings(fields)
		probe.FieldsByEvent[et] = unique(fields)
	}
	log.Printf("  DxLink events: %v", probe.EventsSeen)
	return nil
}

func ingestDxFields(data json.RawMessage, probe *DxLinkProbe) {
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
	fields := dxAcceptFields[eventType]
	if len(fields) == 0 || len(flat)%len(fields) != 0 {
		probe.EventsSeen[eventType+"#malformed"]++
		return
	}
	probe.EventsSeen[eventType]++
	seen := map[string]bool{}
	for _, f := range probe.FieldsByEvent[eventType] {
		seen[f] = true
	}
	// Record every accepted field name (presence), plus which values were non-null/non-NaN.
	for i := 0; i < len(flat); i += len(fields) {
		for j, name := range fields {
			key := name
			if !isEmptyDx(flat[i+j]) {
				key = name + " (non-empty)"
			}
			if !seen[key] {
				probe.FieldsByEvent[eventType] = append(probe.FieldsByEvent[eventType], key)
				seen[key] = true
			}
			if !seen[name] {
				probe.FieldsByEvent[eventType] = append(probe.FieldsByEvent[eventType], name)
				seen[name] = true
			}
		}
	}
}

func isEmptyDx(v any) bool {
	if v == nil {
		return true
	}
	switch t := v.(type) {
	case string:
		return t == "" || strings.EqualFold(t, "NaN") || t == "Infinity" || t == "-Infinity"
	case float64:
		return false
	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// Account streamer
// ---------------------------------------------------------------------------

func probeAccountStream(ctx context.Context, client *ttclient.Client, cfg *config.Config, report *Report) error {
	token, err := client.AccessToken(ctx)
	if err != nil {
		return err
	}
	probe := &AccountProbe{
		URL:              cfg.AccountStreamerURL,
		Accounts:         cfg.AccountNumbers,
		ControlActions:   []string{},
		NotificationType: map[string]int{},
		FieldPathsByType: map[string][]string{},
		Duration:         "15s",
	}
	report.AccountWS = probe

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, cfg.AccountStreamerURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	auth := token
	if !strings.HasPrefix(auth, "Bearer ") {
		auth = "Bearer " + auth
	}
	if err := conn.WriteJSON(map[string]any{
		"action": "connect", "value": cfg.AccountNumbers, "auth-token": auth, "request-id": 1,
	}); err != nil {
		return err
	}

	var samples []json.RawMessage
	deadline := time.Now().Add(15 * time.Second)
	_ = conn.SetReadDeadline(deadline)
	hbCtx, hbCancel := context.WithCancel(ctx)
	defer hbCancel()
	hb := time.NewTicker(5 * time.Second)
	defer hb.Stop()

	go func() {
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-hb.C:
				_ = conn.WriteJSON(map[string]any{
					"action": "heartbeat", "auth-token": auth, "request-id": time.Now().UnixNano(),
				})
			}
		}
	}()

	for time.Now().Before(deadline) {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			probe.LastError = err.Error()
			break
		}
		if len(samples) < 100 {
			samples = append(samples, append(json.RawMessage(nil), msg...))
		}
		var meta struct {
			Action    string          `json:"action"`
			Status    string          `json:"status"`
			SessionID string          `json:"web-socket-session-id"`
			Type      string          `json:"type"`
			Data      json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(msg, &meta); err != nil {
			continue
		}
		if meta.Action != "" {
			probe.ControlActions = append(probe.ControlActions, meta.Action)
			if meta.SessionID != "" {
				probe.SessionID = meta.SessionID
			}
			if meta.Action == "connect" {
				probe.Connected = true
			}
			continue
		}
		if meta.Type != "" {
			probe.NotificationType[meta.Type]++
			paths := sortedKeys(collectPaths(meta.Data, "data"))
			probe.FieldPathsByType[meta.Type] = mergePaths(probe.FieldPathsByType[meta.Type], paths)
		}
	}
	hbCancel()

	samplePath := filepath.Join(outDir, "account_stream_samples.json")
	_ = os.WriteFile(samplePath, pretty(samples), 0o644)
	probe.SampleFile = samplePath
	probe.ControlActions = unique(probe.ControlActions)
	log.Printf("  account notifications: %v", probe.NotificationType)
	return nil
}

// ---------------------------------------------------------------------------
// JSON helpers
// ---------------------------------------------------------------------------

func collectPaths(v any, prefix string) map[string]struct{} {
	out := map[string]struct{}{}
	var walk func(any, string)
	walk = func(x any, p string) {
		switch t := x.(type) {
		case map[string]any:
			if len(t) == 0 && p != "" {
				out[p] = struct{}{}
			}
			for k, child := range t {
				np := k
				if p != "" {
					np = p + "." + k
				}
				walk(child, np)
			}
		case []any:
			key := p + "[]"
			if p == "" {
				key = "[]"
			}
			if len(t) == 0 {
				out[key] = struct{}{}
				return
			}
			for _, child := range t {
				walk(child, key)
			}
		default:
			if p != "" {
				out[p] = struct{}{}
			}
		}
	}
	// Normalize raw JSON into any.
	switch r := v.(type) {
	case json.RawMessage:
		var anyV any
		if err := json.Unmarshal(r, &anyV); err != nil {
			return out
		}
		walk(anyV, prefix)
	default:
		walk(v, prefix)
	}
	return out
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func mergePaths(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range append(append([]string{}, a...), b...) {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}

func unique(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func pretty(v any) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return []byte("{}")
	}
	return b
}

func urlPath(s string) string { return url.PathEscape(s) }

func productRoot(fut string) string {
	root, _, err := fetcher.FutureProductRoot(fut)
	if err != nil || root == "" {
		return fut
	}
	return root
}

func sanitizePath(p string) string {
	// Keep path shape; truncate very long option symbols in the report path display.
	if len(p) > 180 {
		return p[:180] + "…"
	}
	return p
}

func renderMarkdown(r *Report) string {
	var b strings.Builder
	b.WriteString("# tastytrade live field inventory\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s → %s\n\n", r.StartedAt.Format(time.RFC3339), r.FinishedAt.Format(time.RFC3339)))
	if len(r.Errors) > 0 {
		b.WriteString("## Errors\n\n")
		for _, e := range r.Errors {
			b.WriteString("- " + e + "\n")
		}
		b.WriteString("\n")
	}
	if len(r.Notes) > 0 {
		b.WriteString("## Notes\n\n")
		for _, n := range r.Notes {
			b.WriteString("- " + n + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## REST endpoints\n\n")
	for _, ep := range r.REST {
		status := "OK"
		if !ep.StatusOK {
			status = "FAIL: " + ep.Error
		}
		b.WriteString(fmt.Sprintf("### `%s %s` — %s (%s)\n\n", ep.Method, ep.Path, ep.Name, status))
		if len(ep.FieldPaths) > 0 {
			b.WriteString("Fields:\n\n")
			for _, f := range ep.FieldPaths {
				b.WriteString("- `" + f + "`\n")
			}
			b.WriteString("\n")
		}
		if ep.SampleFile != "" {
			b.WriteString(fmt.Sprintf("Sample: `%s`\n\n", ep.SampleFile))
		}
	}
	if r.DxLink != nil {
		b.WriteString("## DxLink WebSocket\n\n")
		b.WriteString(fmt.Sprintf("- URL: `%s`\n", r.DxLink.URL))
		b.WriteString(fmt.Sprintf("- Symbols: %v\n", r.DxLink.Subscribed))
		b.WriteString(fmt.Sprintf("- Event types requested: %v\n", r.DxLink.EventTypes))
		b.WriteString(fmt.Sprintf("- Events seen: %v\n\n", r.DxLink.EventsSeen))
		for _, et := range r.DxLink.EventTypes {
			fields := r.DxLink.FieldsByEvent[et]
			b.WriteString(fmt.Sprintf("### Event `%s`\n\n", et))
			if len(fields) == 0 {
				b.WriteString("_no events observed_\n\n")
				continue
			}
			for _, f := range fields {
				b.WriteString("- `" + f + "`\n")
			}
			b.WriteString("\n")
		}
	}
	if r.AccountWS != nil {
		b.WriteString("## Account streamer WebSocket\n\n")
		b.WriteString(fmt.Sprintf("- URL: `%s`\n", r.AccountWS.URL))
		b.WriteString(fmt.Sprintf("- Connected: %v session=%s\n", r.AccountWS.Connected, r.AccountWS.SessionID))
		b.WriteString(fmt.Sprintf("- Control actions: %v\n", r.AccountWS.ControlActions))
		b.WriteString(fmt.Sprintf("- Notifications: %v\n\n", r.AccountWS.NotificationType))
		for typ, paths := range r.AccountWS.FieldPathsByType {
			b.WriteString(fmt.Sprintf("### Notification `%s`\n\n", typ))
			for _, f := range paths {
				b.WriteString("- `" + f + "`\n")
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}
