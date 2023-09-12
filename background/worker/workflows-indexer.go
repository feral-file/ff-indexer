package worker

import (
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"

	indexer "github.com/bitmark-inc/nft-indexer"
)

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
