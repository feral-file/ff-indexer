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

var transferLastStoppedBlock = uint64(0)
var bigmapUpdatesLastStoppedBlock = uint64(0)

var isFirstTransferEventOnConnected = true
var isFirstBigmapEventOnConnected = true

type TezosEventsEmitter struct {
	transfersLastBlockKeyName     string
	bigmapUpdateslastBlockKeyName string

	parameterStore *ssm.ParameterStore

	grpcClient processor.EventProcessorClient
	emitter.EventsEmitter
	tzktWebsocketURL string
	tzkt             *tzkt.TZKT

	eventChan chan TokenEvent
}

func NewTezosEventsEmitter(
	transfersLastBlockKeyName string,
	bigmapUpdateslastBlockKeyName string,

	parameterStore *ssm.ParameterStore,
	grpcClient processor.EventProcessorClient,
	tzktWebsocketURL string,
	tzktClient *tzkt.TZKT,
) *TezosEventsEmitter {
	return &TezosEventsEmitter{
		transfersLastBlockKeyName:     transfersLastBlockKeyName,
		bigmapUpdateslastBlockKeyName: bigmapUpdateslastBlockKeyName,

		parameterStore:   parameterStore,
		grpcClient:       grpcClient,
		EventsEmitter:    emitter.New(grpcClient),
		tzktWebsocketURL: tzktWebsocketURL,
		tzkt:             tzktClient,

		eventChan: make(chan TokenEvent, 100),
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

	if isFirstTransferEventOnConnected {
		isFirstTransferEventOnConnected = false
		if transferLastStoppedBlock > 0 {
			e.fetchTransfersFromLastStoppedLevel(transferLastStoppedBlock)
		}
	}

	for _, t := range res.Data {
		e.eventChan <- e.tokenTransferToEvent(t)
	}
}

// Bigmaps is a callback function for handling events from `bigmaps` channel
// According to https://github.com/philippseith/signalr#client-side-go, we need to create a same name
// function according to the server response channel. See https://api.tzkt.io/#section/SubscribeToBigMaps
func (e *TezosEventsEmitter) Bigmaps(data json.RawMessage) {
	var res BigmapUpdateResponse

	err := json.Unmarshal(data, &res)
	if err != nil {
		log.Error("fail to unmarshal bigmaps data", zap.Error(err))
		return
	}

	if len(res.Data) == 0 {
		return
	}

	if isFirstBigmapEventOnConnected {
		isFirstBigmapEventOnConnected = false
		if bigmapUpdatesLastStoppedBlock > 0 {
			e.fetchTokenBigmapUpdateFromLastStoppedLevel(bigmapUpdatesLastStoppedBlock)
		}
	}

	for _, t := range res.Data {
		e.eventChan <- e.tokenMetadataUpdateToEvent(t)
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

				e.processBigmapUpdatesSinceLastStoppedLevel(ctx)
				isFirstBigmapEventOnConnected = true
				result = <-client.Invoke("SubscribeToBigMaps", struct {
					Tags []string
				}{
					Tags: []string{"token_metadata"},
				})
				if result.Error != nil {
					log.Panic("fail to SubscribeToBigMaps", zap.Error(err))
				}
			case signalr.ClientClosed:
				log.Panic("client closed", zap.Error(err))
			}
		}
	}()

	for t := range e.eventChan {
		e.processTranferEvent(ctx, t)
	}
}

func (e *TezosEventsEmitter) processTransfersSinceLastStoppedLevel(ctx context.Context) {
	lastStopLevel, err := e.parameterStore.GetString(ctx, e.transfersLastBlockKeyName)
	if err != nil {
		log.Error("failed to read transfer last stop block from parameter store: ", zap.Error(err), log.SourceTZKT)
		return
	}

	fromLevel, err := strconv.ParseUint(lastStopLevel, 10, 64)
	if err != nil {
		log.Error("failed to parse transfer last stop block: ", zap.Error(err), log.SourceETHClient)
		return
	}

	e.fetchTransfersFromLastStoppedLevel(fromLevel)
}

func (e *TezosEventsEmitter) fetchTransfersFromLastStoppedLevel(lastStoppedLevel uint64) {
	offset := 0
	pageSize := 100

	for {
		transfers, err := e.tzkt.GetTokenTransfersByLevel(fmt.Sprintf("%d", lastStoppedLevel), offset, pageSize)
		if err != nil {
			log.Error("failed to fetch token transfer from last level: ",
				zap.Error(err), zap.Uint64("lastLevel", lastStoppedLevel), zap.Int("offset", offset), log.SourceTZKT)
			return
		}

		for _, transfer := range transfers {
			e.eventChan <- e.tokenTransferToEvent(transfer)
		}

		if len(transfers) < pageSize {
			break
		}

		offset += pageSize
	}
}

