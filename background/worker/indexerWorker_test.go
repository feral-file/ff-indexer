package worker

import (
	"testing"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
	"github.com/stretchr/testify/assert"
)

func TestGetBalanceDiffFromTezosTransaction(t *testing.T) {
	indexerEngine := indexer.New("", nil, tzkt.New(""), nil, nil)
	w := NFTIndexerWorker{
		Environment:   "",
		indexerEngine: indexerEngine,
	}

	// Transfer 1
	pendingAccountToken := indexer.AccountToken{
		IndexID:      "tez-KT1QWATcHYpBVDxvNoxw5CbkaMGpjm4SxXa2-17184878501790703549168165515956430051664778290448030394743847114378028095281",
		OwnerAccount: "tz1UYxQdehAZUA7xVY9JqyiPomeG8qqKt6Z7",
	}

	transactionDetails, err := w.indexerEngine.GetTransactionDetailsByPendingTx("opKHp2yLnk3Uv8jG4WXRs4tyEUUAXNmrHcvh5tzDZNBqBr7PnNb")
	assert.NoError(t, err)

	accountTokens, err := w.GetBalanceDiffFromTezosTransaction(transactionDetails[0], pendingAccountToken)
	assert.NoError(t, err)
	assert.Len(t, accountTokens, 2)
	assert.Equal(t, accountTokens[0].OwnerAccount, "tz1gDahtzboNACtGTkubR5RNXdtp2dANTNUu", true)
	assert.Equal(t, accountTokens[0].Balance, int64(1), true)
	assert.Equal(t, accountTokens[1].OwnerAccount, "tz1UYxQdehAZUA7xVY9JqyiPomeG8qqKt6Z7", true)
	assert.Equal(t, accountTokens[1].Balance, int64(-1), true)

	// Transfer 2
	pendingAccountToken = indexer.AccountToken{
		IndexID:      "tez-KT1PJNTpgxTsf91o93RPb7wSuw727bnUNoUt-0",
		OwnerAccount: "tz2KvnsT81omevxUr4JfsU8by1Ywpwk833zA",
	}

	transactionDetails, err = w.indexerEngine.GetTransactionDetailsByPendingTx("opWg8K5r9TBy6BQuQHwbnPYpntNdwt6izZvRPVvcC9PZB1vWMRf")
	assert.NoError(t, err)

	accountTokens, err = w.GetBalanceDiffFromTezosTransaction(transactionDetails[0], pendingAccountToken)
	assert.NoError(t, err)
	assert.Len(t, accountTokens, 251)
	assert.Equal(t, accountTokens[0].OwnerAccount, "tz1KnQk3X4qmQcXHtoeoXcj2WcLBVfYK99sv", true)
	assert.Equal(t, accountTokens[0].Balance, int64(1), true)
	assert.Equal(t, accountTokens[1].OwnerAccount, "tz1M6hvecxpEeN7ryQpdJmWRExeQRtn4UoAx", true)
	assert.Equal(t, accountTokens[1].Balance, int64(1), true)
	assert.Equal(t, accountTokens[249].OwnerAccount, "tz1egmQg7rv5yyzgEvNtAxBFwqtpfGGantqN", true)
	assert.Equal(t, accountTokens[249].Balance, int64(1), true)
	assert.Equal(t, accountTokens[250].OwnerAccount, "tz2KvnsT81omevxUr4JfsU8by1Ywpwk833zA", true)
	assert.Equal(t, accountTokens[250].Balance, int64(-250), true)

	transactionDetails, err = w.indexerEngine.GetTransactionDetailsByPendingTx("0x913ea0359128c7a17b0213bf485694a3f3a94656b2370a16984075966ab54dfc")
	assert.Error(t, err)
	assert.Len(t, transactionDetails, 0)

}

func TestGetBalanceDiffFromETHTransaction(t *testing.T) {
	indexerEngine := indexer.New("", nil, tzkt.New(""), nil, nil)
	w := NFTIndexerWorker{
		Environment:   "",
		indexerEngine: indexerEngine,
	}

	// transfer
	transactionDetails := []indexer.TransactionDetails{{
		From:    "0xE83C750b2708320bb134796c555b80DF39A3D97B",
		To:      "0xE6Be1ebD1B0A56EdF38b4E5F8AFa55aa40d8Afdd",
		IndexID: "7529",
	}}

	accountTokens, err := w.GetBalanceDiffFromETHTransaction(transactionDetails)
	assert.NoError(t, err)
	assert.Len(t, accountTokens, 2)
	assert.Equal(t, accountTokens[0].OwnerAccount, transactionDetails[0].To, true)
	assert.Equal(t, accountTokens[0].Balance, int64(1), true)
	assert.Equal(t, accountTokens[0].IndexID, "7529", true)
	assert.Equal(t, accountTokens[1].OwnerAccount, transactionDetails[0].From, true)
	assert.Equal(t, accountTokens[1].Balance, int64(-1), true)
	assert.Equal(t, accountTokens[1].IndexID, "7529", true)

	// minting
	transactionDetails = []indexer.TransactionDetails{{
		From:    "0x0000000000000000000000000000000000000000",
		To:      "0xFA73954137D7ef439efb9E87B0C86FFa44BA75A2",
		IndexID: "91758420999999695262527978151514422877926998985541613039379462996033863019025",
	}}

	accountTokens, err = w.GetBalanceDiffFromETHTransaction(transactionDetails)
	assert.NoError(t, err)
	assert.Len(t, accountTokens, 1)
	assert.Equal(t, accountTokens[0].OwnerAccount, transactionDetails[0].To, true)
	assert.Equal(t, accountTokens[0].Balance, int64(1), true)
	assert.Equal(t, accountTokens[0].IndexID, "91758420999999695262527978151514422877926998985541613039379462996033863019025", true)

}
