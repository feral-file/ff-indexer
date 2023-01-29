package main

import (
	"context"
	"fmt"

	"github.com/bitmark-inc/config-loader"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var SelectedOwners = []string{
	"tz1hhG3vN2NCoPZepvEbKHBFQ5denwNJwHXP",
	"tz1fobhvuEWmHjGrjdmxm64rpkKFtJNKsNdo",
	"0xb858A3F45840E76076c6c4DBa9f0f8958F11C1E8",
	"0x0f87bac53e6CaDF4DBaeB9C2cECB52ceAD675BeF",
}

func main() {

	config.LoadConfig("NFT_INDEXER")

	ctx := context.Background()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(viper.GetString("store.db_uri")))
	if err != nil {
		panic(err)
	}

	db := mongoClient.Database(viper.GetString("store.db_name"))
	tokenCollection := db.Collection("tokens")

	// clean up demo tokens
	fmt.Printf("Clean all demo tokens\n")
	if _, err := tokenCollection.DeleteMany(ctx, bson.M{"is_demo": true}); err != nil {
		panic(err)
	}

	for _, owner := range SelectedOwners {
		var tokens []map[string]interface{}
		cursor, err := tokenCollection.Find(ctx, bson.M{fmt.Sprintf("owners.%s", owner): bson.M{"$gt": 0}})
		if err != nil {
			panic(err)
		}
		if err := cursor.All(ctx, &tokens); err != nil {
			panic(err)
		}

		fmt.Printf("Create %d demo tokens from address: %s\n", len(tokens), owner)

		demoTokens := []interface{}{}
		for i, token := range tokens {
			delete(token, "_id")
			indexID := token["indexID"].(string)
			token["indexID"] = fmt.Sprintf("demo-%d-%s", i, indexID)
			token["owners"] = map[string]int{"demo": 1, "demo2": 1}
			token["ownersArray"] = []string{"demo", "demo2"}
			token["is_demo"] = true

			demoTokens = append(demoTokens, token)
		}

		if _, err := tokenCollection.InsertMany(ctx, demoTokens); err != nil {
			panic(err)
		}
	}
}
