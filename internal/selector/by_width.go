package selector

import "math"

// ByWidth finds the contract closest to (anchorStrike + signedWidth).
// signedWidth may be negative (lower strike). Empty input => (Contract{}, false).
func ByWidth(contracts []Contract, anchorStrike, signedWidth float64) (Contract, bool) {
	target := anchorStrike + signedWidth
	best := Contract{}
	found := false
	bestDiff := math.Inf(1)
	for _, c := range contracts {
		diff := math.Abs(c.Strike - target)
		if diff < bestDiff {
			best, bestDiff, found = c, diff, true
		}
	}
	return best, found
}
