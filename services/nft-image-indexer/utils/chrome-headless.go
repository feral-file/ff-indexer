package utils

import (
	"bytes"
	"context"
	"errors"
	"github.com/chromedp/chromedp"
	"time"
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

	return nil, errors.New("can not convert SVG to PNG")
}

func ScreenShoot(url string, selector string) []byte {
	var buf []byte

	ctx, cancel := chromedp.NewContext(
		context.Background(),
	)
	defer cancel()

	ctx2, _ := context.WithTimeout(ctx, CropImageTimeout)

	var screenshotTask = chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.Screenshot(selector, &buf, chromedp.NodeVisible),
	}

	if err := chromedp.Run(ctx2, screenshotTask); err != nil {
		return nil
	}

	return buf
}
