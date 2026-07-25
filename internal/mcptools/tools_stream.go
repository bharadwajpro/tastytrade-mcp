package mcptools

import (
	"context"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/bharadwajpro/tastytrade-mcp/fetcher"
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
		mcp.WithDescription("Subscribe to DxLink Quote/Trade/Summary/Profile/Greeks and return the merged cached snapshot for each symbol. Accepts DxLink streamer symbols (e.g. SPY, /ESU6:XCME, ./OGV26C4340:XCEC) or trade symbols (e.g. /GCV6, ./GCV6 OGV6  260924C4340) which are resolved automatically. Use streamer-symbol from list_positions when available."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithArray("symbols", mcp.Required(), mcp.Description("Symbols to quote (streamer or trade format)"), mcp.WithStringItems()),
		accountArg(),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if d.MarketStream == nil {
			return mcp.NewToolResultError("market streamer not running"), nil
		}
		syms := req.GetStringSlice("symbols", nil)
		if len(syms) == 0 {
			return mcp.NewToolResultError("symbols is required"), nil
		}

		posMap := map[string]string{}
		if account, errRes := d.acct(req); errRes == nil {
			if pos, err := fetcher.ListPositions(ctx, d.Client, account); err == nil {
				posMap = fetcher.StreamerSymbolsFromPositions(pos)
			}
		}

		type resolvedSym struct {
			requested string
			streamer  string
		}
		resolved := make([]resolvedSym, len(syms))
		streamerSyms := make([]string, len(syms))
		for i, s := range syms {
			streamer := s
			if ss, ok := posMap[s]; ok {
				streamer = ss
			} else if rs, err := fetcher.ResolveStreamerSymbol(ctx, d.Client, s); err == nil {
				streamer = rs
			}
			resolved[i] = resolvedSym{requested: s, streamer: streamer}
			streamerSyms[i] = streamer
		}

		d.MarketStream.Subscribe(streamerSyms)
		waitCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
		d.MarketStream.WaitForQuotes(waitCtx, streamerSyms)
		cancel()
		now := time.Now()
		st := d.MarketStream.Status()
		out := make([]map[string]any, 0, len(resolved))
		missing := 0
		for _, r := range resolved {
			q, found, stale := d.MarketStream.Quote(r.streamer, now)
			if !found {
				missing++
			}
			out = append(out, map[string]any{
				"symbol":          r.requested,
				"streamer_symbol": r.streamer,
				"found":           found,
				"stale":           stale,
				"data":            q,
			})
		}
		if missing > 0 || !st.FeedReady {
			log.Printf("dxlink: get_streaming_quote missing=%d/%d feed_ready=%v connected=%v authorized=%v cached_quotes=%d last_error=%q",
				missing, len(resolved), st.FeedReady, st.Connected, st.Authorized, st.CachedQuotes, st.LastError)
		}
		return jsonOK(map[string]any{"quotes": out, "status": st}), nil
	}
}
