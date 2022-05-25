package indexer

import (
	"fmt"
	"math/big"
	"path/filepath"
	"strings"

	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// artblocksContracts indexes the addresses which are ERC721 contracts of Artblocks
var artblocksContracts = map[string]struct{}{
	"0x059EDD72Cd353dF5106D2B9cC5ab83a52287aC3a": {},
	"0xa7d8d9ef8D8Ce8992Df33D8b8CF4Aebabd5bD270": {},
}

// getTokenSourceByContract token source name by inspecting a contract address
func getTokenSourceByContract(contractAddress string) string {
	switch DetectContractBlockchain(contractAddress) {
	case EthereumBlockchain:
		if _, ok := artblocksContracts[contractAddress]; ok {
			return "Art Blocks"

		} else if contractAddress == "0x70e6b3f9d99432fCF35274d6b24D83Ef5Ba3dE2D" {
			return "Crayon Codes"
		}

		return "OpenSea"
	case TezosBlockchain:
		// WIP
		return ""
	default:
		return ""
	}
}

// mediumByPreviewFileExtension returns token medium by detecting file extension
func mediumByPreviewFileExtension(url string) string {
	ext := filepath.Ext(url)

	switch ext {
	case ".jpg", ".jpeg", ".png", ".svg":
		return "image"
	case ".mp4", ".mov":
		return "video"
	default:
		return "other"
	}
}

// IndexETHToken prepares indexing data for a specific asset read from opensea
func IndexETHToken(a *opensea.Asset) (*AssetUpdates, error) {
	dataSource := "opensea"

	var sourceURL string
	var artistURL string
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
			artistName = EthereumChecksumAddress(a.Creator.Address)
		}
	}

	metadata := ProjectMetadata{
		ArtistName:          artistName,
		ArtistURL:           artistURL,
		AssetID:             contractAddress,
		Title:               a.Name,
		Description:         a.Description,
		Medium:              "unknown",
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
			metadata.Medium = "software"
		} else {
			medium := mediumByPreviewFileExtension(metadata.PreviewURL)
			log.WithField("previewURL", metadata.PreviewURL).WithField("medium", medium).Debug("fallback medium check")
			metadata.Medium = medium
		}
	} else if a.ImageURL != "" {
		metadata.Medium = "image"
	}

	// token id from opensea is a decimal integer string
	tokenID, ok := big.NewInt(0).SetString(a.TokenID, 10)
	if !ok {
		return nil, fmt.Errorf("fail to parse token id from opensea")
	}

	return &AssetUpdates{
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
	}, nil
}
