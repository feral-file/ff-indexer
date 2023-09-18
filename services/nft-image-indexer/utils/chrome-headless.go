package utils

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/chromedp/chromedp"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	imageStore "github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/store"
)

const CropImageTimeout = 5 * time.Second

var SVGSupportTags = []string{
	"rect",
	"svg",
}

func ConvertSVGToPNG(url string) (*bytes.Buffer, error) {
	for _, tag := range SVGSupportTags {
		buf, err := ScreenShoot(url, tag)

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			log.Warn("fail to take screenshot", zap.Error(err))
			return nil, imageStore.NewUnsupportedSVG(url)
		}

		if buf != nil {
			return bytes.NewBuffer(buf), nil
		}
	}

	// fail back to fullscreen screenshot
	buf, err := ScreenShoot(url, "")
	if err != nil {
		log.Warn("fail to take screenshot", zap.Error(err))
		return nil, imageStore.NewUnsupportedSVG(url)
	}

	return bytes.NewBuffer(buf), nil
}

func ScreenShoot(url string, selector string) ([]byte, error) {
	var buf []byte

	ctx := context.Background()
	ctx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	ctx2, cancel := context.WithTimeout(ctx, CropImageTimeout)
	defer cancel()

	var screenshotTask = chromedp.Tasks{
		chromedp.EmulateViewport(2048, 2048),
		chromedp.Navigate(url),
	}

	if selector != "" {
		screenshotTask = append(screenshotTask,
			chromedp.Screenshot(selector, &buf, chromedp.ByQuery))
	} else {
		screenshotTask = append(screenshotTask,
			chromedp.CaptureScreenshot(&buf))
	}

	if err := chromedp.Run(ctx2, screenshotTask); err != nil {
		return nil, err
	}

	return buf, nil
}
