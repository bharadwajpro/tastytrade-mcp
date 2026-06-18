package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Gates // embeds ReadOnly + Enable* (task 013)

	ClientSecret, RefreshToken              string
	Environment                             Environment
	APIBaseURL, AccountStreamerURL          string
	AccountNumbers                          []string
	DefaultAccount                          string
	UserAgent, APIVersion                   string
	DxLinkCachePath                         string
	Transport, Port                         string
	MCPAuthToken                            string
	OAuthClientID, OAuthClientSecret        string
	ServerURL                               string
	ToolTimeout, StaleQuoteThreshold        time.Duration
}

// getenv is the injected lookup: returns value (or "" if unset).
type getenv func(string) string

func Load() (*Config, error) {
	_ = godotenv.Load() // best-effort
	return loadFrom(osGetenv)
}

func loadFrom(get getenv) (*Config, error) {
	c := &Config{}
	c.ClientSecret = firstNonEmpty(get("TT_CLIENT_SECRET"), get("client_secret"))
	c.RefreshToken = firstNonEmpty(get("TT_REFRESH_TOKEN"), get("refresh_token"))
	if c.ClientSecret == "" || c.RefreshToken == "" {
		return nil, fmt.Errorf("missing TT_CLIENT_SECRET or TT_REFRESH_TOKEN")
	}

	c.Environment = Environment(firstNonEmpty(get("TT_ENVIRONMENT"), string(EnvProduction)))
	c.APIBaseURL = resolveHost(get("TT_API_BASE_URL"), c.Environment, ProdAPI, SandboxAPI)
	c.AccountStreamerURL = resolveHost(get("TT_ACCOUNT_STREAMER_URL"), c.Environment, ProdStream, SandboxStream)

	c.AccountNumbers = parseAccounts(get("TT_ACCOUNT_NUMBER"), get("TT_ACCOUNT_NUMBERS"))
	if len(c.AccountNumbers) > 0 {
		c.DefaultAccount = c.AccountNumbers[0]
	}

	c.UserAgent = firstNonEmpty(get("TT_USER_AGENT"), "tastytrade-mcp/0.1")
	c.APIVersion = get("TT_API_VERSION")
	c.DxLinkCachePath = get("TT_DXLINK_CACHE_PATH") // "" => disabled

	c.ReadOnly = parseBool(get("TT_READ_ONLY"), false)
	c.EnableTrading = parseBool(get("TT_ENABLE_TRADING"), false)
	c.EnableWatchlistWrites = parseBool(get("TT_ENABLE_WATCHLIST_WRITES"), false)
	c.EnableQuoteAlerts = parseBool(get("TT_ENABLE_QUOTE_ALERTS"), false)

	c.Transport = firstNonEmpty(get("MCP_TRANSPORT"), "http")
	if c.Transport == "http" {
		c.Port = firstNonEmpty(get("PORT"), "8080")
		c.ServerURL = get("SERVER_URL")
		if c.ServerURL == "" {
			return nil, fmt.Errorf("SERVER_URL is required for MCP_TRANSPORT=http")
		}
		c.MCPAuthToken = get("MCP_AUTH_TOKEN")
		c.OAuthClientID = get("OAUTH_CLIENT_ID")
		c.OAuthClientSecret = get("OAUTH_CLIENT_SECRET")
	}

	c.ToolTimeout = parseDuration(get("TT_TOOL_TIMEOUT"), 30*time.Second)
	c.StaleQuoteThreshold = parseDuration(get("TT_STALE_QUOTE_THRESHOLD"), 5*time.Second)
	return c, nil
}
