package main

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EventStatus string

const (
	EventStatusCreated    EventStatus = "created"
	EventStatusProcessing EventStatus = "processing"
	EventStatusProcessed  EventStatus = "processed"
)

type NFTEvent struct {
	ID         string `gorm:"primaryKey;size:255;default:uuid_generate_v4()"`
	EventType  string
	Blockchain string
	Contract   string
	TokenID    string
	From       string
	To         string
	Stage      string
	Status     EventStatus `gorm:"index"`
	CreatedAt  time.Time   `gorm:"default:now()"`
	UpdatedAt  time.Time   `gorm:"default:now()"`
}

type EventStore interface {
	CreateEvent(event NFTEvent) error
	UpdateEvent(id string, updates map[string]interface{}) error
	GetQueuedEvent() (*NFTEvent, error)
	CompleteEvent(id string) error
}

type PostgresEventStore struct {
	db *gorm.DB
}

func NewPostgresEventStore(db *gorm.DB) *PostgresEventStore {
	return &PostgresEventStore{
		db: db,
	}
}

// TODO: Do dedup for duplicated events.
// CreateEvent add a new event into event store.
func (s *PostgresEventStore) CreateEvent(event NFTEvent) error {
	return s.db.Save(&event).Error
}

// UpdateEvent updates attributes for a event.
func (s *PostgresEventStore) UpdateEvent(id string, updates map[string]interface{}) error {
	return s.db.Updates(&NFTEvent{ID: id, Status: EventStatusProcessed}).Error
}

// CompleteEvent marks an event to be done
func (s *PostgresEventStore) CompleteEvent(id string) error {
	return s.db.Updates(&NFTEvent{ID: id, Status: EventStatusProcessed}).Error
}

// GetQueuedEvent returns all queued events which need to process
func (s *PostgresEventStore) GetQueuedEvent() (*NFTEvent, error) {
	var event NFTEvent

	// TODO: return outdated queued events as well
	err := s.db.Transaction(func(db *gorm.DB) error {
		if err := db.Debug().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(&NFTEvent{Status: EventStatusCreated}).
			Order("created_at asc").First(&event).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}

		event.Status = EventStatusProcessing
		return db.Save(event).Error
	})
	if err != nil {
		return nil, err
	}

	if event.ID == "" {
		return nil, nil
	}

	return &event, nil
}

// AutoMigrate is a help function that update db when the schema changed.
func (s *PostgresEventStore) AutoMigrate() error {
	return s.db.AutoMigrate(&NFTEvent{})
}
