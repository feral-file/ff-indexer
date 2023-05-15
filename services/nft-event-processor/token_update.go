package main

import (
	"context"

	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
)

func (e *EventProcessor) refreshTokenData(ctx context.Context, event NFTEvent) error {
	indexerWorker.StartIndexTokenWorkflow(ctx, e.worker, event.To, event.Contract, event.TokenID, true, false)
	return nil
}

// UpdateOwnerAndProvenance trigger cadence to update owner and provenance of token
func (e *EventProcessor) RefreshTokenData(ctx context.Context) {
	e.StartWorker(ctx,
		1, 0,
		[]EventType{EventTypeTokenUpdated},
		e.refreshTokenData,
	)
}
