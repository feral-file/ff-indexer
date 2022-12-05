package rpc_services

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
)

type EventProcessor struct {
	processor.UnimplementedEventProcessorServer
}

func NewEventProcessor() *EventProcessor {
	return &EventProcessor{}
}

func (t *EventProcessor) PushEvent(
	ctx context.Context,
	i *processor.EventInput,
) (*processor.EventOutput, error) {

	log.WithFields(log.Fields{
		"input": i,
	}).Info("receive event input")

	// TODO: implement insert data to queue
	output := &processor.EventOutput{
		Result: "successfully",
		Status: 200,
	}

	return output, nil
}
