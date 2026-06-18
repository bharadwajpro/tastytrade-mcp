package config

// Gates holds the safety flags (subset of the full Config; embed in Config later).
type Gates struct {
	ReadOnly              bool
	EnableTrading         bool
	EnableWatchlistWrites bool
	EnableQuoteAlerts     bool
}

// WritesAllowed is false whenever the master kill-switch is on.
func (g Gates) WritesAllowed() bool { return !g.ReadOnly }

func (g Gates) TradingAllowed() bool         { return !g.ReadOnly && g.EnableTrading }
func (g Gates) WatchlistWritesAllowed() bool { return !g.ReadOnly && g.EnableWatchlistWrites }
func (g Gates) QuoteAlertsAllowed() bool     { return !g.ReadOnly && g.EnableQuoteAlerts }
