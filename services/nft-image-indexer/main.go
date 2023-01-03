package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/config-loader"
	"github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/imageStore"
)

func main() {
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config.LoadConfig("NFT_INDEXER")

	if err := sentry.Init(sentry.ClientOptions{
		Dsn: viper.GetString("sentry.dsn"),
	}); err != nil {
		logrus.WithError(err).Panic("Sentry initialization failed")
	}

	store := imageStore.New(
		viper.GetString("image_db.dsn"),
		viper.GetString("cloudflare.account_id"), viper.GetString("cloudflare.api_token"))
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

	pinataIPFS := NewPinataIPFSPinService()

	ctx, stop := signal.NotifyContext(mainCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	thumbnailCachePeriod, err := time.ParseDuration(viper.GetString("thumbnail.cache_period"))
	if err != nil {
		logrus.WithError(err).Error("invalid duration. use default value 72h")
		thumbnailCachePeriod = 72 * time.Hour
	}
	thumbnailCacheRetryInterval, err := time.ParseDuration(viper.GetString("thumbnail.cache_retry_interval"))
	if err != nil {
		logrus.WithError(err).Error("invalid duration. use default value 24h")
		thumbnailCacheRetryInterval = 24 * time.Hour
	}

	imageIndexer := NewNFTContentIndexer(store, assetCollection, pinataIPFS,
		thumbnailCachePeriod, thumbnailCacheRetryInterval)
	imageIndexer.Start(ctx)

	logrus.Info("Content indexer terminated")
}
