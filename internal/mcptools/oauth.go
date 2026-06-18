package mcptools

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// oauthGateway implements a minimal OAuth2 authorization-code + PKCE flow that bridges
// MCP clients to this server's static bearer token. It does NOT touch tastytrade's
// OAuth — that is handled server-side by ttclient. Here we only authenticate the MCP
// client connecting to /mcp.
type oauthGateway struct {
	serverURL string
	mcpToken  string

	mu    sync.Mutex
	codes map[string]authCode
}

type authCode struct {
	challenge   string
	redirectURI string
	exp         time.Time
}

func newOAuthGateway(serverURL, mcpToken string) *oauthGateway {
	return &oauthGateway{serverURL: serverURL, mcpToken: mcpToken, codes: map[string]authCode{}}
}

func randToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// authorize issues a one-time authorization code bound to the client's PKCE challenge,
// then redirects back to the client's redirect_uri.
func (g *oauthGateway) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	redirectURI := q.Get("redirect_uri")
	if redirectURI == "" {
		http.Error(w, "missing redirect_uri", http.StatusBadRequest)
		return
	}
	if q.Get("response_type") != "code" {
		http.Error(w, "unsupported response_type (want code)", http.StatusBadRequest)
		return
	}
	challenge := q.Get("code_challenge")
	if challenge == "" || q.Get("code_challenge_method") != "S256" {
		http.Error(w, "PKCE S256 code_challenge required", http.StatusBadRequest)
		return
	}
	code := randToken()
	g.mu.Lock()
	g.codes[code] = authCode{challenge: challenge, redirectURI: redirectURI, exp: time.Now().Add(5 * time.Minute)}
	g.mu.Unlock()

	u, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	rq := u.Query()
	rq.Set("code", code)
	if state := q.Get("state"); state != "" {
		rq.Set("state", state)
	}
	u.RawQuery = rq.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

// token exchanges a valid authorization code + PKCE verifier for the MCP bearer token.
func (g *oauthGateway) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, "invalid_request")
		return
	}
	if r.Form.Get("grant_type") != "authorization_code" {
		writeOAuthError(w, "unsupported_grant_type")
		return
	}
	code := r.Form.Get("code")
	verifier := r.Form.Get("code_verifier")

	g.mu.Lock()
	ac, ok := g.codes[code]
	delete(g.codes, code)
	g.mu.Unlock()

	if !ok || time.Now().After(ac.exp) {
		writeOAuthError(w, "invalid_grant")
		return
	}
	if ru := r.Form.Get("redirect_uri"); ru != "" && ru != ac.redirectURI {
		writeOAuthError(w, "invalid_grant")
		return
	}
	if !verifyPKCE(verifier, ac.challenge) {
		writeOAuthError(w, "invalid_grant")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": g.mcpToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"scope":        "mcp",
	})
}

// register is a permissive Dynamic Client Registration endpoint. The gateway uses a
// single shared bearer, so it accepts any client and echoes back a public client_id.
func (g *oauthGateway) register(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	resp := map[string]any{
		"client_id":                  "tastytrade-mcp",
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code"},
		"response_types":             []string{"code"},
	}
	if v, ok := body["redirect_uris"]; ok {
		resp["redirect_uris"] = v
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func writeOAuthError(w http.ResponseWriter, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}
