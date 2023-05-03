package main

import (
	"context"
	"time"

	"go.uber.org/zap"

	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
	"github.com/bitmark-inc/nft-indexer/log"
)

// UpdateOwnerAndProvenance trigger cadence to update owner and provenance of token
func (e *EventProcessor) RefreshTokenData(ctx context.Context) {

	for {
		hasEvent, err := e.eventQueue.ProcessTokenUpdatedEvent(ctx, func(event NFTEvent) error {
			indexerWorker.StartIndexTokenWorkflow(ctx, e.worker, event.To, event.Contract, event.TokenID, true, false)
			return nil
		})

		if err != nil {
			log.Error("fail to process token updated", zap.Error(err))
		}

		if !hasEvent {
			time.Sleep(WaitingTime)
		}
	}

}
