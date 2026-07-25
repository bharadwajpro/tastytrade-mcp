// Package config loads tastytrade credentials and settings from environment variables.
//
// Load reads TT_* (and optional .env). LoadMCP adds MCP transport/OAuth fields for
// the server binary only — library consumers should use Load alone.
package config
