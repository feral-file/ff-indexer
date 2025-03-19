package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"
	"github.com/bitmark-inc/config-loader/external/aws/ssm"
	"github.com/bitmark-inc/nft-indexer/emitter"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
	"github.com/bitmark-inc/tzkt-go"
	"github.com/philippseith/signalr"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const maxMessageSize = 1 << 20 // 1MiB

var lastStoppedBlock = uint64(0)

var isFirstTransferEventOnConnected = true
var isFirstBigmapEventOnConnected = true

type TezosEventsEmitter struct {
	lastBlockKeyName string
	parameterStore   *ssm.ParameterStore

	grpcClient processor.EventProcessorClient
	emitter.EventsEmitter
	tzktWebsocketURL string
	tzkt             *tzkt.TZKT

	eventChan chan TokenEvent
}

func NewTezosEventsEmitter(
	lastBlockKeyName string,
	parameterStore *ssm.ParameterStore,
	grpcClient processor.EventProcessorClient,
	tzktWebsocketURL string,
	tzktClient *tzkt.TZKT,
) *TezosEventsEmitter {
	return &TezosEventsEmitter{
		lastBlockKeyName: lastBlockKeyName,
		parameterStore:   parameterStore,
		grpcClient:       grpcClient,
		EventsEmitter:    emitter.New(grpcClient),
		tzktWebsocketURL: tzktWebsocketURL,
		tzkt:             tzktClient,

		eventChan: make(chan TokenEvent, 100),
	}
}

type SignalrLogger struct {
	ctx context.Context
}

func (s *SignalrLogger) Log(keyVals ...interface{}) error {
	if len(keyVals)%2 != 0 {
		// Suppose this should not happen
		log.WarnWithContext(s.ctx, "signalr log", zap.Any("keyVals", keyVals))
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
		log.InfoWithContext(s.ctx, "signalr log", zapFields...)
	}

	return nil
}

func (e *TezosEventsEmitter) Run(ctx context.Context) {
	client, err := signalr.NewClient(ctx,
		signalr.Logger(&SignalrLogger{ctx: ctx}, viper.GetBool("debug")),
		signalr.MaximumReceiveMessageSize(maxMessageSize),
		signalr.WithConnector(func() (signalr.Connection, error) {
			creationCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			return signalr.NewHTTPConnection(creationCtx, e.tzktWebsocketURL)
		}),
		signalr.WithReceiver(e))
	if err != nil {
		log.ErrorWithContext(ctx, "fail to create signalr client", zap.Error(err))
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
				e.processSinceLastStoppedLevel(ctx)
				isFirstTransferEventOnConnected = true
				isFirstBigmapEventOnConnected = true

				result := <-client.Invoke("SubscribeToTokenTransfers", struct{}{})
				if result.Error != nil {
					log.Panic("fail to SubscribeToTokenTransfers", zap.Error(err))
				}

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
		e.processTokenEvent(ctx, t)
	}
}

func (e *TezosEventsEmitter) processSinceLastStoppedLevel(ctx context.Context) {
	lastStopLevel, err := e.parameterStore.GetString(ctx, e.lastBlockKeyName)
	if err != nil {
		log.ErrorWithContext(ctx, "failed to read last stop block from parameter store: ", zap.Error(err), log.SourceTZKT)
		return
	}

	fromLevel, err := strconv.ParseUint(lastStopLevel, 10, 64)
	if err != nil {
		log.ErrorWithContext(ctx, "failed to parse last stop block: ", zap.Error(err), log.SourceTZKT)
		return
	}

	e.fetchFromByLastStoppedLevel(ctx, fromLevel)
}

func (e *TezosEventsEmitter) fetchFromByLastStoppedLevel(ctx context.Context, fromLevel uint64) {
	latestLevel, err := e.tzkt.GetLevelByTime(time.Now())
	if err != nil {
		log.ErrorWithContext(ctx, "failed to get lastest block level: ", zap.Error(err), log.SourceTZKT)
		return
	}

	for i := fromLevel; i <= latestLevel; i++ {
		e.fetchTokenTransfersByLevel(ctx, i)
		e.fetchTokenBigmapUpdateByLevel(ctx, i)
	}
}

func (e *TezosEventsEmitter) fetchTokenTransfersByLevel(ctx context.Context, level uint64) {
	offset := 0
	pageSize := 100

	for {
		transfers, err := e.tzkt.GetTokenTransfersByLevel(fmt.Sprintf("%d", level), offset, pageSize)
		if err != nil {
			log.ErrorWithContext(ctx, "failed to fetch token transfer from level: ",
				zap.Error(err), zap.Uint64("level", level), zap.Int("offset", offset), log.SourceTZKT)
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

func (e *TezosEventsEmitter) fetchTokenBigmapUpdateByLevel(ctx context.Context, level uint64) {
	offset := 0
	pageSize := 100

	for {
		updates, err := e.tzkt.GetTokenMetadataBigmapUpdatesByLevel(fmt.Sprintf("%d", level), offset, pageSize)
		if err != nil {
			log.ErrorWithContext(ctx, "failed to fetch token metadata bigmap updates from level: ",
				zap.Error(err), zap.Uint64("level", level), zap.Int("offset", offset), log.SourceTZKT)
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

func (e *TezosEventsEmitter) processTokenEvent(ctx context.Context, event TokenEvent) {
	log.Debug("received event on tezos",
		zap.String("eventType", string(event.EventType)),
		zap.String("from", event.From),
		zap.String("to", event.To),
		zap.String("contractAddress", event.ContractAddress),
		zap.String("tokenID", event.TokenID),
		zap.String("txID", event.TxID),
		zap.String("txTime", event.TxTime.String()),
	)

	if err := e.PushNftEvent(ctx, string(event.EventType), event.From, event.To,
		event.ContractAddress, event.Blockchain, event.TokenID,
		event.TxID, 0, event.TxTime); err != nil {
		log.ErrorWithContext(ctx, "gRPC request failed", zap.Error(err), log.SourceGRPC)
		return
	}

	if event.Level > lastStoppedBlock {
		lastStoppedBlock = event.Level
		if err := e.parameterStore.PutString(ctx, e.lastBlockKeyName, strconv.FormatUint(lastStoppedBlock, 10)); err != nil {
			log.ErrorWithContext(ctx, "error put parameterStore", zap.Error(err), log.SourceGRPC)
			return
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
