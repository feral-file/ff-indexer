package indexer

import (
	"encoding/json"
	"time"

	utils "github.com/bitmark-inc/autonomy-utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Medium string

const (
	MediumUnknown  = "unknown"
	MediumVideo    = "video"
	MediumImage    = "image"
	MediumSoftware = "software"
	MediumOther    = "other"
)

// BlockchainAddress is a type of blockchain addresses supported in indexer
type BlockchainAddress string

func (a BlockchainAddress) String() string {
	return string(a)
}

func (a *BlockchainAddress) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	if s == "" {
		return nil
	}

	switch utils.GetBlockchainByAddress(s) {
	case utils.EthereumBlockchain:
		s = EthereumChecksumAddress(s)
	case utils.UnknownBlockchain:
		return ErrUnsupportedBlockchain
	}

	*a = BlockchainAddress(s)

	return nil
}

type Provenance struct {
	// this field is only for ownership validating
	FormerOwner *string `json:"formerOwner,omitempty" bson:"-"`

	Type        string    `json:"type" bson:"type"`
	Owner       string    `json:"owner" bson:"owner"`
	Blockchain  string    `json:"blockchain" bson:"blockchain"`
	BlockNumber *uint64   `json:"blockNumber,omitempty" bson:"blockNumber,omitempty"` // TODO: make it non-nullable, just temporarily support the compatibility
	Timestamp   time.Time `json:"timestamp" bson:"timestamp"`
	TxID        string    `json:"txid" bson:"txid"`
	TxURL       string    `json:"txURL" bson:"txURL"`
}

type BaseTokenInfo struct {
	ID              string `json:"id" bson:"id"`
	Blockchain      string `json:"blockchain" bson:"blockchain"`
	Fungible        bool   `json:"fungible" bson:"fungible"`
	ContractType    string `json:"contractType" bson:"contractType"`
	ContractAddress string `json:"contractAddress,omitempty" bson:"contractAddress"`
}

// Token is a structure for token information
type Token struct {
	BaseTokenInfo   `bson:",inline"` // the latest token info
	Edition         int64            `json:"edition" bson:"edition"`
	EditionName     string           `json:"editionName" bson:"editionName"`
	MintedAt        time.Time        `json:"mintedAt" bson:"mintedAt"`
	Balance         int64            `json:"balance" bson:"-"` // a temporarily state of balance for a specific owner
	Owner           string           `json:"owner" bson:"owner"`
	Owners          map[string]int64 `json:"owners" bson:"owners"`
	OwnersArray     []string         `json:"-" bson:"ownersArray"`
	AssetID         string           `json:"-" bson:"assetID"`
	OriginTokenInfo []BaseTokenInfo  `json:"originTokenInfo" bson:"originTokenInfo"`
	IsDemo          bool             `json:"-" bson:"isDemo"`

	IndexID           string       `json:"indexID" bson:"indexID"`
	Source            string       `json:"source" bson:"source"`
	Swapped           bool         `json:"swapped" bson:"swapped"`
	SwappedFrom       *string      `json:"-" bson:"swappedFrom,omitempty"`
	SwappedTo         *string      `json:"-" bson:"swappedTo,omitempty"`
	Burned            bool         `json:"burned" bson:"burned"`
	Provenances       []Provenance `json:"provenance" bson:"provenance"`
	LastActivityTime  time.Time    `json:"lastActivityTime" bson:"lastActivityTime"`
	LastRefreshedTime time.Time    `json:"lastRefreshedTime" bson:"lastRefreshedTime"`
}

type AssetAttributes struct {
	Scrollable bool `json:"scrollable" bson:"scrollable"`
}

