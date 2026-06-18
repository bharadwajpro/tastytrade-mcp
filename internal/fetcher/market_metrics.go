package fetcher

import (
	"context"
	"net/url"
	"strings"
)

type MarketMetric struct {
	Symbol                      string `json:"symbol"`
	ImpliedVolatilityRank       string `json:"implied-volatility-index-rank"`
	ImpliedVolatilityPercentile string `json:"implied-volatility-percentile"`
	Beta                        string `json:"beta"`
	Liquidity                   string `json:"liquidity-rating"`
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
