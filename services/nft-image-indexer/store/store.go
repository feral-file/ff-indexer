package store

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	log "github.com/bitmark-inc/autonomy-logger"
	"github.com/cloudflare/cloudflare-go"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

const CloudflareImageDeilverURL = "https://imagedelivery.net/%s/%s/public"

type Metadata map[string]interface{}

type ImageReader interface {
	Read() (io.Reader, string, int, error)
}

const ImageSizeThreshold = 10 * 1024 * 1024 // 10MB

// IsSupportedImageType validates if an image is supported
func IsSupportedImageType(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

type ImageStore struct {
	db *gorm.DB

	// cloudflare
	cloudflareAccountHash string
	cloudflareAccountID   *cloudflare.ResourceContainer
	cloudflareAPI         *cloudflare.API
}

// imageIDExisted checks if a given image id is presence in cloudflare
func (s *ImageStore) imageIDExisted(imageID string) (bool, error) {
	// FIXME: not use default client
	resp, err := http.Head(fmt.Sprintf(CloudflareImageDeilverURL, s.cloudflareAccountHash, imageID))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, fmt.Errorf("incorrect http status(%d) on checking image existence", resp.StatusCode)
}

func New(dsn string, cloudflareAccountHash, cloudflareAccountID, cloudflareAPIToken string) *ImageStore {
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

	cloudflareAPI, err := cloudflare.NewWithAPIToken(cloudflareAPIToken,
		cloudflare.Debug(viper.GetBool("debug")), cloudflare.UsingLogger(log.CloudflareLogger()))
	if err != nil {
		panic(err)
	}

	accRes := &cloudflare.ResourceContainer{
		Level:      cloudflare.AccountRouteLevel,
		Identifier: cloudflareAccountID,
		Type:       cloudflare.AccountType,
	}

	return &ImageStore{
		db:                    db,
		cloudflareAccountHash: cloudflareAccountHash,
		cloudflareAccountID:   accRes,
		cloudflareAPI:         cloudflareAPI,
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
func (s *ImageStore) UploadImage(ctx context.Context, assetID string, imageReader ImageReader, metadata map[string]interface{}) (ImageMetadata, error) {
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
			imageExisted, err := s.imageIDExisted(image.ImageID)
			if err != nil {
				return err
			}
			log.Debug("check thumbnail cache existent",
				zap.Bool("imageExisted", imageExisted),
				zap.String("assetID", assetID))
			if imageExisted {
				// remove the existent image before create a new one
				if err := s.cloudflareAPI.DeleteImage(ctx, s.cloudflareAccountID, image.ImageID); err != nil {
					log.Warn("fail to delete image cache",
						zap.String("assetID", assetID), zap.Error(err))
					return err
				}
			}
		}

		downloadStartTime := time.Now()
		file, mimeType, imageSize, err := imageReader.Read()
		if err != nil {
			return NewImageCachingError(ReasonDownloadFileFailed)
		}
		log.Debug("download thumbnail finished",
			zap.String("mimeType", mimeType),
			zap.Int("imageSize", imageSize),
			zap.Duration("duration", time.Since(downloadStartTime)),
			zap.String("assetID", assetID))

		if !IsSupportedImageType(mimeType) {
			return NewImageCachingError(ReasonUnsupportedImageType)
		}

		if metadata == nil {
			metadata = Metadata{}
		}

		if strings.HasPrefix(mimeType, "image/svg") {
			metadata["mime_type"] = "image/png"
		} else {
			metadata["mime_type"] = mimeType
		}

		if imageSize > ImageSizeThreshold {
			file, err = compressImage(file)
			if err != nil {
				log.Error("cannot compress the thumbnail with ffmpeg", zap.Error(err))
			}
		}

		uploadRequest := cloudflare.UploadImageParams{
			File:     io.NopCloser(file),
			Name:     assetID,
			Metadata: metadata,
		}

		log.Debug("upload image to cloudflare", zap.String("assetID", assetID))

		i, err := s.cloudflareAPI.UploadImage(ctx, s.cloudflareAccountID, uploadRequest)
		if err != nil {
			switch cerr := err.(type) {
			case *cloudflare.RatelimitError:
				log.Debug("caught cloudflare ratelimit error", zap.String("type", string(cerr.Type())))
				return err
			case *cloudflare.RequestError:
				log.Debug("caught cloudflare request error", zap.String("type", string(cerr.Type())),
					zap.Any("codes", cerr.ErrorCodes()), zap.Any("msg", cerr.ErrorMessages()))
				for _, code := range cerr.ErrorCodes() {
					switch code {
					case 5455: // Unsupported content type
						return NewImageCachingError(ReasonUnsupportedImageType)
					case 9422:
						return NewImageCachingError(ReasonBrokenImage)
					case 5443: // The animation is too large
						return NewImageCachingError(ReasonFileSizeTooLarge)
					default:
						return err
					}
				}
				return err
			case *cloudflare.ServiceError:
				log.Debug("caught cloudflare serivce error", zap.Any("codes", cerr.ErrorCodes()), zap.Any("msg", cerr.ErrorMessages()))
				return err
			}

			isErrSizeTooLarge, _ := regexp.MatchString("entity.*too large", err.Error())
			if isErrSizeTooLarge {
				return NewImageCachingError(ReasonFileSizeTooLarge)
			}

			if strings.Contains(err.Error(), "error unmarshalling the JSON response error body") {
				return NewImageCachingError(ReasonUnknownCloudflareAPIFailure)
			}

			return err
		}

		cloudflareImageID = i.ID
		image.ImageID = i.ID

		return tx.Where("asset_id = ?", assetID).Save(&image).Error
	})

	// Clean up uploaded files when a transaction is failed.
	// It can not 100% ensure the file is cleaned up due to service broken
	if err != nil && cloudflareImageID != "" {
		log.Warn("clean uploaded file due to rollback", zap.String("assetID", assetID))
		err = s.cloudflareAPI.DeleteImage(ctx, s.cloudflareAccountID, cloudflareImageID)
		if err != nil {
			log.Warn("fail to clean uploaded file", zap.String("cloudflareImageID", cloudflareImageID))
		}
	}

	return image, err
}

