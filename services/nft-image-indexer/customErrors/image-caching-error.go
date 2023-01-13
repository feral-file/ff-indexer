package errors

import (
	"errors"
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
