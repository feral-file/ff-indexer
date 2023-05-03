package main

import (
	"context"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/bitmark-inc/nft-indexer/log"
)

type EventType string

const (
	EventTypeMint         EventType = "mint"
	EventTypeTransfer     EventType = "transfer"
	EventTypeTokenUpdated EventType = "token_updated"
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
	UpdateEventByStatus(id string, status EventStatus, updates map[string]interface{}) error
	GetEventByStage(stage int8) (*NFTEvent, error)
	ProcessTokenUpdatedEvent(ctx context.Context, processor func(event NFTEvent) error) (bool, error)
}

type PostgresEventStore struct {
	db *gorm.DB
}

func NewPostgresEventStore(db *gorm.DB) *PostgresEventStore {
	if viper.GetBool("debug") {
		db = db.Debug()
	}

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
	return s.db.Model(&NFTEvent{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateEvent updates attributes for a event.
func (s *PostgresEventStore) UpdateEventByStatus(id string, status EventStatus, updates map[string]interface{}) error {
	return s.db.Model(&NFTEvent{}).Where("id = ?", id).Where("status = ?", status).Updates(updates).Error
}

// ProcessTokenUpdatedEvent searches and processes a token_updated event
func (s *PostgresEventStore) ProcessTokenUpdatedEvent(ctx context.Context,
	processor func(event NFTEvent) error) (bool, error) {
	var hasEvent bool
	return hasEvent, s.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		var event NFTEvent
		if err := db.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("type = ?", EventTypeTokenUpdated).
			Where("status <> ?", EventStatusProcessed).
			Order("created_at asc").First(&event).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}

		hasEvent = true
		if err := processor(event); err != nil {
			// FIXME provide a fine-grain EventStatus. For example, EventStatusError
			log.Error("fail to process event", zap.Error(err))
		}

		return db.Model(&NFTEvent{}).Where("id = ?", event.ID).Update("status", EventStatusProcessed).Error
	})
}

// GetEventByStage returns all queued events which need to process
func (s *PostgresEventStore) GetEventByStage(stage int8) (*NFTEvent, error) {
	var event NFTEvent

	// TODO: return outdated queued events as well
	err := s.db.Transaction(func(db *gorm.DB) error {
		if err := db.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("stage = ?", EventStages[stage]).
			Where("type <> ?", EventTypeTokenUpdated).
			Where("status <> ?", EventStatusProcessed).
			Order("created_at asc").First(&event).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}

		event.Status = EventStatusProcessing
		return db.Model(&NFTEvent{}).Where("id = ?", event.ID).Update("status", EventStatusProcessing).Error
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
