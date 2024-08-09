package coinbase

import (
	"context"
	"fmt"
	"time"
)

const (
	getCandlesEndpoint = "/products/%s/candles"
	candleTimeIndex    = 0
	candleLowIndex     = 1
	candleHighIndex    = 2
	candleOpenIndex    = 3
	candleCloseIndex   = 4
)

type CoinBaseHistoricalExchangeRate struct {
	Time         time.Time `json:"time"`
	Low          float64   `json:"low"`
	High         float64   `json:"high"`
	Open         float64   `json:"open"`
	Close        float64   `json:"close"`
	CurrencyPair string    `json:"currencyPair"`
}

func (c *CoinBaseHistoricalExchangeRate) Scan(
	candle []interface{},
	currencyPair string) error {
	unixTime, ok := candle[candleTimeIndex].(float64)
	if !ok {
		return fmt.Errorf("failed to parse unix time")
	}
	c.Time = time.Unix(int64(unixTime), 0).UTC()

	low, ok := candle[candleLowIndex].(float64)
	if !ok {
		return fmt.Errorf("failed to parse low")
	}
	c.Low = low

	high, ok := candle[candleHighIndex].(float64)
	if !ok {
		return fmt.Errorf("failed to parse low")
	}
	c.High = high

	open, ok := candle[candleOpenIndex].(float64)
	if !ok {
		return fmt.Errorf("failed to parse low")
	}
	c.Open = open

	close, ok := candle[candleCloseIndex].(float64)
	if !ok {
		return fmt.Errorf("failed to parse low")
	}
	c.Close = close

	c.CurrencyPair = currencyPair
	return nil
}

func (c *Client) GetCandles(
	ctx context.Context,
	currencyPair string,
	granularity string,
	start int64,
	end int64,
) ([]CoinBaseHistoricalExchangeRate, error) {
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
	var rates []CoinBaseHistoricalExchangeRate
	for _, candle := range rawData {
		if len(candle) < 5 {
			return nil, fmt.Errorf("incomplete candle data")
		}

		var rate CoinBaseHistoricalExchangeRate
		if err := rate.Scan(candle, currencyPair); err != nil {
			return nil, err
		}

		rates = append(rates, rate)
	}

	return rates, nil
}
