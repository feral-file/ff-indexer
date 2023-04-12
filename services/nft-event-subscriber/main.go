package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	ethereum "github.com/bitmark-inc/account-vault-ethereum"
	"github.com/bitmark-inc/autonomy-account/storage"
	notification "github.com/bitmark-inc/autonomy-notification/sdk"
	bitmarksdk "github.com/bitmark-inc/bitmark-sdk-go"
	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/bitmark-inc/tzkt-go"
)

func main() {
	config.LoadConfig("NFT_INDEXER")

	ctx := context.Background()

	environment := viper.GetString("environment")
	if err := log.Initialize(viper.GetString("log.level"), viper.GetBool("debug")); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	bitmarksdk.Init(&bitmarksdk.Config{
		Network: bitmarksdk.Network(viper.GetString("network.bitmark")),
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		APIToken: viper.GetString("bitmarksdk.apikey"),
	})

	w, err := ethereum.NewWalletFromMnemonic(
		viper.GetString("ethereum.worker_account_mnemonic"),
		viper.GetString("network.ethereum"),
		viper.GetString("ethereum.rpc_url"),
	)
	if err != nil {
		log.Panic(err.Error(), zap.Error(err))
	}

	wsClient, err := ethclient.Dial(viper.GetString("ethereum.ws_url"))
	if err != nil {
		log.Panic(err.Error(), zap.Error(err))
	}

	cadenceClient := cadence.NewWorkerClient(viper.GetString("cadence.domain"))
	cadenceClient.AddService(indexerWorker.ClientName)

	db, err := gorm.Open(postgres.Open(viper.GetString("account.db_uri")))
	if err != nil {
		log.Fatal("fail to connect database", zap.Error(err))
	}

	accountStore := storage.NewAccountInformationStorage(db)

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"))
	if err != nil {
		log.Panic("fail to initiate indexer store", zap.Error(err))
	}

	nc := notification.New(viper.GetString("notification.endpoint"), nil)

	bitmarkListener, err := NewListener(viper.GetString("bitmark.db_uri"))
	if err != nil {
		log.Panic("fail to initiate bitmark listener", zap.Error(err))
	}

	engine := indexer.New(
		environment,
		opensea.New(viper.GetString("network.ethereum"), viper.GetString("opensea.api_key"), viper.GetInt("opensea.ratelimit")),
		tzkt.New(viper.GetString("network.tezos")),
		fxhash.New(viper.GetString("fxhash.api_endpoint")),
		objkt.New(viper.GetString("objkt.api_endpoint")),
	)

	feed := NewFeedClient(viper.GetString("feed.endpoint"), viper.GetString("feed.api_token"), viper.GetBool("feed.debug"))

	service := New(w, environment, wsClient,
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

	if err := NewEventSubscriberAPI(service, feed).Run(); err != nil {
		panic(err)
	}
}
