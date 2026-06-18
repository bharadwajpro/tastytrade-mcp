package fetcher

import (
	"context"
	"net/url"
)

type Customer struct {
	ID        string `json:"id"`
	FirstName string `json:"first-name"`
	LastName  string `json:"last-name"`
}

func GetCustomer(ctx context.Context, d Doer, id string) (*Customer, error) {
	if id == "" {
		id = "me"
	}
	var env struct{ Data Customer `json:"data"` }
	if err := d.DoJSON(ctx, "GET", "/customers/"+url.PathEscape(id), nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

type Account struct {
	AccountNumber string `json:"account-number"`
	Nickname      string `json:"nickname"`
	AccountType   string `json:"account-type-name"`
}

func ListAccounts(ctx context.Context, d Doer, customerID string) ([]Account, error) {
	if customerID == "" {
		customerID = "me"
	}
	var env struct {
		Data struct {
			Items []struct {
				Account Account `json:"account"`
			} `json:"items"`
		} `json:"data"`
	}
	path := "/customers/" + url.PathEscape(customerID) + "/accounts"
	if err := d.DoJSON(ctx, "GET", path, nil, &env); err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(env.Data.Items))
	for _, it := range env.Data.Items {
		out = append(out, it.Account)
	}
	return out, nil
}

type TradingStatus struct {
	AccountNumber  string `json:"account-number"`
	IsClosed       bool   `json:"is-closed"`
	DayTradeCount  int    `json:"day-trade-count"`
	IsInMarginCall bool   `json:"is-in-margin-call"`
}

func GetTradingStatus(ctx context.Context, d Doer, account string) (*TradingStatus, error) {
	var env struct{ Data TradingStatus `json:"data"` }
	path := "/accounts/" + url.PathEscape(account) + "/trading-status"
	if err := d.DoJSON(ctx, "GET", path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}
