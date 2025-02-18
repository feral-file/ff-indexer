package emitter

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
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

// PushNftEvent submits nft events to event processor
func (e *EventsEmitter) PushNftEvent(ctx context.Context, eventType, fromAddress, toAddress, contractAddress, blockchain, tokenID, txID string, eventIndex uint, txTime time.Time) error {
	eventInput := processor.NftEventInput{
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

	r, err := e.grpcClient.PushNftEvent(ctx, &eventInput)
	if err != nil {
		return err
	}

	if r.Status != 200 {
		return fmt.Errorf("gRPC response status not 200 ")
	}

	return nil
}

// PushSeriesEvent submits series events to event processor
func (e *EventsEmitter) PushSeriesEvent(ctx context.Context, eventType, contractAddress, txID string, data *map[string]interface{}, eventIndex uint, txTime time.Time) error {
	var sd *structpb.Struct
	if data != nil {
		nd, err := structpb.NewStruct(*data)
		if err != nil {
			return err
		}
		sd = nd
	}
	eventInput := processor.SeriesEventInput{
		Type:       eventType,
		Contract:   contractAddress,
		Data:       sd,
		TXID:       txID,
		EventIndex: uint64(eventIndex),
		TXTime:     timestamppb.New(txTime),
	}

	r, err := e.grpcClient.PushSeriesEvent(ctx, &eventInput)
	if err != nil {
		return err
	}

	if r.Status != 200 {
		return fmt.Errorf("gRPC response status not 200 ")
	}

	return nil
}
