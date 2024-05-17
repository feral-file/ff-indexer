package main

// this is left for gRPC integration

import (
	"context"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/timestamppb"

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

// PushEvent handles PushEvent requests and save it to event store
func (t *GRPCHandler) PushEvent(
	_ context.Context,
	i *processor.EventInput,
) (*processor.EventOutput, error) {
	log.Debug("receive event input", zap.Any("input", i))

	if err := t.queueProcessor.PushEvent(NFTEvent{
		Type:       i.Type,
		Blockchain: i.Blockchain,
		Contract:   i.Contract,
		TokenID:    i.TokenID,
		From:       i.From,
		To:         i.To,
		TXID:       i.TXID,
		TXTime:     i.TXTime.AsTime(),
		EventIndex: uint(i.EventIndex),
		Stage:      EventStages[1],
		Status:     EventStatusCreated,
	}); err != nil {
		return nil, err
	}

	output := &processor.EventOutput{
		Result: "successfully",
		Status: 200,
	}

	return output, nil
}

// GetArchivedEvents handles GetArchivedEvents requests and return archived events
func (t *GRPCHandler) GetArchivedEvents(
	_ context.Context,
	i *processor.ArchivedEventInput) (*processor.ArchivedEvents, error) {
	filters := mapGrpcArchivedEventInputToFilters(i)
	var pagination *Pagination
	if nil != i.Pagination {
		pagination = &Pagination{
			Limit:  int(i.Pagination.Limit),
			Offset: int(i.Pagination.Offset),
		}
	}
	events, err := t.queueProcessor.GetArchivedEvents(
		context.Background(),
		pagination,
		filters...)

	if err != nil {
		return nil, err
	}

	output := &processor.ArchivedEvents{
		Events: make([]*processor.ArchivedEvent, len(events)),
	}

	for i, event := range events {
		output.Events[i] = &processor.ArchivedEvent{
			ID:         event.ID,
			Type:       event.Type,
			Blockchain: event.Blockchain,
			Contract:   event.Contract,
			TokenID:    event.TokenID,
			From:       event.From,
			To:         event.To,
			TxID:       event.TXID,
			TxTime:     timestamppb.New(event.TXTime),
			CreatedAt:  timestamppb.New(event.CreatedAt),
			UpdatedAt:  timestamppb.New(event.UpdatedAt),
			Status:     string(event.Status),
		}
	}

	return output, nil
}

func mapGrpcArchivedEventInputToFilters(i *processor.ArchivedEventInput) []FilterOption {
	if nil == i {
		return nil
	}

	filters := []FilterOption{}
	if nil != i.Blockchain {
		filters = append(filters, Filter("blockchain = ?", *i.Blockchain))
	}
	if nil != i.Contract {
		filters = append(filters, Filter("contract = ?", *i.Contract))
	}
	if nil != i.TokenID {
		filters = append(filters, Filter("token_id = ?", *i.TokenID))
	}
	if nil != i.From {
		filters = append(filters, Filter("from = ?", *i.From))
	}
	if nil != i.To {
		filters = append(filters, Filter("to = ?", *i.To))
	}
	if nil != i.Type {
		filters = append(filters, Filter("type = ?", *i.Type))
	}
	if nil != i.Status {
		filters = append(filters, Filter("status = ?", *i.Status))
	}
	if nil != i.Stage {
		filters = append(filters, Filter("stage = ?", *i.Stage))
	}
	if nil != i.TxID {
		filters = append(filters, Filter("tx_id = ?", *i.TxID))
	}
	if nil != i.TxFromTime {
		filters = append(filters, Filter("tx_time >= ?", i.TxFromTime.AsTime()))
	}
	if nil != i.TxToTime {
		filters = append(filters, Filter("tx_time <= ?", i.TxToTime.AsTime()))
	}

	return filters
}
