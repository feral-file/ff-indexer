package indexer

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitmark-inc/nft-indexer/externals/bettercall"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
)

const (
	// broken-image.svg
	DEFAULT_DISPLAY_URI = "ipfs://QmX5rRzkZQfvEyaYc1Q78YZ83pFj3AgpFVSK8SmxUmZ85M"
)

// artblocksContracts indexes the addresses which are ERC721 contracts of Artblocks
var artblocksContracts = map[string]struct{}{
	"0x059EDD72Cd353dF5106D2B9cC5ab83a52287aC3a": {},
	"0xa7d8d9ef8D8Ce8992Df33D8b8CF4Aebabd5bD270": {},
}

var ErrUnsupportedBlockchain = fmt.Errorf("unsupported blockchain")

// getTokenSourceByContract token source name by inspecting a contract address
func getTokenSourceByContract(contractAddress string) string {
	switch DetectContractBlockchain(contractAddress) {
	case EthereumBlockchain:
		if _, ok := artblocksContracts[contractAddress]; ok {
			return "Art Blocks"

		} else if contractAddress == "0x70e6b3f9d99432fCF35274d6b24D83Ef5Ba3dE2D" {
			return "Crayon Codes"
		}

		return SourceOpensea
	case TezosBlockchain:
		// WIP
		return ""
	default:
		return ""
	}
}

// mediumByPreviewFileExtension returns token medium by detecting file extension
// this only work for opensea since files over IPFS normally does not have extensions
func mediumByPreviewFileExtension(url string) Medium {
	ext := filepath.Ext(url)

	switch ext {
	case ".jpg", ".jpeg", ".png", ".svg":
		return MediumImage
	case ".mp4", ".mov":
		return MediumVideo
	case "":
		return MediumUnknown
	default:
		return MediumOther
	}
}

// mediumByMIMEType returns medium by detecting mime-type
func mediumByMIMEType(mimeType string) Medium {
	if mimeItems := strings.Split(mimeType, "/"); len(mimeItems) > 0 {
		switch mimeItems[0] {
		case "image":
			return MediumImage
		case "video":
			return MediumVideo
		case "":
			return MediumUnknown
		default:
			return MediumOther
		}
	}
	return MediumUnknown
}

// defaultIPFSLink converts an IPFS link to a HTTP link by using ipfs.io gateway.
func defaultIPFSLink(ipfsLink string) string {
	return strings.ReplaceAll(ipfsLink, "ipfs://", "https://ipfs.io/ipfs/")
}

type MarketplaceProfile struct {
	Source    string
	SourceURL string
	AssetURL  string
}

// AssetMetadataDetail is a structure what contains the basic source
// information of the underlying asset
type AssetMetadataDetail struct {
	AssetID string

	// marketplace information
	Source    string
	SourceURL string
	AssetURL  string

	Name        string
	Description string
	MIMEType    string
	Medium      Medium

	ArtistID   string
	ArtistName string
	ArtistURL  string
	MaxEdition int64

	DisplayURI string
	PreviewURI string
}

func NewAssetMetadataDetail(assetID string) *AssetMetadataDetail {
	return &AssetMetadataDetail{
		AssetID: assetID,
		Medium:  MediumUnknown,
	}
}

// SetMarketplace sets marketplace property
func (detail *AssetMetadataDetail) SetMarketplace(profile MarketplaceProfile) {
	detail.Source = profile.Source
	detail.SourceURL = profile.SourceURL
	detail.AssetURL = profile.AssetURL
}

func (detail *AssetMetadataDetail) SetMedium(m Medium) {
	detail.Medium = m
}

// FromBetterCallDev reads asset detail from an better call dev API object
func (detail *AssetMetadataDetail) FromBetterCallDev(t bettercall.Token, m bettercall.TokenMetadata) {
	var mimeType string
	for _, f := range t.Formats {
		if f.URI == t.ArtifactURI {
			mimeType = f.MIMEType
			break
		}
	}
	if m.TokenInfo.MimeType != "" {
		mimeType = m.TokenInfo.MimeType
	}

	detail.Name = t.Name
	detail.Description = t.Description
	detail.MIMEType = mimeType
	detail.Medium = mediumByMIMEType(mimeType)

	if len(t.Creators) > 0 {
		detail.ArtistID = t.Creators[0]
		detail.ArtistName = t.Creators[0] // creator tezos address
		detail.ArtistURL = fmt.Sprintf("https://objkt.com/profile/%s", t.Creators[0])
	}

	detail.MaxEdition = m.Supply

	var displayURI, previewURI string
	if t.DisplayURI != "" {
		displayURI = t.DisplayURI
	} else if t.ThumbnailURI != "" {
		displayURI = t.ThumbnailURI
	} else {
		displayURI = DEFAULT_DISPLAY_URI
	}

	if t.ArtifactURI != "" {
		previewURI = t.ArtifactURI
	} else {
		previewURI = displayURI
	}
	detail.DisplayURI = defaultIPFSLink(displayURI)
	detail.PreviewURI = defaultIPFSLink(previewURI)
}

// FromFxhashObject reads asset detail from an fxhash API object
func (detail *AssetMetadataDetail) FromFxhashObject(o fxhash.FxHashObjectDetail) {
	detail.Name = o.Name
	detail.Description = o.Metadata.Description
	detail.ArtistID = o.Issuer.Author.ID
	detail.ArtistName = o.Issuer.Author.ID
	detail.ArtistURL = fmt.Sprintf("https://www.fxhash.xyz/u/%s", o.Issuer.Author.Name)
	detail.MaxEdition = o.Issuer.Supply
	detail.DisplayURI = fxhashLink(o.Metadata.DisplayURI)
	detail.PreviewURI = fxhashLink(o.Metadata.ArtifactURI)
}

// FromObjktObject reads asset detail from an objkt API object
func (detail *AssetMetadataDetail) FromObjktObject(o objkt.ObjktTokenDetails) {
	detail.Name = o.Name
	detail.Description = o.Description
	if o.Contract.CreatorAddress != "" {
		detail.ArtistID = o.Contract.CreatorAddress
		detail.ArtistName = o.Contract.CreatorAddress
		detail.ArtistURL = fmt.Sprintf("https://objkt.com/profile/%s", o.Contract.CreatorAddress)
	}
	detail.MaxEdition = o.Supply

	detail.MIMEType = o.MIMEType
	detail.Medium = mediumByMIMEType(o.MIMEType)
	detail.DisplayURI = defaultIPFSLink(o.DisplayURI)
	detail.PreviewURI = defaultIPFSLink(o.ArtifactURI)
}

// TokenDetail saves token specific detail from different sources
type TokenDetail struct {
	Edition  int64
	MintedAt time.Time
}

func (e *IndexEngine) IndexToken(c context.Context, owner, contract, tokenID string) (*AssetUpdates, error) {
	switch DetectContractBlockchain(contract) {
	case EthereumBlockchain:
		return e.IndexETHToken(c, owner, contract, tokenID)
	case TezosBlockchain:
		return e.IndexTezosToken(c, owner, contract, tokenID)
	default:
		return nil, ErrUnsupportedBlockchain
	}
}
