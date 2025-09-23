package store

import (
	"errors"
	"fmt"
)

// Reason keys for unsupported errors
const (
	ReasonBrokenImage                 = "ErrBrokenImage"
	ReasonUnsupportedImageType        = "ErrUnsupportedImageType"
	ReasonUnsupportedSVGFile          = "ErrUnsupportedSVGFile"
	ReasonDownloadFileFailed          = "ErrDownloadFileFailed"
	ReasonFileSizeTooLarge            = "ErrSizeTooLarge"
	ReasonUnknownCloudflareAPIFailure = "ErrUnknownCloudflareAPIFailure"
)

var ImageCachingErrorReasons = map[string]string{ // string string
	ReasonBrokenImage:                 "broken image",
	ReasonUnsupportedImageType:        "unsupported image type",
	ReasonUnsupportedSVGFile:          "unsupported SVG File",
	ReasonDownloadFileFailed:          "download file error",
	ReasonFileSizeTooLarge:            "size too large",
	ReasonUnknownCloudflareAPIFailure: "unknown cloudflare api error",
}

type UnsupportedImageCachingError interface {
	Error() string
	Reason() string
}

type ImageCachingError struct {
	reason string
}

func (e *ImageCachingError) Reason() string {
	return e.reason
}

func (e *ImageCachingError) Error() string {
	return fmt.Sprintf("known image caching error: %s", ImageCachingErrorReasons[e.Reason()])
}

// NewImageCachingError returns ImageCachingError if a reson is given.
// Otherwise, it returns a regular error.
func NewImageCachingError(reason string) error {
	if _, ok := ImageCachingErrorReasons[reason]; !ok {
		return errors.New(reason)
	}

	return &ImageCachingError{
		reason: reason,
	}
}

type UnsupportedSVGError struct {
	URL string
}

func (e *UnsupportedSVGError) Reason() string {
	return ReasonUnsupportedSVGFile
}

func (e *UnsupportedSVGError) Error() string {
	return fmt.Sprintf("unsupported SVG Url: %v", e.URL)
}

func NewUnsupportedSVG(url string) error {
	return &UnsupportedSVGError{
		URL: url,
	}
}
