package main

import (
	"context"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/config-loader"
	"github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/imageStore"
)

func main() {
	ctx := context.Background()

	config.LoadConfig("NFT_INDEXER")

	store := imageStore.New(
		viper.GetString("image_db.dsn"),
		viper.GetString("cloudflare.account_id"), viper.GetString("cloudflare.api_token"))
	if err := store.AutoMigrate(); err != nil {
		panic(err)
	}

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(viper.GetString("store.db_uri")))
	if err != nil {
		panic(err)
	}

	// nft indexer store
	db := mongoClient.Database(viper.GetString("store.db_name"))
	assetCollection := db.Collection("assets")

	imageIndexer := NewNFTImageIndexer(store, assetCollection)
	imageIndexer.Start(ctx)

	// TODO: detect signal the close the process gracefully
	// detect the signal to close and stop the process
	// close(assets)
	// s.wg.Wait()
}