type ProjectMetadata struct {
	// Common attributes
	ArtistID            string   `json:"artistID" structs:"artistID" bson:"artistID"`       // Artist blockchain address
	ArtistName          string   `json:"artistName" structs:"artistName" bson:"artistName"` // <creator.user.username>,
	ArtistURL           string   `json:"artistURL" structs:"artistURL" bson:"artistURL"`    // <OpenseaAPI/creator.address>,
	Artists             []Artist `json:"artists" structs:"artists" bson:"artists"`
	AssetID             string   `json:"assetID" structs:"assetID" bson:"assetID"`                                     // <asset_contract.address>,
	Title               string   `json:"title" structs:"title" bson:"title"`                                           // <name>,
	Description         string   `json:"description" structs:"description" bson:"description"`                         // <description>,
	MIMEType            string   `json:"mimeType" structs:"mimeType" bson:"mimeType"`                                  // <mime_type from file extension or metadata>,
	Medium              Medium   `json:"medium" structs:"medium" bson:"medium"`                                        // <"image" if image_url is present; "other" if animation_url is present> ,
	MaxEdition          int64    `json:"maxEdition" structs:"maxEdition" bson:"maxEdition"`                            // 0,
	BaseCurrency        string   `json:"baseCurrency,omitempty" structs:"baseCurrency" bson:"baseCurrency"`            // null,
	BasePrice           float64  `json:"basePrice,omitempty" structs:"basePrice" bson:"basePrice"`                     // null,
	Source              string   `json:"source" structs:"source" bson:"source"`                                        // <Opeasea/Artblock>,
	SourceURL           string   `json:"sourceURL" structs:"sourceURL" bson:"sourceURL"`                               // <linktoSourceWebsite>,
	PreviewURL          string   `json:"previewURL" structs:"previewURL" bson:"previewURL"`                            // <image_url or animation_url>,
	ThumbnailURL        string   `json:"thumbnailURL" structs:"thumbnailURL" bson:"thumbnailURL"`                      // <image_thumbnail_url>,
	GalleryThumbnailURL string   `json:"galleryThumbnailURL" structs:"galleryThumbnailURL" bson:"galleryThumbnailURL"` // <image_thumbnail_url>,
	AssetData           string   `json:"assetData" structs:"assetData" bson:"assetData"`                               // null,
	AssetURL            string   `json:"assetURL" structs:"assetURL" bson:"assetURL"`                                  // <permalink>

	// autonomy customized attributes
	Attributes *AssetAttributes `json:"attributes,omitempty" structs:"attributes,omitempty" bson:"attributes,omitempty"`

	// artwork metadata from source. currently on for Feral File
	ArtworkMetadata map[string]interface{} `json:"artworkMetadata" structs:"artworkMetadata" bson:"artworkMetadata"`

	// Operation attributes
	LastUpdatedAt time.Time `json:"lastUpdatedAt" structs:"lastUpdatedAt" bson:"lastUpdatedAt"`

	// Feral File attributes
	InitialSaleModel string `json:"initialSaleModel" structs:"initialSaleModel" bson:"initialSaleModel"` // airdrop|fix-price|highest-bid-auction|group-auction

	// Deprecated attributes
	OriginalFileURL string `json:"originalFileURL" structs:"-" bson:"-"`
}

type Artist struct {
	ID   string `json:"id" structs:"id" bson:"id"`       // Artist blockchain address
	Name string `json:"name" structs:"name" bson:"name"` // <creator.user.username>,
	URL  string `json:"url" structs:"url" bson:"url"`    // <OpenseaAPI/creator.address>,
}

