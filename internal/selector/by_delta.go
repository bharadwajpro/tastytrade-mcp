package selector

import "math"

// Contract is the minimal option contract the selectors operate on.
type Contract struct {
	Symbol string  // streamer or OCC symbol
	Strike float64
	Delta  float64 // signed (calls +, puts -)
}

// ByDelta returns the contract whose |delta| is nearest to targetAbsDelta.
// Ties break toward the lower strike (deterministic). Empty input => (Contract{}, false).
func ByDelta(contracts []Contract, targetAbsDelta float64) (Contract, bool) {
	best := Contract{}
	found := false
	bestDiff := math.Inf(1)
	for _, c := range contracts {
		diff := math.Abs(math.Abs(c.Delta) - targetAbsDelta)
		if diff < bestDiff || (diff == bestDiff && c.Strike < best.Strike) {
			best, bestDiff, found = c, diff, true
		}
	}
	return best, found
}
