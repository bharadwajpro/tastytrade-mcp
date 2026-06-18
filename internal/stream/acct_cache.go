package stream

import "encoding/json"

type AcctCache struct {
	Orders    map[string]json.RawMessage // key: order id
	Balances  map[string]json.RawMessage // key: account-number
	Positions map[string]json.RawMessage // key: account-number|symbol
}

func NewAcctCache() *AcctCache {
	return &AcctCache{
		Orders:    map[string]json.RawMessage{},
		Balances:  map[string]json.RawMessage{},
		Positions: map[string]json.RawMessage{},
	}
}

type notification struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// route updates the cache from one raw notification frame. Returns the type handled
// (or "" if ignored/unknown). Unknown types are ignored, not errors.
func (c *AcctCache) route(raw []byte) (string, error) {
	var n notification
	if err := json.Unmarshal(raw, &n); err != nil {
		return "", err
	}
	switch n.Type {
	case "Order":
		var d struct {
			ID json.Number `json:"id"`
		}
		if err := json.Unmarshal(n.Data, &d); err != nil {
			return "", err
		}
		c.Orders[d.ID.String()] = n.Data
	case "AccountBalance":
		var d struct {
			Acct string `json:"account-number"`
		}
		if err := json.Unmarshal(n.Data, &d); err != nil {
			return "", err
		}
		c.Balances[d.Acct] = n.Data
	case "CurrentPosition":
		var d struct {
			Acct   string `json:"account-number"`
			Symbol string `json:"symbol"`
		}
		if err := json.Unmarshal(n.Data, &d); err != nil {
			return "", err
		}
		c.Positions[d.Acct+"|"+d.Symbol] = n.Data
	default:
		return "", nil // ignore unknown
	}
	return n.Type, nil
}
