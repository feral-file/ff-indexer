package sdk

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	indexer "github.com/feral-file/ff-indexer"
)

const serverURL = "localhost:8888"

func TestGetTokenByIndexID(t *testing.T) {
	indexID := "tez-KT1VZ6Zkoae9DtXkbuw4wtFCg9WH8eywcvEX-23798030035473632618901897089878275372960165372586891230635421889000008911882"

	client, err := NewGRPCClient(serverURL)
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

	client, err := NewGRPCClient(serverURL)
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
	client, err := NewGRPCClient(serverURL)
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
	client, err := NewGRPCClient(serverURL)
	if err != nil {
		fmt.Println(err)
		return
	}

	indexID := "tez-KT1VZ6Zkoae9DtXkbuw4wtFCg9WH8eywcvEX-23798030035473632618901897089878275372960165372586891230635421889000008911882"

	_, err = client.GetDetailedToken(context.Background(), indexID, false)

	assert.NoError(t, err)
}

// TODO: implement tests: TestIndexAccountTokens, TestUpdateOwnerForFungibleToken, TestUpdateOwner

// TestGetOwnerAccountsByIndexIDs a test for GetOwnerAccountsByIndexIDs
func TestGetOwnerAccountsByIndexIDs(t *testing.T) {
	client, err := NewGRPCClient(serverURL)
	if err != nil {
		fmt.Println(err)
		return
	}

	indexIDs := []string{"tez-KT1ESGez4dEuDjjNt4k2HPAK5Nzh7e8X8jyX-1683031758835", "tez-KT1ESGez4dEuDjjNt4k2HPAK5Nzh7e8X8jyX-1683863093457"}

	owners, err := client.GetOwnerAccountsByIndexIDs(context.Background(), indexIDs)

	fmt.Println(owners)
	assert.NoError(t, err)
}

// TestCheckAddressOwnTokenByCriteria a test for CheckAddressOwnTokenByCriteria
func TestCheckAddressOwnTokenByCriteria(t *testing.T) {
	client, err := NewGRPCClient(serverURL)
	if err != nil {
		fmt.Println(err)
		return
	}

	check, err := client.CheckAddressOwnTokenByCriteria(
		context.Background(),
		"0x51E92B35a5a182B2d62b2E22f431D8e0797aB60e",
		indexer.Criteria{
			IndexID: "eth-0xb43c51447405008AEBf7a35B4D15e1f29b7Ce823-84379833228553110502734947101839209675161105358737778734002435191848727499610",
		},
	)

	assert.NoError(t, err)
	assert.Equal(t, true, check)

	check, err = client.CheckAddressOwnTokenByCriteria(
		context.Background(),
		"tz1ZRtM64raLrUBFPFfxAWHXpiGrB2KmW4kL",
		indexer.Criteria{
			Source: "feralfile",
		},
	)

	assert.NoError(t, err)
	assert.Equal(t, true, check)
}

// TestGetOwnersByBlockchainAndContracts a test for GetOwnersByBlockchainContracts
func TestGetOwnersByBlockchainAndContracts(t *testing.T) {
	client, err := NewGRPCClient("localhost:8889")
	if err != nil {
		fmt.Println(err)
		return
	}

	owners, err := client.GetOwnersByBlockchainContracts(context.Background(), map[string][]string{
		"tezos":    {"KT1ESGez4dEuDjjNt4k2HPAK5Nzh7e8X8jyX"},
		"ethereum": {"0xb43c51447405008AEBf7a35B4D15e1f29b7Ce823"},
	})

	fmt.Println(owners)
	assert.NoError(t, err)
}

// TestGetETHBlockTime a test for GetETHBlockTime
func TestGetETHBlockTime(t *testing.T) {
	client, err := NewGRPCClient("localhost:8889")
	if err != nil {
		fmt.Println(err)
		return
	}

	blockTime, err := client.GetETHBlockTime(context.Background(), "0xc220dc38d77f6105631461f85d9e6ca0b7048047f030d631e9f95d6c5eaa3fd2")

	assert.Equal(t, blockTime.Format(time.RFC3339Nano), "2023-06-23T09:45:12+07:00")
	assert.NoError(t, err)
}
