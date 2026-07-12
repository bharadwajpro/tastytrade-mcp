package stream

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"
)

// DxLinkCredentialsFile is the fixed path (relative to the process working directory)
// where a cached DxLink quote token is stored between restarts.
const DxLinkCredentialsFile = ".dxlink-credentials"

// dxLinkTokenTTL matches the tastytrade api-quote-token lifetime (24h).
const dxLinkTokenTTL = 24 * time.Hour

// CachedQuoteProvider wraps fetch in a disk-backed cache. On each call it reads
// DxLinkCredentialsFile when the stored token is still valid; otherwise it calls
// fetch, writes the new token + expiration, and returns it.
func CachedQuoteProvider(fetch func(ctx context.Context) (token, dxlinkURL string, err error)) QuoteProvider {
	return func(ctx context.Context) (string, string, error) {
		now := time.Now()
		if t, err := loadCredentials(); err == nil && t != nil && t.valid(now) {
			return t.Token, t.DxLinkURL, nil
		} else if err != nil {
			log.Printf("dxlink cache: read %s: %v (fetching fresh token)", DxLinkCredentialsFile, err)
		}
		token, url, err := fetch(ctx)
		if err != nil {
			return "", "", err
		}
		if err := saveCredentials(&DxLinkToken{
			Token: token, DxLinkURL: url, Expiration: now.Add(dxLinkTokenTTL),
		}); err != nil {
			log.Printf("dxlink cache: write %s: %v", DxLinkCredentialsFile, err)
		}
		return token, url, nil
	}
}

func loadCredentials() (*DxLinkToken, error) {
	return loadCredentialsAt(DxLinkCredentialsFile)
}

func saveCredentials(t *DxLinkToken) error {
	return saveCredentialsAt(DxLinkCredentialsFile, t)
}

// invalidateCredentials removes the on-disk DxLink token cache so the next
// provider call fetches a fresh api-quote-token from tastytrade.
func invalidateCredentials() error {
	return invalidateCredentialsAt(DxLinkCredentialsFile)
}

func invalidateCredentialsAt(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func loadCredentialsAt(path string) (*DxLinkToken, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var t DxLinkToken
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func saveCredentialsAt(path string, t *DxLinkToken) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
