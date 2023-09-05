package worker

import (
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"

	indexer "github.com/bitmark-inc/nft-indexer"
)

const TokenRefreshingDelay = 7 * time.Minute

// refreshTokenProvenanceByOwnerDetachedWorkflow creates a detached workflow to trigger token provenance check
func (w *NFTIndexerWorker) refreshTokenProvenanceByOwnerDetachedWorkflow(ctx workflow.Context, caller, owner string) {
	log := workflow.GetLogger(ctx)

	cwo := ContextDetachedChildWorkflow(ctx, WorkflowIDRefreshTokenProvenanceByOwner(caller, owner), ProvenanceTaskListName)

	var cw workflow.Execution
	if err := workflow.
		ExecuteChildWorkflow(cwo, w.RefreshTokenProvenanceByOwnerWorkflow, owner).
		GetChildWorkflowExecution().Get(ctx, &cw); err != nil {
		log.Error("fail to start workflow RefreshTokenProvenanceByOwnerWorkflow", zap.Error(err), zap.String("owner", owner))
	}
	log.Info("workflow RefreshTokenProvenanceByOwnerWorkflow started", zap.String("workflow_id", cw.ID), zap.String("owner", owner))
}

// IndexETHTokenWorkflow is a workflow to index and summarize ETH tokens for a owner.
// The data now comes from OpenSea.
func (w *NFTIndexerWorker) IndexETHTokenWorkflow(ctx workflow.Context, tokenOwner string, includeHistory bool) error {
	log := workflow.GetLogger(ctx)

	if includeHistory {
		defer w.refreshTokenProvenanceByOwnerDetachedWorkflow(ctx, "nft-indexer-background", tokenOwner)
	}

	ethTokenOwner := indexer.EthereumChecksumAddress(tokenOwner)
	if ethTokenOwner == indexer.EthereumZeroAddress {
		log.Warn("invalid ethereum token owner", zap.String("owner", tokenOwner))
		var err = fmt.Errorf("invalid ethereum token owner")
		sentry.CaptureException(err)
		return err
	}

	var offset = 0
	for {
		var updateCounts int

		if err := workflow.ExecuteActivity(ContextRetryActivity(ctx, ""), w.IndexETHTokenByOwner, ethTokenOwner, offset).Get(ctx, &updateCounts); err != nil {
			sentry.CaptureException(err)
			return err
		}

		if updateCounts == 0 {
			log.Debug("[loop] no token found from ethereum", zap.String("owner", ethTokenOwner), zap.Int("offset", offset))
			break
		}

		offset += updateCounts
	}

	log.Info("ETH tokens indexed", zap.String("owner", ethTokenOwner))
	return nil
}

// IndexTezosTokenWorkflow is a workflow to index and summarized Tezos tokens for a owner
func (w *NFTIndexerWorker) IndexTezosTokenWorkflow(ctx workflow.Context, tokenOwner string, includeHistory bool) error {
	log := workflow.GetLogger(ctx)

	if includeHistory {
		defer w.refreshTokenProvenanceByOwnerDetachedWorkflow(ctx, "nft-indexer-background", tokenOwner)
	}

	var isFirstPage = true
	for {
		var shouldContinue bool

		if err := workflow.ExecuteActivity(ContextRetryActivity(ctx, ""), w.IndexTezosTokenByOwner, tokenOwner, isFirstPage).Get(ctx, &shouldContinue); err != nil {
			sentry.CaptureException(err)
			return err
		}

		if !shouldContinue {
			log.Debug("[loop] no token found from tezos", zap.String("owner", tokenOwner))
			break
		}

		isFirstPage = false
	}
	log.Info("TEZOS tokens indexed", zap.String("owner", tokenOwner))
	return nil
}

