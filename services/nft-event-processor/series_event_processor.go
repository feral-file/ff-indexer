package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"regexp"

	utils "github.com/bitmark-inc/autonomy-utils"
	seriesRegistry "github.com/bitmark-inc/feralfile-exhibition-smart-contract/go-binding/series-registry"
	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/spf13/viper"
	"go.uber.org/cadence/client"
)

func (e *EventProcessor) indexSeriesCollection(ctx context.Context, event SeriesEvent) error {
	var data map[string]interface{}
	if err := json.Unmarshal(event.Data.RawMessage, &data); err != nil {
		return err
	}

	seriesID, ok := new(big.Int).SetString(data["series_id"].(string), 10)
	if !ok {
		return errors.New("invalid series ID")
	}

	sr, err := newSeriesRegistryContract(e.rpcClient)
	if err != nil {
		return err
	}

	metadataURI, err := sr.GetSeriesMetadataURI(nil, seriesID)
	if err != nil {
		return err
	}
	bytesMetadata, err := ipfsCat(ipfsURIToCID(metadataURI))
	if err != nil {
		return err
	}
	var metadata indexer.SeriesMetadata
	if err = json.Unmarshal(bytesMetadata, &metadata); err != nil {
		return err
	}

	tokenMapURI, err := sr.GetSeriesContractTokenDataURI(nil, seriesID)
	if err != nil {
		return err
	}
	bytesTokenMap, err := ipfsCat(ipfsURIToCID(tokenMapURI))
	if err != nil {
		return err
	}
	var tokenMap indexer.TokenRegistry
	if err = json.Unmarshal(bytesTokenMap, &tokenMap); err != nil {
		return err
	}

	artists, err := sr.GetSeriesArtistAddresses(nil, seriesID)
	if err != nil {
		return err
	}

	artistAddresses := make([]string, len(artists))
	for i, a := range artists {
		artistAddresses[i] = a.Hex()
	}

	collection, err := e.indexerStore.GetNewCollectionByID(ctx, fmt.Sprint("seriesRegistry-", data["series_id"].(string)))
	if err != nil {
		return err
	}

	if collection == nil {
		collection = &indexer.NewCollection{
			ID:         fmt.Sprint("seriesRegistry-", data["series_id"].(string)),
			ExternalID: data["series_id"].(string),
			Source:     "seriesRegistry",
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
	if err := json.Unmarshal(event.Data.RawMessage, &data); err != nil {
		return err
	}

	collection, err := e.indexerStore.GetNewCollectionByID(ctx, fmt.Sprint("seriesRegistry-", data["series_id"].(string)))
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
	if err := json.Unmarshal(event.Data.RawMessage, &data); err != nil {
		return err
	}
	oldAddress := data["old_address"].(string)
	newAddress := data["new_address"].(string)
	return e.indexerStore.UpdateNewCollectionArtistAddress(ctx, oldAddress, newAddress)
}

func (e *EventProcessor) collaboratorConfirmed(ctx context.Context, event SeriesEvent) error {
	var data map[string]interface{}
	if err := json.Unmarshal(event.Data.RawMessage, &data); err != nil {
		return err
	}

	collaboratorID, ok := new(big.Int).SetString(data["confirmed_artist_id"].(string), 10)
	if !ok {
		return errors.New("invalid collaboratorID")
	}

	sr, err := newSeriesRegistryContract(e.rpcClient)
	if err != nil {
		return err
	}

	collaboratorAddress, err := sr.GetArtistAddress(nil, collaboratorID)
	if err != nil {
		return err
	}
	return e.indexerStore.UpdateNewCollectionCollaborator(ctx, data["series_id"].(string), collaboratorAddress.Hex())
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
		[]SeriesEventType{SeriesEventTypeRegistered},
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

func ipfsURIToCID(ipfsLink string) string {
	return regexp.MustCompile("^ipfs://").
		ReplaceAllString(ipfsLink, "")
}

func ipfsCat(cid string) ([]byte, error) {
	reader, err := shell.NewShell(viper.GetString("ipfs.addr")).Cat(cid)
	if nil != err {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}
