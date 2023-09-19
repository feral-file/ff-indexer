package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
)

var screenshotSupportedTags = []string{
	"rect",
	"svg",
}

var ErrTakeScreenshot = fmt.Errorf("fail to take screenshot")

// ScreenshotTags takes screenshots by supported tags and fall back to
// take a full screenshot if nothing is targeted.
func ScreenshotTags(url string) (*bytes.Buffer, error) {
	for _, tag := range screenshotSupportedTags {
		buf, err := ScreenShoot(url, tag)

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			log.Warn("fail to take screenshot", zap.Error(err))
			return nil, ErrTakeScreenshot
		}

		if buf != nil {
			return bytes.NewBuffer(buf), nil
		}
	}

	// fail back to fullscreen screenshot
	buf, err := ScreenShoot(url, "")
	if err != nil {
		log.Warn("fail to take screenshot", zap.Error(err))
		return nil, ErrTakeScreenshot
	}

	return bytes.NewBuffer(buf), nil
}

func ScreenshotLink(url string) (io.Reader, string, int, error) {
	f, err := ScreenshotTags(url)
	if err != nil {
		return nil, "", 0, err
	}

	return f, "image/png", f.Len(), nil
}

type URLImageReader struct {
	url string
}

func NewURLImageReader(url string) *URLImageReader {
	return &URLImageReader{
		url: url,
	}
}

func (d *URLImageReader) Read() (file io.Reader, mimeType string, fileSize int, err error) {
	log.Debug("download image from source", zap.String("sourceURL", d.url))

	if strings.HasSuffix(d.url, ".svg") {
		return ScreenshotLink(d.url)
	}

	file, mimeType, fileSize, err = DownloadFile(d.url)

	if strings.HasPrefix(mimeType, "image/svg") ||
		strings.HasPrefix(mimeType, "application/octet-stream") {
		return ScreenshotLink(d.url)
	}

	return
}
