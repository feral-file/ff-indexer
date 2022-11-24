package indexer

import (
	"context"
	"errors"
	"fmt"
	"github.com/fatih/structs"
	"net/http"
	"path/filepath"
	"strings"
	"time"

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

// CheckDataIsEnough check metadata has enough data or not
func (detail *AssetMetadataDetail) CheckDataIsEnough() bool {
	if detail.Name != "" &&
		detail.Description != "" &&
		detail.MIMEType != "" &&
		detail.Medium != "" &&
		detail.ArtistID != "" &&
		detail.ArtistName != "" &&
		detail.ArtistURL != "" &&
		detail.MaxEdition != 0 &&
		detail.DisplayURI != DEFAULT_DISPLAY_URI &&
		detail.PreviewURI != DEFAULT_DISPLAY_URI {
		return true
	}

	return false
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
func (detail *AssetMetadataDetail) FromTZKT(t tzkt.Token, creatorInfo tzkt.UserInfo) {
	detail.MaxEdition = int64(t.TotalSupply)

	detail.UpdateMetadataFromTZKT(t.Metadata, creatorInfo)
}

// UpdateMetadataFromTZKT update TZKT token metadata to AssetMetadataDetail
func (detail *AssetMetadataDetail) UpdateMetadataFromTZKT(metadata tzkt.TokenMetadata, creatorInfo tzkt.UserInfo) {
	var mimeType string

	for _, f := range metadata.Formats {
		if f.URI == metadata.ArtifactURI {
			mimeType = f.MIMEType
			break
		}
	}

	detail.Name = metadata.Name
	detail.Description = metadata.Description
	detail.MIMEType = mimeType
	detail.Medium = mediumByMIMEType(mimeType)

	var optimizedFileSize = 0
	var optimizedDisplayURI string

	for _, format := range metadata.Formats {
		if strings.Contains(format.MIMEType, "image") && format.FileSize > optimizedFileSize {
			optimizedDisplayURI = format.URI
			optimizedFileSize = format.FileSize
		}
	}

	var displayURI, previewURI string

	if optimizedDisplayURI != "" {
		displayURI = optimizedDisplayURI
	} else if metadata.DisplayURI != "" {
		displayURI = metadata.DisplayURI
	} else if metadata.ThumbnailURI != "" {
		displayURI = metadata.ThumbnailURI
	} else {
		displayURI = DEFAULT_DISPLAY_URI
	}

	if metadata.ArtifactURI != "" {
		previewURI = metadata.ArtifactURI
	} else {
		previewURI = displayURI
	}

	detail.DisplayURI = displayURI
	detail.PreviewURI = previewURI

	detail.ArtistID = creatorInfo.Address
	detail.ArtistName = creatorInfo.Alias
	detail.ArtistURL = getTZKTArtistURL(creatorInfo)
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

	switch DetectContractBlockchain(contract) {
	case TezosBlockchain:
		tokenOwners, err := e.tzkt.GetTokenOwners(contract, tokenID)
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

		return tokenOwners[0].Owner.Address, nil
	default:
		return "", ErrUnsupportedBlockchain
	}

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

func (e *IndexEngine) GetTransactionDetailsByPendingTx(pendingTx string) ([]tzkt.TransactionDetails, error) {
	detailedTransactions, err := e.tzkt.GetTransactionByTx(pendingTx)
	if err != nil {
		return nil, err
	}

	return detailedTransactions, nil
}

// GetCDNURLFromObjkt get CDN URL of Objkt
func (d *AssetMetadataDetail) GetCDNURLFromObjkt() {
	for _, assetType := range ObjktCDNTypes {
		d.ReplaceIPFSURIByObjktCDNURI(assetType)
	}
}

// FromObjkt reads asset detail from Objkt API object
func (d *AssetMetadataDetail) FromObjkt(objktToken objkt.Token) {
	d.UpdateMetadataFromObjkt(objktToken)
	d.GetCDNURLFromObjkt()
}

// UpdateMetadataFromObjkt update Objkt metadata to AssetMetadataDetail
func (d *AssetMetadataDetail) UpdateMetadataFromObjkt(token objkt.Token) {
	d.Name = token.Name
	d.Description = token.Description
	d.MIMEType = token.Mime
	d.Medium = mediumByMIMEType(token.Mime)

	if len(token.Creators) > 0 {
		d.ArtistID = token.Creators[0].Holder.Address
		d.ArtistName = token.Creators[0].Holder.Alias
		d.ArtistURL = getObjktArtistURL(token.Creators[0].Holder)
	}

	if token.Thumbnail_uri != "" {
		d.DisplayURI = token.Thumbnail_uri
	} else if token.Display_uri != "" {
		d.DisplayURI = token.Display_uri
	} else {
		d.DisplayURI = DEFAULT_DISPLAY_URI
	}

	if token.Artifact_uri != "" {
		d.PreviewURI = token.Artifact_uri
	} else {
		d.PreviewURI = DEFAULT_DISPLAY_URI
	}
}

func getTZKTArtistURL(a tzkt.UserInfo) string {
	m := a.Metadata
	s := structs.Map(m)

	site := s["Site"].(string)
	if site != "" {
		return site
	}

	delete(s, "Site")

	for k, v := range s {
		name := v.(string)
		if name != "" {
			return SocialMediaLink[k] + name
		}
	}

	return fmt.Sprintf("https://objkt.com/profile/%s", a.Address)
}

// getObjktArtistURL get social media url of Artist from Objkt api
func getObjktArtistURL(h objkt.Holder) string {
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

// ReplaceIPFSURIByObjktCDNURI get cid from IPFS uri and make Objkt CND uri
func (d *AssetMetadataDetail) ReplaceIPFSURIByObjktCDNURI(assetType string) {
	if assetType == ObjktCDNDisplayType || assetType == ObjktCDNThumbnailType {
		if strings.Contains(d.DisplayURI, "assets.objkt.media/file/assets-003") {
			return
		}

		uri, err := MakeCDNURIFromIPFSURI(d.DisplayURI, assetType)

		if err == nil {
			d.DisplayURI = uri
		} else {
			d.DisplayURI = defaultIPFSLink(d.DisplayURI)
		}

		return
	}

	if assetType == ObjktCDNArtifactType {
		uri, err := MakeCDNURIFromIPFSURI(d.PreviewURI, assetType)

		if err == nil {
			d.PreviewURI = uri
		} else {
			d.PreviewURI = defaultIPFSLink(d.PreviewURI)
		}

		return
	}
}

// MakeCDNURIFromIPFSURI create Objkt CDN uri from IPFS Uri(extract cid)
func MakeCDNURIFromIPFSURI(sURI string, assetType string) (string, error) {
	splitUri := strings.Split(sURI, "/")
	cid := splitUri[len(splitUri)-1]

	url := ObjktCDNURL + cid + "/" + assetType
	res, err := http.Get(url)

	if err == nil && res.StatusCode >= 200 && res.StatusCode < 400 {
		return url, nil
	}

	return "", errors.New("can not reach CDN url")
}

func (p *ProjectMetadata) CheckDataIsEnough() bool {
	if p.Title != "" &&
		p.Description != "" &&
		p.MIMEType != "" &&
		p.Medium != "" &&
		p.ArtistID != "" &&
		p.ArtistName != "" &&
		p.ArtistURL != "" &&
		p.MaxEdition != 0 &&
		p.PreviewURL != DEFAULT_DISPLAY_URI &&
		p.ThumbnailURL != DEFAULT_DISPLAY_URI {
		return true
	}

	return false
}