// compressImage compresses an image by reducing width (iw), height (ih) and quality (compression_level)
func compressImage(file io.Reader) (io.Reader, error) {
	resultBuffer := bytes.NewBuffer(make([]byte, 0))
	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, file)
	if err != nil {
		return file, err
	}

	// Compress the image
	// If it is a gif, we select the first frame
	// To make sure the size is less than a threshold, the new image has 90% of the width and height
	// of the input image
	// FIXME: adapt the command with various image size to make sure we get the best quality with output size < 10MB
	cmd := exec.Command("ffmpeg", "-y",
		"-f", "image2pipe",
		"-i", "pipe:0",
		"-vsync", "0",
		"-vf", "select=eq(n\\,0)",
		"-vf", "scale=-1:min'(10000,ih*0.9)':force_original_aspect_ratio=decrease",
		"-q:v", "5",
		"-update", "1",
		"-f", "image2", "pipe:1")

	var execErr bytes.Buffer
	cmd.Stderr = &execErr
	cmd.Stdout = resultBuffer

	stdin, _ := cmd.StdinPipe()
	if err = cmd.Start(); err != nil {
		return file, err
	}

	if _, err = stdin.Write(buf.Bytes()); err != nil {
		return file, err
	}

	if err = stdin.Close(); err != nil {
		return file, err
	}

	if err = cmd.Wait(); err != nil {
		return file, err
	}

	if resultBuffer.Len() > ImageSizeThreshold {
		return compressImage(resultBuffer)
	}

	return resultBuffer, nil
}
