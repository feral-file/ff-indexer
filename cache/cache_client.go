package cache

import (
	"context"
	"time"

	log "github.com/bitmark-inc/autonomy-logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type Client struct {
	ethClient *ethclient.Client
	store     Store
}

func NewCacheClient(ethClient *ethclient.Client, store Store) *Client {
	return &Client{
		ethClient: ethClient,
		store:     store,
	}
}

func (c *Client) GetETHBlockTime(ctx context.Context, blockHash common.Hash) (time.Time, error) {
	data, err := c.store.Get(ctx, blockHash.Hex())

	if err == nil {
		if t, ok := data.(primitive.DateTime); ok {
			return t.Time(), nil
		}
	}

	// Fallback using rpc
	block, err := c.ethClient.BlockByHash(ctx, blockHash)
	if err != nil {
		return time.Time{}, err
	}

	blockTime := time.Unix(int64(block.Time()), 0)
	err = c.store.Set(ctx, blockHash.Hex(), blockTime)
	if err != nil {
		log.Warn("failed to save cache data", zap.Error(err))
	}

	return blockTime, err

}
