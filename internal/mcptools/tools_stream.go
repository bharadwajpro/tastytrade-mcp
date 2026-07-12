package mcptools

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (d *Deps) toolAccountStreamStatus() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("get_account_stream_status",
		mcp.WithDescription("Get the health of the account streamer (connection, last heartbeat, cached notification counts)."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if d.AcctStream == nil {
			return jsonOK(map[string]any{"enabled": false, "message": "account streamer not running"}), nil
		}
		return jsonOK(d.AcctStream.Status()), nil
	}
}

func (d *Deps) toolStreamingQuote() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("get_streaming_quote",
		mcp.WithDescription("Subscribe to and return the latest live DxLink bid/ask for symbols. Waits briefly for first events after a new subscription."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithArray("symbols", mcp.Required(), mcp.Description("DxLink streamer symbols (e.g. SPY, or option streamer symbols)"), mcp.WithStringItems()),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if d.MarketStream == nil {
			return mcp.NewToolResultError("market streamer not running"), nil
		}
		syms := req.GetStringSlice("symbols", nil)
		if len(syms) == 0 {
			return mcp.NewToolResultError("symbols is required"), nil
		}
		d.MarketStream.Subscribe(syms)
		waitCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
		d.MarketStream.WaitForQuotes(waitCtx, syms)
		cancel()
		now := time.Now()
		out := make([]map[string]any, 0, len(syms))
		for _, s := range syms {
			q, found, stale := d.MarketStream.Quote(s, now)
			out = append(out, map[string]any{
				"symbol": s, "found": found, "stale": stale,
				"bid": q.Bid, "ask": q.Ask, "updated_at": q.UpdatedAt,
			})
		}
		return jsonOK(map[string]any{"quotes": out, "status": d.MarketStream.Status()}), nil
	}
}
