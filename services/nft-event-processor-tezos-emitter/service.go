package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/philippseith/signalr"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"
	"github.com/bitmark-inc/config-loader/external/aws/ssm"
	"github.com/bitmark-inc/nft-indexer/emitter"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
	"github.com/bitmark-inc/tzkt-go"
)

const maxMessageSize = 1 << 20 // 1MiB

var lastStoppedBlock = uint64(0)
var isFirstTransferEventOnConnected = true

type TezosEventsEmitter struct {
	lastBlockKeyName string
	parameterStore   *ssm.ParameterStore

	grpcClient processor.EventProcessorClient
	emitter.EventsEmitter
	tzktWebsocketURL string
	tzkt             *tzkt.TZKT

	transferChan chan tzkt.TokenTransfer
}

func NewTezosEventsEmitter(
	lastBlockKeyName string,
	parameterStore *ssm.ParameterStore,
	grpcClient processor.EventProcessorClient,
	tzktWebsocketURL string,
	tzktObj *tzkt.TZKT,
) *TezosEventsEmitter {
	return &TezosEventsEmitter{
		lastBlockKeyName: lastBlockKeyName,
		parameterStore:   parameterStore,
		grpcClient:       grpcClient,
		EventsEmitter:    emitter.New(grpcClient),
		tzktWebsocketURL: tzktWebsocketURL,
		tzkt:             tzktObj,

		transferChan: make(chan tzkt.TokenTransfer, 100),
	}
}

// Transfers is a callback function for handling events from `transfers` channel
// According to https://github.com/philippseith/signalr#client-side-go, we need to create a same name
// function according to the server response channel. See https://api.tzkt.io/#section/SubscribeToTokenTransfers
func (e *TezosEventsEmitter) Transfers(data json.RawMessage) {
	var res TokenTransferResponse

	err := json.Unmarshal(data, &res)
	if err != nil {
		log.Error("fail to unmarshal transfers data", zap.Error(err))
		return
	}

	if len(res.Data) == 0 {
		return
	}

	if isFirstTransferEventOnConnected && lastStoppedBlock > 0 {
		isFirstTransferEventOnConnected = false
		e.fetchTransfersFromLastStoppedLevel(lastStoppedBlock)
	}

	for _, t := range res.Data {
		e.transferChan <- t
	}
}

type SignalrLogger struct{}

func (s *SignalrLogger) Log(keyVals ...interface{}) error {
	if len(keyVals)%2 != 0 {
		// Suppose this should not happen
		log.Warn("signalr log", zap.Any("keyVals", keyVals))
		return nil
	}

	signalrDebug := false
	zapFields := []zap.Field{}
	for i := 0; i < len(keyVals); i += 2 {
		key := fmt.Sprint(keyVals[i])
		value := keyVals[i+1]
		if key == "level" {
			if fmt.Sprint(value) == "debug" {
				signalrDebug = true
			}
			// omit the level in the log fields
			continue
		}
		zapFields = append(zapFields, zap.Any(key, value))
	}
	if signalrDebug {
		log.Debug("signalr log", zapFields...)
	} else {
		log.Info("signalr log", zapFields...)
	}

	return nil
}

func (e *TezosEventsEmitter) Run(ctx context.Context) {
	client, err := signalr.NewClient(ctx,
		signalr.Logger(&SignalrLogger{}, viper.GetBool("debug")),
		signalr.MaximumReceiveMessageSize(maxMessageSize),
		signalr.WithConnector(func() (signalr.Connection, error) {
			creationCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			return signalr.NewHTTPConnection(creationCtx, e.tzktWebsocketURL)
		}),
		signalr.WithReceiver(e))
	if err != nil {
		log.Error("fail to create signalr client", zap.Error(err))
		return
	}

	client.Start()

	//handle state changed
	stateChan := make(chan signalr.ClientState, 1)
	_ = client.ObserveStateChanged(stateChan)

	go func() {
		for state := range stateChan {
			switch state {
			case signalr.ClientConnected:
				e.processTransfersSinceLastStoppedLevel(ctx)
				isFirstTransferEventOnConnected = true

				result := <-client.Invoke("SubscribeToTokenTransfers", struct{}{})
				if result.Error != nil {
					log.Panic("fail to SubscribeToTokenTransfers", zap.Error(err))
				}
			case signalr.ClientClosed:
				log.Panic("client closed", zap.Error(err))
			}
		}
	}()

	for t := range e.transferChan {
		e.processTranferEvent(ctx, t)
	}
}

func (e *TezosEventsEmitter) processTransfersSinceLastStoppedLevel(ctx context.Context) {
	lastStopLevel, err := e.parameterStore.GetString(ctx, e.lastBlockKeyName)
	if err != nil {
		log.Error("failed to read last stop block from parameter store: ", zap.Error(err), log.SourceTZKT)
		return
	}

	fromLevel, err := strconv.ParseUint(lastStopLevel, 10, 64)
	if err != nil {
		log.Error("failed to parse last stop block: ", zap.Error(err), log.SourceETHClient)
		return
	}

	e.fetchTransfersFromLastStoppedLevel(fromLevel)
}

func (e *TezosEventsEmitter) fetchTransfersFromLastStoppedLevel(lastStoppedLevel uint64) {
	lastLevel := lastStoppedLevel
	pageSize := 100

	for {
		transfers, err := e.tzkt.GetTokenTransfersByLevel(fmt.Sprintf("%d", lastLevel), pageSize)
		if err != nil {
			log.Error("failed to fetch token transfer from last level: ",
				zap.Error(err), zap.Uint64("lastLevel", lastLevel), log.SourceTZKT)
			return
		}

		for _, transfer := range transfers {
			e.transferChan <- transfer
		}

		if len(transfers) < pageSize {
			break
		}

		lastLevel = transfers[len(transfers)-1].Level
	}
}

func (e *TezosEventsEmitter) processTranferEvent(ctx context.Context, transfer tzkt.TokenTransfer) {
	var fromAddress string
	eventType := "mint"

	if transfer.From != nil {
		fromAddress = transfer.From.Address
		eventType = "transfer"
	}

	log.Debug("received transfer event on tezos",
		zap.String("eventType", eventType),
		zap.String("from", fromAddress),
		zap.String("to", transfer.To.Address),
		zap.String("contractAddress", transfer.Token.Contract.Address),
		zap.String("tokenID", transfer.Token.ID.String()),
		zap.String("txID", strconv.FormatUint(transfer.TransactionID, 10)),
		zap.String("txTime", transfer.Timestamp.String()),
	)

	if err := e.PushEvent(ctx, eventType, fromAddress, transfer.To.Address,
		transfer.Token.Contract.Address, utils.TezosBlockchain, transfer.Token.ID.String(),
		strconv.FormatUint(transfer.TransactionID, 10), 0, transfer.Timestamp); err != nil {
		log.Error("gRPC request failed", zap.Error(err), log.SourceGRPC)
		return
	}

	if transfer.Level > lastStoppedBlock {
		lastStoppedBlock = transfer.Level
		if err := e.parameterStore.Put(ctx, e.lastBlockKeyName, strconv.FormatUint(lastStoppedBlock, 10)); err != nil {
			log.Error("error put parameterStore", zap.Error(err), log.SourceGRPC)
			return
		}
	}
}
