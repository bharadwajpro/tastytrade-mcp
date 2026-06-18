package fetcher

import (
	"context"
	"net/url"
)

type MarginGroup struct {
	Description       string `json:"description"`
	MarginRequirement string `json:"margin-requirement"`
	Code              string `json:"code"`
}

type MarginReport struct {
	AccountNumber     string        `json:"account-number"`
	MarginRequirement string        `json:"margin-requirement"`
	MarginEquity      string        `json:"margin-equity"`
	Groups            []MarginGroup `json:"groups"`
}

func GetMarginRequirements(ctx context.Context, d Doer, account string) (*MarginReport, error) {
	var env struct{ Data MarginReport `json:"data"` }
	path := "/margin/accounts/" + url.PathEscape(account) + "/requirements"
	if err := d.DoJSON(ctx, "GET", path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}
