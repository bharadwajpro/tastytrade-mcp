package watchlist

import (
	"context"
	"net/url"
)

// Doer is the minimal HTTP dependency (satisfied by *ttclient.Client).
type Doer interface {
	DoJSON(ctx context.Context, method, path string, body, out any) error
}

// ListWatchlists returns the user's watchlists.
func ListWatchlists(ctx context.Context, d Doer) ([]map[string]any, error) {
	var env struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := d.DoJSON(ctx, "GET", "/watchlists", nil, &env); err != nil {
		return nil, err
	}
	return env.Data.Items, nil
}

// GetWatchlist fetches a single watchlist by name.
func GetWatchlist(ctx context.Context, d Doer, name string) (map[string]any, error) {
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := d.DoJSON(ctx, "GET", "/watchlists/"+url.PathEscape(name), nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// CreateWatchlist creates a new watchlist.
func CreateWatchlist(ctx context.Context, d Doer, b Body) (map[string]any, error) {
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := d.DoJSON(ctx, "POST", "/watchlists", b, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// UpdateWatchlist replaces an existing watchlist (keyed by its current name).
func UpdateWatchlist(ctx context.Context, d Doer, name string, b Body) (map[string]any, error) {
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := d.DoJSON(ctx, "PUT", "/watchlists/"+url.PathEscape(name), b, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// DeleteWatchlist removes a watchlist by name.
func DeleteWatchlist(ctx context.Context, d Doer, name string) error {
	return d.DoJSON(ctx, "DELETE", "/watchlists/"+url.PathEscape(name), nil, nil)
}

// ListQuoteAlerts returns the user's configured quote alerts.
func ListQuoteAlerts(ctx context.Context, d Doer) ([]map[string]any, error) {
	var env struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := d.DoJSON(ctx, "GET", "/quote-alerts", nil, &env); err != nil {
		return nil, err
	}
	return env.Data.Items, nil
}

// CreateQuoteAlert creates a user-level quote alert.
func CreateQuoteAlert(ctx context.Context, d Doer, b QuoteAlertBody) (map[string]any, error) {
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := d.DoJSON(ctx, "POST", "/quote-alerts", b, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// DeleteQuoteAlert removes a quote alert by its external id.
func DeleteQuoteAlert(ctx context.Context, d Doer, externalID string) error {
	return d.DoJSON(ctx, "DELETE", "/quote-alerts/"+url.PathEscape(externalID), nil, nil)
}
