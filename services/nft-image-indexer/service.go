package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/bitmark-inc/autonomy-logger"
	indexer "github.com/bitmark-inc/nft-indexer"
	imageStore "github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/store"
	"github.com/getsentry/sentry-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type Type string

const (
	TypeCollection = "collection"
	TypeAsset      = "asset"
)

type NFTAsset struct {
	ID              string                           `bson:"id"`
	IndexID         string                           `bson:"indexID"`
	ProjectMetadata indexer.VersionedProjectMetadata `bson:"projectMetadata"`
}

type ThumbnailIndexInfo struct {
	ID       string
	ImageURL string
	Metadata map[string]interface{}
	Type     Type
}

type NFTContentIndexer struct {
	wg sync.WaitGroup

	thumbnailCachePeriod        time.Duration
	thumbnailCacheRetryInterval time.Duration

	cloudflareURLPrefix string

	db               *imageStore.ImageStore
	nftAssets        *mongo.Collection
	nftTokens        *mongo.Collection
	nftAccountTokens *mongo.Collection
	nftCollections   *mongo.Collection
}

func NewNFTContentIndexer(db *imageStore.ImageStore, nftAssets, nftTokens, nftAccountTokens, nftCollections *mongo.Collection,
	thumbnailCachePeriod, thumbnailCacheRetryInterval time.Duration, cloudflareURLPrefix string) *NFTContentIndexer {
	return &NFTContentIndexer{
		thumbnailCachePeriod:        thumbnailCachePeriod,
		thumbnailCacheRetryInterval: thumbnailCacheRetryInterval,

		cloudflareURLPrefix: cloudflareURLPrefix,

		db:               db,
		nftAssets:        nftAssets,
		nftTokens:        nftTokens,
		nftAccountTokens: nftAccountTokens,
		nftCollections:   nftCollections,
	}
}

// spawnThumbnailWorker spawn worker for generate thumbnails from source images
func (s *NFTContentIndexer) spawnThumbnailWorker(ctx context.Context, infos <-chan ThumbnailIndexInfo, count int) {
	for i := 0; i < count; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for info := range infos {
				log.Debug("start generating thumbnail cache for an asset", zap.String("indexID", info.ID))

				if _, err := s.db.CreateOrGetImage(ctx, info.ID); err != nil {
					log.Error("fail to get or create image record", zap.Error(err))
					continue
				}

				uploadImageStartTime := time.Now()
				img, err := s.db.UploadImage(ctx, info.ID, NewURLImageReader(info.ImageURL),
					info.Metadata,
				)
				if err != nil {
					if uerr, ok := err.(imageStore.UnsupportedImageCachingError); ok {
						// add failure to the asset
						if uerr.Reason() == imageStore.ReasonBrokenImage {
							log.Error("broken image",
								zap.String("id", info.ID),
								zap.String("type", string(info.Type)),
								zap.String("thumbnailURL", info.ImageURL))
						}

						if err := s.markAssetThumbnailFailed(ctx, info.ID, uerr.Reason()); err != nil {
							log.Error("add thumbnail failure was failed", zap.String("id", info.ID), zap.Error(err))
						}
					}

					sentry.CaptureMessage("assetId: " + info.ID + " - " + err.Error())
					log.Error("fail to upload image", zap.String("id", info.ID), zap.Error(err))
					continue
				}
				log.Debug("thumbnail image uploaded",
					zap.Duration("duration", time.Since(uploadImageStartTime)),
					zap.String("id", info.ID))

				// Update the thumbnail by image ID returned from cloudflare, it the whol process is succeed.
				// Otherwise, it would update to an empty value
				switch info.Type {
				case TypeAsset:
					if err := s.updateAssetThumbnail(ctx, img.AssetID, img.ImageID); err != nil {
						log.Error("fail to update token thumbnail back to indexer", zap.Error(err))
						continue
					}
				case TypeCollection:
					if err := s.updateCollectionThumbnail(ctx, img.AssetID, img.ImageID); err != nil {
						log.Error("fail to update token thumbnail back to indexer", zap.Error(err))
						continue
					}
				default:
					log.Error("type is not supported", zap.Error(err))
					continue
				}

				log.Info("thumbnail generating process finished", zap.String("indexID", info.ID))
			}
			log.Debug("ThumbnailWorker stopped")
		}()
	}
}

