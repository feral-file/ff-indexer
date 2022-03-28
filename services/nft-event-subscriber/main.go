package main

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	ethereum "github.com/bitmark-inc/account-vault-ethereum"
	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/cadence"
)

// FIXME: prevent the map from increasing infinitely
var blockTimes = map[string]time.Time{}

func getBlockTime(ctx context.Context, rpcClient *ethclient.Client, hash common.Hash) (time.Time, error) {
	if blockTime, ok := blockTimes[hash.Hex()]; ok {
		return blockTime, nil
	} else {
		block, err := rpcClient.BlockByHash(ctx, hash)
		if err != nil {
			return time.Time{}, err
		}
		logrus.WithField("blockNumber", block.NumberU64()).Debug("set new block")
		blockTimes[hash.Hex()] = time.Unix(int64(block.Time()), 0)

		return blockTimes[hash.Hex()], nil
	}
}

func main() {
	config.LoadConfig("NFT_INDEXER")

	ctx := context.Background()

	w, err := ethereum.NewWalletFromMnemonic(
		viper.GetString("ethereum.worker_account_mnemonic"),
		viper.GetString("network"),
		viper.GetString("ethereum.rpc_url"),
	)
	if err != nil {
		logrus.WithError(err).Panic(err)
	}

	wsClient, err := ethclient.Dial(viper.GetString("ethereum.ws_url"))
	if err != nil {
		logrus.WithError(err).Panic(err)
	}

	cadenceClient := cadence.NewWorkerClient(viper.GetString("cadence.domain"))
	cadenceClient.AddService(indexerWorker.ClientName)

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"))
	if err != nil {
		logrus.WithError(err).Panic("fail to initiate indexer store")
	}

	service := New(w, wsClient, indexerStore, *cadenceClient)
	if err := service.Subscribe(ctx); err != nil {
		panic(err)
	}
}