// CollectionUpdates is the inputs payload of IndexCollection
type CollectionUpdates struct {
	ID          string      `json:"id"`
	ExternalID  string      `json:"externalID"`
	Blockchain  string      `json:"blockchain"`
	Owner       string      `json:"owner"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	ImageURL    string      `json:"imageURL"`
	Contract    string      `json:"contract"`
	Metadata    interface{} `json:"metadata"`
	Published   bool        `json:"published"`
	Source      string      `json:"source"`
	SourceURL   string      `json:"source_url"`
}

// AssetUpdates is the inputs payload of IndexAsset. It includes project metadata, blockchain metadata and
// tokens that is attached to it
type AssetUpdates struct {
	ID                 string          `json:"id"`
	Source             string          `json:"source"`
	ProjectMetadata    ProjectMetadata `json:"projectMetadata"`
	BlockchainMetadata interface{}     `json:"blockchainMetadata"`
	Tokens             []Token         `json:"tokens"`
}

// SwapUpdate is the inputs payload for swap a token
type SwapUpdate struct {
	OriginalTokenID         string          `json:"originalTokenID"`
	OriginalBlockchain      string          `json:"originalBlockchain"`
	OriginalContractAddress string          `json:"originalContractAddress"`
	NewTokenID              string          `json:"newTokenID"`
	NewBlockchain           string          `json:"newBlockchain"`
	NewContractAddress      string          `json:"newContractAddress"`
	NewContractType         string          `json:"newContractType"`
	ProjectMetadata         ProjectMetadata `json:"projectMetadata"`
	BlockchainMetadata      interface{}     `json:"blockchainMetadata"`
}

type TokenFeedbackUpdate struct {
	IndexID  string `json:"indexID"`
	MimeType string `json:"mimeType"`
}

// VersionedProjectMetadata is a structure that manages different versions of project metadata.
// Currently, it maintains two version: the original one and the latest one.
type VersionedProjectMetadata struct {
	Origin ProjectMetadata `json:"origin" structs:"origin" bson:"origin"`
	Latest ProjectMetadata `json:"latest" structs:"latest" bson:"latest"`
}

// DetailedToken is the summarized information of a token. It includes asset information
// that this token is linked to.
type DetailedToken struct {
	Token           `bson:",inline"`
	ThumbnailID     string                   `json:"thumbnailID"`
	IPFSPinned      bool                     `json:"ipfsPinned"`
	Attributes      *AssetAttributes         `json:"attributes,omitempty"`
	ProjectMetadata VersionedProjectMetadata `json:"projectMetadata" bson:"projectMetadata"`
}

type DetailedTokenV2 struct {
	Token      `bson:",inline"`
	IPFSPinned bool    `json:"ipfsPinned"`
	Asset      AssetV2 `json:"asset" bson:"asset"`
}

type AssetV2 struct {
	IndexID                   string           `json:"indexID" bson:"indexID"`
	ThumbnailID               string           `json:"thumbnailID" bson:"thumbnailID"`
	LastRefreshedTime         time.Time        `json:"lastRefreshedTime" bson:"lastRefreshedTime"`
	Attributes                *AssetAttributes `json:"attributes" bson:"attributes,omitempty"`
	Metadata                  AssetMetadata    `json:"metadata" bson:"metadata"`
	StaticPreviewURLLandscape *string          `json:"staticPreviewURLLandscape" bson:"staticPreviewURLLandscape"`
	StaticPreviewURLPortrait  *string          `json:"staticPreviewURLPortrait" bson:"staticPreviewURLPortrait"`
}

type AssetMetadata struct {
	Project VersionedProjectMetadata `json:"project" bson:"project"`
}

type AbsentMIMETypeToken struct {
	IndexID    string `json:"indexID"`
	PreviewURL string `json:"previewURL"`
}

type TokenFeedback struct {
	IndexID         string    `json:"indexID" bson:"indexID"`
	MimeType        string    `json:"mimeType" bson:"mimeType"`
	LastUpdatedTime time.Time `json:"lastUpdatedTime" bson:"lastUpdatedTime"`
	DID             string    `json:"did" bson:"did"`
}

type GrouppedTokenFeedback struct {
	IndexID   string              `bson:"_id" json:"indexID,omitempty"`
	MimeTypes []MimeTypeWithCount `bson:"mimeTypes" json:"mimeTypes"`
}

type MimeTypeWithCount struct {
	MimeType string `bson:"mimeType" json:"mimeType"`
	Count    int    `bson:"count" json:"count"`
}

type Account struct {
	Account          string    `json:"account" bson:"account"`
	Blockchain       string    `json:"blockchain" bson:"blockchain"`
	LastUpdatedTime  time.Time `json:"lastUpdateTime" bson:"lastUpdateTime"`
	LastActivityTime time.Time `json:"lastActivityTime" bson:"lastActivityTime"`
}

type AccountToken struct {
	BaseTokenInfo     `bson:",inline"` // the latest token info
	IndexID           string           `json:"indexID" bson:"indexID"`
	OwnerAccount      string           `json:"ownerAccount" bson:"ownerAccount"`
	Balance           int64            `json:"balance" bson:"balance"`
	LastActivityTime  time.Time        `json:"lastActivityTime" bson:"lastActivityTime"`
	LastRefreshedTime time.Time        `json:"lastRefreshedTime" bson:"lastRefreshedTime"`
	LastPendingTime   []time.Time      `json:"lastPendingTime" bson:"lastPendingTime,omitempty"`
	PendingTxs        []string         `json:"pendingTxs" bson:"pendingTxs,omitempty"`
}

type TotalBalance struct {
	ID    string `bson:"_id"`
	Total int    `bson:"total"`
}

type Collection struct {
	ID           string   `json:"id" bson:"id"`
	ExternalID   string   `json:"externalID" bson:"externalID"`
	Creator      string   `json:"creator" bson:"creator"`
	Name         string   `json:"name" bson:"name"`
	Description  string   `json:"description" bson:"description"`
	Items        int      `json:"items" bson:"items"`
	ImageURL     string   `json:"imageURL" bson:"imageURL"`
	Blockchain   string   `json:"blockchain" bson:"blockchain"`
	Contracts    []string `json:"contracts" bson:"contracts"`
	Published    bool     `json:"published" bson:"published"`
	Source       string   `json:"source" bson:"source"`
	SourceURL    string   `json:"sourceURL" bson:"sourceURL"`
	ProjectURL   string   `json:"projectURL" bson:"projectURL"`
	ThumbnailURL string   `json:"thumbnailURL" bson:"thumbnailURL"`

	LastUpdatedTime  time.Time `json:"lastUpdatedTime" bson:"lastUpdatedTime"`
	LastActivityTime time.Time `json:"lastActivityTime" bson:"lastActivityTime"`
	CreatedAt        time.Time `json:"createdAt" bson:"createdAt"`
}

type CollectionAsset struct {
	CollectionID     string    `json:"collectionID" bson:"collectionID"`
	TokenIndexID     string    `json:"tokenIndexID" bson:"tokenIndexID"`
	Edition          int64     `json:"edition" bson:"edition"`
	LastActivityTime time.Time `json:"lastActivityTime" bson:"lastActivityTime"`

	RunID string `json:"-" bson:"runID"`
}

type NewCollection struct {
	ID          string                 `json:"id" bson:"id"`
	ExternalID  string                 `json:"externalID" bson:"externalID"`
	Creators    []string               `json:"creators" bson:"creators"`
	Name        string                 `json:"name" bson:"name"`
	Description string                 `json:"description" bson:"description"`
	Items       int                    `json:"items" bson:"items"`
	ImageURL    string                 `json:"imageURL" bson:"imageURL"`
	Contracts   ContractAddresses      `json:"contracts" bson:"contracts"`
	Published   bool                   `json:"published" bson:"published"`
	Source      string                 `json:"source" bson:"source"`
	ExternalURL string                 `json:"externalURL" bson:"externalURL"`
	Metadata    map[string]interface{} `json:"metadata" bson:"metadata"`

	LastUpdatedTime time.Time `json:"lastUpdatedTime" bson:"lastUpdatedTime"`
	CreatedAt       time.Time `json:"createdAt" bson:"createdAt"`
}

type NewCollectionAsset struct {
	CollectionID string `json:"collectionID" bson:"collectionID"`
	TokenIndexID string `json:"tokenIndexID" bson:"tokenIndexID"`
	Edition      int64  `json:"edition" bson:"edition"`

	RunID string `json:"-" bson:"runID"`
}

type GenericSalesTimeSeries struct {
	Timestamp string                 `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
	Values    map[string]string      `json:"values"`
	Shares    map[string]string      `json:"shares"`
}

