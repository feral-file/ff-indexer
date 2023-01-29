package main

// this is left for gRPC integration

import (
	"context"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
)

type GRPCServer struct {
	server         *grpc.Server
	queueProcessor *EventQueueProcessor

	network string
	address string
}

func NewGRPCServer(network, address string, queueProcessor *EventQueueProcessor) *GRPCServer {
	grpcServer := grpc.NewServer()
	grpcHandler := NewGRPCHandler(queueProcessor)

	reflection.Register(grpcServer)
	processor.RegisterEventProcessorServer(grpcServer, grpcHandler)

	return &GRPCServer{
		server:         grpcServer,
		queueProcessor: queueProcessor,

		network: network,
		address: address,
	}
}

// Run starts a gRPC server for the event processor
func (s *GRPCServer) Run() error {
	grpcListener, err := net.Listen(s.network, s.address)
	if err != nil {
		log.Panic("server interrupted", zap.Error(err))
	}

	err = s.server.Serve(grpcListener)

	return err
}

type GRPCHandler struct {
	processor.UnimplementedEventProcessorServer

	queueProcessor *EventQueueProcessor
}

func NewGRPCHandler(queueProcessor *EventQueueProcessor) *GRPCHandler {
	return &GRPCHandler{
		queueProcessor: queueProcessor,
	}
}

// PushEvent handles PushEvent requests and save it to event store
func (t *GRPCHandler) PushEvent(
	ctx context.Context,
	i *processor.EventInput,
) (*processor.EventOutput, error) {
	log.Debug("receive event input", zap.Any("input", i))

	if err := t.queueProcessor.PushEvent(NFTEvent{
		EventType:  i.EventType,
		Blockchain: i.Blockchain,
		Contract:   i.Contract,
		TokenID:    i.TokenID,
		From:       i.From,
		To:         i.To,
		Status:     EventStatusCreated,
		Stage:      EventStages[1],
	}); err != nil {
		return nil, err
	}

	output := &processor.EventOutput{
		Result: "successfully",
		Status: 200,
	}

	return output, nil
}