// IndexTokenWorkflow is a workflow to index a single token
func (w *NFTIndexerWorker) IndexTokenWorkflow(ctx workflow.Context, owner, contract, tokenID string, indexProvenance, indexPreview bool) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    time.Hour,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	log := workflow.GetLogger(ctx)

	var update indexer.AssetUpdates
	if err := workflow.ExecuteActivity(ctx, w.IndexToken, contract, tokenID).Get(ctx, &update); err != nil {
		sentry.CaptureException(err)
		return err
	}

	if err := workflow.ExecuteActivity(ctx, w.IndexAsset, update).Get(ctx, nil); err != nil {
		sentry.CaptureException(err)
		return err
	}

	if owner != "" {
		var balance int64
		if err := workflow.ExecuteActivity(ctx, w.GetTokenBalanceOfOwner, contract, tokenID, owner).Get(ctx, &balance); err != nil {
			sentry.CaptureException(err)
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
			sentry.CaptureException(err)
			return err
		}
	}

	if indexPreview && indexer.IsIPFSLink(update.ProjectMetadata.PreviewURL) {
		switch update.ProjectMetadata.Medium {
		case "video", "image":
			log.Debug("start indexing preview for the token",
				zap.String("medium", string(update.ProjectMetadata.Medium)),
				zap.String("medium", update.ProjectMetadata.PreviewURL),
				zap.String("indexID: ", tokenID))
			if err := workflow.ExecuteActivity(ctx, w.CacheArtifact, update.ProjectMetadata.PreviewURL).Get(ctx, nil); err != nil {
				sentry.CaptureException(err)
				return fmt.Errorf("IndexTokenWorkflow-preview: %w", err)
			}
		default:
			log.Debug("unsupported preview file", zap.String("medium", string(update.ProjectMetadata.Medium)), zap.String("indexID: ", tokenID))
		}
	}

	if indexProvenance {
		tokenID := update.Tokens[0].IndexID
		if update.Tokens[0].Fungible {
			log.Debug("Start child workflow to update token ownership", zap.String("owner", owner), zap.String("indexID: ", tokenID))

			if err := workflow.ExecuteChildWorkflow(
				ContextNamedRegularChildWorkflow(ctx, WorkflowIDIndexTokenOwnershipByIndexID("background-IndexTokenWorkflow", tokenID), ProvenanceTaskListName),
				w.RefreshTokenOwnershipWorkflow, []string{tokenID}, 0,
			).Get(ctx, nil); err != nil {
				sentry.CaptureException(err)
				return err
			}
		} else {
			log.Debug("Start child workflow to update token provenance", zap.String("owner", owner), zap.String("indexID: ", tokenID))

			if err := workflow.ExecuteChildWorkflow(
				ContextNamedRegularChildWorkflow(ctx, WorkflowIDIndexTokenProvenanceByIndexID("background-IndexTokenWorkflow", tokenID), ProvenanceTaskListName),
				w.RefreshTokenProvenanceWorkflow, []string{tokenID}, 0,
			).Get(ctx, nil); err != nil {
				sentry.CaptureException(err)
				return err
			}
		}
	}

	log.Info("token indexed", zap.String("owner", owner),
		zap.String("contract", contract), zap.String("tokenID", tokenID))
	return nil
}

// RefreshTokenProvenanceByOwnerWorkflow is a workflow to refresh provenance for a specific owner
func (w *NFTIndexerWorker) RefreshTokenProvenanceByOwnerWorkflow(ctx workflow.Context, owner string) error {
	log := workflow.GetLogger(ctx)

	var ownedTokenIDs []string

	if err := workflow.ExecuteActivity(
		ContextRegularActivity(ctx, w.ProvenanceTaskListName),
		w.GetOwnedTokenIDsByOwner, owner,
	).Get(ctx, &ownedTokenIDs); err != nil {
		log.Error("fail to refresh provenance for indexIDs", zap.Error(err), zap.String("owner", owner))
	}

	if err := workflow.ExecuteActivity(
		ContextRegularActivity(ctx, w.ProvenanceTaskListName),
		w.FilterTokenIDsWithInconsistentProvenanceForOwner, ownedTokenIDs, owner,
	).Get(ctx, &ownedTokenIDs); err != nil {
		log.Error("fail to refresh provenance for indexIDs", zap.Error(err), zap.String("owner", owner))
	}

	batchIndexingTokens := 25

	for i := 0; i < len(ownedTokenIDs); i += batchIndexingTokens {
		endIndex := i + batchIndexingTokens
		if endIndex > len(ownedTokenIDs) {
			endIndex = len(ownedTokenIDs)
		}

		if err := workflow.ExecuteChildWorkflow(
			ContextSlowChildWorkflow(ctx, ProvenanceTaskListName),
			w.RefreshTokenProvenanceWorkflow, ownedTokenIDs[i:endIndex], 0,
		).Get(ctx, nil); err != nil {
			log.Error("fail to refresh provenance for indexIDs", zap.Error(err), zap.String("owner", owner))
			return err
		}
	}

	return nil
}

