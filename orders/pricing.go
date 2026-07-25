package orders

import "strings"

type LegQuote struct {
	Action   string // "Buy to Open" etc.
	Bid, Ask float64
	Quantity int
}

type NetMid struct {
	Price       float64 // per-contract net price, always >= 0
	PriceEffect string  // "Debit" | "Credit"
}

func legSign(action string) float64 {
	if strings.HasPrefix(action, "Buy") {
		return -1 // buying spends cash
	}
	return 1 // selling receives cash
}

// NetMidPrice computes the net mid across legs. The price is per one spread unit
// (one copy of each leg), so the divisor is the per-leg quantity (min across legs).
func NetMidPrice(legs []LegQuote) NetMid {
	if len(legs) == 0 {
		return NetMid{Price: 0, PriceEffect: "Debit"}
	}
	var netCash float64
	minQty := legs[0].Quantity
	for _, l := range legs {
		mid := (l.Bid + l.Ask) / 2
		netCash += legSign(l.Action) * mid * float64(l.Quantity)
		if l.Quantity < minQty {
			minQty = l.Quantity
		}
	}
	if minQty == 0 {
		return NetMid{Price: 0, PriceEffect: "Debit"}
	}
	if netCash <= 0 {
		return NetMid{Price: -netCash / float64(minQty), PriceEffect: "Debit"}
	}
	return NetMid{Price: netCash / float64(minQty), PriceEffect: "Credit"}
}
