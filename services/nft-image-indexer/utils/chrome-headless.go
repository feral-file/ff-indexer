package utils

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/customErrors"
)

const CropImageTimeout = 5 * time.Second

var SVGSupportTags = []string{
	"rect",
	"svg",
}

func ConvertSVGToPNG(url string) (*bytes.Buffer, error) {
	for _, tag := range SVGSupportTags {
		buf, err := ScreenShoot(url, tag)

		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		} else if buf != nil {
			return bytes.NewBuffer(buf), nil
		}
	}
	
	return nil, customErrors.NewUnsupportedSVG(url)
}

func ScreenShoot(url string, selector string) ([]byte, error) {
	var buf []byte

	ctx, cancel := chromedp.NewContext(
		context.Background(),
	)
	defer cancel()

	ctx2, cancel := context.WithTimeout(ctx, CropImageTimeout)
	defer cancel()

	var screenshotTask = chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.Screenshot(selector, &buf, chromedp.NodeVisible),
	}

	if err := chromedp.Run(ctx2, screenshotTask); err != nil {
		return nil, err
	}

	return buf, nil
}
