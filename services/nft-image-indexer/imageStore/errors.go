package imageStore

import (
	"errors"
)

const (
	ErrUnsupportImageType = "ErrUnsupportImageType"
	ErrUnsupportedSVGURL  = "ErrUnsupportedSVGURL"
	ErrDownloadFileError  = "ErrDownloadFileEror"
	ErrSizeTooLarge       = "ErrSizeTooLarge"
)

var UploadErrorTypes = map[string]error{
	ErrUnsupportImageType: errors.New("unsupported image type"),
	ErrUnsupportedSVGURL:  errors.New("unsupported SVG URL"),
	ErrDownloadFileError:  errors.New("download file error"),
	ErrSizeTooLarge:       errors.New("size too large"),
}
