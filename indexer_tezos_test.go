package indexer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/nft-indexer/externals/bettercall"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
)

func TestIndexTezosTokenProvenance(t *testing.T) {
	engine := New(nil, tzkt.New("api.mainnet.tzkt.io"), nil, nil)
	provenances, err := engine.IndexTezosTokenProvenance(context.Background(), "KT1KEa8z6vWXDJrVqtMrAeDVzsvxat3kHaCE", "178227")
	assert.NoError(t, err)

	b, _ := json.MarshalIndent(provenances, "", "  ")
	t.Log(string(b))
}

func TestIndexRetriveTokens(t *testing.T) {
	bcd := bettercall.New()
	tokens1, _ := bcd.RetrieveTokens("tz1bpvbjRGW1XHkALp4hFee6PKbnZCcoN9hE", 0)

	tzktClient := tzkt.New("api.mainnet.tzkt.io")
	tokens2, _ := tzktClient.RetrieveTokens("tz1bpvbjRGW1XHkALp4hFee6PKbnZCcoN9hE", 0)

	tJson1, _ := json.MarshalIndent(tokens1, "", "  ")
	tJson2, _ := json.MarshalIndent(tokens2, "", "  ")
	t.Log(string(tJson1))
	t.Log(string(tJson2))

	metadata1, _ := bcd.GetTokenMetadata("KT1RJ6PbjHpwc3M5rw5s2Nbmefwbuwbdxton", "538543")
	mJson1, _ := json.MarshalIndent(metadata1, "", "  ")
	metadata2, _ := tzktClient.GetContractToken("KT1RJ6PbjHpwc3M5rw5s2Nbmefwbuwbdxton", "538543")
	mJson2, _ := json.MarshalIndent(metadata2, "", "  ")
	t.Log(string(mJson1))
	t.Log(string(mJson2))

}
