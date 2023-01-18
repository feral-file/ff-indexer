package indexer

import (
	"context"
	"sync"
	"time"

	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

var blockTimes = map[string]time.Time{}
var getBlockHashLock sync.Mutex

func GetETHBlockTime(ctx context.Context, rpcClient *ethclient.Client, hash common.Hash) (time.Time, error) {
	getBlockHashLock.Lock()
	defer getBlockHashLock.Unlock()

	if blockTime, ok := blockTimes[hash.Hex()]; ok {
		return blockTime, nil
	} else {
		block, err := rpcClient.BlockByHash(ctx, hash)
		if err != nil {
			return time.Time{}, err
		}

		log.Debug("set new block", zap.Uint64("blockNumber", block.NumberU64()))
		blockTimes[hash.Hex()] = time.Unix(int64(block.Time()), 0)

		return blockTimes[hash.Hex()], nil
	}
}
