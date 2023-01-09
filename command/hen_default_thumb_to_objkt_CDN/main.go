package main

import (
	"context"
	"fmt"
	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strings"
)

type NFTAsset struct {
	IndexID string `bson:"indexID"`
}

func main() {
	var asset NFTAsset
	var indexIDs []string

	config.LoadConfig("NFT_INDEXER")
	ctx := context.Background()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(viper.GetString("store.db_uri")))
	if err != nil {
		panic(err)
	}

	db := mongoClient.Database(viper.GetString("store.db_name"))
	assetsCollection := db.Collection("assets")

	cursor, err := assetsCollection.Find(ctx, bson.M{
		"$and": bson.A{
			bson.M{"$or": bson.A{
				bson.M{"projectMetadata.latest.thumbnailURL": bson.M{"$regex": "QmNrhZHUaEqxhyLfqoq1mtHSipkWHeT31LNHb1QEbDHgnc"}},
				bson.M{"projectMetadata.latest.galleryThumbnailURL": bson.M{"$regex": "QmNrhZHUaEqxhyLfqoq1mtHSipkWHeT31LNHb1QEbDHgnc"}},
			}},
			bson.M{"$or": bson.A{
				bson.M{"indexID": bson.M{"$regex": "tez"}},
			}},
		},
	})

	if err != nil {
		defer cursor.Close(ctx)
		panic(err)
	}

	for cursor.Next(ctx) {
		err := cursor.Decode(&asset)
		if err != nil {
			panic(err)
		}
		indexIDs = append(indexIDs, asset.IndexID)
	}

	cursor.Close(ctx)

	ad := indexer.AssetMetadataDetail{}

	for _, indexID := range indexIDs {
		s := strings.Split(indexID, "-")
		thumbnailCDN := ad.ReplaceIPFSURIByObjktCDNURI(indexer.ObjktCDNArtifactThumbnailType, "ipfs://", s[1], s[2])

		if thumbnailCDN != "ipfs://" {
			_, err := assetsCollection.UpdateOne(
				ctx,
				bson.M{"indexID": bson.M{"$eq": indexID}},
				bson.M{"$set": bson.M{
					"projectMetadata.latest.thumbnailURL":        thumbnailCDN,
					"projectMetadata.latest.galleryThumbnailURL": thumbnailCDN,
				}},
			)

			if err != nil {
				fmt.Println(err)
			}

			fmt.Println("update thumbnail for asset have indexID: ", indexID)
		}
	}
}
