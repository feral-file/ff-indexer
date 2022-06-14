package indexerWorker

import (
	"fmt"
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

// IndexOpenseaTokenWorkflow is a workflow to summarize NFT data from OpenSea and save it to the storage.
func (w *NFTIndexerWorker) IndexOpenseaTokenWorkflow(ctx workflow.Context, tokenOwner string) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: time.Second * 60,
		StartToCloseTimeout:    time.Hour * 24,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	log := workflow.GetLogger(ctx)

	var offset = 0

	ethTokenOwner := indexer.EthereumChecksumAddress(tokenOwner)

	if ethTokenOwner == indexer.EthereumZeroAddress {
		log.Warn("invalid ethereum token owner", zap.String("owner", tokenOwner))
		var err = fmt.Errorf("invalid ethereum token owner")
		sentry.CaptureException(err)
		return err
	}

	var tokenIndexIDs []string
	if err := workflow.ExecuteActivity(ctx, w.GetTokenIDsByOwner, ethTokenOwner).Get(ctx, &tokenIndexIDs); err != nil {
		sentry.CaptureException(err)
		return err
	}

	log.Info("tokens to check existence token provenance", zap.Any("tokenIndexIDs", tokenIndexIDs))
	if err := workflow.ExecuteActivity(ctx, w.RefreshTokenProvenance, tokenIndexIDs, 2*time.Hour).Get(ctx, nil); err != nil {
		sentry.CaptureException(err)
		return err
	}

	for {
		tokenUpdates := []indexer.AssetUpdates{}
		if err := workflow.ExecuteActivity(ctx, w.IndexOwnerTokenDataFromOpensea, ethTokenOwner, offset).Get(ctx, &tokenUpdates); err != nil {
			sentry.CaptureException(err)
			return err
		}

		if len(tokenUpdates) == 0 {
			log.Debug("no token found from opensea", zap.String("owner", ethTokenOwner), zap.Int("offset", offset))
			break
		}

		for _, u := range tokenUpdates {
			if err := workflow.ExecuteActivity(ctx, w.IndexAsset, u).Get(ctx, nil); err != nil {
				sentry.CaptureException(err)
				return err
			}

			for _, t := range u.Tokens {
				log.Info("tokens to check indexed token provenance", zap.Any("tokenIndexIDs", tokenIndexIDs))
				if err := workflow.ExecuteActivity(ctx, w.RefreshTokenProvenance, []string{t.IndexID}, time.Hour).Get(ctx, nil); err != nil {
					sentry.CaptureException(err)
					return err
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

	var offset = 0
	for {
		tokenUpdates := []indexer.AssetUpdates{}
		if err := workflow.ExecuteActivity(ctx, w.IndexOwnerTokenDataFromTezos, tokenOwner, offset).Get(ctx, &tokenUpdates); err != nil {
			sentry.CaptureException(err)
			return err
		}

		if len(tokenUpdates) == 0 {
			log.Debug("no token found from tezos", zap.String("owner", tokenOwner), zap.Int("offset", offset))
			break
		}

		for _, u := range tokenUpdates {
			if err := workflow.ExecuteActivity(ctx, w.IndexAsset, u).Get(ctx, nil); err != nil {
				sentry.CaptureException(err)
				return err
			}
		}

		offset += len(tokenUpdates)

		workflow.Sleep(ctx, time.Second)
	}
	log.Info("TEZOS tokens indexed", zap.String("owner", tokenOwner))
	return nil
}

// IndexTokenWorkflow is a worlflow to index a single token
func (w *NFTIndexerWorker) IndexTokenWorkflow(ctx workflow.Context, owner, contract, tokenID string) error {
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

	log.Info("token indexed", zap.String("owner", owner),
		zap.String("contract", contract), zap.String("tokenID", tokenID))
	return nil
}

// RefreshTokenProvenanceWorkflow is a workflow to refresh provenance for a specific token
func (w *NFTIndexerWorker) RefreshTokenProvenanceWorkflow(ctx workflow.Context, indexIDs []string, delay time.Duration) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: time.Second * 60,
		StartToCloseTimeout:    time.Hour * 24,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	log.Debug("start RefreshTokenProvenanceWorkflow")

	return workflow.ExecuteActivity(ctx, w.RefreshTokenProvenance, indexIDs, delay).Get(ctx, nil)
}

// RefreshTokenProvenanceWorkflow is a workflow to refresh provenance for a specific token
func (w *NFTIndexerWorker) RefreshTokenProvenancePeriodicallyWorkflow(ctx workflow.Context, size int64) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: time.Second * 60,
		StartToCloseTimeout:    time.Hour * 24,
	}

	log.Debug("start RefreshTokenProvenancePeriodicallyWorkflow")

	var tokens []indexer.Token
	if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, ao), w.GetOutdatedTokens, size).Get(ctx, &tokens); err != nil {
		sentry.CaptureException(err)
		return err
	}

	log.WithField("token_counts", len(tokens)).Debug("outdated tokens")

	indexIDs := []string{}
	for _, t := range tokens {
		indexIDs = append(indexIDs, t.IndexID)
	}

	if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, ao), w.RefreshTokenProvenance, indexIDs, 0).Get(ctx, nil); err != nil {
		log.WithError(err).Error("fail to refresh token provenance")
		sentry.CaptureException(err)
		return err
	}

	return nil
}
