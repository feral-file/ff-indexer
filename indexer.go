package indexer

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/structs"

	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
)

const (
	// broken-image.svg
	DEFAULT_DISPLAY_URI  = "ipfs://QmX5rRzkZQfvEyaYc1Q78YZ83pFj3AgpFVSK8SmxUmZ85M"
	DEFAULT_IPFS_GATEWAY = "https://ipfs.io/ipfs/"
)

// artblocksContracts indexes the addresses which are ERC721 contracts of Artblocks
var artblocksContracts = map[string]struct{}{
	"0x059EDD72Cd353dF5106D2B9cC5ab83a52287aC3a": {},
	"0xa7d8d9ef8D8Ce8992Df33D8b8CF4Aebabd5bD270": {},
}

var ErrUnsupportedBlockchain = fmt.Errorf("unsupported blockchain")

// getTokenSourceByContract token source name by inspecting a contract address
func getTokenSourceByContract(contractAddress string) string {
	switch GetBlockchainByAddress(contractAddress) {
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
	return strings.ReplaceAll(ipfsLink, "ipfs://", DEFAULT_IPFS_GATEWAY)
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

// FromTZKT reads asset detail from tzkt API object
func (detail *AssetMetadataDetail) FromTZKT(t tzkt.Token) {
	detail.MaxEdition = int64(t.TotalSupply)

	detail.UpdateMetadataFromTZKT(t.Metadata)
}

// UpdateMetadataFromTZKT update TZKT token metadata to AssetMetadataDetail
func (detail *AssetMetadataDetail) UpdateMetadataFromTZKT(md tzkt.TokenMetadata) {
	var mimeType string

	for _, f := range md.Formats {
		if f.URI == md.ArtifactURI {
			mimeType = string(f.MIMEType)
			break
		}
	}

	detail.Name = md.Name
	detail.Description = md.Description
	detail.MIMEType = mimeType
	detail.Medium = mediumByMIMEType(mimeType)

	var optimizedFileSize = 0
	var optimizedDisplayURI string

	for _, format := range md.Formats {
		if strings.Contains(string(format.MIMEType), "image") && format.FileSize > optimizedFileSize {
			optimizedDisplayURI = format.URI
			optimizedFileSize = format.FileSize
		}
	}

	var displayURI, previewURI string

	if optimizedDisplayURI != "" {
		displayURI = optimizedDisplayURI
	} else if md.DisplayURI != "" {
		displayURI = md.DisplayURI
	} else if md.ThumbnailURI != "" {
		displayURI = md.ThumbnailURI
	} else {
		displayURI = DEFAULT_DISPLAY_URI
	}

	if md.ArtifactURI != "" {
		previewURI = md.ArtifactURI
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

// TokenDetail saves token specific detail from different sources
type TokenDetail struct {
	Edition  int64
	Fungible bool
	MintedAt time.Time
}

// GetTokenOwnerAddress get token owners of a specific contract and tokenID
func (e *IndexEngine) GetTokenOwnerAddress(contract, tokenID string) (string, error) {
	if contract == "" {
		return "", fmt.Errorf("contract must not be empty")
	}

	switch GetBlockchainByAddress(contract) {
	case TezosBlockchain:
		tokenOwners, err := e.tzkt.GetTokenOwners(contract, tokenID, 1, time.Time{})
		if err != nil {
			return "", err
		}

		if len(tokenOwners) == 0 {
			return "", fmt.Errorf("no token owners found")
		}

		return tokenOwners[0].Address, nil
	case EthereumBlockchain:
		switch EthereumChecksumAddress(contract) {
		case ENSContractAddress:
			return "", fmt.Errorf("this contract is in the black list")
		}

		tokenOwners, _, err := e.opensea.RetrieveTokenOwners(contract, tokenID, nil)
		if err != nil {
			return "", err
		}

		if len(tokenOwners) == 0 {
			return "", fmt.Errorf("no token owners found")
		}

		return EthereumChecksumAddress(tokenOwners[0].Owner.Address), nil
	default:
		return "", ErrUnsupportedBlockchain
	}

}

func (e *IndexEngine) IndexToken(c context.Context, owner, contract, tokenID string) (*AssetUpdates, error) {
	switch GetBlockchainByAddress(contract) {
	case EthereumBlockchain:
		return e.IndexETHToken(c, owner, contract, tokenID)
	case TezosBlockchain:
		return e.IndexTezosToken(c, owner, contract, tokenID)
	default:
		return nil, ErrUnsupportedBlockchain
	}
}

func (e *IndexEngine) GetTransactionDetailsByPendingTx(pendingTx string) ([]tzkt.TransactionDetails, error) {
	detailedTransactions, err := e.tzkt.GetTransactionByTx(pendingTx)
	if err != nil {
		return nil, err
	}

	return detailedTransactions, nil
}

// FromObjkt reads asset detail from Objkt API object
func (d *AssetMetadataDetail) FromObjkt(objktToken objkt.Token) {
	d.UpdateMetadataFromObjkt(objktToken)

	if len(objktToken.Creators) > 0 {
		d.ArtistID = objktToken.Creators[0].Holder.Address
		d.ArtistName = objktToken.Creators[0].Holder.Alias
		d.ArtistURL = getArtistURL(objktToken.Creators[0].Holder)
	}
}

// UpdateMetadataFromObjkt update Objkt metadata to AssetMetadataDetail
func (d *AssetMetadataDetail) UpdateMetadataFromObjkt(token objkt.Token) {
	d.Name = token.Name
	d.Description = token.Description
	d.MIMEType = token.Mime
	d.Medium = mediumByMIMEType(token.Mime)

	if token.DisplayUri != "" {
		d.DisplayURI = d.ReplaceIPFSURIByObjktCDNURI(ObjktCDNDisplayType, token.DisplayUri, token.FaContract, token.TokenID)
	} else if token.ThumbnailUri == hicetnuncDefaultThumbnailURL {
		d.DisplayURI = d.ReplaceIPFSURIByObjktCDNURI(ObjktCDNArtifactThumbnailType, token.ThumbnailUri, token.FaContract, token.TokenID)
	} else if token.ThumbnailUri != "" {
		d.DisplayURI = d.ReplaceIPFSURIByObjktCDNURI(ObjktCDNThumbnailType, token.ThumbnailUri, token.FaContract, token.TokenID)
	}

	if d.DisplayURI == "" || d.DisplayURI == hicetnuncDefaultThumbnailURL {
		d.DisplayURI = defaultIPFSLink(DEFAULT_DISPLAY_URI)
	}

	if token.ArtifactUri != "" {
		d.PreviewURI = d.ReplaceIPFSURIByObjktCDNURI(ObjktCDNArtifactType, token.ArtifactUri, token.FaContract, token.TokenID)
	} else {
		d.PreviewURI = defaultIPFSLink(DEFAULT_DISPLAY_URI)
	}
}

// getArtistURL get social media url of Artist from Objkt api
func getArtistURL(h objkt.Holder) string {
	s := structs.Map(h)

	for k, v := range s {
		if k == "Alias" || k == "Address" {
			continue
		}
		if v != "" {
			return v.(string)
		}
	}

	return fmt.Sprintf("https://objkt.com/profile/%s", h.Address)
}

// ReplaceIPFSURIByObjktCDNURI return CDN uri if exist, if not this function will return ipfs link
func (d *AssetMetadataDetail) ReplaceIPFSURIByObjktCDNURI(assetType, assetUri, contract, tokenID string) string {
	if !strings.HasPrefix(assetUri, "ipfs://") {
		return assetUri
	}

	uri, err := MakeCDNURIFromIPFSURI(assetUri, assetType, contract, tokenID)

	if err == nil {
		return uri
	}

	return assetUri
}

// MakeCDNURIFromIPFSURI create Objkt CDN uri from IPFS Uri(extract cid)
func MakeCDNURIFromIPFSURI(assetURI, assetType, contract, tokenID string) (string, error) {
	var uri string
	var cid string

	urlParsed, err := url.Parse(assetURI)
	if err != nil {
		return "", err
	}

	cid = urlParsed.Host

	urlParsed.Scheme = "https"
	urlParsed.Host = ObjktCDNHost

	if assetType == ObjktCDNArtifactThumbnailType {
		urlParsed.Path, err = url.JoinPath(ObjktCDNBasePath, contract, tokenID, assetType)
		if err != nil {
			return "", err
		}

		uri = urlParsed.String()

		if CheckCDNURLIsExist(uri) {
			return uri, nil
		} else {
			return "", fmt.Errorf("CDN URL is not exist")
		}
	}

	urlParsed.Path, err = url.JoinPath(ObjktCDNBasePath, cid, assetType)
	if err != nil {
		return "", err
	}

	if assetType == ObjktCDNArtifactType && urlParsed.RawQuery != "" {
		urlParsed.Path, err = url.JoinPath(urlParsed.Path, "/index.html")
		if err != nil {
			return "", err
		}
	}

	uri = urlParsed.String()

	if CheckCDNURLIsExist(uri) {
		return uri, nil
	} else if assetType == ObjktCDNArtifactType && urlParsed.RawQuery == "" {
		uri = uri + "/index.html"

		if CheckCDNURLIsExist(uri) {
			return uri, nil
		}
	}

	return assetURI, nil
}
