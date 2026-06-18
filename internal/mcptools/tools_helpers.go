package mcptools

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// acct resolves the effective account from the request, returning an error result if
// resolution fails.
func (d *Deps) acct(req mcp.CallToolRequest) (string, *mcp.CallToolResult) {
	a, err := resolveAccount(d.Accounts, req.GetString("account", ""))
	if err != nil {
		return "", mcp.NewToolResultError(err.Error())
	}
	return a, nil
}

// jsonOK renders v as a pretty-JSON text result.
func jsonOK(v any) *mcp.CallToolResult {
	return mcp.NewToolResultText(marshalResult(v))
}

// accountArg is the shared optional account parameter (required only when multiple
// accounts are configured).
func accountArg() mcp.ToolOption {
	return mcp.WithString("account", mcp.Description("Account number (required only if multiple accounts are configured)"))
}
