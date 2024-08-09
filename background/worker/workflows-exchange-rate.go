package worker

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bitmark-inc/nft-indexer/externals/coinbase"
	"go.uber.org/cadence"
	cadenceClient "go.uber.org/cadence/client"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

const (
	granularity          = 60
	maxCandlesPerRequest = 300
	chunkSize            = 50
)

type RequestChunk struct {
	currencyPair string
	startTime    int64
	endTime      int64
}

func (w *NFTIndexerWorker) CrawlHistoricalExchangeRate(
	ctx workflow.Context,
	currencyPairs []string,
	start int64,
	end int64,
) error {
	log := workflow.GetLogger(ctx)
	log.Debug("start CrawlHistoricalExchangeRate")

	supportedCurrencyPairs := map[string]bool{
		"ETH-USD": true,
		"XTZ-USD": true,
	}
	// Check if all currencyPairs is supported
	for _, currencyPair := range currencyPairs {
		if !supportedCurrencyPairs[currencyPair] {
			log.Error("unsupported currency pair", zap.String("currencyPair", currencyPair))
			return nil
		}
	}

	if start > end {
		log.Error("Start must be before end")
		return nil
	}

	workflowDataChunks := make([][]RequestChunk, 0)
	requestBatches := make([]RequestChunk, 0, chunkSize)
	for i := start; i < end; i += int64(maxCandlesPerRequest * granularity) {
		_startTime := i
		_endTime := i + int64(maxCandlesPerRequest*granularity)
		if _endTime > end {
			_endTime = end
		}

		for _, currencyPair := range currencyPairs {
			requestBatches = append(requestBatches, RequestChunk{currencyPair, _startTime, _endTime})
			if len(requestBatches) == chunkSize {
				workflowDataChunks = append(workflowDataChunks, requestBatches)
				requestBatches = make([]RequestChunk, 0, chunkSize)
			}
		}
	}

	if len(requestBatches) > 0 {
		workflowDataChunks = append(workflowDataChunks, requestBatches)
	}

	for _, requestBatch := range workflowDataChunks {
		futures := make([]workflow.Future, 0, len(requestBatch))
		for _, request := range requestBatch {
			cwo := workflow.ChildWorkflowOptions{
				ExecutionStartToCloseTimeout: 5 * time.Minute,
				WorkflowID:                   fmt.Sprintf("CrawlExchangeRateByCurrencyPair-%s-%d-%d", request.currencyPair, request.startTime, request.endTime),
				WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
			}
			ctxwo := workflow.WithChildOptions(ctx, cwo)
			futures = append(futures, workflow.ExecuteChildWorkflow(
				ctxwo,
				w.CrawlExchangeRateByCurrencyPair,
				request.currencyPair,
				request.startTime,
				request.endTime))
		}

		for _, future := range futures {
			if err := future.Get(ctx, nil); err != nil {
				log.Error("Failed to crawl exchange rate", zap.Error(err))
				return err
			}
		}

	}

	return nil
}

func (w *NFTIndexerWorker) CrawlExchangeRateByCurrencyPair(
	ctx workflow.Context,
	currencyPair string,
	start int64,
	end int64,
) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    10,
		},
	}
	log := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, ao)
	log.Debug("start CrawlExchangeRateByCurrencyPair")

	var rates []coinbase.CoinBaseHistoricalExchangeRate

	if err := workflow.ExecuteActivity(
		ctx,
		w.CrawlExchangeRateFromCoinbase,
		currencyPair,
		strconv.Itoa(granularity),
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
