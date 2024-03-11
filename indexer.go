package indexer

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/structs"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	"github.com/bitmark-inc/tzkt-go"
)

const (
	// broken-image.svg
	DefaultDisplayURI    = "ipfs://QmX5rRzkZQfvEyaYc1Q78YZ83pFj3AgpFVSK8SmxUmZ85M"
	DefaultIPFSGateway   = "ipfs.nftstorage.link"
	FxhashGateway        = "gateway.fxhash.xyz"
	FxhashDevIPFSGateway = "gateway.fxhash-dev2.xyz"
	FxhashOnchfsGateway  = "onchfs.fxhash2.xyz"
)

var inhouseMinter = map[string]struct{}{
	"tz1d6EdHCR6YSpW1dNcbF9BqG1SaY1nCxrLx": {},
	"tz1hQbuRax3op9knY3YDxqNnqxzcmoxmv1qa": {},
}

// artblocksContracts indexes the addresses which are ERC721 contracts of Artblocks
var artblocksContracts = map[string]struct{}{
	"0x059EDD72Cd353dF5106D2B9cC5ab83a52287aC3a": {},
	"0xa7d8d9ef8D8Ce8992Df33D8b8CF4Aebabd5bD270": {},
}

var (
	sourceArtBlocks   = "Art Blocks"
	sourceCrayonCodes = "Crayon Codes"
	sourceFxHash      = "fxhash"
)

var (
	ErrTXNotFound            = fmt.Errorf("transaction is not found")
	ErrUnsupportedBlockchain = fmt.Errorf("unsupported blockchain")
)

// getTokenSourceByContract token source name by inspecting a contract address
func getTokenSourceByContract(contractAddress string) string {
	switch utils.GetBlockchainByAddress(contractAddress) {
	case utils.EthereumBlockchain:
		if _, ok := artblocksContracts[contractAddress]; ok {
			return sourceArtBlocks

		} else if contractAddress == "0x70e6b3f9d99432fCF35274d6b24D83Ef5Ba3dE2D" {
			return sourceCrayonCodes
		}

		return SourceOpensea
	case utils.TezosBlockchain:
		// WIP
		return ""
	default:
		return ""
	}
}

// getTokenSourceByPreviewURL returns the token source name by inspecting the preview url
func getTokenSourceByPreviewURL(url string) string {
	if artblocksMatched, _ := regexp.MatchString(`generator.artblocks.io`, url); artblocksMatched {
		return sourceArtBlocks
	}

	if fxhashMatched, _ := regexp.MatchString(`fxhash\d+\.xyz`, url); fxhashMatched {
		return sourceFxHash
	}

	return ""
}

// getTokenSourceByMetadataURL returns the token source name by inspecting the metadata url
func getTokenSourceByMetadataURL(url string) string {
	if fxhashMatched, _ := regexp.MatchString(`media.fxhash.xyz`, url); fxhashMatched {
		return sourceFxHash
	}

	return ""
}

// OptimizedOpenseaImageURL get the filename by inspect opensea's cdn link.
func OptimizedOpenseaImageURL(imageURL string) (string, error) {
	u, err := url.Parse(imageURL)
	if err != nil {
		return imageURL, err
	}

	if u.Host == "i.seadn.io" {
		u.RawQuery = url.Values{
			"auto": []string{"format"},
			"dpr":  []string{"1"},
			"w":    []string{"3840"},
		}.Encode()
	}

	return u.String(), nil
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
		// if sheme is onchfs fallback to fxhash onchfs gateway
		if u.Scheme == "onchfs" {
			return onchfsURLToGatewayURL(FxhashOnchfsGateway, ipfsURL)
		}

		// not a valid URL
		return ipfsURL
	}

	// remove the leading "/" from the path
	p := strings.TrimLeft(u.Path, "/")

	u.Path = fmt.Sprintf("ipfs/%s/%s", u.Host, p)
	u.Host = gateway
	u.Scheme = "https"

	return u.String()
}

