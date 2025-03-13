package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
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

func (e *EventProcessor) indexCollection(ctx context.Context, event SeriesRegistryEvent) error {
	// Unmarshal event data
	var data map[string]interface{}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return err
	}

	// Get series ID
	seriesID, ok := data["series_id"].(string)
	if !ok {
		return errors.New("series_id cannot be found in event data or is not a string")
	}

	// Convert series ID to big int
	seriesIDInt, ok := new(big.Int).SetString(seriesID, 10)
	if !ok {
		return errors.New("cannot convert series_id to big int")
	}

	// Initialize contract instance
	contract, err := e.newSeriesRegistryContract(e.rpcClient)
	if err != nil {
		return err
	}

	// Get metadata URI
	metadataURI, err := contract.GetSeriesMetadataURI(nil, seriesIDInt)
	if err != nil {
		return err
	}

	// Read metadata
	metadataBytes, err := e.ReadDataURI(metadataURI)
	if err != nil {
		return err
	}
	var metadata indexer.SeriesMetadata
	if err = json.Unmarshal(metadataBytes, &metadata); err != nil {
		return err
	}

	// Get token data URI
	tokenDataURI, err := contract.GetSeriesTokenDataURI(nil, seriesIDInt)
	if err != nil {
		return err
	}

	// Read token data
	tokenDataBytes, err := e.ReadDataURI(tokenDataURI)
	if err != nil {
		return err
	}
	var tokenData indexer.TokenRegistry
	if err = json.Unmarshal(tokenDataBytes, &tokenData); err != nil {
		return err
	}

	// Get artists
	artists, err := contract.GetSeriesArtistAddresses(nil, seriesIDInt)
	if err != nil {
		return err
	}

	artistAddresses := make([]string, len(artists))
	for i, a := range artists {
		artistAddresses[i] = a.Hex()
	}

	// Get collection
	collectionID := collectionID(seriesID)
	collection, err := e.indexerStore.GetCollectionByID(ctx, collectionID)
	if err != nil {
		return err
	}

	if collection == nil {
		collection = &indexer.Collection{
			ID:         collectionID,
			ExternalID: seriesID,
			Source:     "SeriesRegistry",
			Published:  true,
			CreatedAt:  event.CreatedAt,
		}
	}

	collection.Creators = artistAddresses
	collection.Name = metadata.Name
	collection.Description = metadata.Description
	collection.ImageURL = metadata.Image
	collection.Contracts = tokenData.AllContractAddresses()
	collection.ExternalURL = metadata.ExternalURL
	collection.Metadata = metadata.Metadata
	collection.Items = tokenData.TotalSupply()
	collection.LastUpdatedTime = event.CreatedAt

	if err := e.indexerStore.IndexCollection(ctx, *collection); err != nil {
		return err
	}

	// Index the collection tokens
	runID := uuid.New().String()
	collectionAssetMap := make(map[string]indexer.CollectionAsset)
	var tokenIndexIDs []string
	var workflowFutures []client.WorkflowRun
	for contract, tokens := range tokenData.Ethereum.ERC721 {
		for _, tokenID := range tokens {
			future, err := indexerWorker.ExecuteIndexTokenWorkflow(ctx, e.worker, "", contract, tokenID, false, false)
			if err != nil {
				return err
			}
			workflowFutures = append(workflowFutures, future)

			indexID := indexer.TokenIndexID(utils.EthereumBlockchain, contract, tokenID)
			tokenIndexIDs = append(tokenIndexIDs, indexID)
			collectionAssetMap[indexID] = indexer.CollectionAsset{
				CollectionID: collection.ID,
				TokenIndexID: indexID,
				RunID:        runID,
			}
		}
	}
	for contract, tokens := range tokenData.Ethereum.ERC1155 {
		for _, tokenID := range tokens {
			future, err := indexerWorker.ExecuteIndexTokenWorkflow(ctx, e.worker, "", contract, tokenID, false, false)
			if err != nil {
				return err
			}
			workflowFutures = append(workflowFutures, future)

			indexID := indexer.TokenIndexID(utils.EthereumBlockchain, contract, tokenID)
			tokenIndexIDs = append(tokenIndexIDs, indexID)
			collectionAssetMap[indexID] = indexer.CollectionAsset{
				CollectionID: collection.ID,
				TokenIndexID: indexID,
				RunID:        runID,
			}
		}
	}
	for contract, tokens := range tokenData.Tezos.FA2 {
		for _, tokenID := range tokens {
			future, err := indexerWorker.ExecuteIndexTokenWorkflow(ctx, e.worker, "", contract, tokenID, false, false)
			if err != nil {
				return err
			}
			workflowFutures = append(workflowFutures, future)

			indexID := indexer.TokenIndexID(utils.TezosBlockchain, contract, tokenID)
			tokenIndexIDs = append(tokenIndexIDs, indexID)
			collectionAssetMap[indexID] = indexer.CollectionAsset{
				CollectionID: collection.ID,
				TokenIndexID: indexID,
				RunID:        runID,
			}
		}
	}

	// Wait until all token indexed
	for _, f := range workflowFutures {
		if err := f.Get(ctx, nil); err != nil {
			return err
		}
	}

	// Filter if any of the token is burned
	liveTokenIndexIDs, err := e.indexerStore.FilterBurnedIndexIDs(ctx, tokenIndexIDs)
	if err != nil {
		return err
	}

	var collectionAssets []indexer.CollectionAsset
	for _, id := range liveTokenIndexIDs {
		collectionAssets = append(collectionAssets, collectionAssetMap[id])
	}

	// Index the collection assets
	if err := e.indexerStore.IndexCollectionAsset(ctx, collection.ID, collectionAssets); err != nil {
		return err
	}

	// Delete the deprecated collection assets
	if err := e.indexerStore.DeleteDeprecatedCollectionAsset(ctx, collection.ID, runID); err != nil {
		return err
	}

	// Update the total supply of collection
	collection.Items = len(liveTokenIndexIDs)
	if err := e.indexerStore.IndexCollection(ctx, *collection); err != nil {
		return err
	}

	return nil
}

