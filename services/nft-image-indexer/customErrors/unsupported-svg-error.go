package customErrors

import (
	"errors"
	"fmt"
)

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
