package main

import (
	"context"
	"math/big"
	"strconv"
	"sync"
	"time"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"
	"github.com/bitmark-inc/config-loader/external/aws/ssm"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/cache"
	"github.com/bitmark-inc/nft-indexer/emitter"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
	goethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"

	seriesRegistry "github.com/bitmark-inc/feralfile-exhibition-smart-contract/go-binding/series-registry"
)

var currentLastStoppedBlock = uint64(0)

type EthereumEventsEmitter struct {
	lastBlockKeyName string

	grpcClient processor.EventProcessorClient
	emitter.EventsEmitter
	wsClient       *ethclient.Client
	parameterStore *ssm.ParameterStore
	cacheStore     cache.Store

	nftTransferLogChan      chan types.Log
	seriesIndexLogChan      chan types.Log
	nftTransferSubscription *goethereum.Subscription
	seriesIndexSubscription *goethereum.Subscription
}

func NewEthereumEventsEmitter(
	lastBlockKeyName string,
	wsClient *ethclient.Client,
	parameterStore *ssm.ParameterStore,
	cacheStore cache.Store,
	grpcClient processor.EventProcessorClient,
) *EthereumEventsEmitter {
	return &EthereumEventsEmitter{
		lastBlockKeyName:   lastBlockKeyName,
		grpcClient:         grpcClient,
		parameterStore:     parameterStore,
		cacheStore:         cacheStore,
		EventsEmitter:      emitter.New(grpcClient),
		wsClient:           wsClient,
		nftTransferLogChan: make(chan types.Log, 100),
		seriesIndexLogChan: make(chan types.Log, 100),
	}
}

func (e *EthereumEventsEmitter) Watch(ctx context.Context) {
	log.Info("start watching Ethereum events")

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for {
			if e.nftTransferSubscription != nil {
				(*e.nftTransferSubscription).Unsubscribe()
			}
			nftTransferSubscription, err := e.wsClient.SubscribeFilterLogs(ctx, goethereum.FilterQuery{Topics: [][]common.Hash{
				{common.HexToHash(indexer.TransferEventSignature), common.HexToHash(indexer.TransferSingleEventSignature)}, // transfer event
			}}, e.nftTransferLogChan)
			if err != nil {
				log.Error("fail to start nft transfer subscription connection", zap.Error(err), log.SourceETHClient)
				time.Sleep(time.Second)
				continue
			}

			e.nftTransferSubscription = &nftTransferSubscription

			// Block until an error occurs in the subscription or the context is canceled.
			select {
			case err = <-nftTransferSubscription.Err():
				log.Error("nft transfer subscription stopped with failure", zap.Error(err), log.SourceETHClient)
			case <-ctx.Done():
				log.Info("context done: unsubscribing NFT transfer subscription")
				nftTransferSubscription.Unsubscribe()
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for {
			if e.seriesIndexSubscription != nil {
				(*e.seriesIndexSubscription).Unsubscribe()
			}
			seriesIndexSubscription, err := e.wsClient.SubscribeFilterLogs(ctx, goethereum.FilterQuery{
				Addresses: []common.Address{
					common.HexToAddress(indexer.SeriesRegistryContract),
				},
				Topics: [][]common.Hash{
					{
						common.HexToHash(indexer.SeriesEventRegisteredSignature),
						common.HexToHash(indexer.SeriesEventUpdatedSignature),
						common.HexToHash(indexer.SeriesEventDeletedSignature),
						common.HexToHash(indexer.SeriesEventArtistAddressUpdatedSignature),
						common.HexToHash(indexer.SeriesEventCollaboratorConfirmedSignature),
					},
				},
			}, e.seriesIndexLogChan)
			if err != nil {
				log.Error("fail to start series index subscription connection", zap.Error(err), log.SourceETHClient)
				time.Sleep(time.Second)
				continue
			}
			e.seriesIndexSubscription = &seriesIndexSubscription

			// Block until an error occurs in the subscription or the context is canceled.
			select {
			case err = <-seriesIndexSubscription.Err():
				log.Error("series index subscription stopped with failure", zap.Error(err), log.SourceETHClient)
			case <-ctx.Done():
				log.Info("context done: unsubscribing series index subscription")
				seriesIndexSubscription.Unsubscribe()
				return
			}
		}
	}()

	wg.Wait()
}

func (e *EthereumEventsEmitter) fetchLogsFromLastStoppedBlock(ctx context.Context, lastStopBlock uint64) {
	latestBlock, err := e.wsClient.BlockNumber(ctx)
	if err != nil {
		log.Error("failed to fetch latest block: ", zap.Error(err), log.SourceETHClient)
		return
	}

	// iterate every block to avoid heavy response
	for i := lastStopBlock; i <= latestBlock; i++ {
		block := new(big.Int)
		block.SetUint64(i)
		logs, err := e.wsClient.FilterLogs(ctx, goethereum.FilterQuery{
			FromBlock: block,
			ToBlock:   block,
			Topics: [][]common.Hash{
				{common.HexToHash(indexer.TransferEventSignature), common.HexToHash(indexer.TransferSingleEventSignature)},
			},
		})

		if err != nil {
			log.Error("failed to fetch nft transfer logs from las stopped block: ", zap.Uint64("blockNum", i), zap.Error(err), log.SourceETHClient)
			return
		}

		for _, log := range logs {
			e.processNftTransferLog(ctx, log)
		}

		logs, err = e.wsClient.FilterLogs(ctx, goethereum.FilterQuery{
			FromBlock: block,
			ToBlock:   block,
			Addresses: []common.Address{
				common.HexToAddress(indexer.SeriesRegistryContract),
			},
			Topics: [][]common.Hash{
				{
					common.HexToHash(indexer.SeriesEventRegisteredSignature),
					common.HexToHash(indexer.SeriesEventUpdatedSignature),
					common.HexToHash(indexer.SeriesEventDeletedSignature),
					common.HexToHash(indexer.SeriesEventArtistAddressUpdatedSignature),
					common.HexToHash(indexer.SeriesEventCollaboratorConfirmedSignature),
				},
			},
		})

		if err != nil {
			log.Error("failed to fetch series index logs from las stopped block: ", zap.Uint64("blockNum", i), zap.Error(err), log.SourceETHClient)
			return
		}

		for _, log := range logs {
			e.processSeriesIndexLog(ctx, log)
		}
	}
}

func (e *EthereumEventsEmitter) processLogsSinceLastStoppedBlock(ctx context.Context) {
	lastStopBlock, err := e.parameterStore.GetString(ctx, e.lastBlockKeyName)
	if err != nil {
		log.Error("failed to read last stop bloc from parameter store: ", zap.Error(err), log.SourceETHClient)
	} else {
		fromBlock, err := strconv.ParseUint(lastStopBlock, 10, 64)
		if err != nil {
			log.Error("failed to parse last stop block: ", zap.Error(err), log.SourceETHClient)
		} else {
			e.fetchLogsFromLastStoppedBlock(ctx, fromBlock)
		}
	}
}

func (e *EthereumEventsEmitter) Run(ctx context.Context) {
	// e.processLogsSinceLastStoppedBlock(ctx)

	go e.Watch(ctx)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		log.Debug("start receiving nft transfer event log")
		for eLog := range e.nftTransferLogChan {
			e.processNftTransferLog(ctx, eLog)
		}
	}()

	go func() {
		defer wg.Done()
		log.Debug("start receiving series event log")
		for eLog := range e.seriesIndexLogChan {
			e.processSeriesIndexLog(ctx, eLog)
		}
	}()

	wg.Wait()
}

