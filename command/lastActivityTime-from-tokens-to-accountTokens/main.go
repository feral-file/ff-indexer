package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"time"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/config-loader"
)

type accountToken struct {
	IndexID string `bson:"indexID"`
}

type token struct {
	LastActivityTime time.Time `bson:"lastActivityTime"`
}

func main() {
	config.LoadConfig("NFT_INDEXER")
	ctx := context.Background()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(viper.GetString("store.db_uri")))
	if err != nil {
		panic(err)
	}

	db := mongoClient.Database(viper.GetString("store.db_name"))
	tokensCollection := db.Collection("tokens")
	accountTokensCollection := db.Collection("account_tokens")

	indexIDs, err := findAllAccountTokensWithoutLastActivityTime(ctx, accountTokensCollection)
	if err != nil {
		panic(err)
	}

	for _, indexID := range indexIDs {
		lastActivityTime, err := getTokenLastActivityTime(ctx, tokensCollection, indexID)
		if err != nil {
			fmt.Println("error occur when get token lastActivityTime", indexID, err)
			continue
		}

		_, err = accountTokensCollection.UpdateOne(
			ctx,
			bson.M{"indexID": indexID},
			bson.M{"$set": bson.M{"lastActivityTime": lastActivityTime}},
		)
		if err != nil {
			fmt.Println("error occur when update accountTokens with the latest lastActivityTime", indexID, err)
			continue
		}
	}
}

// getTokenLastActivityTime returns the lastActivityTime of token by indexID
func getTokenLastActivityTime(ctx context.Context, tokensCollection *mongo.Collection, indexID string) (time.Time, error) {
	var token token

	err := tokensCollection.FindOne(
		ctx,
		bson.M{
			"indexID": indexID,
		},
		options.FindOne().SetProjection(bson.M{"lastActivityTime": 1}),
	).Decode(&token)

	if err != nil {
		fmt.Println("error occur when find token by indexID", indexID, err)
		return time.Time{}, err
	}

	return token.LastActivityTime, nil
}

// findAllAccountTokensWithoutLastActivityTime returns all accountTokens that not have lastActivityTime
func findAllAccountTokensWithoutLastActivityTime(ctx context.Context, accountTokensCollection *mongo.Collection) ([]string, error) {
	cursor, err := accountTokensCollection.Find(
		ctx,
		bson.M{"lastActivityTime": bson.M{"$exists": false}},
		nil,
	)

	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx)

	var accountTokens []accountToken

	err = cursor.All(ctx, &accountTokens)
	if err != nil {
		return nil, err
	}

	var indexIDs []string

	for _, accountToken := range accountTokens {
		indexIDs = append(indexIDs, accountToken.IndexID)
	}

	return indexIDs, err
}
