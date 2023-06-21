package emitter

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

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
func (e *EventsEmitter) PushEvent(ctx context.Context, eventType, fromAddress, toAddress, contractAddress, blockchain, tokenID, txID string, eventIndex uint, txTime time.Time) error {
	eventInput := processor.EventInput{
		Type:       eventType,
		Blockchain: blockchain,
		Contract:   contractAddress,
		From:       fromAddress,
		To:         toAddress,
		TokenID:    tokenID,
		TXID:       txID,
		EventIndex: uint64(eventIndex),
		TXTime:     timestamppb.New(txTime),
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
