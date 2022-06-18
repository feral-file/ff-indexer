package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/imageStore"
)

type NFTAsset struct {
	IndexID         string                           `bson:"indexID"`
	ProjectMetadata indexer.VersionedProjectMetadata `bson:"projectMetadata"`
}

type NFTImageIndexer struct {
	wg sync.WaitGroup

	db        *imageStore.ImageStore
	nftAssets *mongo.Collection
}

func NewNFTImageIndexer(db *imageStore.ImageStore, nftAssets *mongo.Collection) *NFTImageIndexer {
	return &NFTImageIndexer{
		db:        db,
		nftAssets: nftAssets,
	}
}

// spawnWorker spawn worker for generate thumbnails from source images
func (s *NFTImageIndexer) spawnWorker(ctx context.Context, assets <-chan NFTAsset, count int) {
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
					} else {
						logrus.WithError(err).WithField("indexID", asset.IndexID).Error("fail to upload image")
					}
				}

				if err := s.updateTokenThumbnail(ctx, img.AssetID, img.ImageID); err != nil {
					logrus.WithError(err).Error("update token thumbnail to indexer")
				}

				logrus.WithField("indexID", asset.IndexID).Info("thumbnail generating process finished")
			}
		}()
	}
}

// getAssetWithoutThumbnailCached looks up assets without thumbnail cached
func (s *NFTImageIndexer) getAssetWithoutThumbnailCached(ctx context.Context) (NFTAsset, error) {
	var asset NFTAsset
	r := s.nftAssets.FindOneAndUpdate(ctx,
		bson.M{
			"indexID":                             bson.M{"$exists": true},
			"thumbnailID":                         bson.M{"$exists": false},
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
func (s *NFTImageIndexer) updateTokenThumbnail(ctx context.Context, indexID, thumbnailID string) error {
	_, err := s.nftAssets.UpdateOne(
		ctx,
		bson.M{"indexID": indexID},
		bson.D{{"$set", bson.D{{"thumbnailID", thumbnailID}}}},
	)

	return err
}

func (s *NFTImageIndexer) Start(ctx context.Context) {
	for {
		assets := make(chan NFTAsset, 200)
		s.spawnWorker(ctx, assets, 100)

		for {
			asset, err := s.getAssetWithoutThumbnailCached(ctx)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					logrus.Info("No token need to generate cache a thumbnail")
				} else {
					logrus.WithError(err).Error("fail to get asset")
				}

				// FIXME: add config for thumbnail checking interval
				time.Sleep(time.Minute)
				continue
			}
			logrus.WithField("indexID", asset.IndexID).Debug("send asset to process")
			assets <- asset
		}
	}
}
