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

// PushNftEvent adds a nft event into event store
func (q *EventQueue) PushNftEvent(event NFTEvent) error {
	return q.store.CreateNftEvent(event)
}

// PushSeriesRegistryEvent adds a series event into event store
func (q *EventQueue) PushSeriesRegistryEvent(event SeriesRegistryEvent) error {
	return q.store.CreateSeriesRegistryEvent(event)
}

func (q *EventQueue) GetNftEventTransaction(ctx context.Context, filters ...FilterOption) (*NftEventTx, error) {
	return q.store.GetNftEventTransaction(ctx, filters...)
}

func (q *EventQueue) GetSeriesRegistryEventTransaction(ctx context.Context, filters ...FilterOption) (*SeriesRegistryEventTx, error) {
	return q.store.GetSeriesRegistryEventTransaction(ctx, filters...)
}
