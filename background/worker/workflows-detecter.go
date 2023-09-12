package worker

import (
	"time"

	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"

	indexer "github.com/bitmark-inc/nft-indexer"
)

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
