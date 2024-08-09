package coinbase

import (
	"context"
	"fmt"
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
)

const (
	getCandlesEndpoint = "/products/%s/candles"
)

func (c *Client) GetCandles(
	ctx context.Context,
	currencyPair string,
	granularity string,
	start int64,
	end int64,
) ([]indexer.CoinBaseHistoricalExchangeRate, error) {
	queryParams := map[string]string{
		"granularity": granularity,
		"start":       time.Unix(start, 0).UTC().Format(time.RFC3339),
		"end":         time.Unix(end, 0).UTC().Format(time.RFC3339),
	}

	endpoint := fmt.Sprintf(getCandlesEndpoint, currencyPair)
	var rawData [][]interface{}
	err := c.MakeRequest(ctx, "GET", "application/json", endpoint, queryParams, nil, &rawData)
	if err != nil {
		return nil, err
	}

	// Process the raw data into the desired format
	var rates []indexer.CoinBaseHistoricalExchangeRate
	for _, candle := range rawData {
		if len(candle) < 5 {
			return nil, fmt.Errorf("incomplete candle data")
		}

		var rate indexer.CoinBaseHistoricalExchangeRate
		if err := rate.Scan(candle, currencyPair); err != nil {
			return nil, err
		}

		rates = append(rates, rate)
	}

	return rates, nil
}