// getAssetWithoutThumbnailCached looks up assets without thumbnail cached
func (s *NFTContentIndexer) getAssetWithoutThumbnailCached(ctx context.Context) (NFTAsset, error) {
	var asset NFTAsset
	ts := time.Now().Add(-s.thumbnailCacheRetryInterval)
	r := s.nftAssets.FindOneAndUpdate(ctx,
		bson.M{ // This is effectively "$and"

			// filter recent assets which have not been processed or are not timestamped
			//"thumbnailLastCheck": bson.M{"$lt": time.Now().Add(-s.thumbnailCacheRetryInterval)},
			"$or": bson.A{
				bson.M{ // this will be false of any non time values
					"thumbnailLastCheck": bson.M{"$lt": ts},
				},
				bson.M{ // include null and empty string to cover both defaults
					"thumbnailLastCheck": bson.M{"$in": bson.A{nil, ""}},
				},
			},

			// filter assets which does not have thumbnailID or the thumbnailID is empty
			"thumbnailID": bson.M{
				// "$not": bson.M{"$exists": true, "$ne": ""},
				"$in": bson.A{nil, ""},
			},

			// filter assets which does not have thumbnailFailure or the thumbnailFailure is empty
			"thumbnailFailedReason": bson.M{
				// "$not": bson.M{"$exists": true},
				"$in": bson.A{nil, ""},
			},
			// filter assets which are qualified to generate thumbnails in cloudflare.
			// "$or": bson.A{
			// 	// filter all tokens that set SVG as the mime-type and their thumbnail URLs start with https
			// 	bson.M{
			// 		"projectMetadata.latest.mimeType":     "image/svg+xml",
			// 		"projectMetadata.latest.thumbnailURL": bson.M{"$regex": "^https://"},
			// 	},
			// 	// For tezos tokens, it parses tokens that starts with `https://ipfs.` which means
			// 	// all token that is uploaded to IPFS but is not cached by objkt.
			// 	bson.M{
			// 		"source":                              "tzkt",
			// 		"projectMetadata.latest.source":       bson.M{"$nin": []string{"fxhash"}},
			// 		"projectMetadata.latest.thumbnailURL": bson.M{"$regex": "^https://ipfs"}, // either ipfs.io or ipfs.bitmark
			// 	},
			// 	// For opensea tokens, we can only check the mime-type of an asset by its file extension.
			// 	bson.M{
			// 		"source": "opensea",
			// 		"projectMetadata.latest.thumbnailURL": bson.M{
			// 			"$regex": ".svg$",
			// 		},
			// 	},
			// },
		},
		bson.M{"$set": bson.M{"thumbnailLastCheck": time.Now()}},
		options.FindOneAndUpdate().
			// SetSort(bson.D{{Key: "projectMetadata.latest.lastUpdatedAt", Value: -1}}).
			SetProjection(bson.M{"id": 1, "indexID": 1, "projectMetadata.latest.thumbnailURL": 1}),
	)

	if err := r.Err(); err != nil {
		return asset, err
	}

	err := r.Decode(&asset)
	return asset, err
}

// getAssetWithoutThumbnailCached looks up assets without thumbnail cached
func (s *NFTContentIndexer) getCollectionWithoutThumbnailCached(ctx context.Context) (indexer.Collection, error) {
	var collection indexer.Collection
	ts := time.Now().Add(-s.thumbnailCacheRetryInterval)
	r := s.nftCollections.FindOneAndUpdate(ctx,
		bson.M{ // This is effectively "$and"

			// filter recent assets which have not been processed or are not timestamped
			//"thumbnailLastCheck": bson.M{"$lt": time.Now().Add(-s.thumbnailCacheRetryInterval)},
			"$or": bson.A{
				bson.M{ // this will be false of any non time values
					"thumbnailLastCheck": bson.M{"$lt": ts},
				},
				bson.M{ // include null and empty string to cover both defaults
					"thumbnailLastCheck": bson.M{"$in": bson.A{nil, ""}},
				},
			},

			// filter assets which does not have thumbnailURL or the thumbnailURL is empty
			"thumbnailURL": bson.M{
				// "$not": bson.M{"$exists": true, "$ne": ""},
				"$in": bson.A{nil, ""},
			},

			// filter assets which does not have thumbnailFailure or the thumbnailFailure is empty
			"thumbnailFailedReason": bson.M{
				// "$not": bson.M{"$exists": true},
				"$in": bson.A{nil, ""},
			},
		},
		bson.M{"$set": bson.M{"thumbnailLastCheck": time.Now()}},
		options.FindOneAndUpdate().
			SetProjection(bson.M{"id": 1, "imageURL": 1}),
	)

	if err := r.Err(); err != nil {
		return collection, err
	}

	err := r.Decode(&collection)
	return collection, err
}

