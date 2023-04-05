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
	"github.com/bitmark-inc/tzkt-go"
)

const (
	// broken-image.svg
	DefaultDisplayURI    = "ipfs://QmX5rRzkZQfvEyaYc1Q78YZ83pFj3AgpFVSK8SmxUmZ85M"
	DefaultIPFSGateway   = "ipfs.nftstorage.link"
	FxhashGateway        = "gateway.fxhash.xyz"
	FxhashDevIPFSGateway = "gateway.fxhash-dev2.xyz"
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
		case "application":
			return MediumSoftware
		case "":
			return MediumUnknown
		default:
			return MediumOther
		}
	}
	return MediumUnknown
}

// ipfsURLToGatewayURL converts an IPFS link to a HTTP link by a given ipfs gateway.
// If a link is failed to parse, it returns the original link
func ipfsURLToGatewayURL(gateway, ipfsURL string) string {
	u, err := url.Parse(ipfsURL)
	if err != nil {
		return ipfsURL
	}

	if u.Scheme != "ipfs" {
		// not a valid URL
		return ipfsURL
	}

	u.Path = fmt.Sprintf("ipfs/%s/", u.Host)
	u.Host = gateway
	u.Scheme = "https"

	return u.String()
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

	ArtworkMetadata map[string]interface{}
	Artists         []Artist
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

	if t.Metadata != nil {
		detail.FromTZIP21TokenMetadata(*t.Metadata)
	}
}

// FromTZIP21TokenMetadata update TZKT token metadata to AssetMetadataDetail
func (detail *AssetMetadataDetail) FromTZIP21TokenMetadata(md tzkt.TokenMetadata) {
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
		displayURI = DefaultDisplayURI
	}

	if md.ArtifactURI != "" {
		previewURI = md.ArtifactURI
	} else {
		previewURI = displayURI
	}

	if len(md.Creators) > 0 {
		var artists []Artist
		for _, v := range md.Creators {
			artists = append(artists, Artist{
				ArtistID:   v,
				ArtistName: v,
			})
		}

		detail.Artists = artists
		detail.ArtistID = md.Creators[0]
		detail.ArtistName = md.Creators[0]
	}

	if len(md.Publishers) > 0 {
		detail.Source = md.Publishers[0]
	}

	detail.DisplayURI = displayURI
	detail.PreviewURI = previewURI

	detail.ArtworkMetadata = md.ArtworkMetadata
}

// FromFxhashObject reads asset detail from an fxhash API object
func (detail *AssetMetadataDetail) FromFxhashObject(o fxhash.ObjectDetail) {
	detail.Name = o.Name
	detail.Description = o.Metadata.Description
	detail.ArtistID = o.Issuer.Author.ID
	detail.ArtistName = o.Issuer.Author.ID
	detail.ArtistURL = fmt.Sprintf("https://www.fxhash.xyz/u/%s", o.Issuer.Author.Name)
	detail.MaxEdition = o.Issuer.Supply
	detail.DisplayURI = ipfsURLToGatewayURL(FxhashGateway, o.Metadata.DisplayURI)
	detail.PreviewURI = ipfsURLToGatewayURL(FxhashGateway, o.Metadata.ArtifactURI)

	var artists []Artist
	artists = append(artists, Artist{
		ArtistID:   o.Issuer.Author.ID,
		ArtistName: o.Issuer.Author.ID,
		ArtistURL:  fmt.Sprintf("https://www.fxhash.xyz/u/%s", o.Issuer.Author.Name),
	})

	for _, v := range o.Issuer.Author.Collaborators {
		artists = append(artists, Artist{
			ArtistID:   v.ID,
			ArtistName: v.Name,
			ArtistURL:  fmt.Sprintf("https://www.fxhash.xyz/u/%s", v.Name),
		})
	}

	detail.Artists = artists
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
		return e.IndexETHToken(owner, contract, tokenID)
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
func (detail *AssetMetadataDetail) FromObjkt(objktToken objkt.Token) {
	detail.UpdateMetadataFromObjkt(objktToken)

	if len(objktToken.Creators) > 0 {
		var artists []Artist
		for _, i := range objktToken.Creators {
			artist := Artist{
				ArtistID:   i.Holder.Address,
				ArtistURL:  getArtistURL(i.Holder),
				ArtistName: i.Holder.Alias,
			}

			if artist.ArtistName == "" && artist.ArtistID != "" {
				artist.ArtistName = artist.ArtistID
			}

			artists = append(artists, artist)
		}

		detail.Artists = artists
		detail.ArtistID = objktToken.Creators[0].Holder.Address
		detail.ArtistURL = getArtistURL(objktToken.Creators[0].Holder)
		detail.ArtistName = objktToken.Creators[0].Holder.Alias

		if detail.ArtistName == "" && detail.ArtistID != "" {
			detail.ArtistName = detail.ArtistID
		}
	}
}

// UpdateMetadataFromObjkt update Objkt metadata to AssetMetadataDetail
func (detail *AssetMetadataDetail) UpdateMetadataFromObjkt(token objkt.Token) {
	detail.Name = token.Name
	detail.Description = token.Description
	detail.MIMEType = token.Mime
	detail.Medium = mediumByMIMEType(token.Mime)

	if token.DisplayURI != "" {
		detail.DisplayURI = detail.ReplaceIPFSURIByObjktCDNURI(ObjktCDNDisplayType, token.DisplayURI, token.FaContract, token.TokenID)
	} else if token.ThumbnailURI == hicetnuncDefaultThumbnailURL {
		detail.DisplayURI = detail.ReplaceIPFSURIByObjktCDNURI(ObjktCDNArtifactThumbnailType, token.ThumbnailURI, token.FaContract, token.TokenID)
	} else if token.ThumbnailURI != "" {
		detail.DisplayURI = detail.ReplaceIPFSURIByObjktCDNURI(ObjktCDNThumbnailType, token.ThumbnailURI, token.FaContract, token.TokenID)
	}

	if detail.DisplayURI == "" || detail.DisplayURI == hicetnuncDefaultThumbnailURL {
		detail.DisplayURI = ipfsURLToGatewayURL(DefaultIPFSGateway, DefaultDisplayURI)
	}

	if token.ArtifactURI != "" {
		detail.PreviewURI = detail.ReplaceIPFSURIByObjktCDNURI(ObjktCDNArtifactType, token.ArtifactURI, token.FaContract, token.TokenID)
	} else {
		detail.PreviewURI = ipfsURLToGatewayURL(DefaultIPFSGateway, DefaultDisplayURI)
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
func (detail *AssetMetadataDetail) ReplaceIPFSURIByObjktCDNURI(assetType, assetURI, contract, tokenID string) string {
	if !strings.HasPrefix(assetURI, "ipfs://") {
		return assetURI
	}

	uri, err := MakeCDNURIFromIPFSURI(assetURI, assetType, contract, tokenID)

	if err == nil {
		return uri
	}

	return ipfsURLToGatewayURL(DefaultIPFSGateway, assetURI)
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
		}

		return "", fmt.Errorf("CDN URL is not exist")
	}

	urlParsed.Path, err = url.JoinPath(ObjktCDNBasePath, cid, urlParsed.Path, assetType)
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

	return ipfsURLToGatewayURL(DefaultIPFSGateway, assetURI), nil
}