// RefreshTokenProvenanceWorkflow is a workflow to refresh provenance for a specific token
func (w *NFTIndexerWorker) RefreshTokenProvenanceWorkflow(ctx workflow.Context, indexIDs []string, delay time.Duration) error {
	ao := workflow.ActivityOptions{

		TaskList:               w.ProvenanceTaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    time.Hour,
	}

	log := workflow.GetLogger(ctx)

	ctx = workflow.WithActivityOptions(ctx, ao)

	log.Debug("start RefreshTokenProvenanceWorkflow")

	err := workflow.ExecuteActivity(ctx, w.RefreshTokenProvenance, indexIDs, delay).Get(ctx, nil)
	if err != nil {
		log.Error("fail to refresh provenance for indexIDs", zap.Error(err), zap.Any("indexIDs", indexIDs))
	}

	return err
}

// RefreshTokenOwnershipWorkflow is a workflow to refresh ownership for a specific token
func (w *NFTIndexerWorker) RefreshTokenOwnershipWorkflow(ctx workflow.Context, indexIDs []string, delay time.Duration) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.ProvenanceTaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    time.Hour,
	}

	log := workflow.GetLogger(ctx)

	ctx = workflow.WithActivityOptions(ctx, ao)

	log.Debug("start RefreshTokenOwnershipWorkflow")

	err := workflow.ExecuteActivity(ctx, w.RefreshTokenOwnership, indexIDs, delay).Get(ctx, nil)
	if err != nil {
		log.Error("fail to refresh ownership for indexIDs", zap.Error(err), zap.Any("indexIDs", indexIDs))
	}

	return err
}

// CacheIPFSArtifactWorkflow is a worlflow to cache an IPFS artifact
func (w *NFTIndexerWorker) CacheIPFSArtifactWorkflow(ctx workflow.Context, fullDataLink string) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    time.Hour,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	log := workflow.GetLogger(ctx)

	if err := workflow.ExecuteActivity(ctx, w.CacheArtifact, fullDataLink).Get(ctx, nil); err != nil {
		// sentry.CaptureException(err)
		log.Error("fail to cache IPFS data", zap.Error(err))
		return err
	}

	return nil
}

