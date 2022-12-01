package main

import (
	"time"

	"github.com/sirupsen/logrus"
)

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

// Run start a loop to continuously consuming queud event
func (p *EventQueueProcessor) Run() {

	for {
		logrus.Trace("start fetching new event")
		// get a queued event from the store
		event, err := p.store.GetQueuedEvent()
		if err != nil {
			// error on query
			logrus.WithError(err).Error("fail to get event from event store")
			time.Sleep(5 * time.Second)
			continue
		}

		if event == nil {
			// no events
			time.Sleep(5 * time.Second)
			continue
		}

		// TODO: items
		// update latest owner into mongodb
		// trigger full updates for the token
		// send notification
		// send to feed server

		if err := p.store.CompleteEvent(event.ID); err != nil {
			logrus.WithError(err).Error("fail to mark an event completed")
		}
		// time.Sleep(5 * time.Second)
		// continue
	}
}
