package mcptools

import (
	"encoding/json"
	"net/http"
)

type HTTPConfig struct {
	ServerURL         string
	OAuthClientID     string
	OAuthClientSecret string
}

func jsonHandler(doc func() any) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc())
	}
}

// NewMux builds the router. mcpHandler is the StreamableHTTP handler (injected so the
// test can pass a stub).
func NewMux(cfg HTTPConfig, mcpHandler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-protected-resource",
		jsonHandler(func() any { return protectedResourceDoc(cfg.ServerURL) }))
	mux.HandleFunc("/.well-known/oauth-authorization-server",
		jsonHandler(func() any { return authServerMetadataDoc(cfg.ServerURL) }))
	store := newTokenStore()
	mux.Handle("/mcp", bearerMiddleware(store, mcpHandler))

	gw := newOAuthGateway(cfg.ServerURL, store, cfg.OAuthClientID, cfg.OAuthClientSecret)
	mux.HandleFunc("/oauth/authorize", gw.authorize)
	mux.HandleFunc("/oauth/token", gw.token)
	return mux
}

func bearerMiddleware(store *tokenStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !checkBearer(r.Header.Get("Authorization"), store) {
			w.Header().Set("WWW-Authenticate", `Bearer`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
