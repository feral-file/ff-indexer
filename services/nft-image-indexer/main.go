package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/bitmark-inc/config-loader"
	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/imageStore"
)

func main() {
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config.LoadConfig("NFT_INDEXER")
	if err := log.Initialize(viper.GetString("log.level"), viper.GetBool("debug")); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn: viper.GetString("sentry.dsn"),
	}); err != nil {
		log.Panic("Sentry initialization failed", zap.Error(err))
	}

	store := imageStore.New(
		viper.GetString("image_db.dsn"),
		viper.GetString("cloudflare.account_hash"),
		viper.GetString("cloudflare.account_id"),
		viper.GetString("cloudflare.api_token"))
	if err := store.AutoMigrate(); err != nil {
		panic(err)
	}

	mongoClient, err := mongo.Connect(mainCtx, options.Client().ApplyURI(viper.GetString("store.db_uri")))
	if err != nil {
		panic(err)
	}

	// nft indexer store
	db := mongoClient.Database(viper.GetString("store.db_name"))
	assetCollection := db.Collection("assets")
	accountTokenCollection := db.Collection("account_tokens")

	pinataIPFS := NewPinataIPFSPinService()

	ctx, stop := signal.NotifyContext(mainCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	thumbnailCachePeriod, err := time.ParseDuration(viper.GetString("thumbnail.cache_period"))
	if err != nil {
		log.Error("invalid duration. use default value 72h", zap.Error(err))
		thumbnailCachePeriod = 72 * time.Hour
	}
	thumbnailCacheRetryInterval, err := time.ParseDuration(viper.GetString("thumbnail.cache_retry_interval"))
	if err != nil {
		log.Error("invalid duration. use default value 24h", zap.Error(err))
		thumbnailCacheRetryInterval = 24 * time.Hour
	}

	imageIndexer := NewNFTContentIndexer(store, assetCollection, accountTokenCollection, pinataIPFS,
		thumbnailCachePeriod, thumbnailCacheRetryInterval)
	imageIndexer.Start(ctx)

	log.Info("Content indexer terminated")
}
