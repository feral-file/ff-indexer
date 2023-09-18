package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	log "github.com/bitmark-inc/autonomy-logger"
	imageStore "github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/store"
	"github.com/bitmark-inc/nft-indexer/services/nft-image-indexer/utils"
)

func TestMain(m *testing.M) {
	if err := log.Initialize("", false); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}
	os.Exit(m.Run())
}

func TestDownloadFile(t *testing.T) {
	url := "https://openseauserdata.com/files/dccefc262d24e3c43c1421efbc0e56f1.svg"
	_, _, _, err := DownloadFile(url)

	assert.NoError(t, err)
}

func TestIsSupportedImageType(t *testing.T) {
	mimeType := "image/svg+xml"
	result := imageStore.IsSupportedImageType(mimeType)

	assert.Equal(t, result, true)
}

func TestConvertSVGToPNG(t *testing.T) {
	url1 := "https://openseauserdata.com/files/dccefc262d24e3c43c1421efbc0e56f1.svg"
	buf1, err := utils.ConvertSVGToPNG(url1)
	assert.NotNil(t, buf1)
	assert.NoError(t, err)

	url2 := "https://openseauserdata.com/files/6e3209d74c6471690348418330f76545.svg"
	buf2, err := utils.ConvertSVGToPNG(url2)
	assert.NotNil(t, buf2)
	assert.NoError(t, err)

	url3 := "https://assets.objkt.media/file/assets-003/QmfYFFvQ5cY7Y7H4mxmeZWmHxum8EhibDfea2ggi2PuEY8/artifact?objkt=1243&creator=tz1exNcVPJdNSqiKDxYkiZRAaxB74jq1m4CQ&viewer=tz1SidNQb9XcwP7L3MzCZD9JHmWw2ebDzgyX&danger=ignored"
	buf3, err := utils.ConvertSVGToPNG(url3)
	assert.NotNil(t, buf3)
	assert.NoError(t, err)
}
