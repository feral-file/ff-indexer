package main

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/utils"
	"github.com/gabriel-vasile/mimetype"
	"github.com/sirupsen/logrus"
)

// DownloadFile downloads a file from a given url and returns a file reader and its mime type
func DownloadFile(url string) (io.Reader, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	fileHeader := make([]byte, 512)
	_, err = resp.Body.Read(fileHeader)
	if err != nil {
		return nil, "", err
	}

	mimeType := mimetype.Detect(fileHeader).String()

	var file *bytes.Buffer
	if strings.HasPrefix(mimeType, "image/svg") {
		if file, err = utils.ConvertSVGToPNG(url); err != nil {
			return nil, "", err
		}
	} else {
		file = bytes.NewBuffer(fileHeader)
		if _, err := io.Copy(file, resp.Body); err != nil {
			return nil, "", err
		}
	}

	logrus.WithField("file_size", file.Len()).Debug("file downloaded")

	return file, mimeType, err
}

type URLImageDownloader struct {
	url string
}

func NewURLImageDownloader(url string) *URLImageDownloader {
	return &URLImageDownloader{
		url: url,
	}
}

func (d *URLImageDownloader) Download() (io.Reader, string, error) {
	logrus.WithField("sourceURL", d.url).Debug("download image from source")
	return DownloadFile(d.url)
}
