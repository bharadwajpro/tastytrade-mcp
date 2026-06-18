package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bharadwajpro/tastytrade-mcp/internal/config"
	"github.com/bharadwajpro/tastytrade-mcp/internal/fetcher"
	"github.com/bharadwajpro/tastytrade-mcp/internal/mcptools"
	"github.com/bharadwajpro/tastytrade-mcp/internal/stream"
	"github.com/bharadwajpro/tastytrade-mcp/internal/ttclient"
)

const Version = "0.1.0-dev"

func main() {
	if err := run(); err != nil {
		log.Fatalf("tastytrade-mcp: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := ttclient.New(ttclient.Config{
		BaseURL:      cfg.APIBaseURL,
		ClientSecret: cfg.ClientSecret,
		RefreshToken: cfg.RefreshToken,
		UserAgent:    cfg.UserAgent,
		APIVersion:   cfg.APIVersion,
	}, nil)

	// Account streamer (drives get_account_stream_status). Started only when accounts
	// are configured.
	var acctStream *stream.AccountStreamer
	if len(cfg.AccountNumbers) > 0 {
		acctStream = stream.NewAccountStreamer(cfg.AccountStreamerURL, cfg.AccountNumbers,
			func(ctx context.Context) (string, error) { return client.AccessToken(ctx) })
		acctStream.Start(ctx)
	}

	// DxLink market streamer (drives get_streaming_quote).
	marketStream := stream.NewMarketStreamer(func(ctx context.Context) (string, string, error) {
		qt, err := fetcher.GetQuoteToken(ctx, client)
		if err != nil {
			return "", "", err
		}
		return qt.Token, qt.DxLinkURL, nil
	}, cfg.StaleQuoteThreshold)
	marketStream.Start(ctx)

	srv := mcptools.NewServer(&mcptools.Deps{
		Gates: mcptools.Gates{
			ReadOnly:              cfg.ReadOnly,
			EnableTrading:         cfg.EnableTrading,
			EnableWatchlistWrites: cfg.EnableWatchlistWrites,
			EnableQuoteAlerts:     cfg.EnableQuoteAlerts,
		},
		ToolTimeout:  cfg.ToolTimeout,
		Accounts:     cfg.AccountNumbers,
		Version:      Version,
		Client:       client,
		AcctStream:   acctStream,
		MarketStream: marketStream,
	})

	switch cfg.Transport {
	case "stdio":
		return mcptools.ServeStdio(srv)
	default: // "http"
		if cfg.MCPAuthToken == "" {
			log.Printf("warning: MCP_AUTH_TOKEN is empty; /mcp will reject all requests until it is set")
		}
		log.Printf("tastytrade-mcp %s listening on :%s (env=%s, read_only=%v)", Version, cfg.Port, cfg.Environment, cfg.ReadOnly)
		return mcptools.ServeHTTP(ctx, srv, mcptools.HTTPConfig{
			ServerURL:    cfg.ServerURL,
			MCPAuthToken: cfg.MCPAuthToken,
		}, ":"+cfg.Port)
	}
}
