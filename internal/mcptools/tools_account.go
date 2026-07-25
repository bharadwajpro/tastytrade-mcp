package mcptools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/bharadwajpro/tastytrade-mcp/fetcher"
)

func (d *Deps) toolListAccounts() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("list_accounts",
		mcp.WithDescription("List the accounts available to the authenticated customer."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		accts, err := fetcher.ListAccounts(ctx, d.Client, "me")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list_accounts failed", err), nil
		}
		return jsonOK(map[string]any{"configured": d.Accounts, "accounts": accts}), nil
	}
}

func (d *Deps) toolGetTradingStatus() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("get_trading_status",
		mcp.WithDescription("Get the trading status (permissions, day-trade count, margin-call flag) for an account."),
		mcp.WithReadOnlyHintAnnotation(true),
		accountArg(),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		account, errRes := d.acct(req)
		if errRes != nil {
			return errRes, nil
		}
		ts, err := fetcher.GetTradingStatus(ctx, d.Client, account)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get_trading_status failed", err), nil
		}
		return jsonOK(ts), nil
	}
}

func (d *Deps) toolGetBalances() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("get_balances",
		mcp.WithDescription("Get cash, net liquidating value, and buying power for an account."),
		mcp.WithReadOnlyHintAnnotation(true),
		accountArg(),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		account, errRes := d.acct(req)
		if errRes != nil {
			return errRes, nil
		}
		bal, err := fetcher.GetBalances(ctx, d.Client, account)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get_balances failed", err), nil
		}
		return jsonOK(bal), nil
	}
}

func (d *Deps) toolListPositions() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("list_positions",
		mcp.WithDescription("List open positions for an account, including streamer-symbol and P/L fields (average-open-price, close-price, realized-day-gain)."),
		mcp.WithReadOnlyHintAnnotation(true),
		accountArg(),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		account, errRes := d.acct(req)
		if errRes != nil {
			return errRes, nil
		}
		pos, err := fetcher.ListPositions(ctx, d.Client, account)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list_positions failed", err), nil
		}
		return jsonOK(pos), nil
	}
}

func (d *Deps) toolListTransactions() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("list_transactions",
		mcp.WithDescription("List account transactions, optionally filtered by symbol/type/date range."),
		mcp.WithReadOnlyHintAnnotation(true),
		accountArg(),
		mcp.WithString("symbol", mcp.Description("Filter by symbol")),
		mcp.WithString("type", mcp.Description("Filter by transaction type")),
		mcp.WithString("start_date", mcp.Description("Start date YYYY-MM-DD")),
		mcp.WithString("end_date", mcp.Description("End date YYYY-MM-DD")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		account, errRes := d.acct(req)
		if errRes != nil {
			return errRes, nil
		}
		f := fetcher.TxFilter{
			Symbol:    req.GetString("symbol", ""),
			Type:      req.GetString("type", ""),
			StartDate: req.GetString("start_date", ""),
			EndDate:   req.GetString("end_date", ""),
			PerPage:   req.GetInt("per_page", 0),
		}
		txs, err := fetcher.ListTransactions(ctx, d.Client, account, f)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list_transactions failed", err), nil
		}
		return jsonOK(txs), nil
	}
}

func (d *Deps) toolGetMarginRequirements() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("get_margin_requirements",
		mcp.WithDescription("Get the margin requirements report for an account."),
		mcp.WithReadOnlyHintAnnotation(true),
		accountArg(),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		account, errRes := d.acct(req)
		if errRes != nil {
			return errRes, nil
		}
		report, err := fetcher.GetMarginRequirements(ctx, d.Client, account)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get_margin_requirements failed", err), nil
		}
		return jsonOK(report), nil
	}
}
