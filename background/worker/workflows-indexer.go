package worker

import (
	"errors"
	"fmt"
	"time"

	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	indexer "github.com/bitmark-inc/nft-indexer"
)

// IndexETHTokenWorkflow is a workflow to index and summarize ETH tokens for a owner.
// The data now comes from OpenSea.
func (w *NFTIndexerWorker) IndexETHTokenWorkflow(ctx workflow.Context, tokenOwner string, includeHistory bool) error {
	logger := log.CadenceWorkflowLogger(ctx)

	if includeHistory {
		defer w.refreshTokenProvenanceByOwnerDetachedWorkflow(ctx, "nft-indexer-background", tokenOwner)
	}

	ethTokenOwner := indexer.EthereumChecksumAddress(tokenOwner)
	if ethTokenOwner == indexer.EthereumZeroAddress {
		err := errors.New("invalid ethereum token owner")
		logger.Error(err, zap.String("tokenOwner", tokenOwner))
		return err
	}

	var next = ""
	for {
		var nextPointer string

		if err := workflow.ExecuteActivity(ContextRetryActivity(ctx, ""), w.IndexETHTokenByOwner, ethTokenOwner, next).Get(ctx, &nextPointer); err != nil {
			logger.Error(errors.New("fail to index ethereum token by owner"), zap.Error(err), zap.String("tokenOwner", tokenOwner), zap.String("next", next))
			return err
		}

		if nextPointer == "" {
			break
		}

		next = nextPointer
	}

	return nil
}

// IndexTezosTokenWorkflow is a workflow to index and summarized Tezos tokens for a owner
func (w *NFTIndexerWorker) IndexTezosTokenWorkflow(ctx workflow.Context, tokenOwner string, includeHistory bool) error {
	logger := log.CadenceWorkflowLogger(ctx)

	if includeHistory {
		defer w.refreshTokenProvenanceByOwnerDetachedWorkflow(ctx, "nft-indexer-background", tokenOwner)
	}

	var isFirstPage = true
	for {
		var shouldContinue bool

		if err := workflow.ExecuteActivity(ContextRetryActivity(ctx, ""), w.IndexTezosTokenByOwner, tokenOwner, isFirstPage).Get(ctx, &shouldContinue); err != nil {
			logger.Error(errors.New("fail to index tezos token by owner"), zap.Error(err), zap.String("tokenOwner", tokenOwner), zap.Bool("isFirstPage", isFirstPage))
			return err
		}

		if !shouldContinue {
			break
		}

		isFirstPage = false
	}
	return nil
}

// IndexTokenWorkflow is a workflow to index a single token
func (w *NFTIndexerWorker) IndexTokenWorkflow(ctx workflow.Context, owner, contract, tokenID string, indexProvenance, indexPreview bool) error {
	logger := log.CadenceWorkflowLogger(ctx)

	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    time.Hour,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	var update indexer.AssetUpdates
	if err := workflow.ExecuteActivity(ctx, w.IndexToken, contract, tokenID).Get(ctx, &update); err != nil {
		logger.Error(errors.New("fail to index token"), zap.Error(err), zap.String("contract", contract), zap.String("tokenID", tokenID))
		return err
	}

	if err := workflow.ExecuteActivity(ctx, w.IndexAsset, update).Get(ctx, nil); err != nil {
		logger.Error(errors.New("fail to index asset"), zap.Error(err), zap.String("contract", contract), zap.String("tokenID", tokenID))
		return err
	}

	if owner != "" {
		var balance int64
		if err := workflow.ExecuteActivity(ctx, w.GetTokenBalanceOfOwner, contract, tokenID, owner).Get(ctx, &balance); err != nil {
			logger.Error(errors.New("fail to get token balance of owner"), zap.Error(err), zap.String("contract", contract), zap.String("tokenID", tokenID), zap.String("owner", owner))
			return err
		}

		accountTokens := []indexer.AccountToken{
			{
				BaseTokenInfo:     update.Tokens[0].BaseTokenInfo,
				IndexID:           update.Tokens[0].IndexID,
				OwnerAccount:      owner,
				Balance:           balance,
				LastActivityTime:  update.Tokens[0].LastActivityTime,
				LastRefreshedTime: update.Tokens[0].LastRefreshedTime,
			}}

		if err := workflow.ExecuteActivity(ctx, w.IndexAccountTokens, owner, accountTokens).Get(ctx, nil); err != nil {
			logger.Error(errors.New("fail to index account tokens"), zap.Error(err), zap.String("owner", owner))
			return err
		}
	}

	if indexPreview && indexer.IsIPFSLink(update.ProjectMetadata.PreviewURL) {
		switch update.ProjectMetadata.Medium {
		case "video", "image":
			if err := workflow.ExecuteActivity(ctx, w.CacheArtifact, update.ProjectMetadata.PreviewURL).Get(ctx, nil); err != nil {
				logger.Error(errors.New("fail to cache artifact"), zap.Error(err), zap.String("url", update.ProjectMetadata.PreviewURL))
				return err
			}
		default:
			// do nothing
		}
	}

	indexID := update.Tokens[0].IndexID
	if indexProvenance {
		if update.Tokens[0].Fungible {
			if err := workflow.ExecuteChildWorkflow(
				ContextNamedRegularChildWorkflow(ctx, WorkflowIDIndexTokenOwnershipByIndexID("background-IndexTokenWorkflow", indexID), ProvenanceTaskListName),
				w.RefreshTokenOwnershipWorkflow, []string{indexID}, 0,
			).Get(ctx, nil); err != nil {
				logger.Error(errors.New("fail to refresh token ownership"), zap.Error(err), zap.String("indexID", indexID))
				return err
			}
		} else {
			if err := workflow.ExecuteChildWorkflow(
				ContextNamedRegularChildWorkflow(ctx, WorkflowIDIndexTokenProvenanceByIndexID("background-IndexTokenWorkflow", indexID), ProvenanceTaskListName),
				w.RefreshTokenProvenanceWorkflow, []string{indexID}, 0,
			).Get(ctx, nil); err != nil {
				logger.Error(errors.New("fail to refresh token provenance"), zap.Error(err), zap.String("indexID", indexID))
				return err
			}
		}
	}

	if err := workflow.ExecuteActivity(ctx, w.MarkAccountTokenChanged, []string{indexID}).Get(ctx, nil); err != nil {
		logger.Error(errors.New("fail to mark account token changed"), zap.Error(err), zap.String("indexID", indexID))
		return err
	}

	return nil
}

