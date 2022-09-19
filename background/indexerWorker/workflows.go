package indexerWorker

import (
	"fmt"
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
	"github.com/getsentry/sentry-go"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

const TokenRefreshingDelay = 60 * time.Minute

// IndexOpenseaTokenWorkflow is a workflow to summarize NFT data from OpenSea and save it to the storage.
func (w *NFTIndexerWorker) IndexOpenseaTokenWorkflow(ctx workflow.Context, tokenOwner string) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: time.Second * 60,
		StartToCloseTimeout:    time.Hour * 24,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	log := workflow.GetLogger(ctx)

	ethTokenOwner := indexer.EthereumChecksumAddress(tokenOwner)

	if ethTokenOwner == indexer.EthereumZeroAddress {
		log.Warn("invalid ethereum token owner", zap.String("owner", tokenOwner))
		var err = fmt.Errorf("invalid ethereum token owner")
		sentry.CaptureException(err)
		return err
	}

	var outdatedTokens []indexer.Token
	if err := workflow.ExecuteActivity(ctx, w.GetOutdatedTokensByOwner, ethTokenOwner).Get(ctx, &outdatedTokens); err != nil {
		sentry.CaptureException(err)
		return err
	}

	log.Debug("outdated tokens for owner", zap.Any("tokens", outdatedTokens), zap.String("owner", ethTokenOwner))

	for _, t := range outdatedTokens {
		if t.Fungible {
			// log.Info("task to check existence token ownership", zap.String("tokenIndexID", t.IndexID))
			// cwo := workflow.ChildWorkflowOptions{
			// 	WorkflowID:                   fmt.Sprintf("index-token-ownership-%s", t.IndexID),
			// 	WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
			// 	ParentClosePolicy:            cadenceClient.ParentClosePolicyAbandon,
			// 	ExecutionStartToCloseTimeout: time.Minute * 10,
			// }
			// _ = workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, cwo),
			// 	w.RefreshTokenOwnershipWorkflow, []string{t.IndexID}, TokenRefreshingDelay)
		} else {
			log.Info("task to check existence token provenance", zap.String("tokenIndexID", t.IndexID))
			// cwo := workflow.ChildWorkflowOptions{
			// 	WorkflowID:                   fmt.Sprintf("index-token-provenance-%s", t.IndexID),
			// 	WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
			// 	ParentClosePolicy:            cadenceClient.ParentClosePolicyAbandon,
			// 	ExecutionStartToCloseTimeout: time.Minute * 10,
			// }
			// _ = workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, cwo),
			// 	w.RefreshTokenProvenanceWorkflow, []string{t.IndexID}, TokenRefreshingDelay)
		}
	}

	var offset = 0

	for {
		tokenUpdates := []indexer.AssetUpdates{}
		if err := workflow.ExecuteActivity(ctx, w.IndexOwnerTokenDataFromOpensea, ethTokenOwner, offset).Get(ctx, &tokenUpdates); err != nil {
			sentry.CaptureException(err)
			return err
		}

		if len(tokenUpdates) == 0 {
			log.Debug("[loop] no token found from opensea", zap.String("owner", ethTokenOwner), zap.Int("offset", offset))
			break
		}

		for _, u := range tokenUpdates {
			if err := workflow.ExecuteActivity(ctx, w.IndexAsset, u).Get(ctx, nil); err != nil {
				sentry.CaptureException(err)
				return err
			}

			for _, t := range u.Tokens {
				if t.Fungible {
					// log.Info("task to check existence token ownership", zap.String("tokenIndexID", t.IndexID))
					// cwo := workflow.ChildWorkflowOptions{
					// 	WorkflowID:                   fmt.Sprintf("index-token-ownership-%s", t.IndexID),
					// 	WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
					// 	ParentClosePolicy:            cadenceClient.ParentClosePolicyAbandon,
					// 	ExecutionStartToCloseTimeout: time.Minute * 10,
					// }
					// _ = workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, cwo),
					// 	w.RefreshTokenOwnershipWorkflow, []string{t.IndexID}, TokenRefreshingDelay)
				} else {
					log.Info("task to check existence token provenance", zap.String("tokenIndexID", t.IndexID))
					// cwo := workflow.ChildWorkflowOptions{
					// 	WorkflowID:                   fmt.Sprintf("index-token-provenance-%s", t.IndexID),
					// 	WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
					// 	ParentClosePolicy:            cadenceClient.ParentClosePolicyAbandon,
					// 	ExecutionStartToCloseTimeout: time.Minute * 10,
					// }
					// _ = workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, cwo),
					// 	w.RefreshTokenProvenanceWorkflow, []string{t.IndexID}, TokenRefreshingDelay)
				}
			}

		}

		offset += len(tokenUpdates)
	}
	log.Info("ETH tokens indexed", zap.String("owner", tokenOwner))
	return nil
}

