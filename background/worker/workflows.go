package worker

import (
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
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
	if err := workflow.ExecuteActivity(ctx, w.IndexToken, owner, contract, tokenID).Get(ctx, &update); err != nil {
		sentry.CaptureException(err)
		return err
	}

	if err := workflow.ExecuteActivity(ctx, w.IndexAsset, update).Get(ctx, nil); err != nil {
		sentry.CaptureException(err)
		return err
	}

	accountTokens := []indexer.AccountToken{
		{
			BaseTokenInfo:     update.Tokens[0].BaseTokenInfo,
			IndexID:           update.Tokens[0].IndexID,
			OwnerAccount:      update.Tokens[0].Owner,
			Balance:           update.Tokens[0].Balance,
			LastActivityTime:  update.Tokens[0].LastActivityTime,
			LastRefreshedTime: update.Tokens[0].LastRefreshedTime,
		}}

	if err := workflow.ExecuteActivity(ctx, w.IndexAccountTokens, owner, accountTokens).Get(ctx, nil); err != nil {
		sentry.CaptureException(err)
		return err
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

// UpdateAccountTokenWorkflow is a workflow to refresh provenance for a specific token
func (w *NFTIndexerWorker) UpdateAccountTokensWorkflow(ctx workflow.Context, delay time.Duration) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.AccountTokenTaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    time.Hour,
	}

	log := workflow.GetLogger(ctx)

	ctx = workflow.WithActivityOptions(ctx, ao)

	log.Debug("start UpdateAccountTokensWorkflow")

	if err := workflow.ExecuteActivity(ctx, w.UpdateAccountTokens).Get(ctx, nil); err != nil {
		log.Error("fail to update account tokens", zap.Error(err))
		return err
	}

	_ = workflow.Sleep(ctx, 1*time.Minute)

	return workflow.NewContinueAsNewError(ctx, w.UpdateAccountTokensWorkflow, delay)

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
