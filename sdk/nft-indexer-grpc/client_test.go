package sdk

import (
	"context"
	"fmt"
	indexer "github.com/bitmark-inc/nft-indexer"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetTokensByIndexID(t *testing.T) {
	indexID := "tez-KT1VZ6Zkoae9DtXkbuw4wtFCg9WH8eywcvEX-23798030035473632618901897089878275372960165372586891230635421889000008911882"

	client, err := NewIndexerClient("localhost:8889")
	if err != nil {
		fmt.Println(err)
		return
	}

	token, err := client.GetTokensByIndexID(
		context.Background(),
		indexID,
	)

	fmt.Println("token: ", token)
	assert.NoError(t, err)
}

func TestPushProvenance(t *testing.T) {
	indexID := "tez-KT1VZ6Zkoae9DtXkbuw4wtFCg9WH8eywcvEX-23798030035473632618901897089878275372960165372586891230635421889000008911882"

	client, err := NewIndexerClient("localhost:8889")
	if err != nil {
		fmt.Println(err)
		return
	}

	FormerOwner := "tz1dBwDL1Ze9zKtfBdiS1WcLZrqDjfgqBUuR"
	var blockNumber uint64 = 123

	provenance := indexer.Provenance{
		FormerOwner: &FormerOwner,
		Type:        "transfer",
		Owner:       "tz1TogGp2Z27pZDGtpNAwUdM9cj9NusLPHUC",
		Blockchain:  "tezos",
		BlockNumber: &blockNumber,
		Timestamp:   time.Now(),
		TxID:        "onxZkY7F6BXnKXdRuWG7fxh2WHdAFR1ZXxXR78FDMiinpuwPHUC",
		TxURL:       "https://tzkt.io/onxZkY7F6BXnKXdRuWG7fxh2WHdAFR1ZXxXR78FDMiinpuwgBYP",
	}

	err = client.PushProvenance(context.Background(), indexID, time.Now(), provenance)

	assert.NoError(t, err)
}
