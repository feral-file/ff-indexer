package indexer

import (
	"context"
	"time"

	log "github.com/bitmark-inc/autonomy-logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/feral-file/ff-indexer/cache"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func GetETHBlockTime(ctx context.Context, store cache.Store, rpcClient *ethclient.Client, hash common.Hash) (time.Time, error) {
	data, err := store.Get(ctx, hash.Hex())

	if err == nil {
		if t, ok := data.(primitive.DateTime); ok {
			return t.Time(), nil
		}
	}

	// Fallback using rpc
	block, err := rpcClient.BlockByHash(ctx, hash)
	if err != nil {
		return time.Time{}, err
	}

	blockTime := time.Unix(int64(block.Time()), 0)
	err = store.Set(ctx, hash.Hex(), blockTime)
	if err != nil {
		log.WarnWithContext(ctx, "failed to save cache data", zap.Error(err))
	}

	return blockTime, err
}

func TransferEventLog(l types.Log) bool {
	evtHash := l.Topics[0]
	return evtHash.Hex() == TransferEventSignature
}

func ERC20Transfer(l types.Log) bool {
	return TransferEventLog(l) && len(l.Topics) == 3
}

func ERC721Transfer(l types.Log) bool {
	return TransferEventLog(l) && len(l.Topics) == 4
}

func ERC1155SingleTransfer(l types.Log) bool {
	return l.Topics[0].Hex() == TransferSingleEventSignature &&
		len(l.Topics) == 4
}
