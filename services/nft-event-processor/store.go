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
	ID         string      `gorm:"primaryKey;size:255;default:uuid_generate_v4()"`
	Type       string      `gorm:"index"`
	Blockchain string      `gorm:"index"`
	Contract   string      `gorm:"index"`
	TokenID    string      `gorm:"index"`
	From       string      `gorm:"index"`
	To         string      `gorm:"index"`
	Stage      string      `gorm:"index"`
	Status     EventStatus `gorm:"index"`
	CreatedAt  time.Time   `gorm:"default:now()"`
	UpdatedAt  time.Time   `gorm:"default:now()"`
}

type EventStore interface {
	CreateEvent(event NFTEvent) error
	UpdateEvent(id string, updates map[string]interface{}) error
	GetQueueEventByStage(stage int8) (*NFTEvent, error)
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
	return s.db.Model(&NFTEvent{}).Where("id = ?", id).Where("status = ?", EventStatusProcessing).Updates(updates).Error
}

// CompleteEvent marks an event to be done
func (s *PostgresEventStore) CompleteEvent(id string) error {
	return s.db.Updates(&NFTEvent{ID: id, Status: EventStatusProcessed}).Error
}

// GetQueueEventByStage returns all queued events which need to process
func (s *PostgresEventStore) GetQueueEventByStage(stage int8) (*NFTEvent, error) {
	var event NFTEvent

	// TODO: return outdated queued events as well
	err := s.db.Transaction(func(db *gorm.DB) error {
		if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("stage = ?", EventStages[stage]).
			Where("status <> ?", EventStatusProcessed).
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
