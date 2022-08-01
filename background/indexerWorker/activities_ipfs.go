package indexerWorker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"

	indexer "github.com/bitmark-inc/nft-indexer"
)

const DEFAULT_DOWNLOAD_RETRY = 3

// CacheIPFSArtifactInS3 is an activity to cache a given file into S3
func (w *NFTIndexerWorker) CacheIPFSArtifactInS3(ctx context.Context, fileURI string) error {
	log := activity.GetLogger(ctx)

	log.Debug("start caching IPFS file", zap.String("fileURI", fileURI))

	var cid string
	if strings.HasPrefix(fileURI, indexer.DEFAULT_IPFS_GATEWAY) {
		cid = strings.Replace(fileURI, indexer.DEFAULT_IPFS_GATEWAY, "", -1)
	} else {
		log.Debug("ignore non IPFS file", zap.String("fileURI", fileURI))
		return nil
	}

	if _, err := s3.New(w.awsSession).HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(w.ipfsCacheBucketName),
		Key:    aws.String(fmt.Sprintf("ipfs/%s", cid)),
	}); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != "NotFound" {
				return err
			}
		}
	} else {
		log.Debug("IPFS data has already cached", zap.String("cid", cid))
		return nil
	}

	d := indexer.NewURLDownloader(fileURI, 5*time.Minute)

	count := 0
	f, mime, err := d.Download()
	for err != nil {
		if count < DEFAULT_DOWNLOAD_RETRY {
			log.Warn("fail to download artwork", zap.Error(err))
			f, mime, err = d.Download()
		} else {
			return err
		}
	}
	defer f.Close()

	if !strings.HasPrefix(mime, "video") && !strings.HasPrefix(mime, "image") {
		log.Debug("ignore non-video and non-image data", zap.String("fileURI", fileURI), zap.String("mime", mime))
		return nil
	}

	uploader := s3manager.NewUploader(w.awsSession)
	if _, err := uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Body:        f,
		Bucket:      aws.String(w.ipfsCacheBucketName),
		Key:         aws.String(fmt.Sprintf("ipfs/%s", cid)),
		ContentType: &mime,
		Metadata:    map[string]*string{},
	}); err != nil {
		return err
	}
	return nil
}