func (e *EventProcessor) deleteCollection(ctx context.Context, event SeriesRegistryEvent) error {
	// Unmarshal event data
	var data map[string]interface{}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return err
	}

	// Get series ID
	seriesID, ok := data["series_id"].(string)
	if !ok {
		return errors.New("series_id cannot be found in event data or is not a string")
	}

	// Get collection
	collectionID := collectionID(seriesID)
	collection, err := e.indexerStore.GetCollectionByID(ctx, collectionID)
	if err != nil {
		return err
	}

	if collection == nil {
		return nil
	}

	return e.indexerStore.DeleteCollection(ctx, collection.ID)
}

func (e *EventProcessor) replaceCollectionCreator(ctx context.Context, event SeriesRegistryEvent) error {
	// Unmarshal event data
	var data map[string]interface{}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return err
	}

	// Get old creator address
	oldAddress, ok := data["old_address"].(string)
	if !ok {
		return errors.New("old_address cannot be found in event data or is not a string")
	}

	// Get new creator address
	newAddress, ok := data["new_address"].(string)
	if !ok {
		return errors.New("new_address cannot be found in event data or is not a string")
	}

	// Update the collection creators
	return e.indexerStore.ReplaceCollectionCreator(ctx, oldAddress, newAddress)
}

func (e *EventProcessor) updateCollectionCreators(ctx context.Context, event SeriesRegistryEvent) error {
	// Unmarshal event data
	var data map[string]interface{}
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return err
	}

	// Get series ID
	seriesID, ok := data["series_id"].(string)
	if !ok {
		return errors.New("series_id cannot be found in event data or is not a string")
	}

	// Convert series ID to big int
	seriesIDInt, ok := new(big.Int).SetString(seriesID, 10)
	if !ok {
		return errors.New("cannot convert series_id to big int")
	}

	// Initialize contract instance
	contract, err := e.newSeriesRegistryContract(e.rpcClient)
	if err != nil {
		return err
	}

	// Get artists
	artists, err := contract.GetSeriesArtistAddresses(nil, seriesIDInt)
	if err != nil {
		return err
	}

	// Convert to hex addresses
	var addresses []string
	for _, addr := range artists {
		addresses = append(addresses, addr.Hex())
	}

	// Update the collection artists
	collectionID := collectionID(seriesID)
	return e.indexerStore.UpdateCollectionCreators(ctx, collectionID, addresses)
}

