package indexerWorker

import (
	"math/big"
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
	"go.uber.org/cadence/workflow"
)

func (w *NFTIndexerWorker) IndexArtblocksTokenWorkflow(ctx workflow.Context, tokenOwner string) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.TaskListName,
		ScheduleToStartTimeout: time.Second * 60,
		StartToCloseTimeout:    time.Hour * 24,
		HeartbeatTimeout:       time.Second * 10,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	// log := workflow.GetLogger(ctx)

	artblocksContracts := []string{
		"0x059EDD72Cd353dF5106D2B9cC5ab83a52287aC3a",
		"0xa7d8d9ef8D8Ce8992Df33D8b8CF4Aebabd5bD270",
	}

	for _, contractAddress := range artblocksContracts {
		var tokenIDs []*big.Int
		if err := workflow.ExecuteActivity(ctx, w.GetOwnedERC721TokenIDByContract, contractAddress, tokenOwner).Get(ctx, &tokenIDs); err != nil {
			return err
		}

		for _, tokenID := range tokenIDs {
			var tokenUpdate indexer.AssetUpdates
			if err := workflow.ExecuteActivity(ctx, w.IndexTokenDataFromArtblocks, contractAddress, tokenOwner, tokenID).Get(ctx, &tokenUpdate); err != nil {
				return err
			}

			// save to mongodb
			if err := workflow.ExecuteActivity(ctx, w.IndexAsset, tokenUpdate).Get(ctx, nil); err != nil {
				return err
			}
		}
	}

	return nil
}
