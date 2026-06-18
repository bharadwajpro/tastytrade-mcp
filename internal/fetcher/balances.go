package fetcher

import (
	"context"
	"net/url"
)

type Balance struct {
	AccountNumber       string `json:"account-number"`
	CashBalance         string `json:"cash-balance"`
	NetLiquidatingValue string `json:"net-liquidating-value"`
	BuyingPower         string `json:"equity-buying-power"`
	Currency            string `json:"currency"`
}

func GetBalances(ctx context.Context, d Doer, account string) (*Balance, error) {
	var env struct{ Data Balance `json:"data"` }
	path := "/accounts/" + url.PathEscape(account) + "/balances"
	if err := d.DoJSON(ctx, "GET", path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func GetBalanceByCurrency(ctx context.Context, d Doer, account, currency string) (*Balance, error) {
	var env struct{ Data Balance `json:"data"` }
	path := "/accounts/" + url.PathEscape(account) + "/balances/" + url.PathEscape(currency)
	if err := d.DoJSON(ctx, "GET", path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}
