package mcptools

import "sort"

// Gates is the subset of config needed here (mirror of config.Gates to avoid import
// cycles; the real wiring passes the values in).
type Gates struct {
	ReadOnly              bool
	EnableTrading         bool
	EnableWatchlistWrites bool
	EnableQuoteAlerts     bool
}

func (g Gates) TradingAllowed() bool { return !g.ReadOnly && g.EnableTrading }

// alwaysOn are read-only tools registered regardless of gates.
var alwaysOn = []string{
	"get_equity", "get_market_data", "get_market_metrics", "get_option_chain",
	"is_market_open", "search_symbols",
	"list_accounts", "get_trading_status", "get_balances", "list_positions",
	"list_transactions", "get_margin_requirements",
	"get_order", "list_live_orders", "search_orders", "dry_run_order",
	"list_watchlists",
	"get_account_stream_status", "get_streaming_quote",
}

// registeredTools returns the sorted set of tool names to register.
func registeredTools(g Gates) []string {
	set := map[string]bool{}
	for _, n := range alwaysOn {
		set[n] = true
	}
	if !g.ReadOnly && g.EnableTrading {
		for _, n := range []string{"place_order", "cancel_order", "replace_order", "patch_order", "cancel_complex_order"} {
			set[n] = true
		}
	}
	if !g.ReadOnly && g.EnableWatchlistWrites {
		for _, n := range []string{"create_watchlist", "update_watchlist", "delete_watchlist"} {
			set[n] = true
		}
	}
	if !g.ReadOnly && g.EnableQuoteAlerts {
		for _, n := range []string{"list_quote_alerts", "create_quote_alert", "delete_quote_alert"} {
			set[n] = true
		}
	}
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func has(list []string, name string) bool {
	for _, n := range list {
		if n == name {
			return true
		}
	}
	return false
}
