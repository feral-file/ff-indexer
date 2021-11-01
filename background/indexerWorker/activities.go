package indexerWorker

import (
	"context"
	"errors"
	"fmt"
	"math/big"
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

// IndexTokenDataFromFromOpensea indexes data from OpenSea into the format of AssetUpdates
func (w *NFTIndexerWorker) IndexTokenDataFromFromOpensea(ctx context.Context, owner string, offset int) ([]indexer.AssetUpdates, error) {
	assets, err := w.opensea.RetrieveAssets(owner, offset)
	if err != nil {
		return nil, err
	}

	tokenUpdates := make([]indexer.AssetUpdates, 0, len(assets))

	for _, a := range assets {
		var source string
		var sourceURL string
		var artistURL string

		if _, ok := artblocksContracts[strings.ToLower(a.AssetContract.Address)]; ok {
			source = "ArtBlocks"
			sourceURL = "https://www.artblocks.io/"
		} else {
			source = "OpenSea"
			if viper.GetString("network") == "testnet" {
				sourceURL = "https://testnets.opensea.io"
			} else {
				sourceURL = "https://opensea.io"
			}
		}

		if a.Creator.Address != "" {
			artistURL = fmt.Sprintf("%s/%s", sourceURL, a.Creator.Address)
		}

		metadata := indexer.ProjectMetadata{
			ArtistName:          a.Creator.User.Username,
			ArtistURL:           artistURL,
			AssetID:             a.AssetContract.Address,
			Title:               a.Name,
			Description:         a.Description,
			Medium:              "unknown",
			Source:              source,
			SourceURL:           sourceURL,
			PreviewURL:          a.ImageURL,
			ThumbnailURL:        a.ImageThumbnailURL,
			GalleryThumbnailURL: a.ImageThumbnailURL,
			AssetURL:            a.Permalink,
		}

		if a.AnimationURL != "" {
			metadata.Medium = "other"
			metadata.PreviewURL = a.AnimationURL
		} else if a.ImageURL != "" {
			metadata.Medium = "image"
		}

		tokenUpdate := indexer.AssetUpdates{
			ID:              fmt.Sprintf("%d", a.ID),
			ProjectMetadata: metadata,
			Tokens: []indexer.Token{
				{
					ID:              a.TokenID,
					Blockchain:      "ethereum",
					Edition:         0,
					ContractType:    strings.ToLower(a.AssetContract.SchemaName),
					ContractAddress: a.AssetContract.Address,
					Owner:           owner,
					MintAt:          a.AssetContract.CreatedDate.Time,
				},
			},
		}

		fmt.Println(tokenUpdate)

		tokenUpdates = append(tokenUpdates, tokenUpdate)
	}

	return tokenUpdates, nil
}

// IndexAsset saves asset data into indexer's storage
func (w *NFTIndexerWorker) IndexAsset(ctx context.Context, updates indexer.AssetUpdates) error {
	return w.indexerStore.IndexAsset(ctx, updates.ID, updates)
}
