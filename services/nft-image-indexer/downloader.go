package main

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/utils"
	"github.com/gabriel-vasile/mimetype"
	"go.uber.org/zap"
)

// DownloadFile downloads a file from a given url and returns a file reader and its mime type
func DownloadFile(url string) (io.Reader, string, int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", 0, err
	}
	defer resp.Body.Close()

	fileHeader := make([]byte, 512)
	_, err = resp.Body.Read(fileHeader)
	if err != nil {
		return nil, "", 0, err
	}

	mimeType := mimetype.Detect(fileHeader).String()
	var action string
	var file *bytes.Buffer

	if strings.HasPrefix(mimeType, "image/svg") {
		action = "chrome_screenshot"
		if file, err = utils.ConvertSVGToPNG(url); err != nil {
			return nil, "", 0, err
		}
	} else {
		action = "url_download"
		file = bytes.NewBuffer(fileHeader)
		if _, err := io.Copy(file, resp.Body); err != nil {
			return nil, "", 0, err
		}
	}
	log.Debug("file downloaded",
		zap.String("action", action),
		zap.String("download_url", url),
		zap.Int("file_size", file.Len()))

	return file, mimeType, file.Len(), err
}

type URLImageDownloader struct {
	url string
}

func NewURLImageDownloader(url string) *URLImageDownloader {
	return &URLImageDownloader{
		url: url,
	}
}

func (d *URLImageDownloader) Download() (io.Reader, string, int, error) {
	log.Debug("download image from source", zap.String("sourceURL", d.url))
	return DownloadFile(d.url)
}
