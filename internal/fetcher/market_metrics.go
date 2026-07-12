package fetcher

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
)

type MarketMetric struct {
	Symbol                      string      `json:"symbol"`
	ImpliedVolatilityRank       string      `json:"implied-volatility-index-rank"`
	ImpliedVolatilityPercentile string      `json:"implied-volatility-percentile"`
	Beta                        string      `json:"beta"`
	Liquidity                   json.Number `json:"liquidity-rating"` // API returns a number (e.g. 4), not a string
}

func GetMarketMetrics(ctx context.Context, d Doer, symbols []string) ([]MarketMetric, error) {
	q := url.Values{}
	q.Set("symbols", strings.Join(symbols, ","))
	var env struct {
		Data struct{ Items []MarketMetric `json:"items"` } `json:"data"`
	}
	if err := d.DoJSON(ctx, "GET", "/market-metrics?"+q.Encode(), nil, &env); err != nil {
		return nil, err
	}
	return env.Data.Items, nil
}
