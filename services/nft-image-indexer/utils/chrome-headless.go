package utils

import (
	"bytes"
	"context"
	"errors"
	"github.com/chromedp/chromedp"
	"time"
)

const timeout = 5 * time.Second

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

	ctx2, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := chromedp.Run(ctx2, elementScreenshot(url, selector, &buf)); err != nil {
		return nil
	}

	return buf
}

// elementScreenshot takes a screenshot of a specific element.
func elementScreenshot(urlstr, sel string, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate(urlstr),
		chromedp.Screenshot(sel, res, chromedp.NodeVisible),
	}
}
