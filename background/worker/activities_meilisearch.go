package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	indexer "github.com/bitmark-inc/nft-indexer"
)

// getMeilisearchClient creates a Meilisearch client from configuration
func getMeilisearchClient() meilisearch.ServiceManager {
	endpoint := viper.GetString("meilisearch.endpoint")
	apiKey := viper.GetString("meilisearch.api_key")

	if endpoint == "" {
		endpoint = "http://localhost:7700" // Default endpoint
	}

	if apiKey == "" {
		// Create client without API key for local development
		return meilisearch.New(endpoint)
	}

	return meilisearch.New(endpoint, meilisearch.WithAPIKey(apiKey))
}

// getMeilisearchIndexName gets the index name from configuration
func getMeilisearchIndexName() string {
	indexName := viper.GetString("meilisearch.index_name")
	if indexName == "" {
		indexName = "nft-tokens" // Default index name
	}
	return indexName
}

// getMeilisearchIndexKeyPrefix gets the index key prefix from configuration
func getMeilisearchIndexKeyPrefix() string {
	return viper.GetString("meilisearch.index_key_prefix")
}

// CreateOrUpdateMeilisearchIndex creates or updates a Meilisearch index with settings
func (w *NFTIndexerWorker) CreateOrUpdateMeilisearchIndex(ctx context.Context) error {
	client := getMeilisearchClient()
	indexName := getMeilisearchIndexName()

	// Ensure index exists with primary key
	_, _ = client.CreateIndex(&meilisearch.IndexConfig{Uid: indexName, PrimaryKey: "indexID"})
	index := client.Index(indexName)

	// Configure index settings for optimal token search
	// Only include human-readable, meaningful content for search
	// Technical identifiers (tokenID, contractAddress, ownerAddresses, mimeType) excluded
	searchableAttrs := []string{
		"title",
		"description",
		"artistName",
		"collectionName",
		"tags",
		"medium",
	}
	filterableAttrs := []string{
		"blockchain",
		"contractType",
		"contractAddress",
		"ownerAddresses",
		"fungible",
		"burned",
		"swapped",
		"medium",
		"mimeType",
		"source",
		"artistID",
		"mintedAt",
		"lastActivityTime",
	}
	sortableAttrs := []string{
		"mintedAt",
		"lastActivityTime",
		"lastRefreshedTime",
		"indexedAt",
		"edition",
		"basePrice",
	}
	settings := &meilisearch.Settings{
		SearchableAttributes: searchableAttrs,
		FilterableAttributes: filterableAttrs,
		SortableAttributes:   sortableAttrs,
	}

	// Update settings
	task, err := index.UpdateSettings(settings)
	if err != nil {
		return fmt.Errorf("failed to update index settings: %w", err)
	}

	log.InfoWithContext(ctx, "Meilisearch index created/updated successfully",
		zap.String("indexName", indexName),
		zap.Int64("taskUID", task.TaskUID))

	return nil
}

