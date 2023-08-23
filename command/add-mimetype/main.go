package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Medium string

const (
	MediumUnknown  = "unknown"
	MediumVideo    = "video"
	MediumImage    = "image"
	MediumSoftware = "software"
	MediumOther    = "other"
)

type Asset struct {
	IndexID         string                   `json:"indexID" structs:"indexID" bson:"indexID"`
	ProjectMetadata VersionedProjectMetadata `json:"projectMetadata" structs:"projectMetadata" bson:"projectMetadata"`
}

type VersionedProjectMetadata struct {
	Origin ProjectMetadata `json:"origin" structs:"origin" bson:"origin"`
	Latest ProjectMetadata `json:"latest" structs:"latest" bson:"latest"`
}

type ProjectMetadata struct {
	MIMEType            string `json:"mimeType" structs:"mimeType" bson:"mimeType"`                                  // <mime_type from file extension or metadata>,
	Medium              string `json:"medium" structs:"medium" bson:"medium"`                                        // <"image" if image_url is present; "other" if animation_url is present> ,
	SourceURL           string `json:"sourceURL" structs:"sourceURL" bson:"sourceURL"`                               // <linktoSourceWebsite>,
	PreviewURL          string `json:"previewURL" structs:"previewURL" bson:"previewURL"`                            // <image_url or animation_url>,
	ThumbnailURL        string `json:"thumbnailURL" structs:"thumbnailURL" bson:"thumbnailURL"`                      // <image_thumbnail_url>,
	GalleryThumbnailURL string `json:"galleryThumbnailURL" structs:"galleryThumbnailURL" bson:"galleryThumbnailURL"` // <image_thumbnail_url>,
	AssetURL            string `json:"assetURL" structs:"assetURL" bson:"assetURL"`                                  // <permalink>

	// Operation attributes
	LastUpdatedAt time.Time `json:"lastUpdatedAt" structs:"lastUpdatedAt" bson:"lastUpdatedAt"`
}

type OpenseaAssetMetadata struct {
	ImageOriginalURL     string `json:"image_original_url" structs:"image_original_url"`
	AnimationOriginalURL string `json:"animation_original_url" structs:"animation_original_url"`
	ImageThumbnailURL    string `json:"image_thumbnail_url" structs:"image_thumbnail_url"`
}

func main() {
	dbURIInput := flag.String("mongouri", "mongodb://localhost:27017", "mongodb uri")
	flag.Parse()
	dbURI := *dbURIInput

	ctx := context.TODO()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dbURI))
	if err != nil {
		panic(err)
	}

	mongoCollection := client.Database("nft_indexer").Collection("assets")
	cursor, err := mongoCollection.Find(ctx, bson.M{
		"source":                          "opensea",
		"projectMetadata.latest.mimeType": "",
	})
	if err != nil {
		panic(err)
	}
	defer cursor.Close(ctx)

	var count int64
	startTime := time.Now()
	for cursor.Next(ctx) {
		var asset Asset
		if err := cursor.Decode(&asset); err != nil {
			continue
		}
		contractAddress, tokenID, err := parseAssetURL(asset.ProjectMetadata.Origin.AssetURL)
		if err != nil {
			continue
		}
		fmt.Printf("\n[%v] %s-%s.\n", time.Now(), contractAddress, tokenID)

		mimeType := indexer.GetMIMETypeByURL(asset.ProjectMetadata.Latest.PreviewURL)

		fmt.Printf("==> Mime type: %s, url: %s\n", mimeType, asset.ProjectMetadata.Latest.PreviewURL)

		result, err := mongoCollection.UpdateOne(
			ctx,
			bson.M{"indexID": asset.IndexID},
			bson.M{"$set": bson.M{"projectMetadata.latest.mimeType": mimeType}},
		)

		if err != nil {
			continue
		}
		count += result.ModifiedCount
	}

	fmt.Printf("Updated %v Documents in %v seconds!\n", count, time.Since(startTime))

}

func parseAssetURL(assetURL string) (string, string, error) {
	temp := strings.Split(assetURL, "/")

	if len(temp) <= 3 {
		return "", "", fmt.Errorf("the struct of assetURL is bad")
	}

	contractAddress := temp[len(temp)-2]
	tokenID := temp[len(temp)-1]

	return contractAddress, tokenID, nil
}
