package stream

import "time"

type DxLinkToken struct {
	Token      string    `json:"token"`
	DxLinkURL  string    `json:"dxlink_url"`
	Expiration time.Time `json:"expiration"`
}

// valid reports whether the token is non-empty, has a URL, and is unexpired at now.
func (t *DxLinkToken) valid(now time.Time) bool {
	if t == nil || t.Token == "" || t.DxLinkURL == "" {
		return false
	}
	return now.Before(t.Expiration)
}
