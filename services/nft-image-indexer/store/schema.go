package store

import "time"

type ImageMetadata struct {
	AssetID string `json:"assetID" gorm:"index:image_asset_id,unique"`
	ImageID string `json:"fileID"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p ImageMetadata) TableName() string {
	return "image_metadata"
}
