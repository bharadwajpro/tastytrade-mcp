package stream

import (
	"context"
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

func TestAuthTokenForUnauthorizedMidSessionRefreshes(t *testing.T) {
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
	m.mu.Unlock()

	tok, err := m.authTokenForUnauthorized(context.Background(), "stale-session-tok")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("fetch called %d times, want 1", calls)
	}
	if tok != "new-tok" {
		t.Fatalf("got %q, want new-tok", tok)
	}
	cached, err := loadCredentialsAt(DxLinkCredentialsFile)
	if err != nil {
		t.Fatal(err)
	}
	if cached == nil || cached.Token != "new-tok" || cached.DxLinkURL != "wss://new" {
		t.Fatalf("disk cache = %+v", cached)
	}

	st := m.Status()
	if st.Authorized || st.FeedReady {
		t.Fatalf("status after refresh: %+v", st)
	}
}
