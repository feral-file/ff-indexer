package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/jackc/pgconn"
	"github.com/spf13/viper"
	"go.uber.org/zap"
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

// NFTEvent is the model for processed token events
type ArchivedNFTEvent struct {
	ID         string      `gorm:"primaryKey;size:255;default:uuid_generate_v4()"`
	Type       string      `gorm:"index"`
	Blockchain string      `gorm:"index"`
	Contract   string      `gorm:"index"`
	TokenID    string      `gorm:"index"`
	From       string      `gorm:"index"`
	To         string      `gorm:"index"`
	TXID       string      `gorm:"index"`
	TXTime     time.Time   `gorm:"index"`
	Stage      string      `gorm:"index"`
	Status     EventStatus `gorm:"index"`
	CreatedAt  time.Time   `gorm:"default:now()"`
	UpdatedAt  time.Time   `gorm:"default:now()"`
}

func (ArchivedNFTEvent) TableName() string {
	return "nft_events"
}

// NFTEvent is the model for token events
type NFTEvent struct {
	ID         string      `gorm:"primaryKey;size:255;default:uuid_generate_v4()"`
	Type       string      `gorm:"index:idx_event,unique"`
	Blockchain string      `gorm:"index:idx_event,unique"`
	Contract   string      `gorm:"index:idx_event,unique"`
	TokenID    string      `gorm:"index:idx_event,unique"`
	From       string      `gorm:"index:idx_event,unique"`
	To         string      `gorm:"index:idx_event,unique"`
	TXID       string      `gorm:"index:idx_event,unique"`
	EventIndex uint        `gorm:"index:idx_event,unique"`
	TXTime     time.Time   `gorm:"index:idx_event,unique"`
	Stage      string      `gorm:"index"`
	Status     EventStatus `gorm:"index"`
	CreatedAt  time.Time   `gorm:"default:now()"`
	UpdatedAt  time.Time   `gorm:"default:now()"`
}

func (NFTEvent) TableName() string {
	return "new_nft_events"
}

// EventTx is an transaction object with event values
type EventTx struct {
	*gorm.DB
	Event NFTEvent
}

// UpdateEvent updates events by given stage or status
func (tx *EventTx) UpdateEvent(stage, status string) error {
	updates := map[string]interface{}{}
	if stage != "" {
		updates["stage"] = stage
	}

	if status != "" {
		updates["status"] = status
	}

	if len(updates) == 0 {
		return fmt.Errorf("nothing for update to a nft event")
	}

	return tx.DB.Model(&NFTEvent{}).Where("id = ?", tx.Event.ID).Updates(updates).Error
}

// DeleteEvent delete the event by the id
func (tx *EventTx) ArchiveNFTEvent() error {
	archivedEvent := ArchivedNFTEvent{
		Type:       tx.Event.Type,
		Blockchain: tx.Event.Blockchain,
		Contract:   tx.Event.Contract,
		TokenID:    tx.Event.TokenID,
		From:       tx.Event.From,
		To:         tx.Event.To,
		TXID:       tx.Event.TXID,
		TXTime:     tx.Event.TXTime,
		Status:     EventStatusProcessed,
	}

	if err := tx.DB.Save(&archivedEvent).Error; err != nil {
		return err
	}

	return tx.DB.Where("id = ?", tx.Event.ID).Delete(&NFTEvent{}).Error
}

func NewEventTx(DB *gorm.DB, event NFTEvent) *EventTx {
	return &EventTx{
		DB:    DB,
		Event: event,
	}
}

type EventStore interface {
	CreateEvent(event NFTEvent) error
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

// CreateEvent add a new event into event store.
func (s *PostgresEventStore) CreateEvent(event NFTEvent) error {
	err := s.db.Save(&event).Error

	var pgError *pgconn.PgError
	if err != nil && errors.As(err, &pgError) {
		if pgError.Code == "23505" { // Unique violation error code
			log.Warn("duplicated event", zap.Error(err))
			return nil
		}
	}

	return err
}

// FilterOption is an abstraction to help filtering events with
// specific conditions
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

// GetEventTransaction returns an EventTx
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
