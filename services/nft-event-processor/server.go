package main

import "github.com/sirupsen/logrus"

type EventProcessor struct {
	grpcServer     *GRPCServer
	queueProcessor *EventQueueProcessor
}

func NewEventProcessor(network, address string, store EventStore) *EventProcessor {
	queueProcessor := NewEventQueueProcessor(store)
	grpcServer := NewGRPCServer(network, address, queueProcessor)

	return &EventProcessor{
		grpcServer:     grpcServer,
		queueProcessor: queueProcessor,
	}
}

// Run starts event processor server. It spawns a queue processor in the
// background routine and starts up a gRPC server to wait new events.
func (p *EventProcessor) Run() {
	go p.queueProcessor.Run()
	if err := p.grpcServer.Run(); err != nil {
		logrus.WithError(err).Error("gRPC stopped with error")
	}
}
