package indexerWorker

import (
	"context"
	"errors"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/viper"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/contracts"
)

var (
	ErrMapKeyNotFound   = errors.New("key is not found")
	ErrValueNotString   = errors.New("value is not of string type")
	ErrInvalidEditionID = errors.New("invalid edition id")
)

// GetOwnedERC721TokenIDByContract returns a list of token id belongs to an owner for a specific contract
func (w *NFTIndexerWorker) GetOwnedERC721TokenIDByContract(ctx context.Context, contractAddress, ownerAddress string) ([]*big.Int, error) {
	rpcClient, err := ethclient.Dial(viper.GetString("ethereum.rpc_url"))
	if err != nil {
		return nil, err
	}

	contractAddr := common.HexToAddress(contractAddress)
	instance, err := contracts.NewERC721Enumerable(contractAddr, rpcClient)
	if err != nil {
		return nil, err
	}

	var tokenIDs []*big.Int

	ownerAddr := common.HexToAddress(ownerAddress)
	n, err := instance.BalanceOf(nil, ownerAddr)
	if err != nil {
		return nil, err
	}
	for i := int64(0); i < n.Int64(); i++ {
		id, err := instance.TokenOfOwnerByIndex(nil, ownerAddr, big.NewInt(i))
		if err != nil {
			return nil, err
		}

		tokenIDs = append(tokenIDs, id)
	}

	return tokenIDs, nil
}

// ensureStringAttribute returns the string value of a key and returns an error if
// the key is missing or is not in type string.
func ensureStringAttribute(m map[string]interface{}, key string) (string, error) {
	v, ok := m[key]
	if !ok {
		return "", ErrMapKeyNotFound
	}

	vv, ok := v.(string)
	if !ok {
		return "", ErrValueNotString
	}

	return vv, nil
}

// IndexTokenDataFromArtblocks reads token data from artblocks and convert it into indexer compatible format
func (w *NFTIndexerWorker) IndexTokenDataFromArtblocks(ctx context.Context, contractAddress, owner string, token *big.Int) (*indexer.AssetUpdates, error) {
	tokenData, err := w.artblocks.GetTokenData(token.String())
	if err != nil {
		return nil, err
	}

	tokenID, err := ensureStringAttribute(tokenData, "tokenID")
	if err != nil {
		return nil, err
	}

	projectID, err := ensureStringAttribute(tokenData, "project_id")
	if err != nil {
		return nil, err
	}

	tokenHash, err := ensureStringAttribute(tokenData, "token_hash")
	if err != nil {
		return nil, err
	}

	editionID, err := strconv.Atoi(
		strings.Replace(tokenID, projectID, "", 1),
	)
	if err != nil {
		return nil, ErrInvalidEditionID
	}

	tokenUpdate := indexer.AssetUpdates{
		ID:              tokenHash,
		ProjectMetadata: tokenData,
		Tokens: []indexer.Token{
			{
				ID:              tokenID,
				Blockchain:      "ethereum",
				Edition:         int64(editionID),
				ContractAddress: contractAddress,
				Owner:           owner,
			},
		},
	}

	return &tokenUpdate, nil
}

// IndexAsset saves asset data into indexer's storage
func (w *NFTIndexerWorker) IndexAsset(ctx context.Context, updates indexer.AssetUpdates) error {
	return w.indexerStore.IndexAsset(ctx, updates.ID, updates)
}
