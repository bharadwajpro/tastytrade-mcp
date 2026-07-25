package fetcher

import "strconv"

type Greeks struct {
	Delta, Gamma, Theta, Vega float64
}

// PositionGreek pairs a position's signed exposure inputs with its per-contract greeks.
type PositionGreek struct {
	QuantityStr string // e.g. "2"
	Direction   string // "Long" | "Short"
	Multiplier  int    // 100 for options
	Greeks      Greeks // per-contract greeks (ok=false legs excluded by caller)
	Missing     bool   // greeks unavailable for this leg
}

// AggregateGreeks sums signed greeks across legs. Returns the net and a count of legs
// skipped for missing greeks.
func AggregateGreeks(legs []PositionGreek) (net Greeks, missing int) {
	for _, l := range legs {
		if l.Missing {
			missing++
			continue
		}
		qty, _ := strconv.ParseFloat(l.QuantityStr, 64)
		sign := 1.0
		if l.Direction == "Short" {
			sign = -1.0
		}
		f := qty * float64(l.Multiplier) * sign
		net.Delta += l.Greeks.Delta * f
		net.Gamma += l.Greeks.Gamma * f
		net.Theta += l.Greeks.Theta * f
		net.Vega += l.Greeks.Vega * f
	}
	return net, missing
}
