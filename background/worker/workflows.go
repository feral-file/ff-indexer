package worker

import (
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/cadence"
	cadenceClient "go.uber.org/cadence/client"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"

	indexer "github.com/bitmark-inc/nft-indexer"
)

const TokenRefreshingDelay = 7 * time.Minute

// triggerIndexOutdatedTokenWorkflow triggers two workflows for checking both ownership and provenance
// func (w *NFTIndexerWorker) triggerIndexOutdatedTokenWorkflow(ctx workflow.Context, owner string, ownedFungibleToken, ownedNonFungibleToken []string) {
// 	log := workflow.GetLogger(ctx)

// 	if len(ownedFungibleToken) > 0 {
// 		log.Debug("Start child workflow to check existence token ownership", zap.String("owner", owner))
// 		cwoOwnership := workflow.ChildWorkflowOptions{
// 			TaskList:                     ProvenanceTaskListName,
// 			WorkflowID:                   WorkflowIDIndexTokenOwnershipByOwner(owner),
// 			WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
// 			ParentClosePolicy:            cadenceClient.ParentClosePolicyAbandon,
// 			ExecutionStartToCloseTimeout: time.Hour,
// 		}
// 		_ = workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, cwoOwnership),
// 			w.RefreshTokenOwnershipWorkflow, ownedFungibleToken, TokenRefreshingDelay)
// 	}

// 	if len(ownedNonFungibleToken) > 0 {
// 		log.Debug("Start child workflow to check existence token provenance", zap.String("owner", owner))
// 		cwoProvenance := workflow.ChildWorkflowOptions{
// 			TaskList:                     ProvenanceTaskListName,
// 			WorkflowID:                   WorkflowIDIndexTokenProvenanceByOwner(owner),
// 			WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
// 			ParentClosePolicy:            cadenceClient.ParentClosePolicyAbandon,
// 			ExecutionStartToCloseTimeout: time.Hour,
// 		}
// 		_ = workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, cwoProvenance),
// 			w.RefreshTokenProvenanceWorkflow, ownedNonFungibleToken, TokenRefreshingDelay)
// 	}
// }

