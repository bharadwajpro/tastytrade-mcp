package mcptools

import (
	"context"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/bharadwajpro/tastytrade-mcp/stream"
)

// Doer is the HTTP dependency every fetcher/service needs (satisfied by *ttclient.Client).
type Doer interface {
	DoJSON(ctx context.Context, method, path string, body, out any) error
}

type Deps struct {
	Gates        Gates
	ToolTimeout  time.Duration
	Accounts     []string
	Version      string
	Client       Doer
	AcctStream   *stream.AccountStreamer
	MarketStream *stream.MarketStreamer
}

// toolReg pairs a tool definition with its handler.
type toolReg struct {
	tool    mcp.Tool
	handler server.ToolHandlerFunc
}

// logTool wraps a handler with a timeout + duration log.
func logTool(timeout time.Duration, name string, h server.ToolHandlerFunc) server.ToolHandlerFunc {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		start := time.Now()
		res, err := h(ctx, req)
		log.Printf("tool=%s dur=%s err=%v", name, time.Since(start), err)
		return res, err
	}
}

// NewServer builds the MCP server, registering exactly the tools allowed by the gates
// (registeredTools is the single gate of truth).
func NewServer(d *Deps) *server.MCPServer {
	s := server.NewMCPServer("tastytrade-mcp", d.Version, server.WithToolCapabilities(false))
	reg := d.registry()
	for _, name := range registeredTools(d.Gates) {
		if r, ok := reg[name]; ok {
			s.AddTool(r.tool, logTool(d.ToolTimeout, name, r.handler))
		}
	}
	return s
}
