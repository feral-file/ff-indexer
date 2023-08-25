package worker

import (
	"context"
	"path"

	"go.uber.org/cadence/activity"
	"go.uber.org/zap"

	indexer "github.com/bitmark-inc/nft-indexer"
)

const DefaultDownloadRetry = 3

// CacheArtifact is an activity to cache a given file into S3
func (w *NFTIndexerWorker) CacheArtifact(ctx context.Context, fileURI string) error {
	log := activity.GetLogger(ctx)
	log.Debug("find CID from link", zap.String("fileURI", fileURI))
	cid, err := indexer.GetCIDFromIPFSLink(fileURI)
	if err != nil {
		log.Warn("fail to get ipfs cid", zap.Error(err), zap.String("fileURI", fileURI))
	}

	filename := cid
	if filename == "" {
		filename = path.Base(fileURI)
	}

	log.Debug("start caching IPFS file", zap.String("filename", filename), zap.String("fileURI", fileURI))

	_, err = w.assetClient.Pin(fileURI, filename, "30d")

	return err
}
