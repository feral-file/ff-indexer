package indexer

import (
	"time"
)

// Token is a structure for token information
type Token struct {
	ID              string    `json:"id" bson:"id"`
	Edition         int64     `json:"edition" bson:"edition"`
	Blockchain      string    `json:"blockchain" bson:"blockchain"`
	MintAt          time.Time `json:"mintedAt" bson:"mintedAt"`
	ContractAddress string    `json:"contractAddress,omitempty" bson:"contractAddress"`
	ContractType    string    `json:"contractType" bson:"contractType"`
	Owner           string    `json:"owner" bson:"owner"`
	AssetID         string    `json:"-" bson:"assetID"`
}

type ProjectMetadata struct {
	ArtistName          string `json:"artistName" bson:"artistName"`                   // <creator.user.username>,
	ArtistURL           string `json:"artistURL" bson:"artistURL"`                     // <OpenseaAPI/creator.address>,
	AssetID             string `json:"assetID" bson:"assetID"`                         // <asset_contract.address>,
	Title               string `json:"title" bson:"title"`                             // <name>,
	Description         string `json:"description" bson:"description"`                 // <description>,
	Medium              string `json:"medium" bson:"medium"`                           // <"image" if image_url is present; "other" if animation_url is present> ,
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

	// Deprecated attributes
	ArtistID        string `json:"artistID" bson:"-"`
	OriginalFileURL string `json:"originalFileURL" bson:"-"`
	FirstMintedAt   string `json:"firstMintedAt" bson:"-"`
}

// AssetUpdates is the inputs payload of IndexAsset. It includes project metadata, blockchain metadata and
// tokens that is attached to it
type AssetUpdates struct {
	ID                 string          `json:"id"`
	ProjectMetadata    ProjectMetadata `json:"projectMetadata"`
	BlockchainMetadata interface{}     `json:"blockchainMetadata"`
	Tokens             []Token         `json:"tokens"`
}

// VersionedProjectMetadata is a structure that manages different versions of project metadata.
// Currently, it maintains two version: the original one and the latest one.
type VersionedProjectMetadata struct {
	Origin ProjectMetadata `json:"origin" bson:"origin"`
	Latest ProjectMetadata `json:"latest" bson:"latest"`
}

// TokenInfo is the summarized information of a token. It includes asset information
// that this token is linked to.
type TokenInfo struct {
	Token           `bson:",inline"`
	ProjectMetadata VersionedProjectMetadata `json:"projectMetadata" bson:"projectMetadata"`
}
