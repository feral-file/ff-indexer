package indexerWorker

import (
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

// artblocksContracts indexes the addresses which are ERC721 contracts of Artblocks
var artblocksContracts = map[string]struct{}{
	"0x059edd72cd353df5106d2b9cc5ab83a52287ac3a": {},
	"0xa7d8d9ef8d8ce8992df33d8b8cf4aebabd5bd270": {},
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
			log.Info("no token found from opensea", zap.String("owner", tokenOwner), zap.Int("offset", offset))
			return nil
		}

		for _, u := range tokenUpdates {
			if err := workflow.ExecuteActivity(ctx, w.IndexAsset, u).Get(ctx, nil); err != nil {
				return err
			}
		}

		offset += len(tokenUpdates)
	}
}
