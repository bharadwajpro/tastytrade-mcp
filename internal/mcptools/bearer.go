package mcptools

import (
	"crypto/subtle"
	"strings"
)

// checkBearer returns true iff the header is "Bearer <expected>" (constant-time).
// An empty expected token always fails (fail closed).
func checkBearer(authHeader, expected string) bool {
	if expected == "" {
		return false
	}
	const p = "Bearer "
	if !strings.HasPrefix(authHeader, p) {
		return false
	}
	got := strings.TrimSpace(authHeader[len(p):])
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}
