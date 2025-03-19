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

var screenshotSupportedSVGTags = []string{
	"rect",
	"svg",
}

var ErrTakeScreenshot = fmt.Errorf("fail to take screenshot")

// screenshotSVGTags takes screenshots for a url using supported SVG tags and falls back to
// take a full screenshot if nothing is targeted.
func screenshotSVGTags(url string) (*bytes.Buffer, error) {
	for _, tag := range screenshotSupportedSVGTags {
		buf, err := ScreenShoot(url, tag)

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			return nil, ErrTakeScreenshot
		}

		if buf != nil {
			return bytes.NewBuffer(buf), nil
		}
	}

	// fail back to fullscreen screenshot
	buf, err := ScreenShoot(url, "")
	if err != nil {
		return nil, ErrTakeScreenshot
	}

	return bytes.NewBuffer(buf), nil
}

func ScreenshotLink(url string) (io.Reader, string, int, error) {
	f, err := screenshotSVGTags(url)
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
