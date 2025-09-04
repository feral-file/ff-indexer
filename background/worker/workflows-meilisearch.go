package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	indexer "github.com/bitmark-inc/nft-indexer"
)

// StreamTokensToMeilisearchWorkflow is the main parent workflow for streaming tokens to Meilisearch
func (w *NFTIndexerWorker) StreamTokensToMeilisearchWorkflow(ctx workflow.Context, request MeilisearchStreamRequest) (*MeilisearchStreamResult, error) {
	logger := log.CadenceWorkflowLogger(ctx)
	startTime := time.Now()

	logger.Info("Starting Meilisearch streaming workflow",
		zap.Int("addressCount", len(request.Addresses)),
		zap.String("indexName", request.Config.IndexName))

	// Initialize result
	result := &MeilisearchStreamResult{
		BatchResults: make([]MeilisearchBatchResult, 0),
		Errors:       make([]MeilisearchStreamError, 0),
	}

	// Step 1: Create or update Meilisearch index
	if err := workflow.ExecuteActivity(
		ContextRegularActivity(ctx, w.TaskListName),
		w.CreateOrUpdateMeilisearchIndex,
	).Get(ctx, nil); err != nil {
		logger.Error(errors.New("failed to create/update Meilisearch index"), zap.Error(err))
		return nil, err
	}

	// Step 2: Get total count of tokens for progress tracking
	var totalTokenCount int64
	if err := workflow.ExecuteActivity(
		ContextRegularActivity(ctx, w.TaskListName),
		w.CountTokensForAddresses,
		request.Addresses,
	).Get(ctx, &totalTokenCount); err != nil {
		logger.Error(errors.New("failed to count tokens"), zap.Error(err))
		return nil, err
	}

	logger.Info("Total tokens to process", zap.Int64("totalTokens", totalTokenCount))

	// Step 3: Process tokens in parallel batches
	batchSize := int64(request.Config.BatchSize)
	if batchSize <= 0 {
		batchSize = 100 // Default batch size
	}

	maxConcurrency := request.Config.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 5 // Default concurrency
	}

	// Create channels for parallel processing
	futures := make([]workflow.Future, 0)
	processedTokens := int64(0)

	for offset := int64(0); offset < totalTokenCount; offset += batchSize {
		// Limit concurrency by waiting for some futures to complete
		if len(futures) >= maxConcurrency {
			// Wait for the first batch to complete
			var batchResult MeilisearchBatchResult
			if err := futures[0].Get(ctx, &batchResult); err != nil {
				logger.Error(errors.New("batch processing failed"), zap.Error(err))
				result.Errors = append(result.Errors, MeilisearchStreamError{
					Error:     err.Error(),
					Timestamp: time.Now(),
					Retryable: true,
				})
				result.TotalTokensErrored += batchResult.DocumentCount
			} else {
				result.BatchResults = append(result.BatchResults, batchResult)
				result.TotalTokensIndexed += batchResult.DocumentCount
			}

			// Remove the completed future
			futures = futures[1:]
		}

		// Start a new child workflow for this batch
		childWorkflowID := fmt.Sprintf("meilisearch-batch-%d-%d", offset, time.Now().UnixNano())
		future := workflow.ExecuteChildWorkflow(
			ContextNamedRegularChildWorkflow(ctx, childWorkflowID, w.TaskListName),
			w.ProcessTokenBatchToMeilisearchWorkflow,
			request.Addresses,
			request.Config,
			offset,
			batchSize,
			request.LastUpdatedAfter,
		)

		futures = append(futures, future)
		processedTokens += batchSize

		logger.Info("Started batch processing",
			zap.Int64("offset", offset),
			zap.Int64("batchSize", batchSize),
			zap.Int64("processedTokens", processedTokens))
	}

	// Wait for all remaining futures to complete
	for _, future := range futures {
		var batchResult MeilisearchBatchResult
		if err := future.Get(ctx, &batchResult); err != nil {
			logger.Error(errors.New("batch processing failed"), zap.Error(err))
			result.Errors = append(result.Errors, MeilisearchStreamError{
				Error:     err.Error(),
				Timestamp: time.Now(),
				Retryable: true,
			})
			result.TotalTokensErrored += batchResult.DocumentCount
		} else {
			result.BatchResults = append(result.BatchResults, batchResult)
			result.TotalTokensIndexed += batchResult.DocumentCount
		}
	}

	// Calculate final statistics
	result.TotalTokensProcessed = int(totalTokenCount)
	result.ProcessingTime = time.Since(startTime)
	result.TotalTokensSkipped = result.TotalTokensProcessed - result.TotalTokensIndexed - result.TotalTokensErrored

	logger.Info("Meilisearch streaming workflow completed",
		zap.Int("totalProcessed", result.TotalTokensProcessed),
		zap.Int("totalIndexed", result.TotalTokensIndexed),
		zap.Int("totalErrored", result.TotalTokensErrored),
		zap.Int("totalSkipped", result.TotalTokensSkipped),
		zap.Duration("processingTime", result.ProcessingTime))

	return result, nil
}

