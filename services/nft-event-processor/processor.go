package main

import "context"

type EventQueueProcessor struct {
	store EventStore
}

func NewEventQueueProcessor(store EventStore) *EventQueueProcessor {
	return &EventQueueProcessor{
		store: store,
	}
}

// PushEvent adds an event into event store
func (p *EventQueueProcessor) PushEvent(event NFTEvent) error {
	return p.store.CreateEvent(event)
}

// PushEvent adds an event into event store
func (p *EventQueueProcessor) ProcessTokenUpdatedEvent(ctx context.Context, processor func(event NFTEvent) error) (bool, error) {
	return p.store.ProcessTokenUpdatedEvent(ctx, processor)
}