type SaleTimeSeries struct {
	Timestamp     time.Time              `json:"timestamp" bson:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata" bson:"metadata"`
	Shares        map[string]interface{} `json:"shares" bson:"shares"`
	NetValue      primitive.Decimal128   `json:"netRevenue" bson:"netRevenue"`
	PaymentAmount primitive.Decimal128   `json:"paymentAmount" bson:"paymentAmount"`
	PlatformFee   primitive.Decimal128   `json:"platformFee" bson:"platformFee"`
	USDQuote      primitive.Decimal128   `json:"exchangeRate" bson:"exchangeRate"`
	Price         primitive.Decimal128   `json:"price" bson:"price"`
}

type ExchangeRate struct {
	Timestamp    time.Time `json:"timestamp" bson:"timestamp"`
	Price        float64   `json:"price" bson:"price"`
	CurrencyPair string    `json:"currencyPair" bson:"currencyPair"`
}

type SeriesMetadata struct {
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Image           string                 `json:"image"`
	ExternalURL     string                 `json:"external_url"`
	Metadata        map[string]interface{} `json:"metadata"`
	MetadataVersion string                 `json:"metadata_version"`
}

// ContractTokens maps contract addresses to their token IDs
type ContractTokens map[string][]string

// EthereumContracts holds Ethereum-specific contract types
type EthereumContracts struct {
	ERC721  ContractTokens `json:"erc721"`
	ERC1155 ContractTokens `json:"erc1155"`
}

