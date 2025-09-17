package worker

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/bitmark-inc/autonomy-logger"
	indexer "github.com/feral-file/ff-indexer"
	"github.com/feral-file/ff-indexer/externals/coinbase"
	"go.uber.org/cadence"
	cadenceClient "go.uber.org/cadence/client"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

const (
	granularity          = 60
	maxCandlesPerRequest = 300
	chunkSize            = 25
)

type RequestChunk struct {
	currencyPair string
	startTime    int64
	endTime      int64
}

func (w *Worker) CrawlHistoricalExchangeRate(
	ctx workflow.Context,
	currencyPairs []string,
	start int64,
	end int64,
) error {
	logger := log.CadenceWorkflowLogger(ctx)

	// Check if all currencyPairs is supported
	for _, currencyPair := range currencyPairs {
		if !indexer.SupportedCurrencyPairs[currencyPair] {
			return nil
		}
	}

	if start == 0 && end == 0 {
		ao := workflow.ActivityOptions{
			TaskList:               w.TaskListName,
			ScheduleToStartTimeout: 10 * time.Minute,
			StartToCloseTimeout:    time.Hour,
		}
		ctxac := workflow.WithActivityOptions(ctx, ao)

		var lastTime time.Time
		if err := workflow.ExecuteActivity(
			ctxac,
			w.GetExchangeRateLastTime,
		).Get(ctx, &lastTime); err != nil {
			logger.Error(errors.New("fail to get exchange rate last time"), zap.Error(err), zap.String("currencyPairs", strings.Join(currencyPairs, ":")))
			return err
		}

		start = lastTime.Unix()
		end = time.Now().Unix()
	}

	if start >= end {
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
				logger.Error(errors.New("fail to crawl exchange rate by currency pair"), zap.Error(err))
				return err
			}
		}

	}

	return nil
}

func (w *Worker) CrawlExchangeRateByCurrencyPair(
	ctx workflow.Context,
	currencyPair string,
	start int64,
	end int64,
) error {
	logger := log.CadenceWorkflowLogger(ctx)
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
	ctx = workflow.WithActivityOptions(ctx, ao)

	var rates []coinbase.HistoricalExchangeRate

	if err := workflow.ExecuteActivity(
		ctx,
		w.CrawlExchangeRateFromCoinbase,
		currencyPair,
		strconv.Itoa(granularity),
		start,
		end).Get(ctx, &rates); err != nil {
		logger.Error(errors.New("fail to crawl exchange rate from coinbase"), zap.Error(err), zap.String("currencyPair", currencyPair), zap.Int64("start", start), zap.Int64("end", end))
		return err
	}

	if len(rates) == 0 {
		return nil
	}

	if err := workflow.ExecuteActivity(
		ctx,
		w.WriteHistoricalExchangeRate,
		rates).Get(ctx, nil); err != nil {
		logger.Error(errors.New("fail to write historical exchange rate"), zap.Error(err), zap.String("currencyPair", currencyPair), zap.Int64("start", start), zap.Int64("end", end))
		return err
	}

	return nil
}
