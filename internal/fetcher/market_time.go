package fetcher

import (
	"context"
	"time"
)

type Session struct {
	State                string    `json:"state"`          // "Open" | "Closed" | "Pre-market" | ...
	InstrumentCollection string    `json:"instrument-collection"`
	OpenAt               time.Time `json:"open-at"`
	CloseAt              time.Time `json:"close-at"`
}

// sessionOpen: true if State=="Open", else fall back to the [OpenAt,CloseAt) window.
func sessionOpen(s Session, now time.Time) bool {
	if s.State == "Open" {
		return true
	}
	if s.State == "Closed" {
		return false
	}
	if s.OpenAt.IsZero() || s.CloseAt.IsZero() {
		return false
	}
	return !now.Before(s.OpenAt) && now.Before(s.CloseAt)
}

func GetCurrentSession(ctx context.Context, d Doer, collection string) (*Session, error) {
	var env struct{ Data Session `json:"data"` }
	path := "/market-time/" + collection + "/sessions/current"
	if err := d.DoJSON(ctx, "GET", path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}
