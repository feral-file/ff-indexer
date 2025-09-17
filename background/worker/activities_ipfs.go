package worker

import (
	"context"
	"path"

	indexer "github.com/feral-file/ff-indexer"
)

const DefaultDownloadRetry = 3

// CacheArtifact is an activity to cache a given file into S3
func (w *Worker) CacheArtifact(ctx context.Context, fileURI string) error {
	cid, err := indexer.GetCIDFromIPFSLink(fileURI)
	if err != nil {
		return err
	}

	filename := cid
	if filename == "" {
		filename = path.Base(fileURI)
	}

	_, err = w.assetClient.Pin(fileURI, filename, "30d")

	return err
}
