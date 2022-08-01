package indexer

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gabriel-vasile/mimetype"
)

type StreamDownloader struct {
	io.Reader
	closeFunc func() error
}

func (d StreamDownloader) Close() error {
	return d.closeFunc()
}

func NewStreamDownloader(reader io.Reader, closeFunc func() error) io.ReadCloser {
	return StreamDownloader{reader, closeFunc}
}

// DownloadFile downloads a file from a given url and returns a file reader and its mime type
func DownloadFile(c *http.Client, url string) (io.ReadCloser, string, error) {
	resp, err := c.Get(url)
	if err != nil {
		return nil, "", err
	}

	fileHeader := make([]byte, 512)
	if _, err = resp.Body.Read(fileHeader); err != nil {
		return nil, "", err
	}

	mimeType := mimetype.Detect(fileHeader).String()

	fileHeaderReader := bytes.NewBuffer(fileHeader)

	file := io.MultiReader(fileHeaderReader, resp.Body)

	return NewStreamDownloader(file, resp.Body.Close), mimeType, err
}

type URLDownloader struct {
	url     string
	timeout time.Duration
}

func NewURLDownloader(url string, timeout time.Duration) *URLDownloader {
	return &URLDownloader{
		url:     url,
		timeout: timeout,
	}
}

func (d *URLDownloader) Download() (io.ReadCloser, string, error) {
	c := &http.Client{
		Timeout: d.timeout,
	}

	return DownloadFile(c, d.url)
}
