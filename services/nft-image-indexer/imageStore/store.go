package imageStore

import (
	"context"
	"io"
	"strings"
	"time"

	log "github.com/bitmark-inc/nft-indexer/zapLog"
	"github.com/cloudflare/cloudflare-go"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type Metadata map[string]interface{}

type ImageDownloader interface {
	Download() (io.Reader, string, error)
}

// IsSupportedImageType validates if an image is supported
func IsSupportedImageType(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

type ImageStore struct {
	db *gorm.DB

	// cloudflare
	cloudflareAccountID string
	cloudflareAPI       *cloudflare.API
}

func New(dsn string, cloudflareAccountID, cloudflareAPIToken string) *ImageStore {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.LogLevel(viper.GetInt("image_db.log_level"))),
	})
	if err != nil {
		panic(err)
	}

	sqldb, err := db.DB()
	if err != nil {
		panic(err)
	}
	sqldb.SetMaxOpenConns(50)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqldb.SetConnMaxLifetime(time.Hour)

	cloudflareAPI, err := cloudflare.NewWithAPIToken(cloudflareAPIToken)
	if err != nil {
		panic(err)
	}

	return &ImageStore{
		db:                  db,
		cloudflareAccountID: cloudflareAccountID,
		cloudflareAPI:       cloudflareAPI,
	}
}

func (s *ImageStore) AutoMigrate() error {
	return s.db.AutoMigrate(&ImageMetadata{})
}

// GetImage returns an image metadata object
func (s *ImageStore) GetImage(ctx context.Context, assetID string) (ImageMetadata, error) {
	var image ImageMetadata

	err := s.db.WithContext(ctx).First(&image, ImageMetadata{
		AssetID: assetID,
	}).Error

	return image, err
}

// CreateOrGetImage creates an image record in db if it is not existed. Otherwise, returns the existed one
func (s *ImageStore) CreateOrGetImage(ctx context.Context, assetID string) (ImageMetadata, error) {
	var image ImageMetadata

	tx := s.db.WithContext(ctx).FirstOrCreate(&image, ImageMetadata{
		AssetID: assetID,
	})

	return image, tx.Error
}

// UploadImage creates a db transaction to download and upload an image to cloudflare.
// After an image is successfully uploaded to cloudflare, it updates the returned image id into image store.
// It locks an image record for updating which prevents from duplicated download precess
// The additional metadata will be attached to the image file when we upload it to cloudflare.
func (s *ImageStore) UploadImage(ctx context.Context, assetID string, imageDownloader ImageDownloader, metadata map[string]interface{}) (ImageMetadata, error) {
	var image ImageMetadata
	var cloudflareImageID string

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{
			Strength: "UPDATE",
			Options:  "NOWAIT",
		}).Where("asset_id = ?", assetID).First(&image).Error; err != nil {
			// not found, locked or any other errors
			return err
		}

		if image.ImageID != "" {
			// remove the existent image before create a new one
			if err := s.cloudflareAPI.DeleteImage(ctx, s.cloudflareAccountID, image.ImageID); err != nil {
				return err
			}
		}

		downloadStartTime := time.Now()
		file, mimeType, err := imageDownloader.Download()
		if err != nil {
			return err
		}
		log.Logger.Debug("download thumbnail finished",
			zap.Duration("duration", time.Since(downloadStartTime)),
			zap.String("assetID", assetID))

		if !IsSupportedImageType(mimeType) {
			return ErrUnsupportImageType
		}

		if metadata == nil {
			metadata = Metadata{}
		}

		if strings.HasPrefix(mimeType, "image/svg") {
			metadata["mime_type"] = "image/png"
		} else {
			metadata["mime_type"] = mimeType
		}

		uploadRequest := cloudflare.ImageUploadRequest{
			File:     io.NopCloser(file),
			Name:     assetID,
			Metadata: metadata,
		}

		log.Logger.Debug("upload image to cloudflare", zap.String("assetID", assetID))

		i, err := s.cloudflareAPI.UploadImage(ctx, s.cloudflareAccountID, uploadRequest)
		if err != nil {
			return err
		}

		cloudflareImageID = i.ID
		image.ImageID = i.ID

		return tx.Where("asset_id = ?", assetID).Save(&image).Error
	})

	// Clean up uploaded files when a transaction is failed.
	// It can not 100% ensure the file is cleaned up due to service broken
	if err != nil && cloudflareImageID != "" {
		log.Logger.Warn("clean uploaded file due to rollback", zap.String("assetID", assetID))
		err = s.cloudflareAPI.DeleteImage(ctx, s.cloudflareAccountID, cloudflareImageID)
		if err != nil {
			log.Logger.Warn("fail to clean uploaded file", zap.String("cloudflareImageID", cloudflareImageID))
		}
	}

	return image, err
}
