package sdk

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	indexer "github.com/bitmark-inc/nft-indexer"
)

func TestGetTokenByIndexID(t *testing.T) {
	indexID := "tez-KT1VZ6Zkoae9DtXkbuw4wtFCg9WH8eywcvEX-23798030035473632618901897089878275372960165372586891230635421889000008911882"

	client, err := NewIndexerClient("localhost:8889")
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = client.GetTokenByIndexID(
		context.Background(),
		indexID,
	)

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

func TestGetTotalBalanceOfOwnerAccounts(t *testing.T) {
	client, err := NewIndexerClient("localhost:8889")
	if err != nil {
		fmt.Println(err)
		return
	}
	addresses := []string{
		"0x1a02c339196597a9AE4515D9C9D49B2195F1C12A",
		"0x1a02c339196597a9AE4515D9C9D49B2195F1C12A",
	}

	balances, err := client.GetTotalBalanceOfOwnerAccounts(context.Background(), addresses)

	fmt.Println("balances: ", balances)

	assert.NoError(t, err)
}

// TestGetDetailedToken is a test for GetDetailedToken
func TestGetDetailedToken(t *testing.T) {
	client, err := NewIndexerClient("localhost:8889")
	if err != nil {
		fmt.Println(err)
		return
	}

	indexID := "tez-KT1VZ6Zkoae9DtXkbuw4wtFCg9WH8eywcvEX-23798030035473632618901897089878275372960165372586891230635421889000008911882"

	_, err = client.GetDetailedToken(context.Background(), indexID)

	assert.NoError(t, err)
}

// TODO: implement tests: TestIndexAccountTokens, TestUpdateOwnerForFungibleToken, TestUpdateOwner

// TestGetOwnerAccountsByIndexIDs a test for GetOwnerAccountsByIndexIDs
func TestGetOwnerAccountsByIndexIDs(t *testing.T) {
	client, err := NewIndexerClient("localhost:8889")
	if err != nil {
		fmt.Println(err)
		return
	}

	indexIDs := []string{"tez-KT1ESGez4dEuDjjNt4k2HPAK5Nzh7e8X8jyX-1683031758835", "tez-KT1ESGez4dEuDjjNt4k2HPAK5Nzh7e8X8jyX-1683863093457"}

	owners, err := client.GetOwnerAccountsByIndexIDs(context.Background(), indexIDs)

	fmt.Println(owners)
	assert.NoError(t, err)
}

// GetRandomIndexIDByContract a test for GetRandomIndexIDByContract
func TestGetRandomIndexIDByContract(t *testing.T) {
	client, err := NewIndexerClient("localhost:8889")
	if err != nil {
		fmt.Println(err)
		return
	}

	contract := "KT1ESGez4dEuDjjNt4k2HPAK5Nzh7e8X8jyX"

	indexID, err := client.GetRandomIndexIDByContract(context.Background(), contract)

	fmt.Println(indexID)
	assert.NoError(t, err)
}
