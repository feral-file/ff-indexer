package main

import (
	"context"
	"time"

	"github.com/chromedp/chromedp"
)

const CropImageTimeout = 5 * time.Second

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
