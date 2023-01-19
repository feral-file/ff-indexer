package indexer

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
)

var blockTimes = map[string]time.Time{}
var getBlockHashLock sync.Mutex

func GetETHBlockTime(ctx context.Context, rpcClient *ethclient.Client, hash common.Hash) (time.Time, error) {
	getBlockHashLock.Lock()
	defer getBlockHashLock.Unlock()

	if blockTime, ok := blockTimes[hash.Hex()]; ok {
		return blockTime, nil
	}

	block, err := rpcClient.BlockByHash(ctx, hash)
	if err != nil {
		return time.Time{}, err
	}

	logrus.WithField("blockNumber", block.NumberU64()).Debug("set new block")
	blockTimes[hash.Hex()] = time.Unix(int64(block.Time()), 0)

	return blockTimes[hash.Hex()], nil
}
