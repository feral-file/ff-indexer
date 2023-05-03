package main

import "context"

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

func (q *EventQueue) ProcessTokenUpdatedEvent(ctx context.Context, processor func(event NFTEvent) error) (bool, error) {
	return q.store.ProcessTokenUpdatedEvent(ctx, processor)
}

func (q *EventQueue) UpdateEvent(id string, updates map[string]interface{}) error {
	return q.store.UpdateEventByStatus(id, EventStatusProcessing, updates)
}

func (q *EventQueue) CompleteEvent(id string) error {
	return q.store.UpdateEvent(id, map[string]interface{}{
		"status": EventStatusProcessed,
	})
}

func (q *EventQueue) GetEventByStage(stage int8) (*NFTEvent, error) {
	return q.store.GetEventByStage(stage)
}
