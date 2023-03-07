package main

import (
	"context"
	"math/big"
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/emitter"
	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
	goethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

type EthereumEventsEmitter struct {
	grpcClient processor.EventProcessorClient
	emitter.EventsEmitter
	wsClient *ethclient.Client

	ethLogChan      chan types.Log
	ethSubscription *goethereum.Subscription
}

func NewEthereumEventsEmitter(wsClient *ethclient.Client,
	grpcClient processor.EventProcessorClient) *EthereumEventsEmitter {
	return &EthereumEventsEmitter{
		grpcClient:    grpcClient,
		EventsEmitter: emitter.New(grpcClient),
		wsClient:      wsClient,
		ethLogChan:    make(chan types.Log, 100),
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

func (e *EthereumEventsEmitter) Run(ctx context.Context) {
	go e.Watch(ctx)

	for eLog := range e.ethLogChan {
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

			log.Debug("receive transfer event on ethereum",
				zap.String("from", fromAddress),
				zap.String("to", toAddress),
				zap.String("contractAddress", contractAddress),
				zap.Any("tokenIDHash", tokenIDHash))

			if eLog.Topics[1].Big().Cmp(big.NewInt(0)) == 0 {
				// ignore minting events
				continue
			}

			eventType := "transfer"
			if fromAddress == indexer.EthereumZeroAddress {
				eventType = "mint"
			} else if toAddress == indexer.EthereumZeroAddress {
				eventType = "burned"
			}

			if err := e.PushEvent(ctx, eventType, fromAddress, toAddress, contractAddress, indexer.EthereumBlockchain, tokenIDHash.Big().Text(10)); err != nil {
				log.Error("gRPC request failed", zap.Error(err), log.SourceGRPC)
				continue
			}
		}
	}
}

func (e *EthereumEventsEmitter) Close() {
	if e.ethSubscription != nil {
		(*e.ethSubscription).Unsubscribe()
		close(e.ethLogChan)
	}
}
