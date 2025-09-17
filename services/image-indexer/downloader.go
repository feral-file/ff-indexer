package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gabriel-vasile/mimetype"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
)

// DownloadFile downloads a file from a given url and returns a file reader and its mime type
func DownloadFile(url string) (io.Reader, string, int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", 0, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	fileHeader := make([]byte, 512)
	if n, err := resp.Body.Read(fileHeader); err != nil {
		if errors.Is(err, io.EOF) {
			// the file size is smaller than the sample bytes (512)
			fileHeader = fileHeader[:n]
		} else {
			return nil, "", 0, fmt.Errorf("%d bytes read. error: %s", n, err.Error())
		}
	}
	mimeType := mimetype.Detect(fileHeader).String()

	file := bytes.NewBuffer(fileHeader)
	if _, err := io.Copy(file, resp.Body); err != nil {
		return nil, "", 0, err
	}

	log.Debug("file downloaded",
		zap.String("download_url", url),
		zap.String("mimeType", mimeType),
		zap.Int("file_size", file.Len()))

	return file, mimeType, file.Len(), nil
}
