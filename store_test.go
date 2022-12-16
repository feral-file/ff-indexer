package indexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPageCount(t *testing.T) {
	assert.Equal(t, 1, getPageCounts(25, 25))
	assert.Equal(t, 2, getPageCounts(26, 25))
	assert.Equal(t, 5, getPageCounts(101, 25))
	assert.Equal(t, 1, getPageCounts(88, 100))
	assert.Equal(t, 2, getPageCounts(101, 100))
}
