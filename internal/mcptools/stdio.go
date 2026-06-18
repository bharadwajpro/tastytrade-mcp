package mcptools

import (
	"io"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
)

// configureLogging points the standard logger at w (os.Stderr in main).
func configureLogging(w io.Writer) {
	log.SetOutput(w)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

// ServeStdio serves the MCP protocol over stdin/stdout. Diagnostics are routed to
// stderr so they never corrupt the protocol stream on stdout.
func ServeStdio(s *server.MCPServer) error {
	configureLogging(os.Stderr)
	return server.ServeStdio(s)
}
