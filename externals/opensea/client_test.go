package opensea

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	log "github.com/bitmark-inc/autonomy-logger"
)

func TestMain(m *testing.M) {
	if err := log.Initialize("", false, nil); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}
	os.Exit(m.Run())
}

func TestGetTokensForOwner(t *testing.T) {
	openseaKey := os.Getenv("OPENSEA_KEY")
	client := New("livenet", openseaKey, 1)

	tokens, err := client.RetrieveAssets("0xb858A3F45840E76076c6c4DBa9f0f8958F11C1E8", "")
	assert.NoError(t, err)
	assert.Len(t, tokens, 50)
}