// BatchIndexTokensToMeilisearch indexes a batch of tokens to Meilisearch
func (w *NFTIndexerWorker) BatchIndexTokensToMeilisearch(ctx context.Context, tokens []indexer.DetailedTokenV2, deleteBurned bool) (*MeilisearchBatchResult, error) {
	if len(tokens) == 0 {
		return &MeilisearchBatchResult{
			BatchID:       fmt.Sprintf("empty-batch-%d", time.Now().UnixNano()),
			DocumentCount: 0,
			Success:       true,
			ProcessedAt:   time.Now(),
		}, nil
	}

	client := getMeilisearchClient()
	indexName := getMeilisearchIndexName()
	index := client.Index(indexName)

	// Convert tokens to Meilisearch documents
	documents := make([]map[string]interface{}, 0, len(tokens))
	for _, token := range tokens {
		// Always skip tokens that are marked burned
		if token.Burned {
			continue
		}
		// Skip tokens owned by burn address (non-fungible tokens)
		if !token.Fungible && indexer.IsBurnAddress(token.Owner, w.Environment) {
			continue
		}
		// Skip fungible tokens if the only owner is a burn address
		if token.Fungible && len(token.Owners) == 1 {
			var onlyOwner string
			for addr := range token.Owners {
				onlyOwner = addr
				break
			}
			if indexer.IsBurnAddress(onlyOwner, w.Environment) {
				continue
			}
		}

		doc := toMeilisearchDocument(token)

		// Convert to map[string]interface{} for Meilisearch SDK
		prefix := getMeilisearchIndexKeyPrefix()
		indexID := doc.IndexID
		if prefix != "" {
			indexID = prefix + ":" + indexID
		}
		docMap := map[string]interface{}{
			"indexID":           indexID,
			"tokenID":           doc.TokenID,
			"contractAddress":   doc.ContractAddress,
			"blockchain":        doc.Blockchain,
			"contractType":      doc.ContractType,
			"ownerAddresses":    doc.OwnerAddresses,
			"ownerBalances":     doc.OwnerBalances,
			"totalSupply":       doc.TotalSupply,
			"fungible":          doc.Fungible,
			"title":             doc.Title,
			"description":       doc.Description,
			"artistName":        doc.ArtistName,
			"artistID":          doc.ArtistID,
			"collectionName":    doc.CollectionName,
			"medium":            doc.Medium,
			"mimeType":          doc.MimeType,
			"assetURL":          doc.AssetURL,
			"thumbnailURL":      doc.ThumbnailURL,
			"previewURL":        doc.PreviewURL,
			"externalURL":       doc.ExternalURL,
			"attributes":        doc.Attributes,
			"tags":              doc.Tags,
			"source":            doc.Source,
			"edition":           doc.Edition,
			"editionName":       doc.EditionName,
			"maxEdition":        doc.MaxEdition,
			"mintedAt":          doc.MintedAt,
			"lastActivityTime":  doc.LastActivityTime,
			"lastRefreshedTime": doc.LastRefreshedTime,
			"indexedAt":         doc.IndexedAt,
			"burned":            doc.Burned,
			"swapped":           doc.Swapped,
			"baseCurrency":      doc.BaseCurrency,
			"basePrice":         doc.BasePrice,
			"searchText":        doc.SearchText,
		}

		documents = append(documents, docMap)
	}

	if len(documents) == 0 {
		return &MeilisearchBatchResult{
			BatchID:       fmt.Sprintf("filtered-batch-%d", time.Now().UnixNano()),
			DocumentCount: 0,
			Success:       true,
			ProcessedAt:   time.Now(),
		}, nil
	}

	// Index documents to Meilisearch using the SDK
	batchID := fmt.Sprintf("batch-%d", time.Now().UnixNano())
	primaryKey := "indexID"
	task, err := index.AddDocuments(documents, &primaryKey) // Use indexID as primary key
	if err != nil {
		return &MeilisearchBatchResult{
			BatchID:       batchID,
			DocumentCount: len(documents),
			Success:       false,
			Error:         err.Error(),
			ProcessedAt:   time.Now(),
		}, err
	}

	log.InfoWithContext(ctx, "Batch indexed to Meilisearch successfully",
		zap.String("batchID", batchID),
		zap.Int("documentCount", len(documents)),
		zap.Int64("taskUID", task.TaskUID))

	return &MeilisearchBatchResult{
		BatchID:       batchID,
		DocumentCount: len(documents),
		Success:       true,
		TaskUID:       task.TaskUID,
		ProcessedAt:   time.Now(),
	}, nil
}

// DeleteTokenFromMeilisearch deletes a token from Meilisearch by indexID
func (w *NFTIndexerWorker) DeleteTokenFromMeilisearch(ctx context.Context, indexID string) error {
	client := getMeilisearchClient()
	indexName := getMeilisearchIndexName()
	index := client.Index(indexName)

	// Apply index key prefix for deletion
	prefix := getMeilisearchIndexKeyPrefix()
	if prefix != "" {
		indexID = prefix + ":" + indexID
	}

	task, err := index.DeleteDocument(indexID)
	if err != nil {
		return fmt.Errorf("failed to delete token from Meilisearch: %w", err)
	}

	log.InfoWithContext(ctx, "Token deleted from Meilisearch",
		zap.String("indexID", indexID),
		zap.Int64("taskUID", task.TaskUID))

	return nil
}

// GetTokensForAddresses retrieves tokens for given addresses using existing store methods
func (w *NFTIndexerWorker) GetTokensForAddresses(ctx context.Context, addresses []string, lastUpdatedAfter *time.Time, offset, size int64) ([]indexer.DetailedTokenV2, error) {
	var lastUpdated time.Time
	if lastUpdatedAfter != nil {
		lastUpdated = *lastUpdatedAfter
	}

	tokens, err := w.indexerStore.GetDetailedAccountTokensByOwners(
		ctx,
		addresses,
		indexer.FilterParameter{}, // No additional filters
		lastUpdated,
		"lastRefreshedTime", // Sort by last refreshed time
		offset,
		size,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get tokens for addresses: %w", err)
	}

	return tokens, nil
}

// CountTokensForAddresses counts total tokens for given addresses
func (w *NFTIndexerWorker) CountTokensForAddresses(ctx context.Context, addresses []string) (int64, error) {
	var totalCount int64

	for _, address := range addresses {
		count, err := w.indexerStore.CountDetailedAccountTokensByOwner(ctx, address)
		if err != nil {
			return 0, fmt.Errorf("failed to count tokens for address %s: %w", address, err)
		}
		totalCount += count
	}

	return totalCount, nil
}

