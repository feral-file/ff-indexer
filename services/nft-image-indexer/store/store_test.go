package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSupportedImageType(t *testing.T) {
	mimeType := "image/svg+xml"
	result := IsSupportedImageType(mimeType)

	assert.Equal(t, result, true)
}
