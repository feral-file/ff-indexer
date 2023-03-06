package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
	"github.com/spf13/viper"
)

type NFTAsset struct {
	ID string `bson:"id"`
}

func main() {
	var asset NFTAsset
	var assetMetadataDetail indexer.AssetMetadataDetail
	engine := indexer.New("", nil, tzkt.New(""), nil, objkt.New("https://data.objkt.com/v3/graphql"))

	config.LoadConfig("NFT_INDEXER")
	ctx := context.Background()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(viper.GetString("store.db_uri")))
	if err != nil {
		panic(err)
	}

	db := mongoClient.Database(viper.GetString("store.db_name"))
	assetsCollection := db.Collection("assets")
	tokenCollection := db.Collection("tokens")

	for {
		err = assetsCollection.FindOne(ctx, bson.M{"$or": bson.A{
			bson.M{"projectMetadata.latest.thumbnailURL": bson.M{"$regex": "https://ipfs.io/ipfs/"}},
			bson.M{"projectMetadata.latest.previewURL": bson.M{"$regex": "https://ipfs.io/ipfs/"}},
			bson.M{"projectMetadata.latest.lastUpdatedAt": bson.M{"$lt": time.Now().Add(-2 * 24 * time.Hour)}},
		}}).Decode(&asset)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				fmt.Println("finish update for all assets")
				break
			}
			panic(err)
		}

		// update lastUpdatedAt of asset
		_, err = assetsCollection.UpdateOne(
			ctx,
			bson.M{"id": asset.ID},
			bson.M{
				"$set": bson.M{
					"projectMetadata.latest.lastUpdatedAt": time.Now(),
				},
			},
		)

		if err != nil {
			fmt.Println(err)
			continue
		}

		// get token by assetID
		var token indexer.Token

		err = tokenCollection.FindOne(
			ctx,
			bson.M{
				"assetID": asset.ID,
			},
		).Decode(&token)

		if err != nil {
			fmt.Println(err)
			continue
		}

		// get contract and token id from token indexID
		s := strings.Split(token.IndexID, "-")
		contract := s[1]
		tokenID := s[2]

		// get objkt cdn
		objktToken, err := engine.GetObjktToken(contract, tokenID)
		if err != nil {
			fmt.Println(err)
			if strings.Contains(fmt.Sprint(err), "there is no token in objkt") {
				continue
			}

			fmt.Println(err)
			continue
		}

		assetMetadataDetail.FromObjkt(objktToken)

		// replace asset thumbnail url
		_, err = assetsCollection.UpdateOne(
			ctx,
			bson.M{"id": asset.ID},
			bson.M{
				"$set": bson.M{
					"projectMetadata.latest.previewURL":          assetMetadataDetail.PreviewURI,
					"projectMetadata.latest.thumbnailURL":        assetMetadataDetail.DisplayURI,
					"projectMetadata.latest.galleryThumbnailURL": assetMetadataDetail.DisplayURI,
				},
			},
		)

		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Println("update asset url for asset ID: ", asset.ID)
	}
}
