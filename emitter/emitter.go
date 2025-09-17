package emitter

import (
	"context"
	"fmt"
	"time"

	pb "github.com/feral-file/ff-indexer/services/event-processor/grpc"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EventsEmitter struct {
	grpcClient pb.EventProcessorClient
}

func New(grpcClient pb.EventProcessorClient) EventsEmitter {
	return EventsEmitter{
		grpcClient: grpcClient,
	}
}

// PushNftEvent submits nft events to event processor
func (e *EventsEmitter) PushNftEvent(ctx context.Context, eventType, fromAddress, toAddress, contractAddress, blockchain, tokenID, txID string, eventIndex uint, txTime time.Time) error {
	eventInput := pb.NftEventInput{
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

// PushSeriesRegistryEvent submits series registry events to event processor
func (e *EventsEmitter) PushSeriesRegistryEvent(ctx context.Context, eventType, contractAddress, txID string, data map[string]interface{}, eventIndex uint, txTime time.Time) error {
	var sd *structpb.Struct
	if data != nil {
		nd, err := structpb.NewStruct(data)
		if err != nil {
			return err
		}
		sd = nd
	}
	eventInput := pb.SeriesRegistryEventInput{
		Type:       eventType,
		Contract:   contractAddress,
		Data:       sd,
		TxID:       txID,
		EventIndex: uint64(eventIndex),
		TxTime:     timestamppb.New(txTime),
	}

	r, err := e.grpcClient.PushSeriesRegistryEvent(ctx, &eventInput)
	if err != nil {
		return err
	}

	if r.Status != 200 {
		return fmt.Errorf("gRPC response status not 200 ")
	}

	return nil
}
