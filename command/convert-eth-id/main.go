package main

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Token struct {
	Id              string           `json:"id" bson:"id"`
	BaseTokenInfo   `bson:",inline"` // the latest token info
	Edition         int64            `json:"edition" bson:"edition"`
	MintAt          time.Time        `json:"mintedAt" bson:"mintedAt"`
	Balance         int64            `json:"balance" bson:"-"` // a temporarily state of balance for a specific owner
	Owner           string           `json:"owner" bson:"owner"`
	Owners          map[string]int64 `json:"owners" bson:"owners"`
	OwnersArray     []string         `json:"-" bson:"ownersArray"`
	AssetID         string           `json:"-" bson:"assetID"`
	OriginTokenInfo []BaseTokenInfo  `json:"originTokenInfo" bson:"originTokenInfo"`

	IndexID           string       `json:"indexID" bson:"indexID"`
	Source            string       `json:"source" bson:"source"`
	Swapped           bool         `json:"swapped" bson:"swapped"`
	SwappedFrom       *string      `json:"-" bson:"swappedFrom,omitempty"`
	SwappedTo         *string      `json:"-" bson:"swappedTo,omitempty"`
	Burned            bool         `json:"burned" bson:"burned"`
	Provenances       []Provenance `json:"provenance" bson:"provenance"`
	LastActivityTime  time.Time    `json:"lastActivityTime" bson:"lastActivityTime"`
	LastRefreshedTime time.Time    `json:"-" bson:"lastRefreshedTime"`
}

type Provenance struct {
	// this field is only for ownership validating
	FormerOwner *string `json:"formerOwner,omitempty" bson:"-"`

	Type       string    `json:"type" bson:"type"`
	Owner      string    `json:"owner" bson:"owner"`
	Blockchain string    `json:"blockchain" bson:"blockchain"`
	Timestamp  time.Time `json:"timestamp" bson:"timestamp"`
	TxID       string    `json:"txid" bson:"txid"`
	TxURL      string    `json:"txURL" bson:"txURL"`
}

type BaseTokenInfo struct {
	ID              string `json:"id" bson:"id"`
	Blockchain      string `json:"blockchain" bson:"blockchain"`
	Fungible        bool   `json:"fungible" bson:"fungible"`
	ContractType    string `json:"contractType" bson:"contractType"`
	ContractAddress string `json:"contractAddress,omitempty" bson:"contractAddress"`
}

func main() {
	ctx := context.TODO()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		panic(err)
	}

	mongoCollection := client.Database("nft_indexer").Collection("tokens")
	cursor, err := mongoCollection.Find(ctx, bson.M{"blockchain": "ethereum"})
	if err != nil {
		panic(err)
	}
	defer cursor.Close(ctx)

	var count int64 = 0
	for cursor.Next(ctx) {
		var token Token
		if err := cursor.Decode(&token); err != nil {
			continue
		}
		blockchain, contractAddress, tokenID, err := parseTokenIndexID(token.IndexID)
		if err != nil {
			continue
		}

		newId, ok := big.NewInt(0).SetString(tokenID, 16)
		if !ok {
			panic(ok)
		}

		newIndexID := fmt.Sprintf("%s-%s-%s", blockchain, contractAddress, newId.String())

		result, err := mongoCollection.UpdateOne(
			ctx,
			bson.M{"indexID": token.IndexID},
			bson.D{
				{"$set", bson.D{{"id", newId.String()}, {"indexID", newIndexID}}},
			},
		)

		if err != nil {
			panic(err)
		}
		count += result.ModifiedCount
	}
	fmt.Printf("Updated %v Documents!\n", count)

}

func parseTokenIndexID(indexID string) (string, string, string, error) {
	temp := strings.Split(indexID, "-")

	if len(temp) != 3 {
		return "", "", "", fmt.Errorf("the struct of indexID is bad")
	}

	blockchain := temp[0]
	contractAddress := temp[1]
	tokenID := temp[2]

	return blockchain, contractAddress, tokenID, nil
}
