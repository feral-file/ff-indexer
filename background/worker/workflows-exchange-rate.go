package worker

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
	cadenceclient "go.uber.org/cadence/client"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

func (w *NFTIndexerWorker) CrawlHistoricalExchangeRate(
	ctx workflow.Context,
	currencyPairs []string,
	granularity string,
	start int64,
	end int64,
) error {
	log := workflow.GetLogger(ctx)
	log.Debug("start CrawlHistoricalExchangeRate")

	supportedCurrencyPairs := []string{
		"ETH-USD",
		"XTZ-USD",
	}
	// Check if all currencyPairs is supported
	for _, currencyPair := range currencyPairs {
		if !indexer.ArrayContains(supportedCurrencyPairs, currencyPair) {
			log.Error("unsupported currency pair", zap.String("currencyPair", currencyPair))
			return nil
		}
	}

	supportedGranularity := []string{
		"60", "300", "900", "3600", "21600", "86400",
	}
	// Check if granularity is supported
	if !indexer.ArrayContains(supportedGranularity, granularity) {
		log.Error("unsupported granularity", zap.String("granularity", granularity))
		return nil
	}

	granularityInt, err := strconv.Atoi(granularity)
	if err != nil {
		log.Error("Invalid granularity", zap.Error(err))
		return nil
	}

	startTime := time.Unix(start, 0).Unix()
	endTime := time.Unix(end, 0).Unix()

	if startTime > endTime {
		log.Error("Start must be before end")
		return nil
	}

	for i := startTime; i < endTime; i += int64(300 * granularityInt) {
		_startTime := i
		_endTime := i + int64(300*granularityInt)
		if _endTime > endTime {
			_endTime = endTime
		}

		cwo := workflow.ChildWorkflowOptions{
			ExecutionStartToCloseTimeout: 30 * time.Minute,
			WorkflowID:                   fmt.Sprintf("CrawlExchangeRate-%d-%d-%s", _startTime, _endTime, granularity),
			WorkflowIDReusePolicy:        cadenceclient.WorkflowIDReusePolicyAllowDuplicate,
		}
		ctxwo := workflow.WithChildOptions(ctx, cwo)
		if err := workflow.ExecuteChildWorkflow(
			ctxwo,
			w.CrawlExchangeRate,
			currencyPairs,
			granularity,
			_startTime,
			_endTime).Get(ctx, nil); err != nil {

			log.Error("Failed to crawl exchange rate", zap.Error(err))
			return err
		}
	}

	return nil
}

func (w *NFTIndexerWorker) CrawlExchangeRate(
	ctx workflow.Context,
	currencyPairs []string,
	granularity string,
	start int64,
	end int64,
) error {
	log := workflow.GetLogger(ctx)
	log.Debug("start CrawlExchangeRate")

	for _, currencyPair := range currencyPairs {
		cwo := workflow.ChildWorkflowOptions{
			ExecutionStartToCloseTimeout: 5 * time.Minute,
			WorkflowID:                   fmt.Sprintf("CrawlExchangeRateByCurrencyPair-%s-%d-%d-%s", currencyPair, start, end, granularity),
			WorkflowIDReusePolicy:        cadenceclient.WorkflowIDReusePolicyAllowDuplicate,
		}
		ctxwo := workflow.WithChildOptions(ctx, cwo)
		if err := workflow.ExecuteChildWorkflow(
			ctxwo,
			w.CrawlExchangeRateByCurrencyPair,
			currencyPair,
			granularity,
			start,
			end).Get(ctx, nil); err != nil {

			log.Error("Failed to crawl exchange rate", zap.Error(err))
			return err
		}
	}

	return nil
}

func (w *NFTIndexerWorker) CrawlExchangeRateByCurrencyPair(
	ctx workflow.Context,
	currencyPair string,
	granularity string,
	start int64,
	end int64,
) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
	}
	log := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, ao)
	log.Debug("start CrawlExchangeRateByCurrencyPair")

	var rates []indexer.CoinBaseHistoricalExchangeRate

	if err := workflow.ExecuteActivity(
		ctx,
		w.CrawlExchangeRateFromCoinbase,
		currencyPair,
		granularity,
		start,
		end).Get(ctx, &rates); err != nil {
		log.Error("Failed to crawl exchange rate", zap.Error(err))
		return err
	}

	if len(rates) == 0 {
		return nil
	}

	if err := workflow.ExecuteActivity(
		ctx,
		w.WriteHistoricalExchangeRate,
		rates).Get(ctx, nil); err != nil {
		log.Error("Failed to write exchange rate", zap.Error(err))
		return err
	}

	return nil
}

func (w *NFTIndexerWorker) CrawlExchangeRateFromCoinbase(
	ctx workflow.Context,
	currencyPair string,
	granularity string,
	start int64,
	end int64,
) ([]indexer.CoinBaseHistoricalExchangeRate, error) {
	log := workflow.GetLogger(ctx)
	log.Debug("start CrawlExchangeRateFromCoinbase")

	url := fmt.Sprintf("https://api.exchange.coinbase.com/products/%s/candles?granularity=%s&start=%s&end=%s", currencyPair, granularity, time.Unix(start, 0).Format(time.RFC3339), time.Unix(end, 0).Format(time.RFC3339))

	resp, err := http.Get(url)
	if err != nil {
		log.Error("Error fetching exchange rate", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("Error reading response body", zap.Error(err))
		return nil, err
	}

	var rawData [][]float64
	if err := json.Unmarshal([]byte(body), &rawData); err != nil {
		log.Error("Error unmarshalling response body", zap.Error(err))
		return nil, err
	}

	var rates []indexer.CoinBaseHistoricalExchangeRate
	for _, item := range rawData {
		rate := indexer.CoinBaseHistoricalExchangeRate{
			Time:         time.Unix(int64(item[0]), 0),
			Low:          item[2],
			High:         item[1],
			Open:         item[3],
			Close:        item[4],
			CurrencyPair: currencyPair,
		}
		rates = append(rates, rate)
	}

	return rates, nil
}
