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
	ProcessEvent(ctx context.Context, option EventProcessOption) (bool, error)
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

type QueryOption interface {
	Apply(tx *gorm.DB) *gorm.DB
}

type FilterOption struct {
	Statement  string
	Argumenets []interface{}
}

func (f FilterOption) Apply(tx *gorm.DB) *gorm.DB {
	return tx.Where(f.Statement, f.Argumenets...)
}

func Filter(statement string, args ...interface{}) FilterOption {
	return FilterOption{
		Statement:  statement,
		Argumenets: args,
	}
}

// EventProcessOption includes information about filters to target an event, a processor for
// process an event and a struct of state updates
type EventProcessOption struct {
	Filters        []QueryOption
	Processor      func(event NFTEvent) error
	CompleteUpdate NFTEvent
}

// ProcessEvent process an event based on processing options
func (s *PostgresEventStore) ProcessEvent(ctx context.Context, option EventProcessOption) (bool, error) {
	var hasEvent bool
	return hasEvent, s.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		var event NFTEvent

		q := db.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
		for _, filter := range option.Filters {
			q = filter.Apply(q)
		}

		// final query
		q = q.Order("created_at asc").First(&event)

		if err := q.Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}

		hasEvent = true
		if err := option.Processor(event); err != nil {
			// FIXME provide a fine-grain EventStatus. For example, EventStatusError
			log.Error("fail to process event", zap.Error(err))
		}

		return db.Model(&NFTEvent{}).Where("id = ?", event.ID).
			Updates(option.CompleteUpdate).Error
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
			Order("created_at asc").Limit(1).Find(&event).Error; err != nil {
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
