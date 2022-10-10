package indexerWorker

import (
	"fmt"
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
	"github.com/getsentry/sentry-go"
	cadenceClient "go.uber.org/cadence/client"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

const TokenRefreshingDelay = 60 * time.Minute

// IndexOpenseaTokenWorkflow is a workflow to summarize NFT data from OpenSea and save it to the storage.
func (w *NFTIndexerWorker) IndexOpenseaTokenWorkflow(ctx workflow.Context, tokenOwner string) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    time.Hour,
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

	ownedFungibleToken := []string{}
	ownedNonFungibleToken := []string{}

	for _, t := range outdatedTokens {
		if t.Fungible {
			ownedFungibleToken = append(ownedFungibleToken, t.IndexID)
		} else {
			ownedNonFungibleToken = append(ownedNonFungibleToken, t.IndexID)
		}
	}

	log.Info("task to check existence token ownership", zap.String("owner", ethTokenOwner))
	cwoOwnership := workflow.ChildWorkflowOptions{
		TaskList:                     ProvenanceTaskListName,
		WorkflowID:                   fmt.Sprintf("index-token-ownership-by-owner-%s", ethTokenOwner),
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
		ParentClosePolicy:            cadenceClient.ParentClosePolicyAbandon,
		ExecutionStartToCloseTimeout: time.Hour,
	}
	_ = workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, cwoOwnership),
		w.RefreshTokenOwnershipWorkflow, ownedFungibleToken, TokenRefreshingDelay)

	log.Info("task to check existence token provenance", zap.String("owner", ethTokenOwner))
	cwoProvenance := workflow.ChildWorkflowOptions{
		TaskList:                     ProvenanceTaskListName,
		WorkflowID:                   fmt.Sprintf("index-token-provenance-by-owner-%s", ethTokenOwner),
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
		ParentClosePolicy:            cadenceClient.ParentClosePolicyAbandon,
		ExecutionStartToCloseTimeout: time.Hour,
	}
	_ = workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, cwoProvenance),
		w.RefreshTokenProvenanceWorkflow, ownedNonFungibleToken, TokenRefreshingDelay)

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
		}

		offset += len(tokenUpdates)
	}
	log.Info("ETH tokens indexed", zap.String("owner", tokenOwner))
	return nil
}

func (w *NFTIndexerWorker) IndexTezosTokenWorkflow(ctx workflow.Context, tokenOwner string) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    time.Hour,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	log := workflow.GetLogger(ctx)

	var outdatedTokens []indexer.Token
	if err := workflow.ExecuteActivity(ctx, w.GetOutdatedTokensByOwner, tokenOwner).Get(ctx, &outdatedTokens); err != nil {
		sentry.CaptureException(err)
		return err
	}

	log.Debug("outdated tokens for owner", zap.Any("tokens", outdatedTokens), zap.String("owner", tokenOwner))

	ownedFungibleToken := []string{}
	ownedNonFungibleToken := []string{}

	for _, t := range outdatedTokens {
		if t.Fungible {
			ownedFungibleToken = append(ownedFungibleToken, t.IndexID)
		} else {
			ownedNonFungibleToken = append(ownedNonFungibleToken, t.IndexID)
		}
	}

	log.Info("task to check existence token ownership", zap.String("owner", tokenOwner))
	cwoOwnership := workflow.ChildWorkflowOptions{
		TaskList:                     ProvenanceTaskListName,
		WorkflowID:                   fmt.Sprintf("index-token-ownership-by-owner-%s", tokenOwner),
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
		ParentClosePolicy:            cadenceClient.ParentClosePolicyAbandon,
		ExecutionStartToCloseTimeout: time.Hour,
	}
	_ = workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, cwoOwnership),
		w.RefreshTokenOwnershipWorkflow, ownedFungibleToken, TokenRefreshingDelay)

	log.Info("task to check existence token provenance", zap.String("owner", tokenOwner))
	cwoProvenance := workflow.ChildWorkflowOptions{
		TaskList:                     ProvenanceTaskListName,
		WorkflowID:                   fmt.Sprintf("index-token-provenance-by-owner-%s", tokenOwner),
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
		ParentClosePolicy:            cadenceClient.ParentClosePolicyAbandon,
		ExecutionStartToCloseTimeout: time.Hour,
	}
	_ = workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, cwoProvenance),
		w.RefreshTokenProvenanceWorkflow, ownedNonFungibleToken, TokenRefreshingDelay)

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

		rawTokens := make([]TezosTokenRawData, 0)
		for i, t := range ownedTokens {
			log.Debug("token raw data before summarizing", zap.String("owner", tokenOwner), zap.Any("token", t))

			rawTokens = append(rawTokens, TezosTokenRawData{
				Token:   t.Token,
				Owner:   tokenOwner,
				Balance: t.Balance,
			})

			if len(rawTokens) >= 50 || i == len(ownedTokens)-1 {
				var updates []indexer.AssetUpdates
				if err := workflow.ExecuteActivity(ctx, w.BatchPrepareTezosTokenFullData, rawTokens).Get(ctx, &updates); err != nil {
					sentry.CaptureException(err)
					return err
				}

				log.Debug("token full data before indexing into DB", zap.String("owner", tokenOwner), zap.Any("assetUpdates", updates))
				if err := workflow.ExecuteActivity(ctx, w.BatchIndexAsset, updates).Get(ctx, nil); err != nil {
					sentry.CaptureException(err)
					return err
				}

				rawTokens = make([]TezosTokenRawData, 0)
			}
		}

		offset += len(ownedTokens)

		workflow.Sleep(ctx, time.Second)
	}
	log.Info("TEZOS tokens indexed", zap.String("owner", tokenOwner))
	return nil
}

// IndexTokenWorkflow is a workflow to index a single token
func (w *NFTIndexerWorker) IndexTokenWorkflow(ctx workflow.Context, owner, contract, tokenID string, indexPreview bool) error {
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
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    time.Hour,
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
