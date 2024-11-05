package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/bitmark-inc/config-loader"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {

	config.LoadConfig("NFT_INDEXER")

	ctx := context.Background()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(viper.GetString("store.db_uri")))
	if err != nil {
		panic(err)
	}

	db := mongoClient.Database(viper.GetString("store.db_name"))
	ffIdentityCollection := db.Collection("ff_identities")

	identities := []struct {
		Alias    string `json:"alias"`
		Tezos    string `json:"tezos"`
		Ethereum string `json:"ethereum"`
		Bitmark  string `json:"bitmark"`
	}{}

	f, err := os.Open("./ff_identities.json")
	if err != nil {
		panic(err)
	}

	if err := json.NewDecoder(f).Decode(&identities); err != nil {
		panic(err)
	}

	var records bson.A
	for _, i := range identities {
		if i.Alias == "" {
			panic("Alias is required")
		}
		if i.Tezos != "" {
			records = append(records, bson.M{"accountNumber": i.Tezos, "blockchain": "tezos", "name": i.Alias})
		}
		if i.Bitmark != "" {
			records = append(records, bson.M{"accountNumber": i.Bitmark, "blockchain": "bitmark", "name": i.Alias})
		}
		if i.Ethereum != "" {
			records = append(records, bson.M{"accountNumber": i.Ethereum, "blockchain": "ethereum", "name": i.Alias})
		}
	}

	if _, err := ffIdentityCollection.InsertMany(ctx, records); err != nil {
		fmt.Println(err)
		panic(err)
	}

	fmt.Println("Imported ff_identities.json")
}
