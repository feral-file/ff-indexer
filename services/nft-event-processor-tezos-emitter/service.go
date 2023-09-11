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
)

const maxMessageSize = 1 << 20 // 1MiB

type TezosEventsEmitter struct {
	lastBlockKeyName string
	parameterStore   *ssm.ParameterStore

	grpcClient processor.EventProcessorClient
	emitter.EventsEmitter
	tzktWebsocketURL string

	transferChan chan TokenTransfer
}

func NewTezosEventsEmitter(
	lastBlockKeyName string,
	parameterStore *ssm.ParameterStore,
	grpcClient processor.EventProcessorClient,
	tzktWebsocketURL string,
) *TezosEventsEmitter {
	return &TezosEventsEmitter{
		lastBlockKeyName: lastBlockKeyName,
		parameterStore:   parameterStore,
		grpcClient:       grpcClient,
		EventsEmitter:    emitter.New(grpcClient),
		tzktWebsocketURL: tzktWebsocketURL,

		transferChan: make(chan TokenTransfer, 100),
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

	for _, t := range res.Data {
		e.transferChan <- t
	}

	//save laststop block
	lastBlock := res.Data[len(res.Data)-1].Level
	if err := e.parameterStore.Put(context.Background(), e.lastBlockKeyName, strconv.FormatUint(lastBlock, 10)); err != nil {
		log.Error("error put parameterStore", zap.Error(err), log.SourceGRPC)
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

		if err := e.PushEvent(ctx, eventType, fromAddress, t.To.Address,
			t.Token.Contract.Address, utils.TezosBlockchain, t.Token.TokenID, strconv.FormatUint(t.TransactionID, 10), 0, t.Timestamp); err != nil {
			log.Error("gRPC request failed", zap.Error(err), log.SourceGRPC)
		}
	}
}