// CacheIPFSArtifactWorkflow is a worlflow to cache an IPFS artifact
func (w *NFTIndexerWorker) CacheIPFSArtifactWorkflow(ctx workflow.Context, fullDataLink string) error {
	logger := log.CadenceWorkflowLogger(ctx)

	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    time.Hour,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	if err := workflow.ExecuteActivity(ctx, w.CacheArtifact, fullDataLink).Get(ctx, nil); err != nil {
		logger.Error(errors.New("fail to cache artifact"), zap.Error(err), zap.String("url", fullDataLink))
		return err
	}

	return nil
}

func (w *NFTIndexerWorker) IndexEthereumTokenSaleInBlockRange(
	ctx workflow.Context,
	fromBlk uint64,
	toBlk uint64,
	blkBatchSize uint64,
	contractAddresses []string,
	skipIndexed bool) error {
	logger := log.CadenceWorkflowLogger(ctx)
	ctx = ContextRegularActivity(ctx, w.TaskListName)

	// TODO remove in the future
	if !skipIndexed {
		return errors.New("skipIndexed must be true until we have a unique index handled properly for sale time series data")
	}

	if fromBlk > toBlk {
		err := errors.New("invalid block range")
		logger.Error(err, zap.Uint64("fromBlk", fromBlk), zap.Uint64("toBlk", toBlk))
		return err
	}
	startBlk := fromBlk
	endBlk := fromBlk + blkBatchSize
	if endBlk > toBlk {
		endBlk = toBlk
	}

	// Query txs
	txIDs := make([]string, 0)
	if err := workflow.ExecuteActivity(
		ctx,
		w.FilterEthereumNFTTxByEventLogs,
		contractAddresses,
		startBlk,
		endBlk).
		Get(ctx, &txIDs); err != nil {
		logger.Error(errors.New("fail to filter ethereum nft tx by event logs"), zap.Error(err), zap.Uint64("fromBlk", fromBlk), zap.Uint64("toBlk", toBlk))
		return err
	}

	// Index token sale
	futures := make([]workflow.Future, 0)
	for _, txID := range txIDs {
		workflowID := fmt.Sprintf("IndexEthereumTokenSale-%s", txID)
		cwctx := ContextNamedRegularChildWorkflow(ctx, workflowID, TaskListName)
		futures = append(
			futures,
			workflow.ExecuteChildWorkflow(
				cwctx,
				w.IndexEthereumTokenSale,
				txID,
				skipIndexed))
	}

	for _, future := range futures {
		if err := future.Get(ctx, nil); err != nil {
			logger.Error(errors.New("fail to index ethereum token sale"), zap.Error(err))
			return err
		}
	}

	if endBlk < toBlk {
		return workflow.NewContinueAsNewError(
			ctx,
			w.IndexEthereumTokenSaleInBlockRange,
			startBlk+blkBatchSize,
			toBlk,
			blkBatchSize,
			contractAddresses,
			skipIndexed)
	}

	return nil
}

func (w *NFTIndexerWorker) IndexTezosObjktTokenSaleFromTime(
	ctx workflow.Context,
	startTime time.Time,
	offset int,
	batchSize int,
	skipIndexed bool) error {
	logger := log.CadenceWorkflowLogger(ctx)
	ctx = ContextRegularActivity(ctx, w.TaskListName)

	// Fetch tx hashes
	hashes := make([]string, 0)
	if err := workflow.ExecuteActivity(
		ctx,
		w.GetObjktSaleTransactionHashes,
		startTime,
		offset,
		batchSize).
		Get(ctx, &hashes); err != nil {
		logger.Error(errors.New("fail to get objkt sale transaction hashes"), zap.Error(err), zap.Time("startTime", startTime), zap.Int("offset", offset), zap.Int("batchSize", batchSize))
		return err
	}

	futures := make([]workflow.Future, 0)
	indexedHashes := make(map[string]bool)
	for _, hash := range hashes {
		if indexedHashes[hash] {
			continue
		}

		indexedHashes[hash] = true
		workflowID := fmt.Sprintf("IndexTezosObjktTokenSale-%s", hash)
		cwctx := ContextNamedRegularChildWorkflow(ctx, workflowID, TaskListName)
		futures = append(
			futures,
			workflow.ExecuteChildWorkflow(
				cwctx,
				w.IndexTezosObjktTokenSale,
				hash,
				skipIndexed,
			))
	}

	for _, future := range futures {
		if err := future.Get(ctx, nil); err != nil {
			logger.Error(errors.New("fail to index tezos objkt token sale"), zap.Error(err))
			return err
		}
	}

	if len(hashes) > 0 {
		return workflow.NewContinueAsNewError(
			ctx,
			w.IndexTezosObjktTokenSaleFromTime,
			startTime,
			offset+len(hashes),
			batchSize,
			skipIndexed)
	}

	return nil
}
