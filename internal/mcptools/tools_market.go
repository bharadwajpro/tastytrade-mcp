package mcptools

import (
	"context"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/bharadwajpro/tastytrade-mcp/internal/fetcher"
)

func (d *Deps) toolGetEquity() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("get_equity",
		mcp.WithDescription("Get instrument details for an equity by ticker symbol."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Equity ticker, e.g. AAPL")),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sym, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		eq, err := fetcher.GetEquity(ctx, d.Client, sym)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get_equity failed", err), nil
		}
		return jsonOK(eq), nil
	}
}

func (d *Deps) toolGetMarketData() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("get_market_data",
		mcp.WithDescription("Get synchronous market quotes for symbols grouped by instrument type (full /market-data/by-type fields: bid/ask/sizes, mid/mark/last, OHLC, volume, halt flags, etc.)."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithArray("equities", mcp.Description("Equity symbols"), mcp.WithStringItems()),
		mcp.WithArray("equity_options", mcp.Description("Equity option (OCC) symbols"), mcp.WithStringItems()),
		mcp.WithArray("futures", mcp.Description("Future symbols, e.g. /ESU6"), mcp.WithStringItems()),
		mcp.WithArray("future_options", mcp.Description("Future option symbols"), mcp.WithStringItems()),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		r := fetcher.MarketDataRequest{
			Equities:      req.GetStringSlice("equities", nil),
			EquityOptions: req.GetStringSlice("equity_options", nil),
			Futures:       req.GetStringSlice("futures", nil),
			FutureOptions: req.GetStringSlice("future_options", nil),
		}
		if len(r.Equities)+len(r.EquityOptions)+len(r.Futures)+len(r.FutureOptions) == 0 {
			return mcp.NewToolResultError("provide at least one symbol in equities/equity_options/futures/future_options"), nil
		}
		quotes, err := fetcher.GetMarketData(ctx, d.Client, r)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get_market_data failed", err), nil
		}
		return jsonOK(quotes), nil
	}
}

func (d *Deps) toolGetMarketMetrics() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("get_market_metrics",
		mcp.WithDescription("Get volatility metrics (IV rank/percentile, beta, liquidity) for symbols."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithArray("symbols", mcp.Required(), mcp.Description("Underlying symbols"), mcp.WithStringItems()),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		syms := req.GetStringSlice("symbols", nil)
		if len(syms) == 0 {
			return mcp.NewToolResultError("symbols is required"), nil
		}
		metrics, err := fetcher.GetMarketMetrics(ctx, d.Client, syms)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get_market_metrics failed", err), nil
		}
		return jsonOK(metrics), nil
	}
}

func (d *Deps) toolGetOptionChain() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("get_option_chain",
		mcp.WithDescription("Get the nested option chain (expirations and strikes). Equity tickers (SPY) return OCC chains. Futures product roots (/CL) return all futures-option series; dated futures (/CLZ6) return only series on that future (multiple roots/weeklies included when listed)."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Equity ticker (SPY), futures root (/CL), or dated future (/CLZ6)")),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sym, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		chain, err := fetcher.GetOptionChain(ctx, d.Client, sym)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get_option_chain failed", err), nil
		}
		return jsonOK(chain), nil
	}
}

func (d *Deps) toolIsMarketOpen() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("is_market_open",
		mcp.WithDescription("Report whether the equities market is currently open."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		status, err := fetcher.IsMarketOpen(ctx, d.Client, "equities", time.Now())
		if err != nil {
			return mcp.NewToolResultErrorFromErr("is_market_open failed", err), nil
		}
		return jsonOK(status), nil
	}
}

func (d *Deps) toolSearchSymbols() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("search_symbols",
		mcp.WithDescription("Search for tradeable symbols matching a query fragment."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search fragment, e.g. APP")),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if strings.TrimSpace(q) == "" {
			return mcp.NewToolResultError("query is required"), nil
		}
		matches, err := fetcher.SearchSymbols(ctx, d.Client, q)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("search_symbols failed", err), nil
		}
		return jsonOK(matches), nil
	}
}
