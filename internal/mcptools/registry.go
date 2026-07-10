package mcptools

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registry returns every tool this server can expose, keyed by name. NewServer then
// registers only the subset allowed by the gates (registeredTools).
func (d *Deps) registry() map[string]toolReg {
	m := map[string]toolReg{}
	add := func(t mcp.Tool, h server.ToolHandlerFunc) { m[t.Name] = toolReg{tool: t, handler: h} }

	// Market / instrument (read-only)
	add(d.toolGetEquity())
	add(d.toolGetMarketData())
	add(d.toolGetMarketMetrics())
	add(d.toolGetOptionChain())
	add(d.toolIsMarketOpen())
	add(d.toolSearchSymbols())
	add(d.toolSelectContractByDelta())
	add(d.toolSelectContractByWidth())

	// Account (read-only)
	add(d.toolListAccounts())
	add(d.toolGetTradingStatus())
	add(d.toolGetBalances())
	add(d.toolListPositions())
	add(d.toolListTransactions())
	add(d.toolGetMarginRequirements())

	// Orders (read-only)
	add(d.toolGetOrder())
	add(d.toolListLiveOrders())
	add(d.toolSearchOrders())
	add(d.toolDryRunOrder())

	// Orders (gated by trading)
	add(d.toolPlaceOrder())
	add(d.toolReplaceOrder())
	add(d.toolPatchOrder())
	add(d.toolCancelOrder())
	add(d.toolCancelComplexOrder())

	// Watchlists (list is read-only; create/update/delete gated)
	add(d.toolListWatchlists())
	add(d.toolCreateWatchlist())
	add(d.toolUpdateWatchlist())
	add(d.toolDeleteWatchlist())

	// Quote alerts (list gated by same flag; create/delete gated)
	add(d.toolListQuoteAlerts())
	add(d.toolCreateQuoteAlert())
	add(d.toolDeleteQuoteAlert())

	// Streaming (read-only)
	add(d.toolAccountStreamStatus())
	add(d.toolStreamingQuote())

	return m
}
