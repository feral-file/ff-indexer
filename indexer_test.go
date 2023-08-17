package indexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
