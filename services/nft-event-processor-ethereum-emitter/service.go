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
	lastBlockKeyName       string
	seriesRegistryContract string

	grpcClient processor.EventProcessorClient
	emitter.EventsEmitter
	wsClient       *ethclient.Client
	parameterStore *ssm.ParameterStore
	cacheStore     cache.Store

	nftTransferLogChan         chan types.Log
	seriesRegistryLogChan      chan types.Log
	nftTransferSubscription    *goethereum.Subscription
	seriesRegistrySubscription *goethereum.Subscription
}

func NewEthereumEventsEmitter(
	lastBlockKeyName string,
	seriesRegistryContract string,
	wsClient *ethclient.Client,
	parameterStore *ssm.ParameterStore,
	cacheStore cache.Store,
	grpcClient processor.EventProcessorClient,
) *EthereumEventsEmitter {
	return &EthereumEventsEmitter{
		lastBlockKeyName:       lastBlockKeyName,
		seriesRegistryContract: seriesRegistryContract,
		grpcClient:             grpcClient,
		parameterStore:         parameterStore,
		cacheStore:             cacheStore,
		EventsEmitter:          emitter.New(grpcClient),
		wsClient:               wsClient,
		nftTransferLogChan:     make(chan types.Log, 100),
		seriesRegistryLogChan:  make(chan types.Log, 100),
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
			if e.seriesRegistrySubscription != nil {
				(*e.seriesRegistrySubscription).Unsubscribe()
			}
			seriesRegistrySubscription, err := e.wsClient.SubscribeFilterLogs(ctx, goethereum.FilterQuery{
				Addresses: []common.Address{
					common.HexToAddress(e.seriesRegistryContract),
				},
				Topics: [][]common.Hash{
					{
						common.HexToHash(indexer.SeriesRegistryEventRegisterSeriesSignature),
						common.HexToHash(indexer.SeriesRegistryEventUpdateSeriesSignature),
						common.HexToHash(indexer.SeriesRegistryEventDeleteSeriesSignature),
						common.HexToHash(indexer.SeriesRegistryEventUpdateArtistAddressSignature),
						common.HexToHash(indexer.SeriesRegistryEventOptInCollaborationSignature),
						common.HexToHash(indexer.SeriesRegistryEventOptOutSeriesSignature),
						common.HexToHash(indexer.SeriesRegistryEventAssignSeriesSignature),
					},
				},
			}, e.seriesRegistryLogChan)
			if err != nil {
				log.Error("fail to start series registry subscription connection", zap.Error(err), log.SourceETHClient)
				time.Sleep(time.Second)
				continue
			}
			e.seriesRegistrySubscription = &seriesRegistrySubscription

			// Block until an error occurs in the subscription or the context is canceled.
			select {
			case err = <-seriesRegistrySubscription.Err():
				log.Error("series registry subscription stopped with failure", zap.Error(err), log.SourceETHClient)
			case <-ctx.Done():
				log.Info("context done: unsubscribing series registry subscription")
				seriesRegistrySubscription.Unsubscribe()
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
				common.HexToAddress(e.seriesRegistryContract),
			},
			Topics: [][]common.Hash{
				{
					common.HexToHash(indexer.SeriesRegistryEventRegisterSeriesSignature),
					common.HexToHash(indexer.SeriesRegistryEventUpdateSeriesSignature),
					common.HexToHash(indexer.SeriesRegistryEventDeleteSeriesSignature),
					common.HexToHash(indexer.SeriesRegistryEventUpdateArtistAddressSignature),
					common.HexToHash(indexer.SeriesRegistryEventOptInCollaborationSignature),
				},
			},
		})

		if err != nil {
			log.Error("failed to fetch series registry logs from las stopped block: ", zap.Uint64("blockNum", i), zap.Error(err), log.SourceETHClient)
			return
		}

		for _, log := range logs {
			e.processSeriesRegistryLog(ctx, log)
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
		log.Debug("start receiving series registry event log")
		for eLog := range e.seriesRegistryLogChan {
			e.processSeriesRegistryLog(ctx, eLog)
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

func (e *EthereumEventsEmitter) processSeriesRegistryLog(ctx context.Context, eLog types.Log) {
	paringStartTime := time.Now()
	log.Debug("start processing series registry log",
		zap.Any("txHash", eLog.TxHash),
		zap.Uint("logIndex", eLog.Index),
		zap.Time("time", paringStartTime))

	contract, err := seriesRegistry.NewSeriesRegistry(common.HexToAddress(e.seriesRegistryContract), e.wsClient)
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

	var data map[string]interface{}
	var eventType string
	switch eLog.Topics[0].Hex() {
	case indexer.SeriesRegistryEventRegisterSeriesSignature:
		eventType = "register_series"
		ev, err := contract.ParseRegisterSeries(eLog)
		if err != nil {
			log.Error(err.Error())
			return
		}
		data = map[string]interface{}{
			"series_id": ev.SeriesID.Text(10),
		}
	case indexer.SeriesRegistryEventUpdateSeriesSignature:
		eventType = "update_series"
		ev, err := contract.ParseUpdateSeries(eLog)
		if err != nil {
			log.Error(err.Error())
			return
		}
		data = map[string]interface{}{
			"series_id": ev.SeriesID.Text(10),
		}
	case indexer.SeriesRegistryEventDeleteSeriesSignature:
		eventType = "delete_series"
		ev, err := contract.ParseDeleteSeries(eLog)
		if err != nil {
			log.Error(err.Error())
			return
		}
		data = map[string]interface{}{
			"series_id": ev.SeriesID.Text(10),
		}
	case indexer.SeriesRegistryEventUpdateArtistAddressSignature:
		eventType = "update_artist_address"
		ev, err := contract.ParseUpdateArtistAddress(eLog)
		if err != nil {
			log.Error(err.Error())
			return
		}
		data = map[string]interface{}{
			"old_address": ev.OldAddress.Hex(),
			"new_address": ev.NewAddress.Hex(),
		}
	case indexer.SeriesRegistryEventOptInCollaborationSignature:
		eventType = "opt_in_collaboration"
		ev, err := contract.ParseOptInCollaboration(eLog)
		if err != nil {
			log.Error(err.Error())
			return
		}
		data = map[string]interface{}{
			"series_id": ev.SeriesID.Text(10),
		}
	case indexer.SeriesRegistryEventOptOutSeriesSignature:
		eventType = "opt_out_series"
		ev, err := contract.ParseOptOutSeries(eLog)
		if err != nil {
			log.Error(err.Error())
			return
		}
		data = map[string]interface{}{
			"series_id": ev.SeriesID.Text(10),
		}
	case indexer.SeriesRegistryEventAssignSeriesSignature:
		eventType = "assign_series"
		ev, err := contract.ParseAssignSeries(eLog)
		if err != nil {
			log.Error(err.Error())
			return
		}
		data = map[string]interface{}{
			"old_address": ev.AssignerAddress.Hex(),
			"new_address": ev.AssigneeAddress.Hex(),
		}
	default:
		log.Error("unsupported event")
		return
	}

	log.Debug("receive series registry event on ethereum",
		zap.String("contractAddress", contractAddress),
		zap.String("eventType", eventType),
		zap.Any("data", data),
		zap.String("txID", eLog.TxHash.Hex()),
		zap.Uint("txIndex", eLog.TxIndex),
		zap.String("txTime", txTime.String()),
	)

	if err := e.PushSeriesRegistryEvent(ctx, eventType, contractAddress, eLog.TxHash.Hex(), data, eLog.Index, txTime); err != nil {
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
