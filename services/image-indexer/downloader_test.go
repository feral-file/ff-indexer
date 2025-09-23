package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	log "github.com/bitmark-inc/autonomy-logger"
)

func TestMain(m *testing.M) {
	if err := log.Initialize(false, nil); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}
	os.Exit(m.Run())
}

func TestDownloadFile(t *testing.T) {
	url := "https://i.seadn.io/gcs/files/84f6bc32f323099f62a8ef81d6b46b91.png?w=500&auto=format"
	_, mimeType, _, err := DownloadFile(url)

	assert.NoError(t, err)
	assert.Equal(t, "image/png", mimeType)
}

func TestDownloadSmallSizeFile(t *testing.T) {
	url := "https://openseauserdata.com/files/224b81f977f415bdd7b5ffccbe28e459.svg"
	_, mimeType, _, err := DownloadFile(url)

	assert.NoError(t, err)
	assert.Equal(t, "image/svg+xml", mimeType)
}
