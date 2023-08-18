package main

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"
	"github.com/bitmark-inc/nft-indexer/emitter"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
	"github.com/philippseith/signalr"
	"go.uber.org/zap"
)

type TezosEventsEmitter struct {
	grpcClient processor.EventProcessorClient
	emitter.EventsEmitter
	tzktWebsocketURL string

	transferChan chan TokenTransfer
}

func NewTezosEventsEmitter(
	grpcClient processor.EventProcessorClient,
	tzktWebsocketURL string,
) *TezosEventsEmitter {
	return &TezosEventsEmitter{
		grpcClient:       grpcClient,
		EventsEmitter:    emitter.New(grpcClient),
		tzktWebsocketURL: tzktWebsocketURL,

		transferChan: make(chan TokenTransfer, 100),
	}
}

// receive event for transfers function
func (e *TezosEventsEmitter) Transfers(data json.RawMessage) {
	var res TokenTransferResponse

	err := json.Unmarshal(data, &res)
	if err != nil {
		log.Error("fail to unmarshal transfers data", zap.Error(err))
		return
	}

	for _, t := range res.Data {
		e.transferChan <- t
	}
}

func (e *TezosEventsEmitter) Run(ctx context.Context) {
	client, err := signalr.NewClient(ctx,
		signalr.WithConnector(func() (signalr.Connection, error) {
			creationCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			return signalr.NewHTTPConnection(creationCtx, e.tzktWebsocketURL)
		}),
		signalr.WithReceiver(e))
	if err != nil {
		log.Error("fail to create signalr client", zap.Error(err))
		return
	}

	client.Start()

	err = <-client.WaitForState(ctx, signalr.ClientConnected)
	if err != nil {
		log.Error("fail to wait for connected state", zap.Error(err))
		return
	}

	result := <-client.Invoke("SubscribeToTokenTransfers", struct{}{})
	if result.Error != nil {
		log.Error("fail to SubscribeToTokenTransfers", zap.Error(err))
		return
	}

	for t := range e.transferChan {
		var fromAddress string
		eventType := "mint"

		if t.From != nil {
			fromAddress = t.From.Address
			eventType = "transfer"
		}

		log.Debug("received transfer event on tezos",
			zap.String("eventType", eventType),
			zap.String("from", fromAddress),
			zap.String("to", t.To.Address),
			zap.String("contractAddress", t.Token.Contract.Address),
			zap.String("tokenID", t.Token.TokenID),
			zap.String("txID", strconv.FormatUint(t.TransactionID, 10)),
			zap.String("txTime", t.Timestamp.String()),
		)

		if err := e.PushEvent(context.Background(), eventType, t.From.Address, t.To.Address,
			t.Token.Contract.Address, utils.TezosBlockchain, t.Token.TokenID, strconv.FormatUint(t.TransactionID, 10), 0, t.Timestamp); err != nil {
			log.Error("gRPC request failed", zap.Error(err), log.SourceGRPC)
			return
		}
	}
}