// ProcessTokenBatchToMeilisearchWorkflow processes a batch of tokens for given addresses
func (w *NFTIndexerWorker) ProcessTokenBatchToMeilisearchWorkflow(
	ctx workflow.Context,
	addresses []string,
	config MeilisearchStreamConfig,
	offset, size int64,
	lastUpdatedAfter *time.Time,
) (MeilisearchBatchResult, error) {
	logger := log.CadenceWorkflowLogger(ctx)

	logger.Info("Processing token batch",
		zap.Int64("offset", offset),
		zap.Int64("size", size),
		zap.Int("addressCount", len(addresses)))

	// Get tokens for the batch
	var tokens []indexer.DetailedTokenV2
	if err := workflow.ExecuteActivity(
		ContextRegularActivity(ctx, w.TaskListName),
		w.GetTokensForAddresses,
		addresses,
		lastUpdatedAfter,
		offset,
		size,
	).Get(ctx, &tokens); err != nil {
		logger.Error(errors.New("failed to get tokens for batch"), zap.Error(err))
		return MeilisearchBatchResult{
			BatchID:       fmt.Sprintf("failed-batch-%d-%d", offset, time.Now().UnixNano()),
			DocumentCount: 0,
			Success:       false,
			Error:         err.Error(),
			ProcessedAt:   time.Now(),
		}, err
	}

	if len(tokens) == 0 {
		return MeilisearchBatchResult{
			BatchID:       fmt.Sprintf("empty-batch-%d-%d", offset, time.Now().UnixNano()),
			DocumentCount: 0,
			Success:       true,
			ProcessedAt:   time.Now(),
		}, nil
	}

	// Filter out tokens owned by burn address or marked burned
	filtered := make([]indexer.DetailedTokenV2, 0, len(tokens))
	for _, t := range tokens {
		if t.Burned {
			continue
		}
		if !t.Fungible && indexer.IsBurnAddress(t.Owner, w.Environment) {
			continue
		}
		filtered = append(filtered, t)
	}

	// Process tokens in smaller sub-batches for better parallelism
	subBatchSize := 50 // Process 50 tokens at a time
	futures := make([]workflow.Future, 0)

	for i := 0; i < len(filtered); i += subBatchSize {
		end := i + subBatchSize
		if end > len(filtered) {
			end = len(filtered)
		}

		subBatch := filtered[i:end]
		future := workflow.ExecuteActivity(
			ContextRegularActivity(ctx, w.TaskListName),
			w.BatchIndexTokensToMeilisearch,
			subBatch,
			false,
		)

		futures = append(futures, future)
	}

	// Collect results from all sub-batches
	var combinedResult MeilisearchBatchResult
	combinedResult.BatchID = fmt.Sprintf("batch-%d-%d", offset, time.Now().UnixNano())
	combinedResult.Success = true
	combinedResult.ProcessedAt = time.Now()

	for i, future := range futures {
		var subResult MeilisearchBatchResult
		if err := future.Get(ctx, &subResult); err != nil {
			logger.Error(errors.New("sub-batch processing failed"),
				zap.Error(err),
				zap.Int("subBatchIndex", i))
			combinedResult.Success = false
			if combinedResult.Error == "" {
				combinedResult.Error = err.Error()
			} else {
				combinedResult.Error += "; " + err.Error()
			}
		} else {
			combinedResult.DocumentCount += subResult.DocumentCount
			if subResult.TaskUID > 0 {
				combinedResult.TaskUID = subResult.TaskUID // Use the last successful task UID
			}
		}
	}

	logger.Info("Batch processing completed",
		zap.String("batchID", combinedResult.BatchID),
		zap.Int("documentCount", combinedResult.DocumentCount),
		zap.Bool("success", combinedResult.Success))

	return combinedResult, nil
}

