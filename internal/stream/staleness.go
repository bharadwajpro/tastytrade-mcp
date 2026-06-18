package stream

import "time"

// Quote is the cached market quote with a receive timestamp.
type Quote struct {
	Symbol    string    `json:"symbol"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	UpdatedAt time.Time `json:"updated_at"`
}

// isStale reports whether the quote is older than threshold at now.
// A zero UpdatedAt (never received) is always stale.
func isStale(q Quote, now time.Time, threshold time.Duration) bool {
	if q.UpdatedAt.IsZero() {
		return true
	}
	return now.Sub(q.UpdatedAt) > threshold
}