// IndexOpenseaTokenWorkflow is a workflow to summarize NFT data from OpenSea and save it to the storage.
func (w *NFTIndexerWorker) IndexOpenseaTokenWorkflow(ctx workflow.Context, tokenOwner string) error {
	log := workflow.GetLogger(ctx)

	ethTokenOwner := indexer.EthereumChecksumAddress(tokenOwner)
	if ethTokenOwner == indexer.EthereumZeroAddress {
		log.Warn("invalid ethereum token owner", zap.String("owner", tokenOwner))
		var err = fmt.Errorf("invalid ethereum token owner")
		sentry.CaptureException(err)
		return err
	}

	// {
	// 	ctx = ContextNoRetryActivity(ctx)
	// 	var outdatedTokens []indexer.Token
	// 	if err := workflow.ExecuteActivity(ctx, w.GetOutdatedTokensByOwner, ethTokenOwner).Get(ctx, &outdatedTokens); err != nil {
	// 		sentry.CaptureException(err)
	// 		return err
	// 	}

	// 	log.Debug("Classify outdated tokens for owner", zap.Any("tokens", outdatedTokens), zap.String("owner", ethTokenOwner))
	// 	ownedFungibleToken := []string{}
	// 	ownedNonFungibleToken := []string{}
	// 	for _, t := range outdatedTokens {
	// 		if t.Fungible {
	// 			ownedFungibleToken = append(ownedFungibleToken, t.IndexID)
	// 		} else {
	// 			ownedNonFungibleToken = append(ownedNonFungibleToken, t.IndexID)
	// 		}
	// 	}
	// 	log.Info("Start workflows to check existence token ownership and provenance", zap.String("owner", ethTokenOwner))
	// 	w.triggerIndexOutdatedTokenWorkflow(ctx, ethTokenOwner, ownedFungibleToken, ownedNonFungibleToken)
	// }

	var offset = 0
	for {
		var updateCounts int

		if err := workflow.ExecuteActivity(ContextRetryActivity(ctx), w.IndexETHTokenByOwner, ethTokenOwner, offset).Get(ctx, &updateCounts); err != nil {
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

func (w *NFTIndexerWorker) IndexTezosTokenWorkflow(ctx workflow.Context, tokenOwner string) error {
	log := workflow.GetLogger(ctx)

	// {
	// 	ctx = ContextNoRetryActivity(ctx)
	// 	var outdatedTokens []indexer.Token
	// 	if err := workflow.ExecuteActivity(ctx, w.GetOutdatedTokensByOwner, tokenOwner).Get(ctx, &outdatedTokens); err != nil {
	// 		sentry.CaptureException(err)
	// 		return err
	// 	}

	// 	log.Debug("Classify outdated tokens for owner", zap.Any("tokens", outdatedTokens), zap.String("owner", tokenOwner))
	// 	ownedFungibleToken := []string{}
	// 	ownedNonFungibleToken := []string{}
	// 	for _, t := range outdatedTokens {
	// 		if t.Fungible {
	// 			ownedFungibleToken = append(ownedFungibleToken, t.IndexID)
	// 		} else {
	// 			ownedNonFungibleToken = append(ownedNonFungibleToken, t.IndexID)
	// 		}
	// 	}
	// 	log.Info("Start workflows to check existence token ownership and provenance", zap.String("owner", tokenOwner))
	// 	w.triggerIndexOutdatedTokenWorkflow(ctx, tokenOwner, ownedFungibleToken, ownedNonFungibleToken)
	// }

	var isFirstPage = true
	for {
		var shouldContinue bool

		if err := workflow.ExecuteActivity(ContextRetryActivity(ctx), w.IndexTezosTokenByOwner, tokenOwner, isFirstPage).Get(ctx, &shouldContinue); err != nil {
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

	if indexPreview {
		if err := workflow.ExecuteActivity(ctx, w.CacheIPFSArtifactInS3, update.ProjectMetadata.PreviewURL).Get(ctx, nil); err != nil {
			sentry.CaptureException(err)
			return fmt.Errorf("IndexTokenWorkflow-preview: %w", err)
		}
	}

	if indexProvenance {
		if update.Tokens[0].Fungible {
			log.Debug("Start child workflow to update token ownership", zap.String("owner", owner), zap.String("indexID: ", update.Tokens[0].IndexID))
			cwoOwnership := workflow.ChildWorkflowOptions{
				TaskList:                     ProvenanceTaskListName,
				WorkflowID:                   WorkflowIDIndexTokenOwnershipByOwner(owner),
				WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
				ParentClosePolicy:            cadenceClient.ParentClosePolicyAbandon,
				ExecutionStartToCloseTimeout: time.Hour,
			}
			if err := workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, cwoOwnership),
				w.RefreshTokenOwnershipWorkflow, []string{update.Tokens[0].IndexID}, 0).Get(ctx, nil); err != nil {
				sentry.CaptureException(err)
				return err
			}
		} else {
			log.Debug("Start child workflow to update token provenance", zap.String("owner", owner), zap.String("indexID: ", update.Tokens[0].IndexID))
			cwoProvenance := workflow.ChildWorkflowOptions{
				TaskList:                     ProvenanceTaskListName,
				WorkflowID:                   WorkflowIDIndexTokenProvenanceByOwner(owner),
				WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
				ParentClosePolicy:            cadenceClient.ParentClosePolicyAbandon,
				ExecutionStartToCloseTimeout: time.Hour,
			}
			if err := workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, cwoProvenance),
				w.RefreshTokenProvenanceWorkflow, []string{update.Tokens[0].IndexID}, 0).Get(ctx, nil); err != nil {
				sentry.CaptureException(err)
				return err
			}
		}
	}

	log.Info("token indexed", zap.String("owner", owner),
		zap.String("contract", contract), zap.String("tokenID", tokenID))
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

	if err := workflow.ExecuteActivity(ctx, w.CacheIPFSArtifactInS3, fullDataLink).Get(ctx, nil); err != nil {
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
		pendindTxCounts := len(a.PendingTxs)

		var hasNewTx bool
		remainingPendingTx := make([]string, 0, pendindTxCounts)
		remainingPendingTxTime := make([]time.Time, 0, pendindTxCounts)

		// The loop checks all new confirmed txs.
		for i := 0; i < pendindTxCounts; i++ {
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
				remainingPendingTx = append(remainingPendingTx, pendingTx)
				remainingPendingTxTime = append(remainingPendingTxTime, pendingTime)
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

			cwo := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
				TaskList:                     ProvenanceTaskListName,
				ExecutionStartToCloseTimeout: 10 * time.Minute,
				RetryPolicy: &cadence.RetryPolicy{
					InitialInterval:    10 * time.Second,
					BackoffCoefficient: 1.0,
					MaximumAttempts:    60,
				},
			})

			if a.Fungible {
				err = workflow.ExecuteChildWorkflow(cwo, w.RefreshTokenOwnershipWorkflow, []string{a.IndexID}, delay).Get(ctx, nil)
			} else {
				err = workflow.ExecuteChildWorkflow(cwo, w.RefreshTokenProvenanceWorkflow, []string{a.IndexID}, delay).Get(ctx, nil)
			}

			if err != nil {
				log.Error("fail to update ownership / provenance for indexID", zap.Error(err),
					zap.Bool("fungible", a.Fungible), zap.String("indexID", a.IndexID))
				// DON'T update the pending list if the refresh failed
				continue
			}
		}

		log.Debug("remaining pending txs", zap.Any("remainingPendingTx", remainingPendingTx), zap.Any("remainingPendingTxTime", remainingPendingTx))

		if err := workflow.ExecuteActivity(ctx, w.UpdatePendingTxsToAccountToken,
			a.OwnerAccount, a.IndexID, remainingPendingTx, remainingPendingTxTime).Get(ctx, nil); err != nil {
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
