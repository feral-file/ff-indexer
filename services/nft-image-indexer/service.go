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

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/customErrors"
	"github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/imageStore"
)

type NFTAsset struct {
	IndexID         string                           `bson:"indexID"`
	ProjectMetadata indexer.VersionedProjectMetadata `bson:"projectMetadata"`
}

type NFTContentIndexer struct {
	wg sync.WaitGroup

	db        *imageStore.ImageStore
	ipfs      IPFSPinService
	nftAssets *mongo.Collection
}

func NewNFTContentIndexer(db *imageStore.ImageStore, nftAssets *mongo.Collection, ipfs IPFSPinService) *NFTContentIndexer {
	return &NFTContentIndexer{
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
				logrus.WithField("indexID", asset.IndexID).Debug("start generating thumbnail cache for an asset")

				if _, err := s.db.CreateOrGetImage(ctx, asset.IndexID); err != nil {
					logrus.WithError(err).Error("fail to get or create image record")
					continue
				}

				img, err := s.db.UploadImage(ctx, asset.IndexID, NewURLImageDownloader(asset.ProjectMetadata.Latest.ThumbnailURL),
					map[string]interface{}{
						"source":   asset.ProjectMetadata.Latest.Source,
						"file_url": asset.ProjectMetadata.Latest.ThumbnailURL,
					},
				)
				if err != nil {
					if errors.Is(err, imageStore.ErrUnsupportImageType) {
						logrus.WithField("indexID", asset.IndexID).Warn("unsupported image type")
						// let the image id remain empty string
					} else if _, ok := err.(*customErrors.UnsupportedSVG); ok {
						logrus.WithError(err).WithField("indexID", asset.IndexID).Error("fail to upload image")
						sentry.CaptureMessage("assetId: " + asset.IndexID + " - " + err.Error())
					} else {
						logrus.WithError(err).WithField("indexID", asset.IndexID).Error("fail to upload image")
					}
				}

				if err := s.updateTokenThumbnail(ctx, img.AssetID, img.ImageID); err != nil {
					logrus.WithError(err).Error("update token thumbnail to indexer")
				}

				logrus.WithField("indexID", asset.IndexID).Info("thumbnail generating process finished")
			}
			logrus.Debug("ThumbnailWorker stopped")
		}()
	}
}

// getAssetWithoutThumbnailCached looks up assets without thumbnail cached
func (s *NFTContentIndexer) getAssetWithoutThumbnailCached(ctx context.Context) (NFTAsset, error) {
	var asset NFTAsset
	r := s.nftAssets.FindOneAndUpdate(ctx,
		bson.M{
			"source":                              "tzkt",
			"thumbnailID":                         bson.M{"$exists": false},
			"projectMetadata.latest.source":       bson.M{"$nin": []string{"fxhash"}},
			"projectMetadata.latest.thumbnailURL": bson.M{"$ne": ""},
			"$or": bson.A{
				bson.M{
					"thumbnailLastCheck": bson.M{"$exists": false},
				},
				bson.M{
					"thumbnailLastCheck": bson.M{"$lt": time.Now().Add(-10 * time.Minute)},
				},
			},
		},
		bson.M{"$set": bson.M{"thumbnailLastCheck": time.Now()}},
		options.FindOneAndUpdate().SetProjection(
			bson.M{"indexID": 1, "projectMetadata.latest.thumbnailURL": 1},
		),
	)

	if err := r.Err(); err != nil {
		return asset, err
	}

	err := r.Decode(&asset)
	return asset, err
}

// updateTokenThumbnail sets the thumbnail id for a specific token
func (s *NFTContentIndexer) updateTokenThumbnail(ctx context.Context, indexID, thumbnailID string) error {
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

	WATCH_ASSETS:
		for {
			asset, err := s.getAssetWithoutThumbnailCached(ctx)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					logrus.Info("No token need to generate cache a thumbnail")
				} else {
					logrus.WithError(err).Error("fail to get asset")
				}

				if done := indexer.SleepWithContext(ctx, 15*time.Second); done {
					break WATCH_ASSETS
				}
				continue
			}
			logrus.WithField("indexID", asset.IndexID).Debug("send asset to process")
			assets <- asset
		}
		logrus.Debug("Thumbnail checker closed")
	}()

}

func (s *NFTContentIndexer) Start(ctx context.Context) {
	s.checkThumbnail(ctx)
	s.wg.Wait()
}
