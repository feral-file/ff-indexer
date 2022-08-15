package main

import (
	"context"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	ethereum "github.com/bitmark-inc/account-vault-ethereum"
	"github.com/bitmark-inc/autonomy-account/storage"
	notification "github.com/bitmark-inc/autonomy-notification/sdk"
	bitmarksdk "github.com/bitmark-inc/bitmark-sdk-go"
	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
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

	network := viper.GetString("network")

	bitmarksdk.Init(&bitmarksdk.Config{
		Network: bitmarksdk.Network(network),
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		APIToken: viper.GetString("bitmarksdk.apikey"),
	})

	w, err := ethereum.NewWalletFromMnemonic(
		viper.GetString("ethereum.worker_account_mnemonic"),
		network,
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

	db, err := gorm.Open(postgres.Open(viper.GetString("account.db_uri")))
	if err != nil {
		logrus.WithError(err).Fatal("fail to connect database")
	}

	accountStore := storage.NewAccountInformationStorage(db)

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"))
	if err != nil {
		logrus.WithError(err).Panic("fail to initiate indexer store")
	}

	nc := notification.New(viper.GetString("notification.endpoint"), nil)

	bitmarkListener, err := NewListener(viper.GetString("bitmark.db_uri"))
	if err != nil {
		logrus.WithError(err).Panic("fail to initiate bitmark listener")
	}

	engine := indexer.New(
		opensea.New(viper.GetString("network"), viper.GetString("opensea.api_key"), viper.GetInt("opensea.ratelimit")),
		tzkt.New("api.mainnet.tzkt.io"),
		fxhash.New(viper.GetString("fxhash.api_endpoint")),
		objkt.New(viper.GetString("objkt.api_endpoint")),
	)

	feed := NewFeedClient(viper.GetString("feed.endpoint"), viper.GetString("feed.api_token"))

	service := New(w, network, wsClient,
		indexerStore, engine,
		accountStore,
		bitmarkListener,
		nc, feed,
		*cadenceClient)

	// Start watching bitmark events
	if err := service.WatchBitmarkEvent(ctx); err != nil {
		panic(err)
	}

	// Start watching ethereum events
	if err := service.WatchEthereumEvent(ctx); err != nil {
		panic(err)
	}

	if err := NewEventSubscriberAPI(service, feed).Run(ctx); err != nil {
		panic(err)
	}
}
