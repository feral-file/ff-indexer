package indexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBlockchainByAddress(t *testing.T) {
	assert.Equal(t, GetBlockchainByAddress("0x8F6ccB4cF3C3bed6830CB6E2824C18AdCFA8eBBd"), EthereumBlockchain)
	assert.Equal(t, GetBlockchainByAddress("tz1MTXXDg7uudxmEieyf2rmZyLBST7ykndWw"), TezosBlockchain)
	assert.Equal(t, GetBlockchainByAddress("aWDT2s4Lba3rrBtqLghY61PLr2gLZuvSy9uvXRmwLmhAixXuNa"), BitmarkBlockchain)
}

func TestParseTokenIndexID(t *testing.T) {
	blockchainAlias, contract, tokenID, err := ParseTokenIndexID("eth-0x82e0b8cdd80af5930c4452c684e71c861148ec8a-20382901")
	assert.NoError(t, err)
	assert.Equal(t, blockchainAlias, "eth")
	assert.Equal(t, contract, "0x82E0b8cDD80Af5930c4452c684E71c861148Ec8A")
	assert.Equal(t, tokenID, "20382901")

	blockchainAlias, contract, tokenID, err = ParseTokenIndexID("tez-tz1MTXXDg7uudxmEieyf2rmZyLBST7ykndWw-1231231212")
	assert.NoError(t, err)
	assert.Equal(t, blockchainAlias, "tez")
	assert.Equal(t, contract, "tz1MTXXDg7uudxmEieyf2rmZyLBST7ykndWw")
	assert.Equal(t, tokenID, "1231231212")
}
