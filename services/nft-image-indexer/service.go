package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/customErrors"
	"github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/imageStore"
	log "github.com/bitmark-inc/nft-indexer/zapLog"
)

type NFTAsset struct {
	IndexID         string                           `bson:"indexID"`
	ProjectMetadata indexer.VersionedProjectMetadata `bson:"projectMetadata"`
}

type NFTContentIndexer struct {
	wg sync.WaitGroup

	thumbnailCachePeriod        time.Duration
	thumbnailCacheRetryInterval time.Duration

	db        *imageStore.ImageStore
	ipfs      IPFSPinService
	nftAssets *mongo.Collection
}

func NewNFTContentIndexer(db *imageStore.ImageStore, nftAssets *mongo.Collection, ipfs IPFSPinService,
	thumbnailCachePeriod, thumbnailCacheRetryInterval time.Duration) *NFTContentIndexer {
	return &NFTContentIndexer{
		thumbnailCachePeriod:        thumbnailCachePeriod,
		thumbnailCacheRetryInterval: thumbnailCacheRetryInterval,

		db:        db,
		ipfs:      ipfs,
		nftAssets: nftAssets,
	}
}

// spawnThumbnailWorker spawn worker for generate thumbnails from source images
func (s *NFTContentIndexer) spawnThumbnailWorker(ctx context.Context, assets <-chan NFTAsset, count int) {
	for i := 0; i < count; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for asset := range assets {
				log.Logger.Debug("start generating thumbnail cache for an asset", zap.String("indexID", asset.IndexID))

				if _, err := s.db.CreateOrGetImage(ctx, asset.IndexID); err != nil {
					log.Logger.Error("fail to get or create image record", zap.Error(err))
					continue
				}

				uploadImageStartTime := time.Now()
				img, err := s.db.UploadImage(ctx, asset.IndexID, NewURLImageDownloader(asset.ProjectMetadata.Latest.ThumbnailURL),
					map[string]interface{}{
						"source":   asset.ProjectMetadata.Latest.Source,
						"file_url": asset.ProjectMetadata.Latest.ThumbnailURL,
					},
				)
				if err != nil {
					if errors.Is(err, imageStore.ErrUnsupportImageType) {
						log.Logger.Warn("unsupported image type", zap.String("indexID", asset.IndexID), zap.String("apiSource", log.ImageCaching))
						// let the image id remain empty string
					} else if _, ok := err.(*customErrors.UnsupportedSVG); ok {
						log.Logger.Error("fail to upload image", zap.Error(err), zap.String("indexID", asset.IndexID), zap.String("apiSource", log.ImageCaching))
						sentry.CaptureMessage("assetId: " + asset.IndexID + " - " + err.Error())
					} else {
						log.Logger.Error("fail to upload image", zap.Error(err), zap.String("indexID", asset.IndexID), zap.String("apiSource", log.ImageCaching))
					}
				}
				log.Logger.Debug("thumbnail image uploaded",
					zap.Duration("duration", time.Since(uploadImageStartTime)),
					zap.String("assetID", asset.IndexID))

				// Update the thumbnail by image ID returned from cloudflare, it the whol process is succeed.
				// Otherwise, it would update to an empty value
				if err := s.updateAssetThumbnail(ctx, img.AssetID, img.ImageID); err != nil {
					logrus.WithError(err).Error("update token thumbnail to indexer")
				}

				log.Logger.Info("thumbnail generating process finished", zap.String("indexID", asset.IndexID))
			}
			log.Logger.Debug("ThumbnailWorker stopped")
		}()
	}
}

// getAssetWithoutThumbnailCached looks up assets without thumbnail cached
func (s *NFTContentIndexer) getAssetWithoutThumbnailCached(ctx context.Context) (NFTAsset, error) {
	var asset NFTAsset
	r := s.nftAssets.FindOneAndUpdate(ctx,
		bson.M{
			// filter assets which have been viewed in the past 7 days.
			"projectMetadata.latest.lastUpdatedAt": bson.M{"$gt": time.Now().Add(-s.thumbnailCachePeriod)},
			// filter assets which have not been processed in the last hour.
			"thumbnailLastCheck": bson.M{
				"$not": bson.M{"$gt": time.Now().Add(-s.thumbnailCacheRetryInterval)},
			},
			// filter assets which does not have thumbnailID or the thumbnailID is empty
			"thumbnailID": bson.M{
				"$not": bson.M{"$exists": true, "$ne": ""},
			},
			// filter assets which are qualified to generate thumbnails in cloudflare.
			"$or": bson.A{
				// filter all tokens that set SVG as the mime-type and their thumbnail URLs start with https
				bson.M{
					"projectMetadata.latest.mimeType":     "image/svg+xml",
					"projectMetadata.latest.thumbnailURL": bson.M{"$regex": "^https://"},
				},
				// For tezos tokens, it parses tokens that starts with `https://ipfs.` which means
				// all token that is uploaded to IPFS but is not cached by objkt.
				bson.M{
					"source":                              "tzkt",
					"projectMetadata.latest.source":       bson.M{"$nin": []string{"fxhash"}},
					"projectMetadata.latest.thumbnailURL": bson.M{"$regex": "^https://ipfs"}, // either ipfs.io or ipfs.bitmark
				},
				// For opensea tokens, we can only check the mime-type of an asset by its file extension.
				bson.M{
					"source": "opensea",
					"projectMetadata.latest.thumbnailURL": bson.M{
						"$regex": ".svg$",
					},
				},
			},
		},
		bson.M{"$set": bson.M{"thumbnailLastCheck": time.Now()}},
		options.FindOneAndUpdate().
			SetSort(bson.D{{Key: "projectMetadata.latest.lastUpdatedAt", Value: 1}}).
			SetProjection(bson.M{"indexID": 1, "projectMetadata.latest.thumbnailURL": 1}),
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
		bson.D{{Key: "$set", Value: bson.D{{Key: "thumbnailID", Value: thumbnailID}}}},
	)

	return err
}

func (s *NFTContentIndexer) checkThumbnail(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		assets := make(chan NFTAsset, 200)
		defer close(assets)

		s.spawnThumbnailWorker(ctx, assets, 100)

		log.Logger.Info("start the loop the get assets without thumbnail cached",
			zap.Duration("thumbnailCachePeriod", s.thumbnailCachePeriod),
			zap.Duration("thumbnailCacheRetryInterval", s.thumbnailCacheRetryInterval))

	WATCH_ASSETS:
		for {
			asset, err := s.getAssetWithoutThumbnailCached(ctx)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					log.Logger.Info("No token need to generate cache a thumbnail")
				} else {
					log.Logger.Error("fail to get asset", zap.Error(err))
				}

				if done := indexer.SleepWithContext(ctx, 15*time.Second); done {
					break WATCH_ASSETS
				}
				continue
			}
			log.Logger.Debug("send asset to process", zap.String("indexID", asset.IndexID))
			assets <- asset
		}
		log.Logger.Debug("Thumbnail checker closed")
	}()

}

func (s *NFTContentIndexer) Start(ctx context.Context) {
	s.checkThumbnail(ctx)
	s.wg.Wait()
}
