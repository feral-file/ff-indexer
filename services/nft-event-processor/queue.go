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

// PushSeriesEvent adds a series event into event store
func (q *EventQueue) PushSeriesEvent(event SeriesEvent) error {
	return q.store.CreateSeriesEvent(event)
}

func (q *EventQueue) GetNftEventTransaction(ctx context.Context, filters ...FilterOption) (*NftEventTx, error) {
	return q.store.GetNftEventTransaction(ctx, filters...)
}

func (q *EventQueue) GetSeriesEventTransaction(ctx context.Context, filters ...FilterOption) (*SeriesEventTx, error) {
	return q.store.GetSeriesEventTransaction(ctx, filters...)
}
