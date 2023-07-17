package main

import (
	"context"
	"math/big"
	"strconv"
	"time"

	goethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"
	"github.com/bitmark-inc/config-loader/external/aws/ssm"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/emitter"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
)

var currentLastStoppedBlock = uint64(0)

type EthereumEventsEmitter struct {
	lastBlockKeyName string

	grpcClient processor.EventProcessorClient
	emitter.EventsEmitter
	wsClient       *ethclient.Client
	parameterStore *ssm.ParameterStore

	ethLogChan      chan types.Log
	ethSubscription *goethereum.Subscription
}

func NewEthereumEventsEmitter(
	lastBlockKeyName string,
	wsClient *ethclient.Client,
	parameterStore *ssm.ParameterStore,
	grpcClient processor.EventProcessorClient,
) *EthereumEventsEmitter {
	return &EthereumEventsEmitter{
		lastBlockKeyName: lastBlockKeyName,
		grpcClient:       grpcClient,
		parameterStore:   parameterStore,
		EventsEmitter:    emitter.New(grpcClient),
		wsClient:         wsClient,
		ethLogChan:       make(chan types.Log, 100),
	}
}

func (e *EthereumEventsEmitter) Watch(ctx context.Context) {
	log.Info("start watching Ethereum events")

	for {
		subscription, err := e.wsClient.SubscribeFilterLogs(ctx, goethereum.FilterQuery{Topics: [][]common.Hash{
			{common.HexToHash(indexer.TransferEventSignature), common.HexToHash(indexer.TransferSingleEventSignature)}, // transfer event
		}}, e.ethLogChan)
		if err != nil {
			log.Error("fail to start subscription connection", zap.Error(err), log.SourceETHClient)
			time.Sleep(time.Second)
			continue
		}

		e.ethSubscription = &subscription
		err = <-subscription.Err()
		log.Error("subscription stopped with failure", zap.Error(err), log.SourceETHClient)

	}
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
			log.Error("failed to fetch logs from las stopped block: ", zap.Uint64("blockNum", i), zap.Error(err), log.SourceETHClient)
			return
		}

		for _, log := range logs {
			e.processETHLog(ctx, log)
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
	e.processLogsSinceLastStoppedBlock(ctx)

	go e.Watch(ctx)

	for eLog := range e.ethLogChan {
		e.processETHLog(ctx, eLog)
	}
}

func (e *EthereumEventsEmitter) processETHLog(ctx context.Context, eLog types.Log) {
	paringStartTime := time.Now()
	log.Debug("start processing ethereum log",
		zap.Any("txHash", eLog.TxHash),
		zap.Uint("logIndex", eLog.Index),
		zap.Time("time", paringStartTime))

	if topicLen := len(eLog.Topics); topicLen == 4 {

		fromAddress := indexer.EthereumChecksumAddress(eLog.Topics[1].Hex())
		toAddress := indexer.EthereumChecksumAddress(eLog.Topics[2].Hex())
		contractAddress := indexer.EthereumChecksumAddress(eLog.Address.String())
		tokenIDHash := eLog.Topics[3]

		txTime, err := indexer.GetETHBlockTime(ctx, e.wsClient, eLog.BlockHash)
		if err != nil {
			log.Error("fail to get the block time", zap.Error(err), log.SourceGRPC)
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

		if err := e.PushEvent(ctx, eventType, fromAddress, toAddress, contractAddress, utils.EthereumBlockchain, tokenIDHash.Big().Text(10), eLog.TxHash.Hex(), eLog.Index, txTime); err != nil {
			log.Error("gRPC request failed", zap.Error(err), log.SourceGRPC)
			return
		}
	}

	if eLog.BlockNumber > currentLastStoppedBlock {
		currentLastStoppedBlock = eLog.BlockNumber
		if err := e.parameterStore.Put(ctx, e.lastBlockKeyName, strconv.FormatUint(currentLastStoppedBlock, 10)); err != nil {
			log.Error("error put parameterStore", zap.Error(err), log.SourceGRPC)
			return
		}
	}
}

func (e *EthereumEventsEmitter) Close() {
	if e.ethSubscription != nil {
		(*e.ethSubscription).Unsubscribe()
		close(e.ethLogChan)
	}
}
