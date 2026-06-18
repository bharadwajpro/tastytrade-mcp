package stream

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// saveToken writes the token as 0600 JSON. Empty path => no-op (caching disabled).
func saveToken(path string, t *DxLinkToken) error {
	if path == "" {
		return nil
	}
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

// loadToken reads a cached token. Empty path or missing file => (nil, nil).
func loadToken(path string) (*DxLinkToken, error) {
	if path == "" {
		return nil, nil
	}
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
