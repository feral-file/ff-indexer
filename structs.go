package indexer

import (
	"time"
)

type Medium string

const (
	MediumUnknown  = "unknown"
	MediumVideo    = "video"
	MediumImage    = "image"
	MediumSoftware = "software"
	MediumOther    = "other"
)

type Provenance struct {
	// this field is only for ownership validating
	FormerOwner *string `json:"formerOwner,omitempty" bson:"-"`

	Type       string    `json:"type" bson:"type"`
	Owner      string    `json:"owner" bson:"owner"`
	Blockchain string    `json:"blockchain" bson:"blockchain"`
	Timestamp  time.Time `json:"timestamp" bson:"timestamp"`
	TxID       string    `json:"txid" bson:"txid"`
	TxURL      string    `json:"txURL" bson:"txURL"`
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
	MintAt          time.Time        `json:"mintedAt" bson:"mintedAt"`
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
	LastRefreshedTime time.Time    `json:"-" bson:"lastRefreshedTime"`
}

type ProjectMetadata struct {
	// Common attributes
	ArtistID            string `json:"artistID" bson:"artistID"`                       // Artist blockchain address
	ArtistName          string `json:"artistName" bson:"artistName"`                   // <creator.user.username>,
	ArtistURL           string `json:"artistURL" bson:"artistURL"`                     // <OpenseaAPI/creator.address>,
	AssetID             string `json:"assetID" bson:"assetID"`                         // <asset_contract.address>,
	Title               string `json:"title" bson:"title"`                             // <name>,
	Description         string `json:"description" bson:"description"`                 // <description>,
	MIMEType            string `json:"mimeType" bson:"mimeType"`                       // <mime_type from file extension or metadata>,
	Medium              Medium `json:"medium" bson:"medium"`                           // <"image" if image_url is present; "other" if animation_url is present> ,
	MaxEdition          int64  `json:"maxEdition" bson:"maxEdition"`                   // 0,
	BaseCurrency        string `json:"baseCurrency,omitempty" bson:"baseCurrency"`     // null,
	BasePrice           int64  `json:"basePrice,omitempty" bson:"basePrice"`           // null,
	Source              string `json:"source" bson:"source"`                           // <Opeasea/Artblock>,
	SourceURL           string `json:"sourceURL" bson:"sourceURL"`                     // <linktoSourceWebsite>,
	PreviewURL          string `json:"previewURL" bson:"previewURL"`                   // <image_url or animation_url>,
	ThumbnailURL        string `json:"thumbnailURL" bson:"thumbnailURL"`               // <image_thumbnail_url>,
	GalleryThumbnailURL string `json:"galleryThumbnailURL" bson:"galleryThumbnailURL"` // <image_thumbnail_url>,
	AssetData           string `json:"assetData" bson:"assetData"`                     // null,
	AssetURL            string `json:"assetURL" bson:"assetURL"`                       // <permalink>

	// Operation attributes
	LastUpdatedAt time.Time `json:"lastUpdatedAt" bson:"lastUpdatedAt"`

	// Feral File attributes
	InitialSaleModel string `json:"initialSaleModel" bson:"initialSaleModel"` // airdrop|fix-price|highest-bid-auction|group-auction

	// Deprecated attributes
	OriginalFileURL string `json:"originalFileURL" bson:"-"`
	FirstMintedAt   string `json:"firstMintedAt" bson:"-"`
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

// VersionedProjectMetadata is a structure that manages different versions of project metadata.
// Currently, it maintains two version: the original one and the latest one.
type VersionedProjectMetadata struct {
	Origin ProjectMetadata `json:"origin" bson:"origin"`
	Latest ProjectMetadata `json:"latest" bson:"latest"`
}

// DetailedToken is the summarized information of a token. It includes asset information
// that this token is linked to.
type DetailedToken struct {
	Token           `bson:",inline"`
	ThumbnailID     string                   `json:"thumbnailID"`
	IPFSPinned      bool                     `json:"ipfsPinned"`
	ProjectMetadata VersionedProjectMetadata `json:"projectMetadata" bson:"projectMetadata"`
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
	LastPendingTime   []time.Time      `json:"lastPendingTime" bson:"lastPendingTime"`
	PendingTx         []string         `json:"pendingTx" bson:"pendingTx"`
}