// updateAssetThumbnail sets the thumbnail id for a specific token
func (s *NFTContentIndexer) updateAssetThumbnail(ctx context.Context, indexID, thumbnailID string) error {
	_, err := s.nftAssets.UpdateOne(
		ctx,
		bson.M{"indexID": indexID},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "thumbnailID", Value: thumbnailID},
			{Key: "lastRefreshedTime", Value: time.Now()},
		}}},
	)
	if err != nil {
		log.Error("update asset thumbnail failed", zap.String("indexID", indexID), zap.Error(err))
		return err
	}

	idSegments := strings.SplitN(indexID, "-", 2)
	if len(idSegments) != 2 {
		log.Error("invalid asset index id",
			zap.String("indexID", indexID),
			zap.Int("segments", len(idSegments)))
		return fmt.Errorf("invalid asset index id")
	}
	assetID := idSegments[1]

	cursor, err := s.nftTokens.Find(ctx,
		bson.M{"assetID": assetID},
		options.Find().SetProjection(bson.M{"indexID": 1}))
	if err != nil {
		return err
	}

	indexIDs := bson.A{}
	for cursor.Next(ctx) {
		var token struct {
			IndexID string `bson:"indexID"`
		}
		if err := cursor.Decode(&token); err != nil {
			return err
		}

		indexIDs = append(indexIDs, token.IndexID)
	}

	_, err = s.nftAccountTokens.UpdateMany(
		ctx,
		bson.M{"indexID": bson.M{"$in": indexIDs}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "lastRefreshedTime", Value: time.Now()},
		}}},
	)

	return err
}

// updateAssetThumbnail sets the thumbnail id for a specific token
func (s *NFTContentIndexer) updateCollectionThumbnail(ctx context.Context, id, thumbnailID string) error {
	thumbnailURL := fmt.Sprintf("%s%s/%s", s.cloudflareURLPrefix, thumbnailID, "thumbnail")
	_, err := s.nftCollections.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "thumbnailURL", Value: thumbnailURL},
			{Key: "lastUpdatedTime", Value: time.Now()},
		}}},
	)
	if err != nil {
		log.Error("update collection thumbnail failed", zap.String("collectionID", id), zap.Error(err))
		return err
	}

	return err
}

// markAssetThumbnailFailed sets thumbnail failure for a specific token
func (s *NFTContentIndexer) markAssetThumbnailFailed(ctx context.Context, indexID, thumbnailFailedReason string) error {
	_, err := s.nftAssets.UpdateOne(
		ctx,
		bson.M{"indexID": indexID},
		bson.D{{Key: "$set", Value: bson.D{{Key: "thumbnailFailedReason", Value: thumbnailFailedReason}}}},
	)

	return err
}

func (s *NFTContentIndexer) checkThumbnail(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		dataChan := make(chan ThumbnailIndexInfo, 20)
		defer close(dataChan)

		s.spawnThumbnailWorker(ctx, dataChan, 10)

		log.Info("start the loop the get assets without thumbnail cached",
			zap.Duration("thumbnailCachePeriod", s.thumbnailCachePeriod),
			zap.Duration("thumbnailCacheRetryInterval", s.thumbnailCacheRetryInterval))

		go func() {
		WATCH_COLLECTION:
			for {
				col, err := s.getCollectionWithoutThumbnailCached(ctx)
				if err != nil {
					if errors.Is(err, mongo.ErrNoDocuments) {
						log.Info("No collection need to generate cache a thumbnail")
					} else {
						log.Error("fail to get collection", zap.Error(err))
					}

					if done := indexer.SleepWithContext(ctx, 15*time.Second); done {
						break WATCH_COLLECTION
					}
					continue
				}
				log.Debug("send collection to process", zap.String("id", col.ID))
				dataChan <- ThumbnailIndexInfo{
					ID:       col.ID,
					ImageURL: col.ImageURL,
					Metadata: map[string]interface{}{
						"source":   col.Source,
						"file_url": col.ImageURL,
					},
					Type: TypeAsset,
				}
			}
		}()

	WATCH_ASSETS:
		for {
			asset, err := s.getAssetWithoutThumbnailCached(ctx)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					log.Info("No token need to generate cache a thumbnail")
				} else {
					log.Error("fail to get asset", zap.Error(err))
				}

				if done := indexer.SleepWithContext(ctx, 15*time.Second); done {
					break WATCH_ASSETS
				}
				continue
			}
			log.Debug("send asset to process", zap.String("indexID", asset.IndexID))
			dataChan <- ThumbnailIndexInfo{
				ID:       asset.IndexID,
				ImageURL: asset.ProjectMetadata.Latest.ThumbnailURL,
				Metadata: map[string]interface{}{
					"source":   asset.ProjectMetadata.Latest.Source,
					"file_url": asset.ProjectMetadata.Latest.ThumbnailURL,
				},
				Type: TypeAsset,
			}
		}
		log.Debug("Thumbnail checker closed")
	}()

}

func (s *NFTContentIndexer) Start(ctx context.Context) {
	s.checkThumbnail(ctx)
	s.wg.Wait()
}
