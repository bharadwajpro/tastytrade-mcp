package mcptools

import (
	"sync"
	"time"
)

const (
	accessTokenTTL  = time.Hour           // short-lived; a leaked access token dies quickly
	refreshTokenTTL = 30 * 24 * time.Hour // long-lived but single-use (rotated on each refresh)
)

// tokenStore holds the access and refresh tokens issued by the OAuth gateway. Access
// tokens authenticate /mcp requests; refresh tokens mint new access tokens. Tokens are
// opaque, 256-bit random, and kept in memory only — a restart invalidates them and clients
// re-authenticate. There is no static long-lived bearer.
//
// Each completed flow mints an independent pair, so multiple clients sharing one client_id
// each hold their own tokens and are all valid concurrently.
type tokenStore struct {
	mu      sync.Mutex
	access  map[string]time.Time // access token -> expiry
	refresh map[string]time.Time // refresh token -> expiry
}

func newTokenStore() *tokenStore {
	return &tokenStore{
		access:  map[string]time.Time{},
		refresh: map[string]time.Time{},
	}
}

// issue mints a fresh access+refresh pair and returns the access TTL in seconds.
func (s *tokenStore) issue() (accessToken, refreshToken string, expiresIn int) {
	now := time.Now()
	at, rt := randToken(), randToken()
	s.mu.Lock()
	s.gcLocked(now)
	s.access[at] = now.Add(accessTokenTTL)
	s.refresh[rt] = now.Add(refreshTokenTTL)
	s.mu.Unlock()
	return at, rt, int(accessTokenTTL.Seconds())
}

// validAccess reports whether token is a known, unexpired access token. Tokens are
// high-entropy random values, so a map lookup (not constant-time) is not a practical
// guessing oracle. Expired entries are pruned on access.
func (s *tokenStore) validAccess(token string) bool {
	if token == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.access[token]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(s.access, token)
		return false
	}
	return true
}

// rotate consumes a refresh token (single-use) and issues a new access+refresh pair.
// ok is false if the refresh token is unknown or expired; in that case nothing is issued.
func (s *tokenStore) rotate(refreshToken string) (accessToken, newRefresh string, expiresIn int, ok bool) {
	if refreshToken == "" {
		return "", "", 0, false
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, exists := s.refresh[refreshToken]
	if !exists || now.After(exp) {
		delete(s.refresh, refreshToken)
		return "", "", 0, false
	}
	delete(s.refresh, refreshToken) // rotation: the presented refresh token is now spent
	s.gcLocked(now)
	at, rt := randToken(), randToken()
	s.access[at] = now.Add(accessTokenTTL)
	s.refresh[rt] = now.Add(refreshTokenTTL)
	return at, rt, int(accessTokenTTL.Seconds()), true
}

// gcLocked drops expired tokens. Caller must hold s.mu.
func (s *tokenStore) gcLocked(now time.Time) {
	for t, exp := range s.access {
		if now.After(exp) {
			delete(s.access, t)
		}
	}
	for t, exp := range s.refresh {
		if now.After(exp) {
			delete(s.refresh, t)
		}
	}
}
