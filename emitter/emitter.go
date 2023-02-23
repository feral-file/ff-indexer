package emitter

import (
	"context"
	"fmt"

	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
)

type EventsEmitter struct {
	grpcClient processor.EventProcessorClient
}

func New(grpcClient processor.EventProcessorClient) EventsEmitter {
	return EventsEmitter{
		grpcClient: grpcClient,
	}
}

// PushEvent submits events to event processor
func (e *EventsEmitter) PushEvent(ctx context.Context, eventType, fromAddress, toAddress, contractAddress, blockchain, tokenID string) error {
	eventInput := processor.EventInput{
		Type:       eventType,
		Blockchain: blockchain,
		Contract:   contractAddress,
		From:       fromAddress,
		To:         toAddress,
		TokenID:    tokenID,
	}

	r, err := e.grpcClient.PushEvent(ctx, &eventInput)
	if err != nil {
		return err
	}

	if r.Status != 200 {
		return fmt.Errorf("gRPC response status not 200 ")
	}

	return nil
}