// ipfsURLToGatewayURL converts an onchfs link to a HTTP link by a given onchfs gateway.
// If a link is failed to parse, it returns the original link
func onchfsURLToGatewayURL(gateway, onchfsURL string) string {
	u, err := url.Parse(onchfsURL)
	if err != nil {
		return onchfsURL
	}

	if u.Scheme != "onchfs" {
		// not a valid URL
		return onchfsURL
	}

	// remove the leading "/" from the path
	p := strings.TrimLeft(u.Path, "/")

	u.Path = fmt.Sprintf("%s/%s", u.Host, p)
	u.Host = gateway
	u.Scheme = "https"

	return u.String()
}

func OptimizeFxHashIPFSURL(url string) string {
	if strings.Contains(url, "https://ipfs.io/ipfs/") {
		return strings.ReplaceAll(url, "ipfs.io", FxhashGateway)
	}

	return url
}

func IsIPFSLink(url string) bool {
	// case: ipfs://QmbbrSdifmQDLyh8ARNkAoC4Sc9mDQaMTwgHSXv5a2FJHd
	if strings.HasPrefix(url, "ipfs://") {
		return true
	}

	// case: https://gateway.ipfs.io/ipfs/QmbbrSdifmQDLyh8ARNkAoC4Sc9mDQaMTwgHSXv5a2FJHd
	if strings.Contains(url, "/ipfs/") {
		return true
	}

	return false
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
	Minter      string

	ArtistID   string
	ArtistName string
	ArtistURL  string
	MaxEdition int64

	ThumbnailURI string
	DisplayURI   string
	PreviewURI   string

	IsBooleanAmount bool

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
	detail.Minter = md.Minter

	var optimizedFileSize = tzkt.FlexInt64(0)
	var optimizedDisplayURI string

	for _, format := range md.Formats {
		if strings.Contains(string(format.MIMEType), "image") && format.FileSize > optimizedFileSize {
			optimizedDisplayURI = format.URI
			optimizedFileSize = format.FileSize
		}
	}

	var thumbnailURI, displayURI, previewURI string

	if optimizedDisplayURI != "" {
		displayURI = optimizedDisplayURI
	} else if md.DisplayURI != "" {
		displayURI = md.DisplayURI
	} else if md.ThumbnailURI != "" {
		displayURI = md.ThumbnailURI
	} else {
		displayURI = DefaultDisplayURI
	}

	// set default thumbnailURI to md.ThumbnailURI and fallback to displayURI if thumbnailURI is empty
	if thumbnailURI = md.ThumbnailURI; thumbnailURI == "" {
		thumbnailURI = displayURI
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
				ID:   v,
				Name: v,
			})
		}

		detail.Artists = artists
		detail.ArtistID = md.Creators[0]
		detail.ArtistName = md.Creators[0]
	}

	if len(md.Publishers) > 0 {
		detail.Source = md.Publishers[0]
	}

	detail.ThumbnailURI = thumbnailURI
	detail.DisplayURI = displayURI
	detail.PreviewURI = previewURI

	detail.IsBooleanAmount = bool(md.IsBooleanAmount)
	detail.ArtworkMetadata = md.ArtworkMetadata
}

