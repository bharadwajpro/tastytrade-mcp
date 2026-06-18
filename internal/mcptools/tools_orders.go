package mcptools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/bharadwajpro/tastytrade-mcp/internal/orders"
)

// parseLegs decodes a JSON array of legs. Each element uses the tastytrade dasherized
// keys: instrument-type, symbol, action, quantity.
func parseLegs(s string) ([]orders.Leg, error) {
	if s == "" {
		return nil, fmt.Errorf("legs is required (JSON array)")
	}
	var legs []orders.Leg
	if err := json.Unmarshal([]byte(s), &legs); err != nil {
		return nil, fmt.Errorf("invalid legs JSON: %w", err)
	}
	return legs, nil
}

func buildOrderFromReq(req mcp.CallToolRequest) (orders.Order, error) {
	legs, err := parseLegs(req.GetString("legs", ""))
	if err != nil {
		return orders.Order{}, err
	}
	o, err := orders.BuildOrder(req.GetString("time_in_force", ""), req.GetString("order_type", ""), legs)
	if err != nil {
		return orders.Order{}, err
	}
	if price := req.GetFloat("price", 0); price != 0 {
		o.Price = price
	}
	if eff := req.GetString("price_effect", ""); eff != "" {
		o.PriceEffect = eff
	}
	return o, nil
}

func orderLegArgs() []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithString("legs", mcp.Required(), mcp.Description(`JSON array of legs, e.g. [{"instrument-type":"Equity","symbol":"AAPL","action":"Buy to Open","quantity":1}]`)),
		mcp.WithString("order_type", mcp.Description("Limit (default) or Market")),
		mcp.WithString("time_in_force", mcp.Description("Day (default) or GTC")),
		mcp.WithNumber("price", mcp.Description("Limit price (per unit)")),
		mcp.WithString("price_effect", mcp.Description("Debit or Credit")),
	}
}

func (d *Deps) toolGetOrder() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("get_order",
		mcp.WithDescription("Get a single order by id."),
		mcp.WithReadOnlyHintAnnotation(true),
		accountArg(),
		mcp.WithString("id", mcp.Required(), mcp.Description("Order id")),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		account, errRes := d.acct(req)
		if errRes != nil {
			return errRes, nil
		}
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		o, err := orders.GetOrder(ctx, d.Client, account, id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get_order failed", err), nil
		}
		return jsonOK(o), nil
	}
}

func (d *Deps) toolListLiveOrders() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("list_live_orders",
		mcp.WithDescription("List currently-live orders for an account."),
		mcp.WithReadOnlyHintAnnotation(true),
		accountArg(),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		account, errRes := d.acct(req)
		if errRes != nil {
			return errRes, nil
		}
		o, err := orders.ListLiveOrders(ctx, d.Client, account)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list_live_orders failed", err), nil
		}
		return jsonOK(o), nil
	}
}

func (d *Deps) toolSearchOrders() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("search_orders",
		mcp.WithDescription("Search historical orders by status, underlying symbol, and date range."),
		mcp.WithReadOnlyHintAnnotation(true),
		accountArg(),
		mcp.WithArray("status", mcp.Description("Order statuses, e.g. [\"Live\",\"Filled\"]"), mcp.WithStringItems()),
		mcp.WithString("underlying_symbol", mcp.Description("Filter by underlying symbol")),
		mcp.WithString("start_date", mcp.Description("Start date YYYY-MM-DD")),
		mcp.WithString("end_date", mcp.Description("End date YYYY-MM-DD")),
		mcp.WithNumber("per_page", mcp.Description("Results per page")),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		account, errRes := d.acct(req)
		if errRes != nil {
			return errRes, nil
		}
		f := orders.OrderFilter{
			Status:           req.GetStringSlice("status", nil),
			UnderlyingSymbol: req.GetString("underlying_symbol", ""),
			StartDate:        req.GetString("start_date", ""),
			EndDate:          req.GetString("end_date", ""),
			PerPage:          req.GetInt("per_page", 0),
		}
		o, err := orders.SearchOrders(ctx, d.Client, account, f)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("search_orders failed", err), nil
		}
		return jsonOK(o), nil
	}
}

func (d *Deps) toolDryRunOrder() (mcp.Tool, server.ToolHandlerFunc) {
	opts := append([]mcp.ToolOption{
		mcp.WithDescription("Validate an order and preview its buying-power effect without placing it."),
		mcp.WithReadOnlyHintAnnotation(true),
		accountArg(),
	}, orderLegArgs()...)
	t := mcp.NewTool("dry_run_order", opts...)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		account, errRes := d.acct(req)
		if errRes != nil {
			return errRes, nil
		}
		o, err := buildOrderFromReq(req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := orders.DryRunOrder(ctx, d.Client, account, o)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("dry_run_order failed", err), nil
		}
		return jsonOK(res), nil
	}
}