// TezosContracts holds Tezos-specific contract types
type TezosContracts struct {
	FA2 ContractTokens `json:"fa2"`
}

// TokenRegistry aggregates contracts across different blockchains
type TokenRegistry struct {
	Ethereum EthereumContracts `json:"ethereum"`
	Tezos    TezosContracts    `json:"tezos"`
}

// ContractAddresses returns all contract addresses from ContractTokens
func (ct ContractTokens) ContractAddresses() []string {
	addresses := make([]string, 0, len(ct))
	for addr := range ct {
		addresses = append(addresses, addr)
	}
	return addresses
}

// ContractAddressMap returns a mapped representation of Ethereum contracts
func (ec EthereumContracts) ContractAddressMap() EthereumContractAddresses {
	return EthereumContractAddresses{
		ERC721:  ec.ERC721.ContractAddresses(),
		ERC1155: ec.ERC1155.ContractAddresses(),
	}
}

// ContractAddressMap returns a mapped representation of Tezos contracts
func (tc TezosContracts) ContractAddressMap() TezosContractAddresses {
	return TezosContractAddresses{
		FA2: tc.FA2.ContractAddresses(),
	}
}

// AllContractAddresses returns a nested map of all contract addresses by blockchain
func (tr TokenRegistry) AllContractAddresses() ContractAddresses {
	return ContractAddresses{
		Ethereum: tr.Ethereum.ContractAddressMap(),
		Tezos:    tr.Tezos.ContractAddressMap(),
	}
}

func (tr TokenRegistry) TotalSupply() int {
	var totalSupply int
	for _, tl := range tr.Ethereum.ERC721 {
		totalSupply += len(tl)
	}
	for _, tl := range tr.Ethereum.ERC1155 {
		totalSupply += len(tl)
	}
	for _, tl := range tr.Tezos.FA2 {
		totalSupply += len(tl)
	}
	return totalSupply
}

type EthereumContractAddresses struct {
	ERC721  []string `json:"erc721"`
	ERC1155 []string `json:"erc1155"`
}

type TezosContractAddresses struct {
	FA2 []string `json:"fa2"`
}

type ContractAddresses struct {
	Ethereum EthereumContractAddresses `json:"ethereum"`
	Tezos    TezosContractAddresses    `json:"tezos"`
}
