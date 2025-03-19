package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/bitmark-inc/autonomy-logger"
	"github.com/bitmark-inc/config-loader"
	imageStore "github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/store"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func main() {
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config.LoadConfig("NFT_INDEXER")
	if err := log.Initialize(viper.GetBool("debug"), &sentry.ClientOptions{
		Dsn: viper.GetString("sentry.dsn"),
	}); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
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
	tokenCollection := db.Collection("tokens")
	accountTokenCollection := db.Collection("account_tokens")
	collectionsCollection := db.Collection("collections")

	ctx, stop := signal.NotifyContext(mainCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	thumbnailCachePeriod, err := time.ParseDuration(viper.GetString("thumbnail.cache_period"))
	if err != nil {
		log.ErrorWithContext(ctx, errors.New("invalid duration. use default value 72h"), zap.Error(err))
		thumbnailCachePeriod = 72 * time.Hour
	}
	thumbnailCacheRetryInterval, err := time.ParseDuration(viper.GetString("thumbnail.cache_retry_interval"))
	if err != nil {
		log.ErrorWithContext(ctx, errors.New("invalid duration. use default value 24h"), zap.Error(err))
		thumbnailCacheRetryInterval = 24 * time.Hour
	}

	log.Debug("cache settings",
		zap.Duration("period", thumbnailCachePeriod),
		zap.Duration("retry", thumbnailCacheRetryInterval),
	)

	imageIndexer := NewNFTContentIndexer(store, assetCollection, tokenCollection, accountTokenCollection, collectionsCollection,
		thumbnailCachePeriod, thumbnailCacheRetryInterval, viper.GetString("cloudflare.url_prefix"))
	imageIndexer.Start(ctx)

	log.InfoWithContext(ctx, "Content indexer terminated")
}
