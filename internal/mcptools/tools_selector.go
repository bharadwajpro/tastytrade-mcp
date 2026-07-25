package mcptools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/bharadwajpro/tastytrade-mcp/selector"
)

type selectedContractResult struct {
	Contract selector.Contract `json:"contract"`
	Target   float64           `json:"target"`
}

func parseSelectorContracts(s string) ([]selector.Contract, error) {
	if s == "" {
		return nil, fmt.Errorf("contracts is required (JSON array)")
	}
	var contracts []selector.Contract
	if err := json.Unmarshal([]byte(s), &contracts); err != nil {
		return nil, fmt.Errorf("invalid contracts JSON: %w", err)
	}
	if len(contracts) == 0 {
		return nil, fmt.Errorf("contracts must contain at least one contract")
	}
	return contracts, nil
}

func (d *Deps) toolSelectContractByDelta() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("select_contract_by_delta",
		mcp.WithDescription("Select the contract whose absolute delta is nearest to the target absolute delta."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("contracts", mcp.Required(), mcp.Description(`JSON array of {"symbol","strike","delta"} contracts`)),
		mcp.WithNumber("target_abs_delta", mcp.Required(), mcp.Description("Target absolute delta, e.g. 0.30")),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		contracts, err := parseSelectorContracts(req.GetString("contracts", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		target := req.GetFloat("target_abs_delta", 0)
		best, ok := selector.ByDelta(contracts, target)
		if !ok {
			return mcp.NewToolResultError("no contracts provided"), nil
		}
		return jsonOK(selectedContractResult{Contract: best, Target: target}), nil
	}
}

func (d *Deps) toolSelectContractByWidth() (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("select_contract_by_width",
		mcp.WithDescription("Select the contract whose strike is nearest to anchor_strike + signed_width."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("contracts", mcp.Required(), mcp.Description(`JSON array of {"symbol","strike","delta"} contracts; delta may be zero if unused`)),
		mcp.WithNumber("anchor_strike", mcp.Required(), mcp.Description("Anchor strike")),
		mcp.WithNumber("signed_width", mcp.Required(), mcp.Description("Signed width from anchor strike; negative selects lower strikes")),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		contracts, err := parseSelectorContracts(req.GetString("contracts", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		anchor := req.GetFloat("anchor_strike", 0)
		width := req.GetFloat("signed_width", 0)
		target := anchor + width
		best, ok := selector.ByWidth(contracts, anchor, width)
		if !ok {
			return mcp.NewToolResultError("no contracts provided"), nil
		}
		return jsonOK(selectedContractResult{Contract: best, Target: target}), nil
	}
}
