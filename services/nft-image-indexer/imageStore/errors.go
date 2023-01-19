package imageStore

import (
	"errors"
	"fmt"
)

// Reason keys for unsupported errors
const (
	ReasonUnsupportedImageType = "ErrUnsupportedImageType"
	ReasonUnsupportedSVGFile   = "ErrUnsupportedSVGFile"
	ReasonDownloadFileFailed   = "ErrDownloadFileFailed"
	ReasonFileSizeTooLarge     = "ErrSizeTooLarge"
)

var ImageCachingErrorReasons = map[string]string{ // string string
	ReasonUnsupportedImageType: "unsupported image type",
	ReasonUnsupportedSVGFile:   "unsupported SVG File",
	ReasonDownloadFileFailed:   "download file error",
	ReasonFileSizeTooLarge:     "size too large",
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
	return ImageCachingErrorReasons[e.Reason()]
}

// NewImageCachingError retuns ImageCachingError if a reson is given.
// Otherwise, it returns a regular error.
func NewImageCachingError(reason string) error {
	if _, ok := ImageCachingErrorReasons[reason]; !ok {
		return errors.New(reason)
	}

	return &ImageCachingError{
		reason: reason,
	}
}

type UnsupportedSVG struct {
	Url string
}

func (e *UnsupportedSVG) Reason() string {
	return ReasonUnsupportedSVGFile
}

func (e *UnsupportedSVG) Error() string {
	return fmt.Sprintf("unsupported SVG Url: %v", e.Url)
}

func NewUnsupportedSVG(url string) error {
	return &UnsupportedSVG{
		Url: url,
	}
}