func (e *EventProcessor) IndexCollection(ctx context.Context) {
	e.StartSeriesRegistryEventWorker(ctx,
		SeriesEventStageInit, SeriesEventStageDone,
		[]SeriesRegistryEventType{
			SeriesRegistryEventTypeRegisterSeries,
			SeriesRegistryEventTypeUpdateSeries},
		0, 0, e.indexCollection,
	)
}

func (e *EventProcessor) DeleteCollection(ctx context.Context) {
	e.StartSeriesRegistryEventWorker(ctx,
		SeriesEventStageInit, SeriesEventStageDone,
		[]SeriesRegistryEventType{SeriesRegistryEventTypeDeleteSeries},
		0, 0, e.deleteCollection,
	)
}

func (e *EventProcessor) ReplaceCollectionCreator(ctx context.Context) {
	e.StartSeriesRegistryEventWorker(ctx,
		SeriesEventStageInit, SeriesEventStageDone,
		[]SeriesRegistryEventType{
			SeriesRegistryEventTypeUpdateArtistAddress,
			SeriesRegistryEventTypeAssignSeries,
		},
		0, 0, e.replaceCollectionCreator,
	)
}

func (e *EventProcessor) UpdateCollectionCreators(ctx context.Context) {
	e.StartSeriesRegistryEventWorker(ctx,
		SeriesEventStageInit, SeriesEventStageDone,
		[]SeriesRegistryEventType{
			SeriesRegistryEventTypeOptInCollaboration,
			SeriesRegistryEventTypeOptOutSeries,
		},
		0, 0, e.updateCollectionCreators,
	)
}

func (e *EventProcessor) newSeriesRegistryContract(ec *ethclient.Client) (*seriesRegistry.SeriesRegistry, error) {
	return seriesRegistry.NewSeriesRegistry(common.HexToAddress(e.seriesRegistryContract), ec)
}

// ReadDataURI reads the data from the given URI
func (e *EventProcessor) ReadDataURI(uri string) ([]byte, error) {
	if !validDataURI(uri) {
		return nil, errors.New("invalid data URI")
	}

	const timeout = 30 * time.Second
	if indexer.IsHTTPSURI(uri) {
		return indexer.ReadFromURL(uri, timeout)
	} else {
		// If no gateways are configured, return error
		if len(e.ipfsGateways) == 0 {
			return nil, errors.New("no IPFS gateways configured")
		}

		// try to read from the IPFS gateways
		var lastErr error
		for _, gateway := range e.ipfsGateways {
			gatewayURL := indexer.ResolveIPFSURI(gateway, uri)
			data, err := indexer.ReadFromURL(gatewayURL, timeout)
			if err == nil {
				log.Debug("Successfully read data from IPFS gateway",
					zap.String("gateway", gateway),
					zap.String("url", gatewayURL))
				return data, nil
			}

			lastErr = err
			log.Warn("Failed to read data from IPFS gateway",
				zap.Error(err),
				zap.String("uri", uri),
				zap.String("gateway", gateway))
		}

		return nil, fmt.Errorf("failed to read data from all %d IPFS gateways; last error: %w", len(e.ipfsGateways), lastErr)
	}
}

func collectionID(seriesID string) string {
	return fmt.Sprint("series-registry-", seriesID)
}

func validDataURI(uri string) bool {
	return indexer.IsIPFSURI(uri) || indexer.IsHTTPSURI(uri)
}
