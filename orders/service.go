package orders

import (
	"context"
	"net/url"
)

// Doer is the minimal HTTP dependency (satisfied by *ttclient.Client).
type Doer interface {
	DoJSON(ctx context.Context, method, path string, body, out any) error
}

func acctPath(account, suffix string) string {
	return "/accounts/" + url.PathEscape(account) + suffix
}

// GetOrder fetches a single order by id.
func GetOrder(ctx context.Context, d Doer, account, id string) (map[string]any, error) {
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := d.DoJSON(ctx, "GET", acctPath(account, "/orders/"+url.PathEscape(id)), nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// ListLiveOrders returns the account's currently-live orders.
func ListLiveOrders(ctx context.Context, d Doer, account string) ([]map[string]any, error) {
	var env struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := d.DoJSON(ctx, "GET", acctPath(account, "/orders/live"), nil, &env); err != nil {
		return nil, err
	}
	return env.Data.Items, nil
}

// SearchOrders returns historical orders matching the filter.
func SearchOrders(ctx context.Context, d Doer, account string, f OrderFilter) ([]map[string]any, error) {
	path := acctPath(account, "/orders")
	if qs := BuildOrderSearchQuery(f); qs != "" {
		path += "?" + qs
	}
	var env struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := d.DoJSON(ctx, "GET", path, nil, &env); err != nil {
		return nil, err
	}
	return env.Data.Items, nil
}

// DryRunOrder submits an order to the dry-run endpoint, which validates it and
// returns buying-power effect / warnings without placing it.
func DryRunOrder(ctx context.Context, d Doer, account string, o Order) (map[string]any, error) {
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := d.DoJSON(ctx, "POST", acctPath(account, "/orders/dry-run"), o, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// PlaceOrder submits a live order.
func PlaceOrder(ctx context.Context, d Doer, account string, o Order) (map[string]any, error) {
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := d.DoJSON(ctx, "POST", acctPath(account, "/orders"), o, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// ReplaceOrder replaces a live order with a new one (PUT).
func ReplaceOrder(ctx context.Context, d Doer, account, id string, o Order) (map[string]any, error) {
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := d.DoJSON(ctx, "PUT", acctPath(account, "/orders/"+url.PathEscape(id)), o, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// PatchOrder applies a partial update (e.g. price) to a live order (PATCH).
func PatchOrder(ctx context.Context, d Doer, account, id string, patch map[string]any) (map[string]any, error) {
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := d.DoJSON(ctx, "PATCH", acctPath(account, "/orders/"+url.PathEscape(id)), patch, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// CancelOrder cancels a live order.
func CancelOrder(ctx context.Context, d Doer, account, id string) (map[string]any, error) {
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := d.DoJSON(ctx, "DELETE", acctPath(account, "/orders/"+url.PathEscape(id)), nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// CancelComplexOrder cancels a live complex order (OTOCO/OCO).
func CancelComplexOrder(ctx context.Context, d Doer, account, id string) (map[string]any, error) {
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := d.DoJSON(ctx, "DELETE", acctPath(account, "/complex-orders/"+url.PathEscape(id)), nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}
