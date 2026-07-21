package stream

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAuthTokenForUnauthorizedInitialHandshake(t *testing.T) {
	calls := 0
	m := NewMarketStreamer(func(ctx context.Context) (string, string, error) {
		calls++
		return "fresh", "wss://x", nil
	}, 0)

	tok, err := m.authTokenForUnauthorized(context.Background(), "session-tok")
	if err != nil {
		t.Fatal(err)
	}
	if tok != "session-tok" {
		t.Fatalf("got %q, want session-tok", tok)
	}
	if calls != 0 {
		t.Fatalf("provider called %d times, want 0", calls)
	}
}

func TestAuthTokenForUnauthorizedMidSessionReconnects(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := saveCredentialsAt(DxLinkCredentialsFile, &DxLinkToken{
		Token: "cached", DxLinkURL: "wss://old", Expiration: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	calls := 0
	provider := CachedQuoteProvider(func(ctx context.Context) (string, string, error) {
		calls++
		return "new-tok", "wss://new", nil
	})
	m := NewMarketStreamer(provider, 0)
	m.mu.Lock()
	m.status.Authorized = true
	m.ready = true
	m.status.FeedReady = true
	m.mu.Unlock()

	tok, err := m.authTokenForUnauthorized(context.Background(), "stale-session-tok")
	if !errors.Is(err, errMidSessionUnauthorized) {
		t.Fatalf("err = %v, want errMidSessionUnauthorized", err)
	}
	if tok != "" {
		t.Fatalf("got token %q, want empty", tok)
	}
	if calls != 0 {
		t.Fatalf("provider called %d times during handle; want 0 (fetch happens on next session)", calls)
	}
	cached, err := loadCredentialsAt(DxLinkCredentialsFile)
	if err != nil {
		t.Fatal(err)
	}
	if cached != nil {
		t.Fatalf("disk cache should be invalidated, got %+v", cached)
	}

	st := m.Status()
	if st.Authorized || st.FeedReady {
		t.Fatalf("status after mid-session UNAUTHORIZED: %+v", st)
	}
}
