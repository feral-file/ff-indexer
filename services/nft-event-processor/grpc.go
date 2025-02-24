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
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
)

type GRPCServer struct {
	server         *grpc.Server
	queueProcessor *EventQueue

	network string
	address string
}

func NewGRPCServer(network, address string, queueProcessor *EventQueue) *GRPCServer {
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

	queueProcessor *EventQueue
}

func NewGRPCHandler(queueProcessor *EventQueue) *GRPCHandler {
	return &GRPCHandler{
		queueProcessor: queueProcessor,
	}
}

// PushNftEvent handles PushNftEvent requests and save it to event store
func (t *GRPCHandler) PushNftEvent(
	_ context.Context,
	i *processor.NftEventInput,
) (*processor.EventOutput, error) {
	log.Debug("receive event input", zap.Any("input", i))

	if err := t.queueProcessor.PushNftEvent(NFTEvent{
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

	output := &processor.EventOutput{
		Result: "successfully",
		Status: 200,
	}

	return output, nil
}

// PushEvent handles PushEvent requests and save it to event store
func (t *GRPCHandler) PushSeriesEvent(
	_ context.Context,
	i *processor.SeriesEventInput,
) (*processor.EventOutput, error) {
	log.Debug("receive event input", zap.Any("input", i))

	se := SeriesEvent{
		Type:       i.Type,
		Contract:   i.Contract,
		TXID:       i.TXID,
		TXTime:     i.TXTime.AsTime(),
		EventIndex: uint(i.EventIndex),
		Stage:      SeriesEventStages[SeriesEventStageInit],
		Status:     SeriesEventStatusCreated,
	}
	if i.Data != nil {
		b, err := json.Marshal(i.Data.AsMap())
		if err != nil {
			return nil, err
		}
		se.Data = b
	}

	if err := t.queueProcessor.PushSeriesEvent(se); err != nil {
		return nil, err
	}

	output := &processor.EventOutput{
		Result: "successfully",
		Status: 200,
	}

	return output, nil
}
