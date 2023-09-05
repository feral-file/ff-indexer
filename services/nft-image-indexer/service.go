package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	indexer "github.com/bitmark-inc/nft-indexer"
	imageStore "github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/store"
)

type NFTAsset struct {
	ID              string                           `bson:"id"`
	IndexID         string                           `bson:"indexID"`
	ProjectMetadata indexer.VersionedProjectMetadata `bson:"projectMetadata"`
}

type NFTContentIndexer struct {
	wg sync.WaitGroup

	thumbnailCachePeriod        time.Duration
	thumbnailCacheRetryInterval time.Duration

	db               *imageStore.ImageStore
	nftAssets        *mongo.Collection
	nftTokens        *mongo.Collection
	nftAccountTokens *mongo.Collection
}

func NewNFTContentIndexer(db *imageStore.ImageStore, nftAssets, nftTokens, nftAccountTokens *mongo.Collection,
	thumbnailCachePeriod, thumbnailCacheRetryInterval time.Duration) *NFTContentIndexer {
	return &NFTContentIndexer{
		thumbnailCachePeriod:        thumbnailCachePeriod,
		thumbnailCacheRetryInterval: thumbnailCacheRetryInterval,

		db:               db,
		nftAssets:        nftAssets,
		nftTokens:        nftTokens,
		nftAccountTokens: nftAccountTokens,
	}
}

// spawnThumbnailWorker spawn worker for generate thumbnails from source images
func (s *NFTContentIndexer) spawnThumbnailWorker(ctx context.Context, assets <-chan NFTAsset, count int) {
	for i := 0; i < count; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for asset := range assets {
				log.Debug("start generating thumbnail cache for an asset", zap.String("indexID", asset.IndexID))

				if _, err := s.db.CreateOrGetImage(ctx, asset.IndexID); err != nil {
					log.Error("fail to get or create image record", zap.Error(err))
					continue
				}

				uploadImageStartTime := time.Now()
				img, err := s.db.UploadImage(ctx, asset.IndexID, NewURLImageReader(asset.ProjectMetadata.Latest.ThumbnailURL),
					map[string]interface{}{
						"source":   asset.ProjectMetadata.Latest.Source,
						"file_url": asset.ProjectMetadata.Latest.ThumbnailURL,
					},
				)
				if err != nil {
					if uerr, ok := err.(imageStore.UnsupportedImageCachingError); ok {
						// add failure to the asset
						if uerr.Reason() == imageStore.ReasonBrokenImage {
							log.Error("broken image",
								zap.String("indexID", asset.IndexID),
								zap.String("thumbnailURL", asset.ProjectMetadata.Latest.ThumbnailURL))
						}

						if err := s.markAssetThumbnailFailed(ctx, asset.IndexID, uerr.Reason()); err != nil {
							log.Error("add thumbnail failure was failed", zap.String("indexID", asset.IndexID), zap.Error(err))
						}
					}

					sentry.CaptureMessage("assetId: " + asset.IndexID + " - " + err.Error())
					log.Error("fail to upload image", zap.String("indexID", asset.IndexID), zap.Error(err))
					continue
				}
				log.Debug("thumbnail image uploaded",
					zap.Duration("duration", time.Since(uploadImageStartTime)),
					zap.String("assetID", asset.IndexID))

				// Update the thumbnail by image ID returned from cloudflare, it the whol process is succeed.
				// Otherwise, it would update to an empty value
				if err := s.updateAssetThumbnail(ctx, img.AssetID, img.ImageID); err != nil {
					log.Error("fail to update token thumbnail back to indexer", zap.Error(err))
					continue
				}
				log.Info("thumbnail generating process finished", zap.String("indexID", asset.IndexID))
			}
			log.Debug("ThumbnailWorker stopped")
		}()
	}
}

// getAssetWithoutThumbnailCached looks up assets without thumbnail cached
func (s *NFTContentIndexer) getAssetWithoutThumbnailCached(ctx context.Context) (NFTAsset, error) {
	var asset NFTAsset
	r := s.nftAssets.FindOneAndUpdate(ctx,
		bson.M{
			// filter assets which have not been processed in the last hour.
			// NOTE: the $lt query will exclude nil values so we need to ensure `thumbnailLastCheck`
			// has a default value
			"thumbnailLastCheck": bson.M{"$lt": time.Now().Add(-s.thumbnailCacheRetryInterval)},
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
		log.Info("update asset thumbnail failed", zap.String("indexID", indexID), zap.Error(err))
		return err
	}

	idSegments := strings.Split(indexID, "-")
	if len(idSegments) != 2 {
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

		assets := make(chan NFTAsset, 20)
		defer close(assets)

		s.spawnThumbnailWorker(ctx, assets, 10)

		log.Info("start the loop the get assets without thumbnail cached",
			zap.Duration("thumbnailCachePeriod", s.thumbnailCachePeriod),
			zap.Duration("thumbnailCacheRetryInterval", s.thumbnailCacheRetryInterval))

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
			assets <- asset
		}
		log.Debug("Thumbnail checker closed")
	}()

}

func (s *NFTContentIndexer) Start(ctx context.Context) {
	s.checkThumbnail(ctx)
	s.wg.Wait()
}
