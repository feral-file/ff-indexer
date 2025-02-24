package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"
	seriesRegistry "github.com/bitmark-inc/feralfile-exhibition-smart-contract/go-binding/series-registry"
	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"go.uber.org/cadence/client"
	"go.uber.org/zap"
)

func (e *EventProcessor) indexSeriesCollection(ctx context.Context, event SeriesEvent) error {
	var data map[string]interface{}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return err
	}

	seriesID := data["series_id"].(string)

	biSeriesID, ok := new(big.Int).SetString(seriesID, 10)
	if !ok {
		return errors.New("invalid series ID")
	}

	sr, err := newSeriesRegistryContract(e.rpcClient)
	if err != nil {
		return err
	}

	metadataURI, err := sr.GetSeriesMetadataURI(nil, biSeriesID)
	if err != nil {
		return err
	}
	bytesMetadata, err := e.FetchIPFSData(metadataURI)
	if err != nil {
		return err
	}
	var metadata indexer.SeriesMetadata
	if err = json.Unmarshal(bytesMetadata, &metadata); err != nil {
		return err
	}

	tokenMapURI, err := sr.GetSeriesContractTokenDataURI(nil, biSeriesID)
	if err != nil {
		return err
	}
	bytesTokenMap, err := e.FetchIPFSData(tokenMapURI)
	if err != nil {
		return err
	}
	var tokenMap indexer.TokenRegistry
	if err = json.Unmarshal(bytesTokenMap, &tokenMap); err != nil {
		return err
	}

	artists, err := sr.GetSeriesArtistAddresses(nil, biSeriesID)
	if err != nil {
		return err
	}

	artistAddresses := make([]string, len(artists))
	for i, a := range artists {
		artistAddresses[i] = a.Hex()
	}

	collection, err := e.indexerStore.GetNewCollectionByID(ctx, seriesCollectionID(seriesID))
	if err != nil {
		return err
	}

	if collection == nil {
		collection = &indexer.NewCollection{
			ID:         seriesCollectionID(seriesID),
			ExternalID: seriesID,
			Source:     "seriesRegistry",
			Published:  true,
			CreatedAt:  event.CreatedAt,
		}
	}

	collection.Creators = artistAddresses
	collection.Name = metadata.Name
	collection.Description = metadata.Description
	collection.ImageURL = metadata.Image
	collection.Contracts = tokenMap.AllContractAddresses()
	collection.ExternalURL = metadata.ExternalURL
	collection.Metadata = metadata.Metadata
	collection.Items = tokenMap.TotalSupply()
	collection.LastUpdatedTime = event.CreatedAt

	if err := e.indexerStore.IndexNewCollection(ctx, *collection); err != nil {
		return err
	}

	// index the collection tokens
	runID := uuid.New().String()
	collectionAssetMap := make(map[string]indexer.NewCollectionAsset)
	var tokenIndexIDs []string
	var workflowRunFutures []client.WorkflowRun
	for contract, tokens := range tokenMap.Ethereum.ERC721 {
		for _, tokenID := range tokens {
			f, err := indexerWorker.ExecuteIndexTokenWorkflow(ctx, e.worker, "", contract, tokenID, false, false)
			if err != nil {
				return err
			}
			workflowRunFutures = append(workflowRunFutures, f)
			indexID := indexer.TokenIndexID(utils.EthereumBlockchain, contract, tokenID)
			tokenIndexIDs = append(tokenIndexIDs, indexID)
			collectionAssetMap[indexID] = indexer.NewCollectionAsset{
				CollectionID: collection.ID,
				TokenIndexID: indexID,
				Edition:      int64(1), // TODO: default to 1
				RunID:        runID,
			}
		}
	}
	for contract, tokens := range tokenMap.Ethereum.ERC1155 {
		for _, tokenID := range tokens {
			f, err := indexerWorker.ExecuteIndexTokenWorkflow(ctx, e.worker, "", contract, tokenID, false, false)
			if err != nil {
				return err
			}
			workflowRunFutures = append(workflowRunFutures, f)
			indexID := indexer.TokenIndexID(utils.EthereumBlockchain, contract, tokenID)
			tokenIndexIDs = append(tokenIndexIDs, indexID)
			collectionAssetMap[indexID] = indexer.NewCollectionAsset{
				CollectionID: collection.ID,
				TokenIndexID: indexID,
				Edition:      int64(1), // TODO: default to 1
				RunID:        runID,
			}
		}
	}
	for contract, tokens := range tokenMap.Tezos.FA2 {
		for _, tokenID := range tokens {
			f, err := indexerWorker.ExecuteIndexTokenWorkflow(ctx, e.worker, "", contract, tokenID, false, false)
			if err != nil {
				return err
			}
			workflowRunFutures = append(workflowRunFutures, f)
			indexID := indexer.TokenIndexID(utils.TezosBlockchain, contract, tokenID)
			tokenIndexIDs = append(tokenIndexIDs, indexID)
			collectionAssetMap[indexID] = indexer.NewCollectionAsset{
				CollectionID: collection.ID,
				TokenIndexID: indexID,
				Edition:      int64(1), // TODO: default to 1
				RunID:        runID,
			}
		}
	}

	// wait until all token indexed
	for _, f := range workflowRunFutures {
		if err := f.Get(ctx, nil); err != nil {
			return err
		}
	}

	// filter if any of the token is burned
	liveTokenIndexIDs, err := e.indexerStore.FilterBurnedIndexIDs(ctx, tokenIndexIDs)
	if err != nil {
		return err
	}
	var collectionAssets []indexer.NewCollectionAsset
	for _, id := range liveTokenIndexIDs {
		collectionAssets = append(collectionAssets, collectionAssetMap[id])
	}

	if err := e.indexerStore.IndexNewCollectionAsset(ctx, collection.ID, collectionAssets); err != nil {
		return err
	}
	if err := e.indexerStore.DeleteDeprecatedNewCollectionAsset(ctx, collection.ID, runID); err != nil {
		return err
	}

	// update the total supply of collection
	collection.Items = len(liveTokenIndexIDs)
	if err := e.indexerStore.IndexNewCollection(ctx, *collection); err != nil {
		return err
	}

	return nil
}

