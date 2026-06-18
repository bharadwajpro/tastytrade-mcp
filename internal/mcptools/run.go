package mcptools

import (
	"context"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// ServeHTTP runs the MCP server over Streamable HTTP behind the OAuth/bearer mux.
// It blocks until the server stops or ctx is cancelled.
func ServeHTTP(ctx context.Context, s *server.MCPServer, cfg HTTPConfig, addr string) error {
	streamHandler := server.NewStreamableHTTPServer(s, server.WithEndpointPath("/mcp"))
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