func (d *Deps) toolPlaceOrder() (mcp.Tool, server.ToolHandlerFunc) {
	opts := append([]mcp.ToolOption{
		mcp.WithDescription("Place an order. Without confirm=true this returns a dry-run preview only; set confirm=true to submit."),
		accountArg(),
		mcp.WithBoolean("confirm", mcp.Description("Set true to actually submit; otherwise a dry-run preview is returned")),
	}, orderLegArgs()...)
	t := mcp.NewTool("place_order", opts...)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		account, errRes := d.acct(req)
		if errRes != nil {
			return errRes, nil
		}
		o, err := buildOrderFromReq(req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		switch orders.DecidePlace(d.Gates.TradingAllowed(), req.GetBool("confirm", false)) {
		case orders.DecisionRejectDisabled:
			return mcp.NewToolResultError("trading is disabled (set TT_ENABLE_TRADING=true and TT_READ_ONLY=false)"), nil
		case orders.DecisionDryRunOnly:
			res, err := orders.DryRunOrder(ctx, d.Client, account, o)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("dry-run failed", err), nil
			}
			return jsonOK(map[string]any{
				"dry_run": true,
				"message": "Preview only. Re-call with confirm=true to submit this order.",
				"preview": res,
				"order":   o,
			}), nil
		default:
			res, err := orders.PlaceOrder(ctx, d.Client, account, o)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("place_order failed", err), nil
			}
			return jsonOK(map[string]any{"submitted": true, "order": res}), nil
		}
	}
}

func (d *Deps) toolReplaceOrder() (mcp.Tool, server.ToolHandlerFunc) {
	opts := append([]mcp.ToolOption{
		mcp.WithDescription("Replace a live order with a new one (PUT)."),
		accountArg(),
		mcp.WithString("id", mcp.Required(), mcp.Description("Order id to replace")),
	}, orderLegArgs()...)
	t := mcp.NewTool("replace_order", opts...)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		account, errRes := d.acct(req)
		if errRes != nil {
			return errRes, nil
		}
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		o, err := buildOrderFromReq(req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := orders.ReplaceOrder(ctx, d.Client, account, id, o)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("replace_order failed", err), nil
		}
		return jsonOK(res), nil
	}
}

func (d *Deps) toolPatchOrder() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("patch_order",
		mcp.WithDescription("Apply a partial update (e.g. price) to a live order (PATCH)."),
		accountArg(),
		mcp.WithString("id", mcp.Required(), mcp.Description("Order id to patch")),
		mcp.WithString("patch", mcp.Required(), mcp.Description(`JSON object of fields to update, e.g. {"price":1.25,"price-effect":"Debit"}`)),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		account, errRes := d.acct(req)
		if errRes != nil {
			return errRes, nil
		}
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		patchStr, err := req.RequireString("patch")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var patch map[string]any
		if err := json.Unmarshal([]byte(patchStr), &patch); err != nil {
			return mcp.NewToolResultErrorf("invalid patch JSON: %v", err), nil
		}
		res, err := orders.PatchOrder(ctx, d.Client, account, id, patch)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("patch_order failed", err), nil
		}
		return jsonOK(res), nil
	}
}

func (d *Deps) toolCancelOrder() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("cancel_order",
		mcp.WithDescription("Cancel a live order by id."),
		accountArg(),
		mcp.WithString("id", mcp.Required(), mcp.Description("Order id to cancel")),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		account, errRes := d.acct(req)
		if errRes != nil {
			return errRes, nil
		}
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := orders.CancelOrder(ctx, d.Client, account, id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("cancel_order failed", err), nil
		}
		return jsonOK(res), nil
	}
}

func (d *Deps) toolCancelComplexOrder() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("cancel_complex_order",
		mcp.WithDescription("Cancel a live complex order (OTOCO/OCO) by id."),
		accountArg(),
		mcp.WithString("id", mcp.Required(), mcp.Description("Complex order id to cancel")),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		account, errRes := d.acct(req)
		if errRes != nil {
			return errRes, nil
		}
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := orders.CancelComplexOrder(ctx, d.Client, account, id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("cancel_complex_order failed", err), nil
		}
		return jsonOK(res), nil
	}
}
