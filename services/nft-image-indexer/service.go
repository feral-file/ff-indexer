package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// spawnIPFSPinWorker spawn worker to pin preview file of assets to IPFS
func (s *NFTContentIndexer) spawnIPFSPinWorker(ctx context.Context, assets <-chan NFTAsset, count int) {
	for i := 0; i < count; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()

			for asset := range assets {
				previewURL := asset.ProjectMetadata.Latest.PreviewURL
				if strings.HasPrefix(previewURL, "https://ipfs.io/ipfs/") {
					ipfsCIDPath := strings.ReplaceAll(previewURL, "https://ipfs.io/ipfs/", "")

					if ipfsCIDPath == "" {
						logrus.WithField("previewURL", previewURL).Error("incorrect file path")
						continue
					}

					cid := strings.Split(ipfsCIDPath, "/")[0]

					if _, err := s.ipfs.Pin(cid); err != nil {
						logrus.WithField("previewURL", previewURL).WithError(err).Error("fail to pin a file into IPFS")
						fmt.Println("www", indexer.SleepWithContext(ctx, 30*time.Second))
						continue
					}

					if err := s.updateTokenPinnedStatus(ctx, asset.IndexID); err != nil {
						logrus.WithError(err).Error("fail to update pin status for a token")
					}

					logrus.WithField("indexID", asset.IndexID).Info("preview url has pinned")
				} else {
					logrus.WithField("previewURL", previewURL).Warn("unsupported preview url")
				}
			}
			logrus.Debug("IPFSPinWorker stopped")
		}()
	}
}

// getAssetWithoutThumbnailCached looks up assets without thumbnail cached
func (s *NFTContentIndexer) getAssetWithoutThumbnailCached(ctx context.Context) (NFTAsset, error) {
	var asset NFTAsset
	r := s.nftAssets.FindOneAndUpdate(ctx,
		bson.M{
			// check only for items has been viewed in the past 7 days
			"projectMetadata.latest.lastUpdatedAt": bson.M{"$gt": time.Now().Add(-168 * time.Hour)},
			"$and": bson.A{
				// thumbnailLastCheck helps filter out assets that have already processed recently.
				bson.M{
					"$or": bson.A{
						bson.M{
							"thumbnailLastCheck": bson.M{"$exists": false},
						},
						bson.M{
							"thumbnailLastCheck": bson.M{"$lt": time.Now().Add(-time.Hour)},
						},
					},
				},
				bson.M{
					"$or": bson.A{
						// For tezos tokens, it parses tokens that starts with `https://ipfs.` which means
						// all token that is uploaded to IPFS but is not cached by objkt.
						bson.M{
							"source":                              "tzkt",
							"thumbnailID":                         bson.M{"$exists": false},
							"projectMetadata.latest.source":       bson.M{"$nin": []string{"fxhash"}},
							"projectMetadata.latest.thumbnailURL": bson.M{"$regex": "^https://ipfs"}, // either ipfs.io or ipfs.bitmark
						},
						// For get all tokens that with the mime-type SVG and the URL starts with https
						bson.M{
							"projectMetadata.latest.mimeType":     "image/svg+xml",
							"projectMetadata.latest.thumbnailURL": bson.M{"$regex": "^https://"},
						},
						// For opensea tokens, it only parses SVG
						bson.M{
							"source": "opensea",
							"projectMetadata.latest.thumbnailURL": bson.M{
								"$regex": ".svg$",
							},
						},
					},
				},
			},
		},
		bson.M{"$set": bson.M{"thumbnailLastCheck": time.Now()}},
		options.FindOneAndUpdate().
			SetSort(bson.D{{Key: "thumbnailLastCheck", Value: 1}}).
			SetProjection(bson.M{"indexID": 1, "projectMetadata.latest.thumbnailURL": 1}),
	)

	if err := r.Err(); err != nil {
		return asset, err
	}

	err := r.Decode(&asset)
	return asset, err
}

// getAssetWithoutIPFSPinned looks up assets without ipfs pinned
func (s *NFTContentIndexer) getAssetWithoutIPFSPinned(ctx context.Context) (NFTAsset, error) {
	var asset NFTAsset
	r := s.nftAssets.FindOneAndUpdate(ctx,
		bson.M{
			"ipfsPinned":                        bson.M{"$ne": true},
			"source":                            indexer.SourceTZKT,
			"projectMetadata.latest.previewURL": bson.M{"$ne": ""},
			"projectMetadata.latest.medium":     indexer.MediumVideo,
			"projectMetadata.latest.mimeType":   bson.M{"$in": bson.A{"video/webm", "video/quicktime", "video/ogg", "video/mp4"}},
			"$or": bson.A{
				bson.M{
					"ipfsPinnedLastCheck": bson.M{"$exists": false},
				},
				bson.M{
					"ipfsPinnedLastCheck": bson.M{"$lt": time.Now().Add(-1 * time.Minute)},
				},
			},
		},
		bson.M{"$set": bson.M{"ipfsPinnedLastCheck": time.Now()}},
		options.FindOneAndUpdate().SetProjection(
			bson.M{"indexID": 1, "projectMetadata.latest.previewURL": 1},
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
		bson.D{{"$set", bson.D{{"thumbnailID", thumbnailID}}}},
	)

	return err
}

// updateTokenPinnedStatus sets a specific token to be pinned
func (s *NFTContentIndexer) updateTokenPinnedStatus(ctx context.Context, indexID string) error {
	_, err := s.nftAssets.UpdateOne(
		ctx,
		bson.M{"indexID": indexID},
		bson.D{{"$set", bson.D{{"ipfsPinned", true}}}},
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

// checkIPFSPinned starts workers to check and pin files to IPFS
func (s *NFTContentIndexer) checkIPFSPinned(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		assets := make(chan NFTAsset, 50)
		defer close(assets)

		s.spawnIPFSPinWorker(ctx, assets, 1)

	WATCH_ASSETS:
		for {
			asset, err := s.getAssetWithoutIPFSPinned(ctx)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					logrus.Info("No token need to pin IPFS")
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
		logrus.Debug("IPFS-pinned checker closed")
	}()
}

func (s *NFTContentIndexer) Start(ctx context.Context) {
	s.checkThumbnail(ctx)
	// s.checkIPFSPinned(ctx)
	s.wg.Wait()
}
