package mcptools

import "strings"

// checkBearer returns true iff the Authorization header is "Bearer <token>" where token is
// a known, unexpired access token in the store. Any other value (including none) fails.
func checkBearer(authHeader string, store *tokenStore) bool {
	const p = "Bearer "
	if !strings.HasPrefix(authHeader, p) {
		return false
	}
	return store.validAccess(strings.TrimSpace(authHeader[len(p):]))
}
