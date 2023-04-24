package infura

import (
	"fmt"
	"testing"

	"github.com/bitmark-inc/config-loader"
	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestGetOwnersAndBalancesByToken(t *testing.T) {
	if err := log.Initialize("", false); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}
	config.LoadConfig("")

	apiKey := viper.GetString("ethereum.infura_api_key")
	apiKeySecret := viper.GetString("ethereum.infura_api_key_secret")
	client := New("testnet", apiKey, apiKeySecret)

	ownerBalances, err := client.GetOwnersAndBalancesByToken("0x8502Aef50609c6b87f12E12a8C04f2650fA86906", "509")
	assert.NoError(t, err)
	assert.NotEqual(t, ownerBalances, []OwnerBalance{})
}
