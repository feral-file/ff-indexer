package indexerWorker

import (
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
	log "github.com/sirupsen/logrus"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

// artblocksContracts indexes the addresses which are ERC721 contracts of Artblocks
var artblocksContracts = map[string]struct{}{
	"0x059EDD72Cd353dF5106D2B9cC5ab83a52287aC3a": {},
	"0xa7d8d9ef8D8Ce8992Df33D8b8CF4Aebabd5bD270": {},
}

// IndexOpenseaTokenWorkflow is a workflow to summarize NFT data from OpenSea and save it to the storage.
func (w *NFTIndexerWorker) IndexOpenseaTokenWorkflow(ctx workflow.Context, tokenOwner string) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: time.Second * 60,
		StartToCloseTimeout:    time.Hour * 24,
		HeartbeatTimeout:       time.Second * 10,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	log := workflow.GetLogger(ctx)

	var offset = 0

	tokenOwner = indexer.EthereumChecksumAddress(tokenOwner)

	var tokenIndexIDs []string
	if err := workflow.ExecuteActivity(ctx, w.GetTokenIDsByOwner, tokenOwner).Get(ctx, &tokenIndexIDs); err != nil {
		return err
	}

	log.Info("tokens to check provenance", zap.Any("tokenIndexIDs", tokenIndexIDs))

	if err := workflow.ExecuteActivity(ctx, w.RefreshTokenProvenance, tokenIndexIDs, time.Hour).Get(ctx, nil); err != nil {
		return err
	}

	for {
		tokenUpdates := []indexer.AssetUpdates{}
		if err := workflow.ExecuteActivity(ctx, w.IndexTokenDataFromFromOpensea, tokenOwner, offset).Get(ctx, &tokenUpdates); err != nil {
			return err
		}

		if len(tokenUpdates) == 0 {
			log.Info("no token found from opensea", zap.String("owner", tokenOwner), zap.Int("offset", offset))
			return nil
		}

		for _, u := range tokenUpdates {
			if err := workflow.ExecuteActivity(ctx, w.IndexAsset, u).Get(ctx, nil); err != nil {
				return err
			}

			for _, t := range u.Tokens {
				if err := workflow.ExecuteActivity(ctx, w.RefreshTokenProvenance, []string{t.IndexID}, 5*time.Minute).Get(ctx, nil); err != nil {
					return err
				}
			}

		}

		offset += len(tokenUpdates)
	}
}

func (w *NFTIndexerWorker) IndexTezosTokenWorkflow(ctx workflow.Context, tokenOwner string) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: time.Second * 60,
		StartToCloseTimeout:    time.Hour * 24,
		HeartbeatTimeout:       time.Second * 10,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	log := workflow.GetLogger(ctx)

	var offset = 0
	for {
		tokenUpdates := []indexer.AssetUpdates{}
		if err := workflow.ExecuteActivity(ctx, w.IndexTokenDataFromFromTezos, tokenOwner, offset).Get(ctx, &tokenUpdates); err != nil {
			return err
		}

		if len(tokenUpdates) == 0 {
			log.Info("no token found from tezos", zap.String("owner", tokenOwner), zap.Int("offset", offset))
			return nil
		}

		for _, u := range tokenUpdates {
			if err := workflow.ExecuteActivity(ctx, w.IndexAsset, u).Get(ctx, nil); err != nil {
				return err
			}
		}

		offset += len(tokenUpdates)

		workflow.Sleep(ctx, time.Second)
	}
}

// RefreshTokenProvenanceWorkflow is a workflow to refresh provenance for a specific token
func (w *NFTIndexerWorker) RefreshTokenProvenanceWorkflow(ctx workflow.Context, indexIDs []string, delay time.Duration) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: time.Second * 60,
		StartToCloseTimeout:    time.Hour * 24,
		HeartbeatTimeout:       time.Second * 10,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	log.Debug("start RefreshTokenProvenanceWorkflow")

	return workflow.ExecuteActivity(ctx, w.RefreshTokenProvenance, indexIDs, delay).Get(ctx, nil)
}

// RefreshTokenProvenanceWorkflow is a workflow to refresh provenance for a specific token
func (w *NFTIndexerWorker) RefreshTokenProvenancePeriodicallyWorkflow(ctx workflow.Context) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: time.Second * 60,
		StartToCloseTimeout:    time.Hour * 24,
		HeartbeatTimeout:       time.Second * 10,
	}

	log.Debug("start RefreshTokenProvenancePeriodicallyWorkflow")

	var tokens []indexer.Token
	if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, ao), w.GetOutdatedTokens).Get(ctx, &tokens); err != nil {
		return err
	}

	log.WithField("token_counts", len(tokens)).Debug("outdated tokens")

	indexIDs := []string{}
	for _, t := range tokens {
		indexIDs = append(indexIDs, t.IndexID)
	}

	if err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, ao), w.RefreshTokenProvenance, indexIDs, 0).Get(ctx, nil); err != nil {
		log.WithError(err).Error("fail to refresh token provenance")
		return err
	}

	return nil
}
