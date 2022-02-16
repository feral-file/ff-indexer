package indexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAccountBlockchain(t *testing.T) {
	assert.Equal(t, DetectAccountBlockchain("0x8F6ccB4cF3C3bed6830CB6E2824C18AdCFA8eBBd"), EthereumBlockchain)
	assert.Equal(t, DetectAccountBlockchain("tz1MTXXDg7uudxmEieyf2rmZyLBST7ykndWw"), TezosBlockchain)
	assert.Equal(t, DetectAccountBlockchain("aWDT2s4Lba3rrBtqLghY61PLr2gLZuvSy9uvXRmwLmhAixXuNa"), BitmarkBlockchain)
}
