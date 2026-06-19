package mcptools

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// oauthGateway implements an OAuth2 authorization-code + PKCE flow that issues access and
// refresh tokens for this server's /mcp endpoint. It does NOT touch tastytrade's OAuth —
// that is handled server-side by ttclient. Here we only authenticate the MCP client.
//
// Client authentication is by a single fixed (clientID, clientSecret) pair shared with the
// operator out of band. There is no dynamic client registration: any other client_id or
// client_secret is rejected. Empty configured credentials fail closed (reject all). The
// credentials gate token issuance only; the tokens themselves live in the shared tokenStore.
type oauthGateway struct {
	serverURL    string
	store        *tokenStore
	clientID     string
	clientSecret string

	mu    sync.Mutex
	codes map[string]authCode
}

type authCode struct {
	challenge   string
	redirectURI string
	exp         time.Time
}

func newOAuthGateway(serverURL string, store *tokenStore, clientID, clientSecret string) *oauthGateway {
	return &oauthGateway{
		serverURL:    serverURL,
		store:        store,
		clientID:     clientID,
		clientSecret: clientSecret,
		codes:        map[string]authCode{},
	}
}

func randToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// ctEqual is a constant-time string compare.
func ctEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// clientIDOK validates a client_id (used at the front-channel authorize step, where no
// secret is sent). Empty configured id fails closed.
func (g *oauthGateway) clientIDOK(id string) bool {
	return g.clientID != "" && ctEqual(id, g.clientID)
}

// clientAuthOK validates client credentials at the token endpoint. Credentials may arrive
// via HTTP Basic auth (client_secret_basic) or as POST body params (client_secret_post).
// Both id and secret must match; empty configured credentials fail closed. Both
// comparisons always run so timing does not reveal which field was wrong.
func (g *oauthGateway) clientAuthOK(r *http.Request) bool {
	if g.clientID == "" || g.clientSecret == "" {
		return false
	}
	id, secret, ok := r.BasicAuth()
	if !ok {
		id = r.Form.Get("client_id")
		secret = r.Form.Get("client_secret")
	}
	idOK := ctEqual(id, g.clientID)
	secretOK := ctEqual(secret, g.clientSecret)
	return idOK && secretOK
}

// authorize issues a one-time authorization code bound to the client's PKCE challenge,
// then redirects back to the client's redirect_uri. The client_id is validated here;
// the client_secret is verified later at the token endpoint (never sent in the browser).
func (g *oauthGateway) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if !g.clientIDOK(q.Get("client_id")) {
		http.Error(w, "unknown client_id", http.StatusUnauthorized)
		return
	}
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

// token is the OAuth token endpoint. The client must authenticate with the fixed
// client_id + client_secret, then exchange either an authorization code or a refresh token.
func (g *oauthGateway) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if !g.clientAuthOK(r) {
		w.Header().Set("WWW-Authenticate", `Basic realm="oauth"`)
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client")
		return
	}
	switch r.Form.Get("grant_type") {
	case "authorization_code":
		g.grantAuthorizationCode(w, r)
	case "refresh_token":
		g.grantRefreshToken(w, r)
	default:
		writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type")
	}
}

// grantAuthorizationCode validates a one-time code + PKCE verifier and issues a new
// access+refresh pair.
func (g *oauthGateway) grantAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	code := r.Form.Get("code")
	verifier := r.Form.Get("code_verifier")

	g.mu.Lock()
	ac, ok := g.codes[code]
	delete(g.codes, code)
	g.mu.Unlock()

	if !ok || time.Now().After(ac.exp) {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant")
		return
	}
	if ru := r.Form.Get("redirect_uri"); ru != "" && ru != ac.redirectURI {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant")
		return
	}
	if !verifyPKCE(verifier, ac.challenge) {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant")
		return
	}
	at, rt, expiresIn := g.store.issue()
	writeTokenResponse(w, at, rt, expiresIn)
}

// grantRefreshToken rotates a refresh token (single-use) into a new access+refresh pair.
func (g *oauthGateway) grantRefreshToken(w http.ResponseWriter, r *http.Request) {
	at, rt, expiresIn, ok := g.store.rotate(r.Form.Get("refresh_token"))
	if !ok {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant")
		return
	}
	writeTokenResponse(w, at, rt, expiresIn)
}

func writeTokenResponse(w http.ResponseWriter, accessToken, refreshToken string, expiresIn int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    expiresIn,
		"scope":         "mcp",
	})
}

func writeOAuthError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}
