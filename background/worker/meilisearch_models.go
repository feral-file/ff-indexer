package worker

import (
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
)

type MeilisearchTokenDocument struct {
	IndexID           string                 `json:"indexID"`
	TokenID           string                 `json:"tokenID"`
	ContractAddress   string                 `json:"contractAddress"`
	Blockchain        string                 `json:"blockchain"`
	ContractType      string                 `json:"contractType"`
	OwnerAddresses    []string               `json:"ownerAddresses"`
	OwnerBalances     map[string]int64       `json:"ownerBalances"`
	TotalSupply       int64                  `json:"totalSupply"`
	Fungible          bool                   `json:"fungible"`
	Title             string                 `json:"title"`
	Description       string                 `json:"description"`
	ArtistName        string                 `json:"artistName"`
	ArtistID          string                 `json:"artistID"`
	CollectionName    string                 `json:"collectionName,omitempty"`
	Medium            string                 `json:"medium"`
	MimeType          string                 `json:"mimeType"`
	AssetURL          string                 `json:"assetURL"`
	ThumbnailURL      string                 `json:"thumbnailURL"`
	PreviewURL        string                 `json:"previewURL"`
	ExternalURL       string                 `json:"externalURL"`
	Attributes        map[string]interface{} `json:"attributes"`
	Tags              []string               `json:"tags"`
	Source            string                 `json:"source"`
	Edition           int64                  `json:"edition"`
	EditionName       string                 `json:"editionName"`
	MaxEdition        int64                  `json:"maxEdition"`
	MintedAt          time.Time              `json:"mintedAt"`
	LastActivityTime  time.Time              `json:"lastActivityTime"`
	LastRefreshedTime time.Time              `json:"lastRefreshedTime"`
	IndexedAt         time.Time              `json:"indexedAt"`
	Burned            bool                   `json:"burned"`
	Swapped           bool                   `json:"swapped"`
	BaseCurrency      string                 `json:"baseCurrency,omitempty"`
	BasePrice         float64                `json:"basePrice,omitempty"`
	SearchText        string                 `json:"searchText"`
}

type MeilisearchStreamConfig struct {
	Endpoint       string `json:"endpoint"`
	APIKey         string `json:"apiKey"`
	IndexName      string `json:"indexName"`
	BatchSize      int    `json:"batchSize"`
	MaxConcurrency int    `json:"maxConcurrency"`
	RetryAttempts  int    `json:"retryAttempts"`
	RetryDelay     int    `json:"retryDelay"`
	UpdateExisting bool   `json:"updateExisting"`
	DeleteBurned   bool   `json:"deleteBurned"`
}

type MeilisearchStreamRequest struct {
	Addresses           []string                `json:"addresses"`
	Config              MeilisearchStreamConfig `json:"config"`
	IncludeHistory      bool                    `json:"includeHistory"`
	FilterByBlockchains []string                `json:"filterByBlockchains"`
	FilterByContracts   []string                `json:"filterByContracts"`
	LastUpdatedAfter    *time.Time              `json:"lastUpdatedAfter"`
	StartOffset         int64                   `json:"startOffset,omitempty"`
}

type MeilisearchStreamResult struct {
	TotalTokensProcessed int                      `json:"totalTokensProcessed"`
	TotalTokensIndexed   int                      `json:"totalTokensIndexed"`
	TotalTokensSkipped   int                      `json:"totalTokensSkipped"`
	TotalTokensErrored   int                      `json:"totalTokensErrored"`
	ProcessingTime       time.Duration            `json:"processingTime"`
	Errors               []MeilisearchStreamError `json:"errors,omitempty"`
	BatchResults         []MeilisearchBatchResult `json:"batchResults,omitempty"`
}

type MeilisearchStreamError struct {
	TokenIndexID string    `json:"tokenIndexID"`
	Error        string    `json:"error"`
	Timestamp    time.Time `json:"timestamp"`
	Retryable    bool      `json:"retryable"`
}

type MeilisearchBatchResult struct {
	BatchID       string    `json:"batchID"`
	DocumentCount int       `json:"documentCount"`
	Success       bool      `json:"success"`
	TaskUID       int64     `json:"taskUID,omitempty"`
	Error         string    `json:"error,omitempty"`
	ProcessedAt   time.Time `json:"processedAt"`
}

func toMeilisearchDocument(token indexer.DetailedTokenV2) MeilisearchTokenDocument {
	doc := MeilisearchTokenDocument{
		IndexID:           token.IndexID,
		TokenID:           token.ID,
		ContractAddress:   token.ContractAddress,
		Blockchain:        token.Blockchain,
		ContractType:      token.ContractType,
		Fungible:          token.Fungible,
		OwnerAddresses:    []string{token.Owner},
		OwnerBalances:     map[string]int64{token.Owner: token.Balance},
		TotalSupply:       token.Balance,
		Title:             token.Asset.Metadata.Project.Latest.Title,
		Description:       token.Asset.Metadata.Project.Latest.Description,
		ArtistName:        token.Asset.Metadata.Project.Latest.ArtistName,
		ArtistID:          token.Asset.Metadata.Project.Latest.ArtistID,
		Medium:            string(token.Asset.Metadata.Project.Latest.Medium),
		MimeType:          token.Asset.Metadata.Project.Latest.MIMEType,
		AssetURL:          token.Asset.Metadata.Project.Latest.AssetURL,
		ThumbnailURL:      token.Asset.Metadata.Project.Latest.ThumbnailURL,
		PreviewURL:        token.Asset.Metadata.Project.Latest.PreviewURL,
		ExternalURL:       token.Asset.Metadata.Project.Latest.SourceURL,
		Source:            token.Asset.Metadata.Project.Latest.Source,
		Edition:           token.Edition,
		EditionName:       token.EditionName,
		MaxEdition:        token.Asset.Metadata.Project.Latest.MaxEdition,
		MintedAt:          token.MintedAt,
		LastActivityTime:  token.LastActivityTime,
		LastRefreshedTime: token.LastRefreshedTime,
		IndexedAt:         time.Now(),
		Burned:            token.Burned,
		Swapped:           token.Swapped,
		BaseCurrency:      token.Asset.Metadata.Project.Latest.BaseCurrency,
		BasePrice:         token.Asset.Metadata.Project.Latest.BasePrice,
	}
	if token.Fungible && len(token.Owners) > 0 {
		doc.OwnerAddresses = make([]string, 0, len(token.Owners))
		doc.OwnerBalances = make(map[string]int64)
		totalSupply := int64(0)
		for address, balance := range token.Owners {
			doc.OwnerAddresses = append(doc.OwnerAddresses, address)
			doc.OwnerBalances[address] = balance
			totalSupply += balance
		}
		doc.TotalSupply = totalSupply
	}
	if token.Asset.Attributes != nil {
		doc.Attributes = make(map[string]interface{})
	}
	doc.SearchText = buildSearchText(doc)
	return doc
}

func buildSearchText(doc MeilisearchTokenDocument) string {
	parts := []string{doc.Title, doc.Description, doc.ArtistName, doc.CollectionName, doc.Medium}
	parts = append(parts, doc.Tags...)
	var result string
	for _, part := range parts {
		if part == "" {
			continue
		}
		if result != "" {
			result += " "
		}
		result += part
	}
	return result
}
