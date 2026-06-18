package ttclient

import "time"

const (
	tokenRefreshBuffer = 60 * time.Second
	tokenFallbackTTL   = 15 * time.Minute
)

// refreshAfter returns how long to wait before refreshing, given expires_in seconds.
//   - expiresIn <= 0      => fallback TTL minus buffer
//   - expiresIn <= buffer => refresh almost immediately (min 1s)
//   - otherwise           => expiresIn - buffer
func refreshAfter(expiresInSeconds int) time.Duration {
	ttl := time.Duration(expiresInSeconds) * time.Second
	if expiresInSeconds <= 0 {
		ttl = tokenFallbackTTL
	}
	d := ttl - tokenRefreshBuffer
	if d < time.Second {
		return time.Second
	}
	return d
}