func (e *TezosEventsEmitter) processBigmapUpdatesSinceLastStoppedLevel(ctx context.Context) {
	lastStopLevel, err := e.parameterStore.GetString(ctx, e.bigmapUpdateslastBlockKeyName)
	if err != nil {
		log.Error("failed to read bigmap last stop block from parameter store: ", zap.Error(err), log.SourceTZKT)
		return
	}

	fromLevel, err := strconv.ParseUint(lastStopLevel, 10, 64)
	if err != nil {
		log.Error("failed to parse bigmap last stop block: ", zap.Error(err), log.SourceETHClient)
		return
	}

	e.fetchTokenBigmapUpdateFromLastStoppedLevel(fromLevel)
}

func (e *TezosEventsEmitter) fetchTokenBigmapUpdateFromLastStoppedLevel(lastStoppedLevel uint64) {
	offset := 0
	pageSize := 100

	for {
		updates, err := e.tzkt.GetTokenMetadataBigmapUpdatesByLevel(fmt.Sprintf("%d", lastStoppedLevel), offset, pageSize)
		if err != nil {
			log.Error("failed to fetch token metadata bigmap updates from last level: ",
				zap.Error(err), zap.Uint64("lastLevel", lastStoppedLevel), zap.Int("offset", offset), log.SourceTZKT)
			return
		}

		for _, update := range updates {
			e.eventChan <- e.tokenMetadataUpdateToEvent(update)
		}

		if len(updates) < pageSize {
			break
		}

		offset += pageSize
	}
}

func (e *TezosEventsEmitter) processTranferEvent(ctx context.Context, event TokenEvent) {
	log.Debug("received event on tezos",
		zap.String("eventType", string(event.EventType)),
		zap.String("from", event.From),
		zap.String("to", event.To),
		zap.String("contractAddress", event.ContractAddress),
		zap.String("tokenID", event.TokenID),
		zap.String("txID", event.TxID),
		zap.String("txTime", event.TxTime.String()),
	)

	if err := e.PushEvent(ctx, string(event.EventType), event.From, event.To,
		event.ContractAddress, event.Blockchain, event.TokenID,
		event.TxID, 0, event.TxTime); err != nil {
		log.Error("gRPC request failed", zap.Error(err), log.SourceGRPC)
		return
	}

	if event.EventType == EventTypeTokenUpdated {
		if event.Level > bigmapUpdatesLastStoppedBlock {
			bigmapUpdatesLastStoppedBlock = event.Level
			if err := e.parameterStore.Put(ctx, e.bigmapUpdateslastBlockKeyName, strconv.FormatUint(bigmapUpdatesLastStoppedBlock, 10)); err != nil {
				log.Error("error put parameterStore for transferLastBlock", zap.Error(err), log.SourceGRPC)
				return
			}
		}
	} else {
		if event.Level > transferLastStoppedBlock {
			transferLastStoppedBlock = event.Level
			if err := e.parameterStore.Put(ctx, e.transfersLastBlockKeyName, strconv.FormatUint(transferLastStoppedBlock, 10)); err != nil {
				log.Error("error put parameterStore bigmapLastBlock", zap.Error(err), log.SourceGRPC)
				return
			}
		}
	}
}

func (e *TezosEventsEmitter) tokenTransferToEvent(transfer tzkt.TokenTransfer) TokenEvent {
	var fromAddress string
	eventType := EventTypeMint

	if transfer.From != nil {
		fromAddress = transfer.From.Address
		eventType = EventTypeTransfer
	}

	return TokenEvent{
		EventType:       eventType,
		From:            fromAddress,
		To:              transfer.To.Address,
		ContractAddress: transfer.Token.Contract.Address,
		Blockchain:      utils.TezosBlockchain,
		TokenID:         transfer.Token.ID.String(),
		TxID:            strconv.FormatUint(transfer.TransactionID, 10),
		TxTime:          transfer.Timestamp,
		Level:           transfer.Level,
	}
}

func (e *TezosEventsEmitter) tokenMetadataUpdateToEvent(bigmap tzkt.BigmapUpdate) TokenEvent {
	return TokenEvent{
		EventType:       EventTypeTokenUpdated,
		From:            "",
		To:              "",
		ContractAddress: bigmap.Contract.Address,
		Blockchain:      utils.TezosBlockchain,
		TokenID:         bigmap.Content.Value.TokenID,
		TxID:            strconv.FormatUint(bigmap.ID, 10),
		TxTime:          bigmap.Timestamp,
		Level:           bigmap.Level,
	}
}
