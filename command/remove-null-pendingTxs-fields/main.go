package main

import (
	"context"
	"flag"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	dbURIInput := flag.String("mongouri", "mongodb://localhost:27017", "mongodb uri")
	flag.Parse()
	dbURI := *dbURIInput

	ctx := context.TODO()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dbURI))
	if err != nil {
		panic(err)
	}

	mongoCollection := client.Database("nft_indexer").Collection("account_tokens")
	result, err := mongoCollection.UpdateMany(ctx,
		bson.M{
			"$or": bson.A{
				bson.M{"pendingTxs": bson.M{"$type": 10}},
				bson.M{"lastPendingTime": bson.M{"$type": 10}},
			}},
		bson.M{
			"$unset": bson.M{"pendingTxs": "", "lastPendingTime": ""},
		})
	if err != nil {
		panic(err)
	}

	fmt.Println("Updated: ", result.ModifiedCount)
}
