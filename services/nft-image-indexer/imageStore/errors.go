package imageStore

import (
	"errors"
	"fmt"
)

const (
	ErrUnsupportImageType = "ErrUnsupportImageType"
	ErrUnsupportedSVGURL  = "ErrUnsupportedSVGURL"
	ErrDownloadFileError  = "ErrDownloadFileEror"
	ErrSizeTooLarge       = "ErrSizeTooLarge"
)

var ImageCachingErrorTypes = map[string]error{
	ErrUnsupportImageType: errors.New("unsupported image type"),
	ErrUnsupportedSVGURL:  errors.New("unsupported SVG URL"),
	ErrDownloadFileError:  errors.New("download file error"),
	ErrSizeTooLarge:       errors.New("size too large"),
}

type ImageCachingError struct {
	Name string
	Err  error
}

func (i *ImageCachingError) Error() string {
	return ImageCachingErrorTypes[i.Name].Error()
}

func NewImageCachingError(name string) error {
	return &ImageCachingError{
		Name: name,
		Err:  ImageCachingErrorTypes[name],
	}
}

type UnsupportedSVG struct {
	Url string

	Err error
}

func (r *UnsupportedSVG) Error() string {
	return fmt.Sprintf("unsupported SVG Url: %v", r.Url)
}

func NewUnsupportedSVG(url string) error {
	return &UnsupportedSVG{
		Url: url,
		Err: errors.New("unsupported SVG URL"),
	}
}
