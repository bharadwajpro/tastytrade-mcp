package stream

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCachedQuoteProviderUsesValidCache(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	exp := time.Now().Add(time.Hour)
	if err := saveCredentialsAt(DxLinkCredentialsFile, &DxLinkToken{
		Token: "cached-tok", DxLinkURL: "wss://example/dxlink", Expiration: exp,
	}); err != nil {
		t.Fatal(err)
	}

	calls := 0
	p := CachedQuoteProvider(func(ctx context.Context) (string, string, error) {
		calls++
		return "fresh-tok", "wss://fresh", nil
	})
	tok, url, err := p(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("fetch called %d times, want 0", calls)
	}
	if tok != "cached-tok" || url != "wss://example/dxlink" {
		t.Fatalf("got (%q, %q), want cached values", tok, url)
	}
}

func TestCachedQuoteProviderFetchesWhenExpired(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := saveCredentialsAt(DxLinkCredentialsFile, &DxLinkToken{
		Token: "old", DxLinkURL: "wss://old", Expiration: time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatal(err)
	}

	calls := 0
	p := CachedQuoteProvider(func(ctx context.Context) (string, string, error) {
		calls++
		return "new-tok", "wss://new", nil
	})
	tok, url, err := p(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("fetch called %d times, want 1", calls)
	}
	if tok != "new-tok" || url != "wss://new" {
		t.Fatalf("got (%q, %q)", tok, url)
	}

	b, err := os.ReadFile(filepath.Join(dir, DxLinkCredentialsFile))
	if err != nil {
		t.Fatal(err)
	}
	var saved DxLinkToken
	if err := json.Unmarshal(b, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Token != "new-tok" || saved.DxLinkURL != "wss://new" {
		t.Fatalf("disk cache = %+v", saved)
	}
	if !saved.Expiration.After(time.Now()) {
		t.Fatalf("expiration not in future: %v", saved.Expiration)
	}
}

func TestCachedQuoteProviderFetchesWhenMissing(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	p := CachedQuoteProvider(func(ctx context.Context) (string, string, error) {
		return "tok", "wss://x", nil
	})
	if _, _, err := p(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(DxLinkCredentialsFile); err != nil {
		t.Fatalf("expected cache file written: %v", err)
	}
}
