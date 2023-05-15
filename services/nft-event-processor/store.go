package main

import (
	"context"
	"time"

	"github.com/spf13/viper"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	EventStatusFailed     EventStatus = "failed"
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

type EventTx struct {
	*gorm.DB
	Event NFTEvent
}

func (tx *EventTx) UpdateEvent(stage, status string) error {
	updates := map[string]interface{}{}
	if stage != "" {
		updates["stage"] = stage
	}

	if status != "" {
		updates["status"] = status
	}

	return tx.DB.Model(&NFTEvent{}).Where("id = ?", tx.Event.ID).Updates(updates).Error
}

func NewEventTx(DB *gorm.DB, event NFTEvent) *EventTx {
	return &EventTx{
		DB:    DB,
		Event: event,
	}
}

type EventStore interface {
	CreateEvent(event NFTEvent) error
	UpdateEvent(id string, updates map[string]interface{}) error
	UpdateEventByStatus(id string, status EventStatus, updates map[string]interface{}) error
	GetEventTransaction(ctx context.Context, filters ...FilterOption) (*EventTx, error)
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
	Processor      processorFunc
	CompleteUpdate NFTEvent
}

// GetEventTransaction returns an event and its lock transaction
func (s *PostgresEventStore) GetEventTransaction(ctx context.Context, filters ...FilterOption) (*EventTx, error) {
	var event NFTEvent

	tx := s.db.WithContext(ctx).Begin()
	if err := tx.Error; err != nil {
		return nil, err
	}

	q := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
	for _, filter := range filters {
		q = filter.Apply(q)
	}

	// final query
	q = q.Order("created_at asc").First(&event)
	if err := q.Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	return NewEventTx(tx, event), nil
}

// AutoMigrate is a help function that update db when the schema changed.
func (s *PostgresEventStore) AutoMigrate() error {
	return s.db.AutoMigrate(&NFTEvent{})
}
