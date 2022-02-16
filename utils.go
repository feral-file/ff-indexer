package indexer

import (
	"fmt"
	"math/big"
	"strings"

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
	blockchainAlias, ok := BlockchianAlias[blockchainType]
	if !ok {
		blockchainAlias = "undefined"
	}

	return fmt.Sprintf("%s-%s-%s", blockchainAlias, contractAddress, id)
}

// DetectAccountBlockchain returns underlying blokchain of a given account number
func DetectAccountBlockchain(accountNumber string) string {
	if strings.HasPrefix(accountNumber, "0x") {
		return EthereumBlockchain
	} else if len(accountNumber) == 50 {
		return BitmarkBlockchain
	} else if strings.HasPrefix(accountNumber, "tz") {
		return TezosBlockchain
	}

	return ""
}