func (detail *AssetMetadataDetail) FromOpenseaAsset(a *opensea.DetailedAssetV2, source string) {
	var sourceURL string
	var artistURL string
	artistID := EthereumChecksumAddress(a.Creator)

	switch source {
	case sourceArtBlocks:
		sourceURL = "https://www.artblocks.io/"
		artistURL = fmt.Sprintf("%s/%s", sourceURL, a.Creator)
	case sourceCrayonCodes:
		sourceURL = "https://openprocessing.org/crayon/"
		artistURL = fmt.Sprintf("https://opensea.io/%s", a.Creator)
	case sourceFxHash:
		sourceURL = "https://www.fxhash.xyz/"
		artistURL = fmt.Sprintf("https://www.fxhash.xyz/u/%s", a.Creator)
	default:
		if viper.GetString("network") == "testnet" {
			sourceURL = "https://testnets.opensea.io"
		} else {
			sourceURL = "https://opensea.io"
		}
		artistURL = fmt.Sprintf("https://opensea.io/%s", a.Creator)
	}

	log.Debug("source debug", zap.String("source", source), zap.String("contract", a.Contract), zap.String("id", a.Identifier))

	// Opensea GET assets API just provide a creator, not multiple creator
	artists := []Artist{
		{
			ID:   artistID,
			Name: artistID,
			URL:  artistURL,
		},
	}

	detail.ArtistID = artists[0].ID
	detail.ArtistName = artists[0].Name
	detail.ArtistURL = artists[0].URL
	detail.Artists = artists

	assetURL := a.OpenseaURL
	animationURL := a.AnimationURL

	imageURL, err := OptimizedOpenseaImageURL(a.ImageURL)
	if err != nil {
		log.Warn("invalid opensea image url", zap.String("imageURL", a.ImageURL))
	}

	// fallback to project origin image url
	if imageURL == "" {
		imageURL = a.ImageURL
	}

	if source == sourceFxHash {
		assetURL = fmt.Sprintf("https://www.fxhash.xyz/gentk/%s-%s", a.Contract, a.Identifier)
		imageURL = OptimizeFxHashIPFSURL(imageURL)
		animationURL = OptimizeFxHashIPFSURL(animationURL)
	}

	detail.Name = a.Name
	detail.Description = a.Description
	detail.MIMEType = GetMIMETypeByURL(imageURL)
	detail.Medium = MediumUnknown
	detail.ThumbnailURI = imageURL
	detail.DisplayURI = imageURL
	detail.PreviewURI = imageURL

	detail.AssetURL = assetURL
	detail.Source = source
	detail.SourceURL = sourceURL

	if animationURL != "" {
		detail.PreviewURI = animationURL
		detail.MIMEType = GetMIMETypeByURL(animationURL)

		if source == sourceArtBlocks {
			detail.Medium = MediumSoftware
		} else {
			medium := mediumByPreviewFileExtension(detail.PreviewURI)
			log.Debug("fallback medium check", zap.String("previewURL", detail.PreviewURI), zap.Any("medium", medium))
			detail.Medium = medium
		}
	} else if imageURL != "" {
		detail.Medium = MediumImage
	}
}

// FromFxhashObject reads asset detail from an fxhash API object
func (detail *AssetMetadataDetail) FromFxhashObject(o fxhash.ObjectDetail) {
	var artists []Artist

	for _, v := range o.Issuer.Author.Collaborators {
		artists = append(artists, Artist{
			ID:   v.ID,
			Name: v.Name,
			URL:  fmt.Sprintf("https://www.fxhash.xyz/u/%s", v.Name),
		})
	}

	// in case just have one artist
	if len(artists) == 0 {
		artists = append(artists, Artist{
			ID:   o.Issuer.Author.ID,
			Name: o.Issuer.Author.Name,
			URL:  fmt.Sprintf("https://www.fxhash.xyz/u/%s", o.Issuer.Author.Name),
		})
	}

	detail.ArtistID = artists[0].ID
	detail.ArtistName = artists[0].Name
	detail.ArtistURL = artists[0].URL

	detail.Name = o.Name
	detail.Description = o.Metadata.Description
	detail.MaxEdition = o.Issuer.Supply
	detail.ThumbnailURI = ipfsURLToGatewayURL(FxhashGateway, o.Metadata.ThumbnailURI)
	detail.DisplayURI = ipfsURLToGatewayURL(FxhashGateway, o.Metadata.DisplayURI)
	detail.PreviewURI = ipfsURLToGatewayURL(FxhashGateway, o.Metadata.ArtifactURI)
	detail.Artists = artists
}

// TokenDetail saves token specific detail from different sources
type TokenDetail struct {
	Edition  int64
	Fungible bool
	MintedAt time.Time
}