func (e *EthereumEventsEmitter) processNftTransferLog(ctx context.Context, eLog types.Log) {
	paringStartTime := time.Now()
	log.Debug("start processing ethereum log",
		zap.Any("txHash", eLog.TxHash),
		zap.Uint("logIndex", eLog.Index),
		zap.Time("time", paringStartTime))

	if topicLen := len(eLog.Topics); topicLen == 4 {
		contractAddress := indexer.EthereumChecksumAddress(eLog.Address.String())
		var fromAddress, toAddress string
		var tokenIDHash common.Hash

		switch eLog.Topics[0].Hex() {
		case indexer.TransferEventSignature:
			fromAddress = indexer.EthereumChecksumAddress(eLog.Topics[1].Hex())
			toAddress = indexer.EthereumChecksumAddress(eLog.Topics[2].Hex())
			tokenIDHash = eLog.Topics[3]
		case indexer.TransferSingleEventSignature:
			fromAddress = indexer.EthereumChecksumAddress(eLog.Topics[2].Hex())
			toAddress = indexer.EthereumChecksumAddress(eLog.Topics[3].Hex())
			tokenIDHash = common.BytesToHash(eLog.Data[0:32])
		default:
			log.Error("unsupported event")
			return
		}

		txTime, err := indexer.GetETHBlockTime(ctx, e.cacheStore, e.wsClient, eLog.BlockHash)
		if err != nil {
			log.Error("fail to get the block time", zap.Error(err), log.SourceGRPC)
			sentry.CaptureException(err)
			return
		}

		log.Debug("receive transfer event on ethereum",
			zap.String("from", fromAddress),
			zap.String("to", toAddress),
			zap.String("contractAddress", contractAddress),
			zap.String("tokenIDHash", tokenIDHash.Hex()),
			zap.String("txID", eLog.TxHash.Hex()),
			zap.Uint("txIndex", eLog.TxIndex),
			zap.String("txTime", txTime.String()),
		)

		eventType := "transfer"
		if fromAddress == indexer.EthereumZeroAddress {
			eventType = "mint"
		} else if toAddress == indexer.EthereumZeroAddress {
			eventType = "burned"
		}

		if err := e.PushNftEvent(ctx, eventType, fromAddress, toAddress, contractAddress, utils.EthereumBlockchain, tokenIDHash.Big().Text(10), eLog.TxHash.Hex(), eLog.Index, txTime); err != nil {
			log.Error("gRPC request failed", zap.Error(err), log.SourceGRPC)
			sentry.CaptureException(err)
			return
		}
	}

	if eLog.BlockNumber > currentLastStoppedBlock {
		currentLastStoppedBlock = eLog.BlockNumber
		if err := e.parameterStore.PutString(ctx, e.lastBlockKeyName, strconv.FormatUint(currentLastStoppedBlock, 10)); err != nil {
			log.Error("error put parameterStore", zap.Error(err), log.SourceGRPC)
			return
		}
	}
}

