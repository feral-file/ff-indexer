package main

import (
	"context"
	"math/big"
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/emitter"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
	goethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
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
	logrus.Info("start watching Ethereum events")

	for {
		subscription, err := e.wsClient.SubscribeFilterLogs(ctx, goethereum.FilterQuery{Topics: [][]common.Hash{
			{common.HexToHash(indexer.TransferEventSignature)}, // transfer event
		}}, e.ethLogChan)
		if err != nil {
			logrus.WithError(err).Error("fail to start subscription connection")
			time.Sleep(time.Second)
			continue
		}

		e.ethSubscription = &subscription
		err = <-subscription.Err()
		logrus.WithError(err).Error("subscription stopped with failure")

	}
}

func (e *EthereumEventsEmitter) Run(ctx context.Context) {
	go e.Watch(ctx)

	for eLog := range e.ethLogChan {
		paringStartTime := time.Now()
		logrus.WithField("txHash", eLog.TxHash).
			WithField("logIndex", eLog.Index).
			WithField("time", paringStartTime).
			Debug("start processing ethereum log")

		if topicLen := len(eLog.Topics); topicLen == 4 {

			fromAddress := indexer.EthereumChecksumAddress(eLog.Topics[1].Hex())
			toAddress := indexer.EthereumChecksumAddress(eLog.Topics[2].Hex())
			contractAddress := indexer.EthereumChecksumAddress(eLog.Address.String())
			tokenIDHash := eLog.Topics[3]

			logrus.WithField("from", fromAddress).
				WithField("to", toAddress).
				WithField("contractAddress", contractAddress).
				WithField("tokenIDHash", tokenIDHash).
				Debug("receive transfer event on ethereum")

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
				logrus.WithError(err).Error("gRPC request failed")
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