// PendingTxFollowUpWorkflow is a workflow to follow up and update pending tokens
func (w *NFTIndexerWorker) PendingTxFollowUpWorkflow(ctx workflow.Context, delay time.Duration) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.AccountTokenTaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
	}

	log := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, ao)
	log.Debug("start PendingTxFollowUpWorkflow")

	var pendingAccountTokens []indexer.AccountToken
	if err := workflow.ExecuteActivity(ctx, w.GetPendingAccountTokens).Get(ctx, &pendingAccountTokens); err != nil {
		log.Error("fail to get pending account tokens", zap.Error(err))
		return err
	}

	if len(pendingAccountTokens) == 0 {
		_ = workflow.Sleep(ctx, 1*time.Minute)
		return workflow.NewContinueAsNewError(ctx, w.PendingTxFollowUpWorkflow, delay)
	}

	for _, a := range pendingAccountTokens {
		log.Debug("start checking txs for pending token", zap.String("indexID", a.IndexID))
		pendindTxCount := len(a.PendingTxs)

		var hasNewTx bool
		remainingPendingTxs := make([]string, 0, pendindTxCount)
		remainingPendingTxTimes := make([]time.Time, 0, pendindTxCount)

		// The loop checks all new confirmed txs.
		for i := 0; i < pendindTxCount; i++ {
			pendingTime := a.LastPendingTime[i]
			pendingTx := a.PendingTxs[i]

			var txComfirmedTime time.Time
			if err := workflow.ExecuteActivity(ctx, w.GetTxTimestamp, a.Blockchain, pendingTx).Get(ctx, &txComfirmedTime); err != nil {
				log.Error("fail to get tx status for the account token", zap.Error(err), zap.String("indexID", a.IndexID))
				switch err.Error() {
				case indexer.ErrTXNotFound.Error():
					// omit not found tx which exceed an hour
					if time.Since(pendingTime) > time.Hour {
						// TODO: should be check if the tx is remaining in the mempool of the blockchain network
						break
					}
				case indexer.ErrUnsupportedBlockchain.Error():
					// omit unsupported pending tx
					break
				default:
					// leave the error pending txs remain
				}
				remainingPendingTxs = append(remainingPendingTxs, pendingTx)
				remainingPendingTxTimes = append(remainingPendingTxTimes, pendingTime)
			} else {
				log.Debug("found a confirme pending tx", zap.String("pendingTx", pendingTx))
				if txComfirmedTime.Sub(a.LastActivityTime) > 0 {
					hasNewTx = true
				}
			}
		}

		log.Debug("finish checking txs for pending token", zap.String("indexID", a.IndexID), zap.Any("hasNewTx", hasNewTx))
		// refresh once only if there is a new updates detected
		if hasNewTx {
			var err error

			if a.Fungible {
				err = workflow.ExecuteChildWorkflow(ContextRegularChildWorkflow(ctx, ProvenanceTaskListName), w.RefreshTokenOwnershipWorkflow, []string{a.IndexID}, delay).Get(ctx, nil)
			} else {
				err = workflow.ExecuteChildWorkflow(ContextRegularChildWorkflow(ctx, ProvenanceTaskListName), w.RefreshTokenProvenanceWorkflow, []string{a.IndexID}, delay).Get(ctx, nil)
			}

			if err != nil {
				log.Error("fail to update ownership / provenance for indexID", zap.Error(err),
					zap.Bool("fungible", a.Fungible), zap.String("indexID", a.IndexID))
				// DON'T update the pending list if the refresh failed
				continue
			}
		}

		log.Debug("remaining pending txs", zap.Any("remainingPendingTx", remainingPendingTxs), zap.Any("remainingPendingTxTime", remainingPendingTxs))

		if err := workflow.ExecuteActivity(ctx, w.UpdatePendingTxsToAccountToken,
			a.OwnerAccount, a.IndexID, remainingPendingTxs, remainingPendingTxTimes).Get(ctx, nil); err != nil {
			// log the error only so the loop will continuously check the next pending token
			log.Error("fail to update remaining pending txs into account token", zap.Error(err),
				zap.String("indexID", a.IndexID), zap.String("ownerAccount", a.OwnerAccount), zap.Time("astRefreshedTime", a.LastRefreshedTime))
		}
	}

	return workflow.NewContinueAsNewError(ctx, w.PendingTxFollowUpWorkflow, delay)
}

// UpdateSuggestedMimeTypeWorkflow is a workflow to update suggested mimeType from token feedback
func (w *NFTIndexerWorker) UpdateSuggestedMIMETypeWorkflow(ctx workflow.Context, _ time.Duration) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.AccountTokenTaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    time.Hour,
	}

	log := workflow.GetLogger(ctx)

	ctx = workflow.WithActivityOptions(ctx, ao)

	log.Debug("start UpdateSuggestedMimeTypeWorkflow")

	if err := workflow.ExecuteActivity(ctx, w.CalculateMIMETypeFromTokenFeedback).Get(ctx, nil); err != nil {
		log.Error("fail to update suggested mimeType", zap.Error(err))
		return err
	}

	return nil
}

func (w *NFTIndexerWorker) DetectAssetChangeWorkflow(ctx workflow.Context) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.AccountTokenTaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    time.Hour,
	}

	log := workflow.GetLogger(ctx)

	ctx = workflow.WithActivityOptions(ctx, ao)

	log.Debug("start DetectAssetChangeWorkflow")

	if err := workflow.ExecuteActivity(ctx, w.UpdatePresignedThumbnailAssets).Get(ctx, nil); err != nil {
		log.Error("fail to update asset", zap.Error(err))
		return err
	}

	return nil
}
