package imageStore

import (
	"errors"
	"fmt"
)

var ErrUnsupportImageType = fmt.Errorf("unsupported image type")

const (
	ErrUnsupportedSVGURL = "ErrUnsupportedSVGURL"
	ErrDownloadFileError = "ErrDownloadFileEror"
	ErrSizeTooLarge      = "ErrSizeTooLarge"
)

var UploadErrorTypes = map[string]error{
	ErrUnsupportedSVGURL: errors.New("unsupported SVG URL"),
	ErrDownloadFileError: errors.New("download file error"),
	ErrSizeTooLarge:      errors.New("size too large"),
}
