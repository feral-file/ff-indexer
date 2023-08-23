package indexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestGetMIMEType(t *testing.T) {
	url := "https://i.seadn.io/gcs/files/bbc0a44987656c60494fd646e0f670d0.gif?w=500&auto=format"
	mimeType := GetMIMETypeByURL(url)
	assert.Equal(t, mimeType, "image/gif")

	url = "https://i.seadn.io/gcs/files/d2fe56690de325daa49cad0600304345.png"
	mimeType = GetMIMETypeByURL(url)
	assert.Equal(t, mimeType, "image/png")

	url = "https://i.seadn.io/gcs/files/669bd5d03542a1a8fb9b6587e2103ae7.png?w=500&auto=format"
	mimeType = GetMIMETypeByURL(url)
	assert.Equal(t, mimeType, "image/png")

	url = "https://ipfs.io/ipfs/QmbNtTu7k2E2UDYDQTyiVzV8rVbCU44hc9js1erKzeSogY"
	mimeType = GetMIMETypeByURL(url)
	assert.Equal(t, mimeType, "")
}

func TestGetCIDFromIPFSLink(t *testing.T) {
	url := "https://ipfs.io/ipfs/QmbNtTu7k2E2UDYDQTyiVzV8rVbCU44hc9js1erKzeSogY"
	cid, err := GetCIDFromIPFSLink(url)
	assert.NoError(t, err)
	assert.Equal(t, cid, "QmbNtTu7k2E2UDYDQTyiVzV8rVbCU44hc9js1erKzeSogY")

	url = "ipfs://QmbNtTu7k2E2UDYDQTyiVzV8rVbCU44hc9js1erKzeSogY"
	cid, err = GetCIDFromIPFSLink(url)
	assert.NoError(t, err)
	assert.Equal(t, cid, "QmbNtTu7k2E2UDYDQTyiVzV8rVbCU44hc9js1erKzeSogY")

	url = "ipfs://QmbNtTu7k2E2UDYDQTyiVzV8rVbCU44hc9js1erKzeSogY?test=true"
	cid, err = GetCIDFromIPFSLink(url)
	assert.NoError(t, err)
	assert.Equal(t, cid, "QmbNtTu7k2E2UDYDQTyiVzV8rVbCU44hc9js1erKzeSogY")

	url = "https://myipfs.test.com/prefix/ipfs/QmbNtTu7k2E2UDYDQTyiVzV8rVbCU44hc9js1erKzeSogY?test=true"
	cid, err = GetCIDFromIPFSLink(url)
	assert.NoError(t, err)
	assert.Equal(t, cid, "QmbNtTu7k2E2UDYDQTyiVzV8rVbCU44hc9js1erKzeSogY")

	url = "https://myipfs.test.com/prefix/ipfs/QmbNtTu7k2E2UDYDQTyiVzV8rVbCU44hc9js1erKzeSogY/index.html?test=true"
	cid, err = GetCIDFromIPFSLink(url)
	assert.NoError(t, err)
	assert.Equal(t, cid, "QmbNtTu7k2E2UDYDQTyiVzV8rVbCU44hc9js1erKzeSogY")
}

func TestGetCIDFromIPFSLinkFailed(t *testing.T) {
	url := "https://ipfs.io/ipfffs/QmbNtTu7k2E2UDYDQTyiVzV8rVbCU44hc9js1erKzeSogY"
	cid, err := GetCIDFromIPFSLink(url)
	assert.Error(t, err, "cid not found")
	assert.Equal(t, cid, "")

	url = "wss://myipfs.test.com/prefix/ipfs/connect"
	cid, err = GetCIDFromIPFSLink(url)
	assert.Error(t, err, "unsupported ipfs link")
	assert.Equal(t, cid, "")
}
