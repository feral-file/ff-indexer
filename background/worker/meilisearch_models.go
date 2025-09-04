package worker

import (
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
)

type MeilisearchTokenDocument struct {
	IndexID           string                  `json:"indexID"`
	Blockchain        string                  `json:"blockchain"`
	ContractType      string                  `json:"contractType"`
	Fungible          bool                    `json:"fungible"`
	Title             string                  `json:"title"`
	Description       string                  `json:"description"`
	ArtistName        string                  `json:"artistName"`
	CollectionName    string                  `json:"collectionName,omitempty"`
	Medium            string                  `json:"medium"`
	MimeType          string                  `json:"mimeType"`
	Source            string                  `json:"source"`
	MintedAt          time.Time               `json:"mintedAt"`
	LastActivityTime  time.Time               `json:"lastActivityTime"`
	LastRefreshedTime time.Time               `json:"lastRefreshedTime"`
	FullToken         indexer.DetailedTokenV2 `json:"fullToken"`
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
		Blockchain:        token.Blockchain,
		ContractType:      token.ContractType,
		Fungible:          token.Fungible,
		Title:             token.Asset.Metadata.Project.Latest.Title,
		Description:       token.Asset.Metadata.Project.Latest.Description,
		ArtistName:        token.Asset.Metadata.Project.Latest.ArtistName,
		Medium:            string(token.Asset.Metadata.Project.Latest.Medium),
		MimeType:          token.Asset.Metadata.Project.Latest.MIMEType,
		Source:            token.Asset.Metadata.Project.Latest.Source,
		Edition:           token.Edition,
		EditionName:       token.EditionName,
		MintedAt:          token.MintedAt,
		LastActivityTime:  token.LastActivityTime,
		LastRefreshedTime: token.LastRefreshedTime,
		IndexedAt:         time.Now(),
		FullToken:         token,
	}
	return doc
}
