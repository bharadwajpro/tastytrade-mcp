package ttclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// httpDoer is the minimal HTTP dependency (satisfied by *http.Client).
type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Config carries the values Client needs (from config.Load or set explicitly).
type Config struct {
	BaseURL      string // e.g. https://api.tastyworks.com
	ClientSecret string
	RefreshToken string
	UserAgent    string
	APIVersion   string // optional Api-Version header (YYYYMMDD)
}

// Client is an authenticated tastytrade REST client. It manages the OAuth2
// refresh-token grant: it exchanges the long-lived refresh token for a short-lived
// (15-minute) access token and refreshes it automatically before expiry.
type Client struct {
	cfg  Config
	http httpDoer

	mu       sync.Mutex
	token    string
	tokenExp time.Time // when the cached access token should be refreshed
}

// New builds a Client. A nil httpClient defaults to a 30s-timeout http.Client.
func New(cfg Config, httpClient *http.Client) *Client {
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	var d httpDoer = httpClient
	if httpClient == nil {
		d = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{cfg: cfg, http: d}
}

// AccessToken returns a valid access token, refreshing it if necessary.
func (c *Client) AccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExp) {
		return c.token, nil
	}
	if err := c.refreshLocked(ctx); err != nil {
		return "", err
	}
	return c.token, nil
}

// refreshLocked performs the refresh-token grant. Caller must hold c.mu.
func (c *Client) refreshLocked(ctx context.Context) error {
	if c.cfg.ClientSecret == "" || c.cfg.RefreshToken == "" {
		return fmt.Errorf("ttclient: missing client secret or refresh token")
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", c.cfg.RefreshToken)
	form.Set("client_secret", c.cfg.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", c.cfg.BaseURL+"/oauth/token",
		strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	setCommonHeaders(req, c.cfg.UserAgent, c.cfg.APIVersion)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ttclient: token request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !IsSuccess(resp.StatusCode) {
		return HTTPError(resp.StatusCode, body)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return fmt.Errorf("ttclient: decode token response: %w", err)
	}
	if tok.AccessToken == "" {
		return fmt.Errorf("ttclient: empty access_token in token response")
	}
	c.token = tok.AccessToken
	c.tokenExp = time.Now().Add(refreshAfter(tok.ExpiresIn))
	return nil
}

// DoJSON performs an authenticated request. body (if non-nil) is JSON-encoded; the
// response is decoded into out (if non-nil). Non-2xx responses become rich errors.
func (c *Client) DoJSON(ctx context.Context, method, path string, body, out any) error {
	token, err := c.AccessToken(ctx)
	if err != nil {
		return err
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("ttclient: marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.cfg.BaseURL+path, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	setCommonHeaders(req, c.cfg.UserAgent, c.cfg.APIVersion)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ttclient: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if !IsSuccess(resp.StatusCode) {
		return HTTPError(resp.StatusCode, respBody)
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("ttclient: decode %s %s response: %w", method, path, err)
	}
	return nil
}