// RefreshTokensInMeilisearchWorkflow refreshes specific tokens in Meilisearch
func (w *NFTIndexerWorker) RefreshTokensInMeilisearchWorkflow(
	ctx workflow.Context,
	indexIDs []string,
) (*MeilisearchStreamResult, error) {
	logger := log.CadenceWorkflowLogger(ctx)
	startTime := time.Now()

	logger.Info("Starting token refresh in Meilisearch",
		zap.Int("tokenCount", len(indexIDs)))

	result := &MeilisearchStreamResult{
		BatchResults: make([]MeilisearchBatchResult, 0),
		Errors:       make([]MeilisearchStreamError, 0),
	}

	// Process tokens in batches
	batchSize := 100 // Default batch size

	for i := 0; i < len(indexIDs); i += batchSize {
		end := i + batchSize
		if end > len(indexIDs) {
			end = len(indexIDs)
		}

		batchIndexIDs := indexIDs[i:end]

		// Get detailed tokens for this batch
		var tokens []indexer.DetailedTokenV2
		if err := workflow.ExecuteActivity(
			ContextRegularActivity(ctx, w.TaskListName),
			w.GetDetailedTokensV2,
			indexer.FilterParameter{IDs: batchIndexIDs},
			0,
			int64(len(batchIndexIDs)),
		).Get(ctx, &tokens); err != nil {
			logger.Error(errors.New("failed to get detailed tokens"), zap.Error(err))
			result.Errors = append(result.Errors, MeilisearchStreamError{
				Error:     err.Error(),
				Timestamp: time.Now(),
				Retryable: true,
			})
			continue
		}

		// Index the batch to Meilisearch
		deleteBurned := false // Don't delete when refreshing
		var batchResult MeilisearchBatchResult
		if err := workflow.ExecuteActivity(
			ContextRegularActivity(ctx, w.TaskListName),
			w.BatchIndexTokensToMeilisearch,
			tokens,
			deleteBurned,
		).Get(ctx, &batchResult); err != nil {
			logger.Error(errors.New("failed to index batch to Meilisearch"), zap.Error(err))
			result.Errors = append(result.Errors, MeilisearchStreamError{
				Error:     err.Error(),
				Timestamp: time.Now(),
				Retryable: true,
			})
			result.TotalTokensErrored += len(batchIndexIDs)
		} else {
			result.BatchResults = append(result.BatchResults, batchResult)
			result.TotalTokensIndexed += batchResult.DocumentCount
		}
	}

	result.TotalTokensProcessed = len(indexIDs)
	result.ProcessingTime = time.Since(startTime)
	result.TotalTokensSkipped = result.TotalTokensProcessed - result.TotalTokensIndexed - result.TotalTokensErrored

	logger.Info("Token refresh completed",
		zap.Int("totalProcessed", result.TotalTokensProcessed),
		zap.Int("totalIndexed", result.TotalTokensIndexed),
		zap.Duration("processingTime", result.ProcessingTime))

	return result, nil
}

// GetDetailedTokensV2 activity wrapper for getting detailed tokens
func (w *NFTIndexerWorker) GetDetailedTokensV2(ctx context.Context, filterParameter indexer.FilterParameter, offset, size int64) ([]indexer.DetailedTokenV2, error) {
	return w.indexerStore.GetDetailedTokensV2(ctx, filterParameter, offset, size)
}

// DeleteBurnedTokensFromMeilisearchWorkflow removes burned tokens from Meilisearch
func (w *NFTIndexerWorker) DeleteBurnedTokensFromMeilisearchWorkflow(
	ctx workflow.Context,
	indexIDs []string,
) (*MeilisearchStreamResult, error) {
	logger := log.CadenceWorkflowLogger(ctx)
	startTime := time.Now()

	logger.Info("Starting burned token cleanup in Meilisearch",
		zap.Int("tokenCount", len(indexIDs)))

	result := &MeilisearchStreamResult{
		Errors: make([]MeilisearchStreamError, 0),
	}

	if len(indexIDs) > 0 {
		var batchResult MeilisearchBatchResult
		if err := workflow.ExecuteActivity(
			ContextRegularActivity(ctx, w.TaskListName),
			w.DeleteBurnedTokensFromMeilisearch,
			indexIDs,
		).Get(ctx, &batchResult); err != nil {
			logger.Error(errors.New("failed to delete burned tokens"), zap.Error(err))
			result.Errors = append(result.Errors, MeilisearchStreamError{
				Error:     err.Error(),
				Timestamp: time.Now(),
				Retryable: true,
			})
		} else {
			result.TotalTokensProcessed = len(indexIDs)
			result.TotalTokensIndexed = batchResult.DocumentCount
		}
	}

	result.ProcessingTime = time.Since(startTime)
	logger.Info("Burned token cleanup completed",
		zap.Duration("processingTime", result.ProcessingTime),
		zap.Int("tokensDeleted", result.TotalTokensIndexed))

	return result, nil
}
