package main

import (
	"context"
)

type EventQueue struct {
	store EventStore
}

func NewEventQueue(store EventStore) *EventQueue {
	return &EventQueue{
		store: store,
	}
}

// PushEvent adds an event into event store
func (q *EventQueue) PushEvent(event NFTEvent) error {
	return q.store.CreateEvent(event)
}

func (q *EventQueue) GetEventTransaction(ctx context.Context, filters ...FilterOption) (*EventTx, error) {
	return q.store.GetEventTransaction(ctx, filters...)
}

// GetArchivedEvents returns all archived events by filters
func (q *EventQueue) GetArchivedEvents(ctx context.Context, pagination *Pagination, filters ...FilterOption) ([]ArchivedNFTEvent, error) {
	return q.store.GetArchivedEvents(ctx, pagination, filters...)
}
