package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

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

	imageIndexer := NewNFTContentIndexer(store, assetCollection, pinataIPFS)
	imageIndexer.Start(ctx)

	logrus.Info("Content indexer terminated")
}
