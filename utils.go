package indexer

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func EthereumChecksumAddress(address string) string {
	return common.HexToAddress(address).Hex()
}

func OpenseaTokenIDToHex(tokenID string) (string, error) {
	tokenIDBig, ok := big.NewInt(0).SetString(tokenID, 10)
	if !ok {
		return "", fmt.Errorf("fail to parse token id")
	}

	return tokenIDBig.Text(16), nil
}

// AssetIndexID returns a source-based unique asset id. It is constructed by
// source of the asset data and the asset id from the source site.
func AssetIndexID(source, id string) string {
	return fmt.Sprintf("%s-%s", source, id)
}

// TokenIndexID returns blockchain-based unique token id. It is constructed by
// blockchain alias, token contract and token id.
func TokenIndexID(blockchainType, contractAddress, id string) string {
	blockchainAlias, ok := BlockchainAlias[blockchainType]
	if !ok {
		blockchainAlias = "undefined"
	}

	return fmt.Sprintf("%s-%s-%s", blockchainAlias, contractAddress, id)
}

// DetectAccountBlockchain returns underlying blockchain of a given account number
func DetectAccountBlockchain(accountNumber string) string {
	if strings.HasPrefix(accountNumber, "0x") {
		return EthereumBlockchain
	} else if len(accountNumber) == 50 {
		return BitmarkBlockchain
	} else if strings.HasPrefix(accountNumber, "tz") {
		return TezosBlockchain
	}

	return UnknownBlockchain
}

// DetectContractBlockchain returns underlying blockchain of a given contract address
func DetectContractBlockchain(contractAddress string) string {
	if strings.HasPrefix(contractAddress, "0x") {
		return EthereumBlockchain
	} else if strings.HasPrefix(contractAddress, "KT1") {
		return TezosBlockchain
	}

	return ""
}

// TxURL returns corresponded blockchain transaction URL
func TxURL(blockchain, environment, txID string) string {
	switch blockchain {
	case BitmarkBlockchain:
		if environment == "production" {
			return fmt.Sprintf("https://registry.bitmark.com/transaction/%s", txID)
		}
		return fmt.Sprintf("https://registry.test.bitmark.com/transaction/%s", txID)
	case EthereumBlockchain:
		if environment == "production" {
			return fmt.Sprintf("https://etherscan.io/tx/%s", txID)
		}
		return fmt.Sprintf("https://goerli.etherscan.io/tx/%s", txID)
	default:
		return ""
	}
}

// SleepWithContext will return whenever the slept time reached or context done
// It returns true if the context is done
func SleepWithContext(ctx context.Context, d time.Duration) bool {
	select {
	case <-time.After(d):
		return false
	case <-ctx.Done():
		return true
	}
}

func ParseIndexID(indexID string) (string, string, string, error) {
	v := strings.Split(indexID, "-")
	if len(v) != 3 {
		return "", "", "", fmt.Errorf("error while parsing indexID: %v", indexID)
	}
	return v[0], v[1], v[2], nil
}
