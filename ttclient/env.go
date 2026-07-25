package ttclient

import (
	"fmt"
	"net/http"

	"github.com/bharadwajpro/tastytrade-mcp/config"
)

// NewFromEnv loads tastytrade credentials via config.Load (TT_* env + optional .env)
// and returns an authenticated Client. Pass an explicit Config to New instead when
// you want to bypass or override environment loading.
func NewFromEnv() (*Client, error) {
	return NewFromEnvHTTP(nil)
}

// NewFromEnvHTTP is like NewFromEnv but accepts a custom *http.Client (nil = default).
func NewFromEnvHTTP(httpClient *http.Client) (*Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("ttclient: %w", err)
	}
	return New(Config{
		BaseURL:      cfg.APIBaseURL,
		ClientSecret: cfg.ClientSecret,
		RefreshToken: cfg.RefreshToken,
		UserAgent:    cfg.UserAgent,
		APIVersion:   cfg.APIVersion,
	}, httpClient), nil
}
