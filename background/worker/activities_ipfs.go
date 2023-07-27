package worker

import (
	"context"
	"strings"
	"time"

	"go.uber.org/cadence/activity"
	"go.uber.org/zap"

	indexer "github.com/bitmark-inc/nft-indexer"
)

const DefaultDownloadRetry = 3

// CacheIPFSArtifactInS3 is an activity to cache a given file into S3
func (w *NFTIndexerWorker) CacheIPFSArtifactInS3(ctx context.Context, fileURI string) error {
	log := activity.GetLogger(ctx)

	log.Debug("find CID from IPFS link", zap.String("fileURI", fileURI))

	cid, err := indexer.GetCIDFromIPFSLink(fileURI)
	if err != nil {
		log.Warn("fail to get ipfs cid", zap.Error(err), zap.String("fileURI", fileURI))
		return nil
	}

	log.Debug("start caching IPFS file", zap.String("cid", cid), zap.String("fileURI", fileURI))
	// TODO: check if the file in asset server

	d := indexer.NewURLDownloader(fileURI, 5*time.Minute)

	counter := 0
	f, mime, err := d.Download()
	for err != nil {
		counter++
		if counter < DefaultDownloadRetry {
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

	_, err = w.assetClient.Upload(f, cid, "30d")

	return err
}
