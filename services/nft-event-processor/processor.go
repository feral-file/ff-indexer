package main

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