func (w *NFTIndexerWorker) IndexTezosTokenWorkflow(ctx workflow.Context, tokenOwner string) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: time.Second * 60,
		StartToCloseTimeout:    time.Hour * 24,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	log := workflow.GetLogger(ctx)

	var outdatedTokens []indexer.Token
	if err := workflow.ExecuteActivity(ctx, w.GetOutdatedTokensByOwner, tokenOwner).Get(ctx, &outdatedTokens); err != nil {
		sentry.CaptureException(err)
		return err
	}

	log.Debug("outdated tokens for owner", zap.Any("tokens", outdatedTokens), zap.String("owner", tokenOwner))

	for _, t := range outdatedTokens {
		if t.Fungible {
			log.Info("task to check existence token ownership", zap.String("owner", tokenOwner), zap.String("tokenIndexID", t.IndexID))
			// cwo := workflow.ChildWorkflowOptions{
			// 	WorkflowID:                   fmt.Sprintf("index-token-ownership-%s", t.IndexID),
			// 	WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
			// 	ParentClosePolicy:            cadenceClient.ParentClosePolicyAbandon,
			// 	ExecutionStartToCloseTimeout: time.Minute * 10,
			// }
			// _ = workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, cwo),
			// 	w.RefreshTokenOwnershipWorkflow, []string{t.IndexID}, TokenRefreshingDelay)
		} else {
			log.Info("task to check existence token provenance", zap.String("owner", tokenOwner), zap.String("tokenIndexID", t.IndexID))
			// cwo := workflow.ChildWorkflowOptions{
			// 	WorkflowID:                   fmt.Sprintf("index-token-provenance-%s", t.IndexID),
			// 	WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
			// 	ParentClosePolicy:            cadenceClient.ParentClosePolicyAbandon,
			// 	ExecutionStartToCloseTimeout: time.Minute * 10,
			// }
			// _ = workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, cwo),
			// 	w.RefreshTokenProvenanceWorkflow, []string{t.IndexID}, TokenRefreshingDelay)
		}
	}

	var offset = 0

	for {
		ownedTokens := []tzkt.OwnedToken{}
		if err := workflow.ExecuteActivity(ctx, w.GetTezosTokenByOwner, tokenOwner, offset).Get(ctx, &ownedTokens); err != nil {
			sentry.CaptureException(err)
			return err
		}

		if len(ownedTokens) == 0 {
			log.Debug("no token found", zap.String("owner", tokenOwner), zap.Int("offset", offset))
			break
		}

		for _, t := range ownedTokens {
			log.Debug("token raw data before summarizing", zap.String("owner", tokenOwner), zap.Any("token", t))

			var u indexer.AssetUpdates
			if err := workflow.ExecuteActivity(ctx, w.PrepareTezosTokenFullData, t.Token, tokenOwner, t.Balance).Get(ctx, &u); err != nil {
				sentry.CaptureException(err)
				return err
			}

			log.Debug("token full data before indexing into DB", zap.String("owner", tokenOwner), zap.Any("assetUpdates", u))
			if err := workflow.ExecuteActivity(ctx, w.IndexAsset, u).Get(ctx, nil); err != nil {
				sentry.CaptureException(err)
				return err
			}

			for _, t := range u.Tokens {
				if t.Fungible {
					log.Info("refresh ownership for indexed token", zap.String("owner", tokenOwner), zap.String("tokenIndexID", t.IndexID))
					// cwo := workflow.ChildWorkflowOptions{
					// 	WorkflowID:                   fmt.Sprintf("index-token-ownership-%s", t.IndexID),
					// 	WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
					// 	ParentClosePolicy:            cadenceClient.ParentClosePolicyAbandon,
					// 	ExecutionStartToCloseTimeout: time.Minute * 10,
					// }
					// _ = workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, cwo),
					// 	w.RefreshTokenOwnershipWorkflow, []string{t.IndexID}, TokenRefreshingDelay)
				} else {
					log.Info("refresh provenance for indexed token", zap.String("owner", tokenOwner), zap.String("tokenIndexID", t.IndexID))
					// cwo := workflow.ChildWorkflowOptions{
					// 	WorkflowID:                   fmt.Sprintf("index-token-provenance-%s", t.IndexID),
					// 	WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
					// 	ParentClosePolicy:            cadenceClient.ParentClosePolicyAbandon,
					// 	ExecutionStartToCloseTimeout: time.Minute * 10,
					// }
					// _ = workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, cwo),
					// 	w.RefreshTokenProvenanceWorkflow, []string{t.IndexID}, TokenRefreshingDelay)
				}
			}
		}

		offset += len(ownedTokens)

		workflow.Sleep(ctx, time.Second)
	}
	log.Info("TEZOS tokens indexed", zap.String("owner", tokenOwner))
	return nil
}

