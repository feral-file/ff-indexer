package main

// this is left for gRPC integration

import (
	"context"
	"encoding/json"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	log "github.com/bitmark-inc/autonomy-logger"
	pb "github.com/feral-file/ff-indexer/services/event-processor/grpc"
)

type GRPCServer struct {
	server     *grpc.Server
	eventQueue *EventQueue

	network string
	address string
}

func NewGRPCServer(network, address string, eventQueue *EventQueue) *GRPCServer {
	server := grpc.NewServer()
	grpcHandler := NewGRPCHandler(eventQueue)

	reflection.Register(server)
	pb.RegisterEventProcessorServer(server, grpcHandler)

	return &GRPCServer{
		server:     server,
		eventQueue: eventQueue,

		network: network,
		address: address,
	}
}

// Run starts a gRPC server for the event processor
func (s *GRPCServer) Run() error {
	listener, err := net.Listen(s.network, s.address)
	if err != nil {
		log.Panic("server interrupted", zap.Error(err))
	}

	err = s.server.Serve(listener)

	return err
}

type GRPCHandler struct {
	pb.UnimplementedEventProcessorServer

	eventQueue *EventQueue
}

func NewGRPCHandler(eventQueue *EventQueue) *GRPCHandler {
	return &GRPCHandler{
		eventQueue: eventQueue,
	}
}

// PushNftEvent handles PushNftEvent requests and save it to event store
func (t *GRPCHandler) PushNftEvent(
	_ context.Context,
	i *pb.NftEventInput,
) (*pb.EventOutput, error) {
	log.Debug("receive event input", zap.Any("input", i))

	if err := t.eventQueue.PushNftEvent(NFTEvent{
		Type:       i.Type,
		Blockchain: i.Blockchain,
		Contract:   i.Contract,
		TokenID:    i.TokenID,
		From:       i.From,
		To:         i.To,
		TXID:       i.TXID,
		TXTime:     i.TXTime.AsTime(),
		EventIndex: uint(i.EventIndex),
		Stage:      NftEventStages[1],
		Status:     NftEventStatusCreated,
	}); err != nil {
		return nil, err
	}

	output := &pb.EventOutput{
		Result: "successfully",
		Status: 200,
	}

	return output, nil
}

// PushSeriesRegistryEvent handles PushSeriesRegistryEvent requests and save it to event store
func (t *GRPCHandler) PushSeriesRegistryEvent(
	_ context.Context,
	i *pb.SeriesRegistryEventInput,
) (*pb.EventOutput, error) {
	log.Debug("receive event input", zap.Any("input", i))

	se := SeriesRegistryEvent{
		Type:       i.Type,
		Contract:   i.Contract,
		TxID:       i.TxID,
		TxTime:     i.TxTime.AsTime(),
		EventIndex: uint(i.EventIndex),
		Stage:      SeriesEventStages[SeriesRegistryEventStageInit],
		Status:     SeriesRegistryEventStatusCreated,
	}
	if i.Data != nil {
		b, err := json.Marshal(i.Data.AsMap())
		if err != nil {
			return nil, err
		}
		se.Data = b
	}

	if err := t.eventQueue.PushSeriesRegistryEvent(se); err != nil {
		return nil, err
	}

	output := &pb.EventOutput{
		Result: "successfully",
		Status: 200,
	}

	return output, nil
}
