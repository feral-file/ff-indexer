package indexer

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/managedblockchainquery"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/tzkt-go"
)

func TestIndexETHToken(t *testing.T) {
	if err := log.Initialize("", false); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	ethClient, err := ethclient.Dial(viper.GetString("ethereum.rpc_url"))
	if err != nil {
		t.Fatalf("fail to initiate eth client: %s", err.Error())
	}

	engine := New(
		"",
		[]string{},
		map[string]string{},
		opensea.New("livenet", "", 1),
		tzkt.New(""),
		fxhash.New("https://api.fxhash.xyz/graphql"),
		objkt.New(""),
		ethClient,
		nil,
		nil,
	)

	assetUpdates, err := engine.IndexETHToken(context.Background(), "0xbc4ca0eda7647a8ab7c2061c2e118a18a936f13d", "9616")

	assert.NoError(t, err)
	assert.Equal(t, assetUpdates.Tokens[0].Balance, int64(0))
	assert.Equal(t, assetUpdates.Tokens[0].Owner, "")
}

func TestIndexTokenOwner(t *testing.T) {
	if err := log.Initialize("", false); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	awsSession, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: aws.String("us-east-1"),
		},
	})
	if err != nil {
		log.Panic("fail to set up aws session", zap.Error(err))
	}

	blockchainQueryClient := managedblockchainquery.New(awsSession)

	engine := New(
		"",
		[]string{},
		map[string]string{},
		opensea.New("livenet", "", 1),
		tzkt.New(""),
		fxhash.New("https://api.fxhash.xyz/graphql"),
		objkt.New(""),
		nil,
		nil,
		blockchainQueryClient,
	)

	owners1, err := engine.IndexETHTokenOwners("0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D", "9616")
	assert.NoError(t, err)
	assert.Equal(t, len(owners1), 1)
	assert.Equal(t, owners1[0].Address, "0x29469395eAf6f95920E59F858042f0e28D98a20B")
	assert.Equal(t, owners1[0].Balance, int64(1))

	owners2, err := engine.IndexETHTokenOwners("0x28472a58A490c5e09A238847F66A68a47cC76f0f", "1")
	assert.NoError(t, err)
	assert.Equal(t, len(owners2), 4706)
	assert.Equal(t, owners2[0].LastTime.String(), "2022-04-29 03:43:37 +0000 UTC")
}
