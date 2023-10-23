package indexer

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/managedblockchainquery"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/tzkt-go"
)

func TestIPFSURLToGatewayURL(t *testing.T) {
	link := ipfsURLToGatewayURL(FxhashGateway, "ipfs://QmWKZY8Qp6U5WrC5Nxzf1xVaZoQgLzuDwbvDrnDdfjkBTV?fxhash=opFymKKHJMKEGuYk5eafpiUS6PpYDKqVRqTUJqztG1F2N6nLVY")
	assert.Equal(t, "https://gateway.fxhash.xyz/ipfs/QmWKZY8Qp6U5WrC5Nxzf1xVaZoQgLzuDwbvDrnDdfjkBTV/?fxhash=opFymKKHJMKEGuYk5eafpiUS6PpYDKqVRqTUJqztG1F2N6nLVY", link)
	link = ipfsURLToGatewayURL(FxhashGateway, "ipfs://QmWKZY8Qp6U5WrC5Nxzf1xVaZoQgLzuDwbvDrnDdfjkBTV/?fxhash=opFymKKHJMKEGuYk5eafpiUS6PpYDKqVRqTUJqztG1F2N6nLVY")
	assert.Equal(t, "https://gateway.fxhash.xyz/ipfs/QmWKZY8Qp6U5WrC5Nxzf1xVaZoQgLzuDwbvDrnDdfjkBTV/?fxhash=opFymKKHJMKEGuYk5eafpiUS6PpYDKqVRqTUJqztG1F2N6nLVY", link)
	link = ipfsURLToGatewayURL(DefaultIPFSGateway, "ipfs://QmWKZY8Qp6U5WrC5Nxzf1xVaZoQgLzuDwbvDrnDdfjkBTV/01/index.html?a=1")
	assert.Equal(t, "https://ipfs.nftstorage.link/ipfs/QmWKZY8Qp6U5WrC5Nxzf1xVaZoQgLzuDwbvDrnDdfjkBTV/01/index.html?a=1", link)
	link = ipfsURLToGatewayURL(DefaultIPFSGateway, "ipfs://QmWKZY8Qp6U5WrC5Nxzf1xVaZoQgLzuDwbvDrnDdfjkBTV/01/?a=1")
	assert.Equal(t, "https://ipfs.nftstorage.link/ipfs/QmWKZY8Qp6U5WrC5Nxzf1xVaZoQgLzuDwbvDrnDdfjkBTV/01/?a=1", link)
}

func TestGetTokenBalanceOfOwner(t *testing.T) {
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

	balance1, err := engine.GetTokenBalanceOfOwner(context.Background(), "0xF903164aa2E070991467F7f9f0464d34B272013F", "22", "0x0F0eAE91990140C560D4156DB4f00c854Dc8F09E")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), balance1)

	balance2, err := engine.GetTokenBalanceOfOwner(context.Background(), "0x33FD426905F149f8376e227d0C9D3340AaD17aF1", "89", "0x0F0eAE91990140C560D4156DB4f00c854Dc8F09E")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), balance2)

	balance3, err := engine.GetTokenBalanceOfOwner(context.Background(), "0xe70659b717112ac4e14284d0db2f5d5703df8e43", "125", "0x0F0eAE91990140C560D4156DB4f00c854Dc8F09E")
	assert.NoError(t, err)
	assert.Equal(t, int64(4), balance3)
}

func TestOptimizedOpenseaImageURL(t *testing.T) {
	newURL1, err := OptimizedOpenseaImageURL("https://i.seadn.io/s/raw/files/aed53e1bcd90ca93b6fd3b0e012fadc5.jpg?w=500&auto=format")
	assert.NoError(t, err)
	assert.Equal(t, "https://i.seadn.io/s/raw/files/aed53e1bcd90ca93b6fd3b0e012fadc5.jpg?auto=format&dpr=1&w=3840", newURL1)

	newURL2, err := OptimizedOpenseaImageURL("https://i.seadn.io/gcs/files/0d06a393468f2e227ed14a6a88f951bc.jpg?w=500&auto=format")
	assert.NoError(t, err)
	assert.Equal(t, "https://i.seadn.io/gcs/files/0d06a393468f2e227ed14a6a88f951bc.jpg?auto=format&dpr=1&w=3840", newURL2)

	newURL3, err := OptimizedOpenseaImageURL("https://i.seadn.io/gae/zxrPTPWKa-uc-oLImHUN_bst5e6v7zeL5AIDXn1LWTpVe_43oCG2i-sZ5IsFHxHt4pkIuoDeaZF1HnApLqdy9wrzSeKSepRyYOr_9Q?w=500&auto=format")
	assert.NoError(t, err)
	assert.Equal(t, "https://i.seadn.io/gae/zxrPTPWKa-uc-oLImHUN_bst5e6v7zeL5AIDXn1LWTpVe_43oCG2i-sZ5IsFHxHt4pkIuoDeaZF1HnApLqdy9wrzSeKSepRyYOr_9Q?auto=format&dpr=1&w=3840", newURL3)

}
