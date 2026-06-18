package orders

import "fmt"

type Leg struct {
	InstrumentType string `json:"instrument-type"` // "Equity" | "Equity Option" | "Future" | "Future Option"
	Symbol         string `json:"symbol"`
	Action         string `json:"action"` // "Buy to Open" | "Sell to Close" | ...
	Quantity       int    `json:"quantity"`
}

type Order struct {
	TimeInForce string  `json:"time-in-force"`          // "Day" | "GTC"
	OrderType   string  `json:"order-type"`             // "Limit" | "Market"
	Price       float64 `json:"price,omitempty"`
	PriceEffect string  `json:"price-effect,omitempty"` // "Debit" | "Credit"
	Legs        []Leg   `json:"legs"`
}

var validActions = map[string]bool{
	"Buy to Open": true, "Buy to Close": true,
	"Sell to Open": true, "Sell to Close": true,
}

// BuildOrder validates and assembles an Order. Price/PriceEffect are filled later by
// the net-mid pricer (task 091) when OrderType is Limit and price is unset.
func BuildOrder(timeInForce, orderType string, legs []Leg) (Order, error) {
	if len(legs) == 0 {
		return Order{}, fmt.Errorf("order requires at least one leg")
	}
	if timeInForce == "" {
		timeInForce = "Day"
	}
	if orderType == "" {
		orderType = "Limit"
	}
	for i, l := range legs {
		if l.Symbol == "" {
			return Order{}, fmt.Errorf("leg %d: symbol required", i)
		}
		if l.Quantity <= 0 {
			return Order{}, fmt.Errorf("leg %d: quantity must be > 0", i)
		}
		if !validActions[l.Action] {
			return Order{}, fmt.Errorf("leg %d: invalid action %q", i, l.Action)
		}
		if l.InstrumentType == "" {
			return Order{}, fmt.Errorf("leg %d: instrument-type required", i)
		}
	}
	return Order{TimeInForce: timeInForce, OrderType: orderType, Legs: legs}, nil
}