func (e *EthereumEventsEmitter) processSeriesIndexLog(ctx context.Context, eLog types.Log) {
	paringStartTime := time.Now()
	log.Debug("start processing series index log",
		zap.Any("txHash", eLog.TxHash),
		zap.Uint("logIndex", eLog.Index),
		zap.Time("time", paringStartTime))

	sr, err := seriesRegistry.NewSeriesRegistry(common.HexToAddress(indexer.SeriesRegistryContract), e.wsClient)
	if err != nil {
		log.Error(err.Error())
		return
	}

	contractAddress := indexer.EthereumChecksumAddress(eLog.Address.String())

	txTime, err := indexer.GetETHBlockTime(ctx, e.cacheStore, e.wsClient, eLog.BlockHash)
	if err != nil {
		log.Error("fail to get the block time", zap.Error(err), log.SourceGRPC)
		sentry.CaptureException(err)
		return
	}

	var data *map[string]interface{}
	var eventType string
	switch eLog.Topics[0].Hex() {
	case indexer.SeriesEventRegisteredSignature:
		eventType = "registered"
		ev, err := sr.ParseSeriesRegistered(eLog)
		if err != nil {
			log.Error(err.Error())
			return
		}
		data = &map[string]interface{}{
			"series_id": ev.SeriesID.Text(10),
		}
	case indexer.SeriesEventUpdatedSignature:
		eventType = "updated"
		ev, err := sr.ParseSeriesUpdated(eLog)
		if err != nil {
			log.Error(err.Error())
			return
		}
		data = &map[string]interface{}{
			"series_id": ev.SeriesID.Text(10),
		}
	case indexer.SeriesEventDeletedSignature:
		eventType = "deleted"
		ev, err := sr.ParseSeriesDeleted(eLog)
		if err != nil {
			log.Error(err.Error())
			return
		}
		data = &map[string]interface{}{
			"series_id": ev.SeriesID.Text(10),
		}
	case indexer.SeriesEventArtistAddressUpdatedSignature:
		eventType = "artist_address_updated"
		ev, err := sr.ParseArtistAddressUpdated(eLog)
		if err != nil {
			log.Error(err.Error())
			return
		}
		data = &map[string]interface{}{
			"artist_id":   ev.ArtistID.Text(10),
			"old_address": ev.OldAddress.Hex(),
			"new_address": ev.NewAddress.Hex(),
		}
	case indexer.SeriesEventCollaboratorConfirmedSignature:
		eventType = "collaborator_confirmed"
		ev, err := sr.ParseCollaboratorConfirmed(eLog)
		if err != nil {
			log.Error(err.Error())
			return
		}
		data = &map[string]interface{}{
			"series_id":           ev.SeriesID.Text(10),
			"confirmed_artist_id": ev.ConfirmedArtistID.Text(10),
		}
	default:
		log.Error("unsupported event")
		return
	}

	log.Debug("receive series event on ethereum",
		zap.String("contractAddress", contractAddress),
		zap.String("eventType", eventType),
		zap.Any("data", data),
		zap.String("txID", eLog.TxHash.Hex()),
		zap.Uint("txIndex", eLog.TxIndex),
		zap.String("txTime", txTime.String()),
	)

	if err := e.PushSeriesEvent(ctx, eventType, contractAddress, eLog.TxHash.Hex(), data, eLog.Index, txTime); err != nil {
		log.Error("gRPC request failed", zap.Error(err), log.SourceGRPC)
		sentry.CaptureException(err)
		return
	}

	if eLog.BlockNumber > currentLastStoppedBlock {
		currentLastStoppedBlock = eLog.BlockNumber
		if err := e.parameterStore.PutString(ctx, e.lastBlockKeyName, strconv.FormatUint(currentLastStoppedBlock, 10)); err != nil {
			log.Error("error put parameterStore", zap.Error(err), log.SourceGRPC)
			return
		}
	}
}

func (e *EthereumEventsEmitter) Close() {
	if e.nftTransferSubscription != nil {
		(*e.nftTransferSubscription).Unsubscribe()
		close(e.nftTransferLogChan)
	}
}