// GetTokenBalanceOfOwner returns the balance of a token for an owner
func (e *IndexEngine) GetTokenBalanceOfOwner(c context.Context, contract, tokenID, owner string) (int64, error) {
	switch utils.GetBlockchainByAddress(contract) {
	case utils.EthereumBlockchain:
		return e.getEthereumTokenBalanceOfOwner(c, contract, tokenID, owner)
	case utils.TezosBlockchain:
		return e.getTezosTokenBalanceOfOwner(c, contract, tokenID, owner)
	default:
		return 0, ErrUnsupportedBlockchain
	}
}

// IndexToken indexes tokens by detecting the blockchain of a contract. This function returns
// an `AssetUpdates“ object for using in `IndexAsset“ function
func (e *IndexEngine) IndexToken(c context.Context, contract, tokenID string) (*AssetUpdates, error) {
	switch utils.GetBlockchainByAddress(contract) {
	case utils.EthereumBlockchain:
		return e.IndexETHToken(c, contract, tokenID)
	case utils.TezosBlockchain:
		return e.IndexTezosToken(c, contract, tokenID)
	default:
		return nil, ErrUnsupportedBlockchain
	}
}

func (e *IndexEngine) GetTransactionDetailsByPendingTx(pendingTx string) ([]tzkt.DetailedTransaction, error) {
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
				ID:   i.Holder.Address,
				URL:  getArtistURL(i.Holder),
				Name: i.Holder.Alias,
			}

			if artist.Name == "" && artist.ID != "" {
				artist.Name = artist.ID
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

	detail.ThumbnailURI = detail.DisplayURI

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

// GetEditionNumberByName inferences the edition number by the format #{numbers}
func (e *IndexEngine) GetEditionNumberByName(name string) int64 {

	r := regexp.MustCompile(`.*#(\d+)$`)

	matchArray := r.FindStringSubmatch(name)

	if len(matchArray) < 2 {
		return 0
	}
	// matchArray[0] is the whole string.
	n, err := strconv.ParseInt(matchArray[1], 10, 0)
	if err != nil {
		return 0
	}

	return n
}

// GetTxTimestamp returns transaction timestamp of a blockchain
func (e *IndexEngine) GetTxTimestamp(ctx context.Context, blockchain, txHash string) (time.Time, error) {
	switch blockchain {
	case utils.TezosBlockchain:
		return e.GetTezosTxTimestamp(ctx, txHash)
	case utils.EthereumBlockchain:
		return e.GetEthereumTxTimestamp(ctx, txHash)
	}

	return time.Time{}, ErrUnsupportedBlockchain
}

// indexTokenFromFXHASH indexes token metadata by a given fxhash objkt id.
// A fxhash objkt id is a new format from fxhash which is unified id but varied by contracts
func (e *IndexEngine) indexTokenFromFXHASH(ctx context.Context, fxhashObjectID string,
	metadataDetail *AssetMetadataDetail, tokenDetail *TokenDetail) {

	metadataDetail.SetMarketplace(
		MarketplaceProfile{
			"fxhash",
			"https://www.fxhash.xyz",
			fmt.Sprintf("https://www.fxhash.xyz/gentk/%s", fxhashObjectID),
		},
	)
	metadataDetail.SetMedium(MediumSoftware)

	if detail, err := e.fxhash.GetObjectDetail(ctx, fxhashObjectID); err != nil {
		log.Error("fail to get token detail from fxhash", zap.Error(err), log.SourceFXHASH)
	} else {
		if !strings.Contains(detail.Metadata.ThumbnailURI, FxhashWaitingToBeSignedCID) {
			metadataDetail.FromFxhashObject(detail)
		} else {
			log.Warn("ignore fxhash waiting to be sign metadata index")
		}
		tokenDetail.MintedAt = detail.CreatedAt
		tokenDetail.Edition = detail.Iteration
	}
}
