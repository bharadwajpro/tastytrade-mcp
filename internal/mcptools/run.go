package mcptools

import (
	"context"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// ServeHTTP runs the MCP server over Streamable HTTP behind the OAuth/bearer mux.
// It blocks until the server stops or ctx is cancelled.
//
// Streaming GET (SSE) is disabled on purpose: the MCP Python SDK's
// OAuthClientProvider holds a process-wide auth lock across the entire HTTP
// response. A long-lived GET SSE never completes that response, so the lock
// is never released and later POSTs (tools/list, tool calls) stall forever.
// Hermes/MCP clients still work over POST request/response without GET SSE.
func ServeHTTP(ctx context.Context, s *server.MCPServer, cfg HTTPConfig, addr string) error {
	streamHandler := server.NewStreamableHTTPServer(s,
		server.WithEndpointPath("/mcp"),
		server.WithDisableStreaming(true),
	)
	mux := NewMux(cfg, streamHandler)
	httpSrv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
