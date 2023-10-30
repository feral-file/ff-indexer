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
	PendingTxChecking:
		for i := 0; i < pendindTxCount; i++ {
			pendingTime := a.LastPendingTime[i]
			pendingTx := a.PendingTxs[i]

			var txComfirmedTime time.Time
			if err := workflow.ExecuteActivity(ctx, w.GetTxTimestamp, a.Blockchain, pendingTx).Get(ctx, &txComfirmedTime); err != nil {
				log.Error("fail to get tx status for the account token", zap.Error(err),
					zap.String("txID", pendingTx), zap.String("indexID", a.IndexID), zap.String("ownerAccount", a.OwnerAccount))
				switch err.Error() {
				case indexer.ErrTXNotFound.Error():
					// drop not found tx which exceed an hour
					if time.Since(pendingTime) > time.Hour {
						// TODO: should be check if the tx is remaining in the mempool of the blockchain network
						continue PendingTxChecking
					}
				case indexer.ErrUnsupportedBlockchain.Error():
					// drop unsupported pending tx
					continue PendingTxChecking
				default:
					// leave non-handled error in the remaining pending txs for next processing
					remainingPendingTxs = append(remainingPendingTxs, pendingTx)
					remainingPendingTxTimes = append(remainingPendingTxTimes, pendingTime)
				}
			} else {
				log.Debug("found a confirme pending tx", zap.String("pendingTx", pendingTx))
				if txComfirmedTime.Sub(a.LastActivityTime) > 0 {
					hasNewTx = true
				}
			}
		}
		log.Debug("finish checking txs for pending token", zap.String("indexID", a.IndexID), zap.Any("hasNewTx", hasNewTx))

		// update the balance of "this" token immediately
		var balance int64
		if err := workflow.ExecuteActivity(ContextFastActivity(ctx, AccountTokenTaskListName), w.GetTokenBalanceOfOwner, a.ContractAddress, a.ID, a.OwnerAccount).
			Get(ctx, &balance); err != nil {
			log.Error("fail to get the latest balance for the account token", zap.Error(err),
				zap.String("indexID", a.IndexID), zap.String("ownerAccount", a.OwnerAccount))
			continue
		}

		accountTokens := []indexer.AccountToken{{
			IndexID:           a.IndexID,
			OwnerAccount:      a.OwnerAccount,
			Balance:           balance,
			LastActivityTime:  a.LastActivityTime,
			LastRefreshedTime: time.Now(),
		}}

		if err := workflow.ExecuteActivity(ContextFastActivity(ctx, AccountTokenTaskListName), w.IndexAccountTokens, a.OwnerAccount, accountTokens).Get(ctx, nil); err != nil {
			log.Error("fail to update the latest balance for the account token", zap.Error(err),
				zap.String("indexID", a.IndexID), zap.String("ownerAccount", a.OwnerAccount))
			continue
		}

		// trigger async token ownership / provenance refreshing for the updated token
		if hasNewTx {
			var childFuture workflow.ChildWorkflowFuture
			if a.Fungible {
				childFuture = workflow.ExecuteChildWorkflow(
					ContextDetachedChildWorkflow(ctx, WorkflowIDIndexTokenOwnershipByIndexID(
						"pending-tx-follower", a.IndexID), ProvenanceTaskListName),
					w.RefreshTokenOwnershipWorkflow, []string{a.IndexID}, delay)
			} else {
				childFuture = workflow.ExecuteChildWorkflow(
					ContextDetachedChildWorkflow(ctx, WorkflowIDIndexTokenProvenanceByIndexID(
						"pending-tx-follower", a.IndexID), ProvenanceTaskListName),
					w.RefreshTokenProvenanceWorkflow, []string{a.IndexID}, delay)
			}

			if err := childFuture.GetChildWorkflowExecution().Get(ctx, nil); err != nil {
				log.Error("fail to spawn ownership / provenance updating workflow for indexID", zap.Error(err),
					zap.Bool("fungible", a.Fungible), zap.String("indexID", a.IndexID))
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
