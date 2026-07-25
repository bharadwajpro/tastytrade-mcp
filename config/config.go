package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Gates // embeds ReadOnly + Enable*

	ClientSecret, RefreshToken       string
	Environment                      Environment
	APIBaseURL, AccountStreamerURL   string
	AccountNumbers                   []string
	DefaultAccount                   string
	UserAgent, APIVersion            string
	ToolTimeout, StaleQuoteThreshold time.Duration

	// MCP server fields — filled by LoadMCP, unused by library consumers.
	Transport, Port, ListenHost      string
	OAuthClientID, OAuthClientSecret string
	ServerURL                        string
}

// getenv is the injected lookup: returns value (or "" if unset).
type getenv func(string) string

// Load reads tastytrade credentials and settings from the process environment
// (best-effort .env in the working directory via godotenv). It does not require
// MCP transport variables — use LoadMCP for those.
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

	c.ReadOnly = parseBool(get("TT_READ_ONLY"), false)
	c.EnableTrading = parseBool(get("TT_ENABLE_TRADING"), false)
	c.EnableWatchlistWrites = parseBool(get("TT_ENABLE_WATCHLIST_WRITES"), false)
	c.EnableQuoteAlerts = parseBool(get("TT_ENABLE_QUOTE_ALERTS"), false)

	c.ToolTimeout = parseDuration(get("TT_TOOL_TIMEOUT"), 30*time.Second)
	c.StaleQuoteThreshold = parseDuration(get("TT_STALE_QUOTE_THRESHOLD"), 5*time.Second)
	return c, nil
}

// LoadMCP fills MCP transport / OAuth fields on an existing Config (from Load).
// Call this only from the MCP server binary — library consumers do not need it.
func LoadMCP(c *Config) error {
	return loadMCPFrom(c, osGetenv)
}

func loadMCPFrom(c *Config, get getenv) error {
	if c == nil {
		return fmt.Errorf("config: LoadMCP requires a non-nil Config")
	}
	c.Transport = firstNonEmpty(get("MCP_TRANSPORT"), "http")
	c.ListenHost = get("LISTEN_HOST")
	if c.Transport == "http" {
		c.Port = firstNonEmpty(get("PORT"), "8080")
		c.ServerURL = get("SERVER_URL")
		if c.ServerURL == "" {
			return fmt.Errorf("SERVER_URL is required for MCP_TRANSPORT=http")
		}
		c.OAuthClientID = get("OAUTH_CLIENT_ID")
		c.OAuthClientSecret = get("OAUTH_CLIENT_SECRET")
	}
	return nil
}