func (e *EventProcessor) deleteSeriesCollection(ctx context.Context, event SeriesEvent) error {
	var data map[string]interface{}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return err
	}

	seriesID := data["series_id"].(string)

	collection, err := e.indexerStore.GetNewCollectionByID(ctx, seriesCollectionID(seriesID))
	if err != nil {
		return err
	}
	if collection == nil {
		return nil
	}

	return e.indexerStore.DeleteNewCollection(ctx, collection.ID)
}

func (e *EventProcessor) artistAddressUpdated(ctx context.Context, event SeriesEvent) error {
	var data map[string]interface{}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return err
	}
	oldAddress := data["old_address"].(string)
	newAddress := data["new_address"].(string)
	return e.indexerStore.UpdateNewCollectionArtistAddress(ctx, oldAddress, newAddress)
}

func (e *EventProcessor) collaboratorConfirmed(ctx context.Context, event SeriesEvent) error {
	var data map[string]interface{}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return err
	}

	seriesID := data["series_id"].(string)

	biSeriesID, ok := new(big.Int).SetString(seriesID, 10)
	if !ok {
		return errors.New("invalid series ID")
	}

	sr, err := newSeriesRegistryContract(e.rpcClient)
	if err != nil {
		return err
	}

	artists, err := sr.GetSeriesArtistAddresses(nil, biSeriesID)
	if err != nil {
		return err
	}

	var addresses []string
	for _, addr := range artists {
		addresses = append(addresses, addr.Hex())
	}
	return e.indexerStore.UpdateCollectionArtists(ctx, seriesCollectionID(seriesID), addresses)
}

