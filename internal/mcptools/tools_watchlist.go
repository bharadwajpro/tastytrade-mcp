package mcptools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/bharadwajpro/tastytrade-mcp/internal/watchlist"
)

func parseEntries(s string) ([]watchlist.Entry, error) {
	if s == "" {
		return nil, fmt.Errorf("entries is required (JSON array)")
	}
	var entries []watchlist.Entry
	if err := json.Unmarshal([]byte(s), &entries); err != nil {
		return nil, fmt.Errorf("invalid entries JSON: %w", err)
	}
	return entries, nil
}

func (d *Deps) toolListWatchlists() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("list_watchlists",
		mcp.WithDescription("List the user's watchlists."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		wls, err := watchlist.ListWatchlists(ctx, d.Client)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list_watchlists failed", err), nil
		}
		return jsonOK(wls), nil
	}
}

func (d *Deps) toolCreateWatchlist() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("create_watchlist",
		mcp.WithDescription("Create a new watchlist."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Watchlist name")),
		mcp.WithString("group_name", mcp.Description("Optional group name")),
		mcp.WithString("entries", mcp.Required(), mcp.Description(`JSON array, e.g. [{"symbol":"AAPL","instrument-type":"Equity"}]`)),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		entries, err := parseEntries(req.GetString("entries", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		body, err := watchlist.BuildBody(name, req.GetString("group_name", ""), entries)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := watchlist.CreateWatchlist(ctx, d.Client, body)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("create_watchlist failed", err), nil
		}
		return jsonOK(res), nil
	}
}

func (d *Deps) toolUpdateWatchlist() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("update_watchlist",
		mcp.WithDescription("Replace an existing watchlist (keyed by its current name)."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Existing watchlist name")),
		mcp.WithString("group_name", mcp.Description("Optional group name")),
		mcp.WithString("entries", mcp.Required(), mcp.Description(`JSON array of {"symbol","instrument-type"}`)),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		entries, err := parseEntries(req.GetString("entries", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		body, err := watchlist.BuildBody(name, req.GetString("group_name", ""), entries)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := watchlist.UpdateWatchlist(ctx, d.Client, name, body)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("update_watchlist failed", err), nil
		}
		return jsonOK(res), nil
	}
}

func (d *Deps) toolDeleteWatchlist() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("delete_watchlist",
		mcp.WithDescription("Delete a watchlist by name."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Watchlist name to delete")),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := watchlist.DeleteWatchlist(ctx, d.Client, name); err != nil {
			return mcp.NewToolResultErrorFromErr("delete_watchlist failed", err), nil
		}
		return jsonOK(map[string]any{"deleted": name}), nil
	}
}

func (d *Deps) toolListQuoteAlerts() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("list_quote_alerts",
		mcp.WithDescription("List the user's configured quote alerts."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		alerts, err := watchlist.ListQuoteAlerts(ctx, d.Client)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list_quote_alerts failed", err), nil
		}
		return jsonOK(alerts), nil
	}
}

func (d *Deps) toolCreateQuoteAlert() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("create_quote_alert",
		mcp.WithDescription("Create a user-level quote alert."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Symbol to watch")),
		mcp.WithString("field", mcp.Required(), mcp.Description("bid | ask | last")),
		mcp.WithString("operator", mcp.Required(), mcp.Description("> | < | >= | <=")),
		mcp.WithString("threshold", mcp.Required(), mcp.Description("Threshold value (decimal string)")),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		body, err := watchlist.BuildQuoteAlert(
			req.GetString("symbol", ""), req.GetString("field", ""),
			req.GetString("operator", ""), req.GetString("threshold", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := watchlist.CreateQuoteAlert(ctx, d.Client, body)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("create_quote_alert failed", err), nil
		}
		return jsonOK(res), nil
	}
}

func (d *Deps) toolDeleteQuoteAlert() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("delete_quote_alert",
		mcp.WithDescription("Delete a quote alert by its external id."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Quote alert external id")),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := watchlist.DeleteQuoteAlert(ctx, d.Client, id); err != nil {
			return mcp.NewToolResultErrorFromErr("delete_quote_alert failed", err), nil
		}
		return jsonOK(map[string]any{"deleted": id}), nil
	}
}
