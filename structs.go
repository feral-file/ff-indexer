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
	Owner           string    `json:"owner" bson:"owner"`

	AssetID string `json:"-" bson:"assetID"`
}

// AssetUpdates is the inputs payload of IndexAsset. It includes project metadata, blockchain metadata and
// tokens that is attached to it
type AssetUpdates struct {
	ID                 string      `json:"id"`
	ProjectMetadata    interface{} `json:"projectMetadata"`
	BlockchainMetadata interface{} `json:"blockchainMetadata"`
	Tokens             []Token     `json:"tokens"`
}

// VersionedProjectMetadata is a structure that manages different versions of project metadata.
// Currently, it maintains two version: the original one and the latest one.
type VersionedProjectMetadata struct {
	Origin map[string]interface{} `json:"origin"`
	Latest map[string]interface{} `json:"latest"`
}

// TokenInfo is the summarized information of a token. It includes asset information
// that this token is linked to.
type TokenInfo struct {
	Token           `bson:",inline"`
	ProjectMetadata VersionedProjectMetadata `json:"projectMetadata"`
}