func (e *EventProcessor) CreateSeriesCollection(ctx context.Context) {
	e.StartSeriesEventWorker(ctx,
		SeriesEventStageInit, SeriesEventStageHandled,
		[]SeriesEventType{SeriesEventTypeRegistered},
		0, 0, e.indexSeriesCollection,
	)
}

func (e *EventProcessor) UpdateSeriesCollection(ctx context.Context) {
	e.StartSeriesEventWorker(ctx,
		SeriesEventStageInit, SeriesEventStageHandled,
		[]SeriesEventType{SeriesEventTypeUpdated},
		0, 0, e.indexSeriesCollection,
	)
}

func (e *EventProcessor) DeleteSeriesCollection(ctx context.Context) {
	e.StartSeriesEventWorker(ctx,
		SeriesEventStageInit, SeriesEventStageHandled,
		[]SeriesEventType{SeriesEventTypeDeleted},
		0, 0, e.deleteSeriesCollection,
	)
}

func (e *EventProcessor) ArtistAddressUpdated(ctx context.Context) {
	e.StartSeriesEventWorker(ctx,
		SeriesEventStageInit, SeriesEventStageHandled,
		[]SeriesEventType{SeriesEventTypeArtistAddressUpdated},
		0, 0, e.artistAddressUpdated,
	)
}

func (e *EventProcessor) CollaboratorConfirmed(ctx context.Context) {
	e.StartSeriesEventWorker(ctx,
		SeriesEventStageInit, SeriesEventStageHandled,
		[]SeriesEventType{SeriesEventTypeCollaboratorConfirmed},
		0, 0, e.collaboratorConfirmed,
	)
}

func newSeriesRegistryContract(ec *ethclient.Client) (*seriesRegistry.SeriesRegistry, error) {
	return seriesRegistry.NewSeriesRegistry(common.HexToAddress(indexer.SeriesRegistryContract), ec)
}

// FetchIPFSData retrieves content from an IPFS URI using available gateways.
// It attempts each gateway sequentially and returns the first successful response.
func (e *EventProcessor) FetchIPFSData(ipfsURI string) ([]byte, error) {
	var lastErr error
	for _, gateway := range e.ipfsGateways {
		gatewayURL := convertIPFSToGatewayURL(gateway, ipfsURI)
		data, err := downloadIPFSContent(gatewayURL)
		if err == nil {
			log.Debug("Successfully retrieved IPFS content",
				zap.String("gateway", gateway),
				zap.String("url", gatewayURL))
			return data, nil
		}
		lastErr = err
		log.Warn("Failed to retrieve IPFS content from gateway",
			zap.Error(err),
			zap.String("uri", ipfsURI),
			zap.String("gateway", gateway))
	}
	return nil, fmt.Errorf("failed to fetch IPFS data from all gateways; last error: %w", lastErr)
}

// ConvertIPFSToGatewayURL transforms an IPFS URI (e.g., ipfs://CID/path) into an HTTPS URL
// using the specified gateway host. Returns the original URI if parsing fails or scheme isn't "ipfs".
func convertIPFSToGatewayURL(gatewayHost, ipfsURI string) string {
	parsedURL, err := url.Parse(ipfsURI)
	if err != nil || parsedURL.Scheme != "ipfs" {
		return ipfsURI // Return original if invalid or not IPFS
	}

	// Clean the path and construct gateway URL
	cid := parsedURL.Host // CID is typically the host in ipfs://CID/path
	path := strings.TrimLeft(parsedURL.Path, "/")
	gatewayPath := fmt.Sprintf("ipfs/%s", cid)
	if path != "" {
		gatewayPath = fmt.Sprintf("%s/%s", gatewayPath, path)
	}

	return fmt.Sprintf("https://%s/%s", gatewayHost, gatewayPath)
}

// downloadIPFSContent retrieves content from a given URL with a timeout.
func downloadIPFSContent(contentURL string) ([]byte, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Get(contentURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return data, nil
}

func seriesCollectionID(seriesID string) string {
	return fmt.Sprint("seriesRegistry-", seriesID)
}
