package main

import (
	"context"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"

	log "github.com/bitmark-inc/autonomy-logger"
	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
)

func (e *EventProcessor) refreshTokenData(ctx context.Context, event NFTEvent) error {
	blockchain := event.Blockchain
	contract := event.Contract
	tokenID := event.TokenID
	indexID := indexer.TokenIndexID(blockchain, contract, tokenID)

	token, err := e.indexerGRPC.GetTokenByIndexID(ctx, indexID)
	if err != nil {
		if grpcError, ok := status.FromError(err); !ok || grpcError.Message() != "token does not exist" {
			log.Error("fail to query token from indexer", zap.Error(err))
			return err
		}
	}

	// check if a token is existent
	// if existent, update the token
	// if not, ignore the process
	if token != nil {
		indexerWorker.StartIndexTokenWorkflow(ctx, e.worker, event.To, event.Contract, event.TokenID, true, false)
	}

	return nil
}

// RefreshTokenData process the EventTypeTokenUpdated and trigger cadence workflow to refresh the token data
func (e *EventProcessor) RefreshTokenData(ctx context.Context) {
	e.StartWorker(ctx,
		StageInit, StageDoubleSync,
		[]EventType{EventTypeTokenUpdated},
		viper.GetInt64("events.check_interval_seconds.token_updated"),
		viper.GetInt64("events.process_delay_seconds.token_updated"),
		e.refreshTokenData,
	)
}

// DoubleSyncTokenData process the EventTypeTokenUpdated and to ensure a token is corrected updates
func (e *EventProcessor) DoubleSyncTokenData(ctx context.Context) {
	e.StartWorker(ctx,
		StageDoubleSync, StageDone,
		[]EventType{EventTypeTokenUpdated},
		0, 60, e.refreshTokenData,
	)
}
