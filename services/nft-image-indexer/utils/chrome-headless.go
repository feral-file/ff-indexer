package utils

import (
	"bytes"
	"context"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/customErrors"
)

const CropImageTimeout = 5 * time.Second

func ConvertSVGToPNG(url string) (*bytes.Buffer, error) {
	bufRect := ScreenShoot(url, "rect")
	if bufRect != nil {
		return bytes.NewBuffer(bufRect), nil
	}

	bufSVG := ScreenShoot(url, "svg")
	if bufSVG != nil {
		return bytes.NewBuffer(bufSVG), nil
	}

	return nil, customErrors.NewUnsupportedSVG(url)
}

func ScreenShoot(url string, selector string) []byte {
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
		return nil
	}

	return buf
}
