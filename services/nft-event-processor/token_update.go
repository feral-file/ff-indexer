package main

import (
	"context"

	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
)

func (e *EventProcessor) refreshTokenData(ctx context.Context, event NFTEvent) error {
	indexerWorker.StartIndexTokenWorkflow(ctx, e.worker, event.To, event.Contract, event.TokenID, true, false)
	return nil
}

// RefreshTokenData process the EventTypeTokenUpdated and trigger cadence workflow to refresh the token data
func (e *EventProcessor) RefreshTokenData(ctx context.Context) {
	e.StartWorker(ctx,
		1, 0,
		[]EventType{EventTypeTokenUpdated},
		e.refreshTokenData,
	)
}
