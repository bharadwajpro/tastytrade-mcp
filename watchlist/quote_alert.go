package watchlist

import "fmt"

type QuoteAlertBody struct {
	Symbol    string `json:"symbol"`
	Field     string `json:"field"`     // "bid" | "ask" | "last"
	Operator  string `json:"operator"`  // ">" | "<" | ">=" | "<="
	Threshold string `json:"threshold"` // decimal as string
}

var validOps = map[string]bool{">": true, "<": true, ">=": true, "<=": true}
var validFields = map[string]bool{"bid": true, "ask": true, "last": true}

func BuildQuoteAlert(symbol, field, operator, threshold string) (QuoteAlertBody, error) {
	if symbol == "" {
		return QuoteAlertBody{}, fmt.Errorf("symbol required")
	}
	if !validFields[field] {
		return QuoteAlertBody{}, fmt.Errorf("invalid field %q (bid|ask|last)", field)
	}
	if !validOps[operator] {
		return QuoteAlertBody{}, fmt.Errorf("invalid operator %q", operator)
	}
	if threshold == "" {
		return QuoteAlertBody{}, fmt.Errorf("threshold required")
	}
	return QuoteAlertBody{Symbol: symbol, Field: field, Operator: operator, Threshold: threshold}, nil
}