// IndexTokenWorkflow is a worlflow to index a single token
func (w *NFTIndexerWorker) IndexTokenWorkflow(ctx workflow.Context, owner, contract, tokenID string, indexPreview bool) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: time.Second * 60,
		StartToCloseTimeout:    time.Hour * 24,
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

	if indexPreview {
		if err := workflow.ExecuteActivity(ctx, w.CacheIPFSArtifactInS3, update.ProjectMetadata.PreviewURL).Get(ctx, nil); err != nil {
			sentry.CaptureException(err)
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
		ScheduleToStartTimeout: time.Second * 60,
		StartToCloseTimeout:    time.Hour * 24,
	}

	log := workflow.GetLogger(ctx)

	ctx = workflow.WithActivityOptions(ctx, ao)

	log.Debug("start RefreshTokenProvenanceWorkflow")

	err := workflow.ExecuteActivity(ctx, w.RefreshTokenProvenance, indexIDs, delay).Get(ctx, nil)
	if err != nil {
		log.Error("fail to refresh procenance for indexIDs", zap.Any("indexIDs", indexIDs))
	}

	return err
}

func (w *NFTIndexerWorker) MaintainProvenanceWorkflow(ctx workflow.Context, indexIDs []string, delay time.Duration) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.ProvenanceTaskListName,
		ScheduleToStartTimeout: time.Second * 60,
		StartToCloseTimeout:    time.Hour * 24,
	}

	log := workflow.GetLogger(ctx)

	ctx = workflow.WithActivityOptions(ctx, ao)

	log.Debug("start MaintainProvenanceWorkflow ")

	err := workflow.ExecuteActivity(ctx, w.MaintainTokenProvenance, indexIDs).Get(ctx, nil)
	if err != nil {
		log.Error("fail to maintain provenance for indexIDs", zap.Any("indexIDs", indexIDs))
	}

	return nil
}

// RefreshTokenOwnershipWorkflow is a workflow to refresh ownership for a specific token
func (w *NFTIndexerWorker) RefreshTokenOwnershipWorkflow(ctx workflow.Context, indexIDs []string, delay time.Duration) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.ProvenanceTaskListName,
		ScheduleToStartTimeout: time.Second * 60,
		StartToCloseTimeout:    time.Hour * 24,
	}

	log := workflow.GetLogger(ctx)

	ctx = workflow.WithActivityOptions(ctx, ao)

	log.Debug("start RefreshTokenOwnershipWorkflow")

	err := workflow.ExecuteActivity(ctx, w.RefreshTezosTokenOwnership, indexIDs, delay).Get(ctx, nil)
	if err != nil {
		log.Error("fail to refresh ownership for indexIDs", zap.Any("indexIDs", indexIDs))
	}

	return err
}

// CacheIPFSArtifactWorkflow is a worlflow to cache an IPFS artifact
func (w *NFTIndexerWorker) CacheIPFSArtifactWorkflow(ctx workflow.Context, fullDataLink string) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: time.Second * 60,
		StartToCloseTimeout:    time.Hour * 24,
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
