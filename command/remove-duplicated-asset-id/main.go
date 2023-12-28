package main

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/config-loader"
	"github.com/spf13/viper"
)

func main() {
	config.LoadConfig("NFT_INDEXER")
	ctx := context.Background()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(viper.GetString("store.db_uri")))
	if err != nil {
		panic(err)
	}

	db := mongoClient.Database(viper.GetString("store.db_name"))
	assetCollection := db.Collection("assets")

	pipeline := []bson.M{
		{"$group": bson.D{
			{Key: "_id", Value: bson.M{
				"id":     "$id",
				"source": "$source",
			}},
			{Key: "count", Value: bson.M{"$sum": 1}},
			{Key: "docs", Value: bson.M{"$push": "$_id"}},
		}},
		{"$match": bson.M{"count": bson.M{"$gt": 1}}},
	}

	cursor, err := assetCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Fatal(err)
	}

	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var duplicate struct {
			Docs []interface{} `bson:"docs"`
		}
		if err := cursor.Decode(&duplicate); err != nil {
			log.Fatal(err)
		}

		// keep first document and remove the rest
		objectIDs := duplicate.Docs[1:]

		// remove the duplicates
		_, err := assetCollection.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": objectIDs}})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Removed duplicates assets")
	}
}
