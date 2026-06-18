package fetcher

import (
	"context"
	"net/url"
	"strconv"
)

type TxFilter struct {
	Symbol         string
	Type           string
	StartDate      string // YYYY-MM-DD
	EndDate        string
	InstrumentType string
	PerPage        int
	PageOffset     int
}

func buildTxQuery(f TxFilter) string {
	q := url.Values{}
	set := func(k, v string) {
		if v != "" {
			q.Set(k, v)
		}
	}
	set("symbol", f.Symbol)
	set("type", f.Type)
	set("start-date", f.StartDate)
	set("end-date", f.EndDate)
	set("instrument-type", f.InstrumentType)
	if f.PerPage > 0 {
		q.Set("per-page", itoa(f.PerPage))
	}
	if f.PageOffset > 0 {
		q.Set("page-offset", itoa(f.PageOffset))
	}
	return q.Encode()
}

type Transaction struct {
	ID              int64  `json:"id"`
	Symbol          string `json:"symbol"`
	TransactionType string `json:"transaction-type"`
	Value           string `json:"value"`
}

func ListTransactions(ctx context.Context, d Doer, account string, f TxFilter) ([]Transaction, error) {
	var env struct {
		Data struct{ Items []Transaction `json:"items"` } `json:"data"`
	}
	path := "/accounts/" + url.PathEscape(account) + "/transactions"
	if qs := buildTxQuery(f); qs != "" {
		path += "?" + qs
	}
	if err := d.DoJSON(ctx, "GET", path, nil, &env); err != nil {
		return nil, err
	}
	return env.Data.Items, nil
}

func itoa(n int) string { return strconv.Itoa(n) }
