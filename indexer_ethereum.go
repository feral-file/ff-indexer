package indexer

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/bitmark-inc/nft-indexer/externals/opensea"
)

// IndexETHTokenByOwner indexes all tokens owned by a specific ethereum address
func (e *IndexEngine) IndexETHTokenByOwner(ctx context.Context, owner string, offset int) ([]AssetUpdates, error) {
	assets, err := e.opensea.RetrieveAssets(owner, offset)
	if err != nil {
		return nil, err
	}

	tokenUpdates := make([]AssetUpdates, 0, len(assets))

	for _, a := range assets {
		update, err := e.indexETHToken(&a)
		if err != nil {
			log.WithError(err).Error("fail to index token data")
		}

		if update != nil {
			tokenUpdates = append(tokenUpdates, *update)
		}
	}

	return tokenUpdates, nil
}

// IndexETHToken indexes an Ethereum token with a specific contract and ID
func (e *IndexEngine) IndexETHToken(ctx context.Context, owner, contract, tokenID string) (*AssetUpdates, error) {
	a, err := e.opensea.RetrieveAsset(contract, tokenID)
	if err != nil {
		return nil, err
	}

	return e.indexETHToken(a)
}

// indexETHToken prepares indexing data for a specific asset read from opensea
func (e *IndexEngine) indexETHToken(a *opensea.Asset) (*AssetUpdates, error) {
	dataSource := "opensea"

	var sourceURL string
	var artistURL string
	artistID := EthereumChecksumAddress(a.Creator.Address)
	artistName := a.Creator.User.Username
	contractAddress := EthereumChecksumAddress(a.AssetContract.Address)
	switch contractAddress {
	case ENSContractAddress:
		return nil, nil
	}

	source := getTokenSourceByContract(contractAddress)

	switch source {
	case "Art Blocks":
		sourceURL = "https://www.artblocks.io/"
		artistURL = fmt.Sprintf("%s/%s", sourceURL, a.Creator.Address)
	case "Crayon Codes":
		sourceURL = "https://openprocessing.org/crayon/"
		artistURL = fmt.Sprintf("https://opensea.io/%s", a.Creator.Address)
	default:
		if viper.GetString("network") == "testnet" {
			sourceURL = "https://testnets.opensea.io"
		} else {
			sourceURL = "https://opensea.io"
		}
		artistURL = fmt.Sprintf("https://opensea.io/%s", a.Creator.Address)
	}

	log.WithField("source", source).WithField("assetID", a.ID).Debug("source debug")

	if a.Creator.Address != "" {
		if artistName == "" {
			artistName = artistID
		}
	}

	metadata := ProjectMetadata{
		ArtistID:            artistID,
		ArtistName:          artistName,
		ArtistURL:           artistURL,
		AssetID:             contractAddress,
		Title:               a.Name,
		Description:         a.Description,
		Medium:              MediumUnknown,
		Source:              source,
		SourceURL:           sourceURL,
		PreviewURL:          a.ImageURL,
		ThumbnailURL:        a.ImageURL,
		GalleryThumbnailURL: a.ImagePreviewURL,
		AssetURL:            a.Permalink,
	}

	if a.AnimationURL != "" {
		metadata.PreviewURL = a.AnimationURL

		if source == "Art Blocks" {
			metadata.Medium = MediumSoftware
		} else {
			medium := mediumByPreviewFileExtension(metadata.PreviewURL)
			log.WithField("previewURL", metadata.PreviewURL).WithField("medium", medium).Debug("fallback medium check")
			metadata.Medium = medium
		}
	} else if a.ImageURL != "" {
		metadata.Medium = MediumImage
	}

	// token id from opensea is a decimal integer string
	tokenID, ok := big.NewInt(0).SetString(a.TokenID, 10)
	if !ok {
		return nil, fmt.Errorf("fail to parse token id from opensea")
	}

	tokenUpdate := &AssetUpdates{
		ID:              fmt.Sprintf("%d", a.ID),
		Source:          dataSource,
		ProjectMetadata: metadata,
		Tokens: []Token{
			{
				BaseTokenInfo: BaseTokenInfo{
					ID:              a.TokenID,
					Blockchain:      EthereumBlockchain,
					ContractType:    strings.ToLower(a.AssetContract.SchemaName),
					ContractAddress: contractAddress,
				},
				IndexID: TokenIndexID(EthereumBlockchain, contractAddress, tokenID.Text(16)),
				Edition: 0,
				Owner:   EthereumChecksumAddress(a.Owner.Address),
				MintAt:  a.AssetContract.CreatedDate.Time, // set minted_at to the contract creation time
			},
		},
	}

	log.WithField("blockchain", EthereumBlockchain).
		WithField("id", TokenIndexID(EthereumBlockchain, contractAddress, tokenID.Text(16))).
		WithField("tokenUpdate", tokenUpdate).
		Trace("asset updating data prepared")

	return tokenUpdate, nil
}