// DeleteBurnedTokensFromMeilisearch removes burned tokens from Meilisearch
func (w *NFTIndexerWorker) DeleteBurnedTokensFromMeilisearch(ctx context.Context, indexIDs []string) (*MeilisearchBatchResult, error) {
	if len(indexIDs) == 0 {
		return &MeilisearchBatchResult{
			BatchID:       fmt.Sprintf("empty-delete-batch-%d", time.Now().UnixNano()),
			DocumentCount: 0,
			Success:       true,
			ProcessedAt:   time.Now(),
		}, nil
	}

	client := getMeilisearchClient()
	indexName := getMeilisearchIndexName()
	index := client.Index(indexName)

	batchID := fmt.Sprintf("delete-batch-%d", time.Now().UnixNano())

	// Apply index key prefix for each ID and delete documents using the SDK
	prefix := getMeilisearchIndexKeyPrefix()
	prefixed := make([]string, 0, len(indexIDs))
	for _, id := range indexIDs {
		if prefix != "" {
			prefixed = append(prefixed, prefix+":"+id)
		} else {
			prefixed = append(prefixed, id)
		}
	}
	task, err := index.DeleteDocuments(prefixed)
	if err != nil {
		return &MeilisearchBatchResult{
			BatchID:       batchID,
			DocumentCount: len(indexIDs),
			Success:       false,
			Error:         err.Error(),
			ProcessedAt:   time.Now(),
		}, err
	}

	log.InfoWithContext(ctx, "Burned tokens deleted from Meilisearch",
		zap.String("batchID", batchID),
		zap.Int("documentCount", len(indexIDs)),
		zap.Int64("taskUID", task.TaskUID))

	return &MeilisearchBatchResult{
		BatchID:       batchID,
		DocumentCount: len(indexIDs),
		Success:       true,
		TaskUID:       task.TaskUID,
		ProcessedAt:   time.Now(),
	}, nil
}

// WaitForMeilisearchTask waits for a Meilisearch task to complete
func (w *NFTIndexerWorker) WaitForMeilisearchTask(ctx context.Context, taskUID int64) error {
	client := getMeilisearchClient()

	// Wait for task completion with timeout
	task, err := client.WaitForTask(taskUID, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to wait for task %d: %w", taskUID, err)
	}

	if task.Status == "failed" {
		return fmt.Errorf("meilisearch task %d failed: %s", taskUID, task.Error.Message)
	}

	log.InfoWithContext(ctx, "Meilisearch task completed",
		zap.Int64("taskUID", taskUID),
		zap.String("status", string(task.Status)))

	return nil
}

// UpdateTokenOwnershipInMeilisearch updates ownership-related fields of a token document in Meilisearch
// - Non-fungible token: set single owner in ownerAddresses and balance 1 in ownerBalances
// - Fungible token: set ownerAddresses and ownerBalances based on current on-chain/store state
func (w *NFTIndexerWorker) UpdateTokenOwnershipInMeilisearch(ctx context.Context, indexID string) error {
	client := getMeilisearchClient()
	indexName := getMeilisearchIndexName()
	index := client.Index(indexName)

	// Fetch latest token data from store
	tokens, err := w.indexerStore.GetDetailedTokensV2(
		ctx,
		indexer.FilterParameter{IDs: []string{indexID}},
		0,
		1,
	)
	if err != nil {
		return fmt.Errorf("failed to get token %s for ownership update: %w", indexID, err)
	}
	if len(tokens) == 0 {
		return fmt.Errorf("token %s not found for ownership update", indexID)
	}

	token := tokens[0]

	// If token is burned or owned by burn address, remove from Meilisearch
	if token.Burned || (!token.Fungible && indexer.IsBurnAddress(token.Owner, w.Environment)) {
		if err := w.DeleteTokenFromMeilisearch(ctx, indexID); err != nil {
			return err
		}
		return nil
	}

	// Build partial document with only ownership fields
	ownerAddresses := []string{}
	ownerBalances := map[string]int64{}
	totalSupply := int64(0)

	if token.Fungible {
		// Use full owners map when available
		if len(token.Owners) > 0 {
			for addr, bal := range token.Owners {
				ownerAddresses = append(ownerAddresses, addr)
				ownerBalances[addr] = bal
				totalSupply += bal
			}
		} else {
			// Fallback to single owner/balance from token fields
			ownerAddresses = []string{token.Owner}
			ownerBalances[token.Owner] = token.Balance
			totalSupply = token.Balance
		}
	} else {
		// Non-fungible
		ownerAddresses = []string{token.Owner}
		ownerBalances[token.Owner] = 1
		totalSupply = 1
	}

	// Apply index key prefix for updates
	prefix := getMeilisearchIndexKeyPrefix()
	prefixedIndexID := indexID
	if prefix != "" {
		prefixedIndexID = prefix + ":" + indexID
	}

	partial := map[string]interface{}{
		"indexID":        prefixedIndexID,
		"ownerAddresses": ownerAddresses,
		"ownerBalances":  ownerBalances,
		"totalSupply":    totalSupply,
	}

	// Update document in Meilisearch
	primaryKey := "indexID"
	task, err := index.UpdateDocuments([]map[string]interface{}{partial}, &primaryKey)
	if err != nil {
		return fmt.Errorf("failed to update ownership in Meilisearch for %s: %w", indexID, err)
	}

	log.InfoWithContext(ctx, "Updated token ownership in Meilisearch",
		zap.String("indexID", indexID),
		zap.Int64("taskUID", task.TaskUID),
		zap.Int("ownerCount", len(ownerAddresses)))

	return nil
}
