package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/jackc/pgconn"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	log "github.com/bitmark-inc/autonomy-logger"
)

type NftEventType string

const (
	NftEventTypeMint         NftEventType = "mint"
	NftEventTypeBurned       NftEventType = "burned"
	NftEventTypeTransfer     NftEventType = "transfer"
	NftEventTypeTokenUpdated NftEventType = "token_updated"
)

type NftEventStatus string

const (
	NftEventStatusCreated    NftEventStatus = "created"
	NftEventStatusProcessing NftEventStatus = "processing"
	NftEventStatusProcessed  NftEventStatus = "processed"
	NftEventStatusFailed     NftEventStatus = "failed"
)

type SeriesRegistryEventType string

const (
	SeriesRegistryEventTypeRegisterSeries      SeriesRegistryEventType = "register_series"
	SeriesRegistryEventTypeUpdateSeries        SeriesRegistryEventType = "update_series"
	SeriesRegistryEventTypeDeleteSeries        SeriesRegistryEventType = "delete_series"
	SeriesRegistryEventTypeUpdateArtistAddress SeriesRegistryEventType = "update_artist_address"
	SeriesRegistryEventTypeOptInCollaboration  SeriesRegistryEventType = "opt_in_collaboration"
	SeriesRegistryEventTypeOptOutSeries        SeriesRegistryEventType = "opt_out_series"
	SeriesRegistryEventTypeAssignSeries        SeriesRegistryEventType = "assign_series"
)

type SeriesRegistryEventStatus string

const (
	SeriesRegistryEventStatusCreated    SeriesRegistryEventStatus = "created"
	SeriesRegistryEventStatusProcessing SeriesRegistryEventStatus = "processing"
	SeriesRegistryEventStatusProcessed  SeriesRegistryEventStatus = "processed"
	SeriesRegistryEventStatusFailed     SeriesRegistryEventStatus = "failed"
)

// NFTEvent is the model for processed token events
type ArchivedNFTEvent struct {
	ID         string         `gorm:"primaryKey;size:255;default:uuid_generate_v4()"`
	Type       string         `gorm:"index"`
	Blockchain string         `gorm:"index"`
	Contract   string         `gorm:"index"`
	TokenID    string         `gorm:"index"`
	From       string         `gorm:"index"`
	To         string         `gorm:"index"`
	TXID       string         `gorm:"index"`
	EventIndex uint           `gorm:"index"`
	TXTime     time.Time      `gorm:"index"`
	Stage      string         `gorm:"index"`
	Status     NftEventStatus `gorm:"index"`
	CreatedAt  time.Time      `gorm:"default:now()"`
	UpdatedAt  time.Time      `gorm:"default:now()"`
}

func (ArchivedNFTEvent) TableName() string {
	return "nft_events"
}

// NFTEvent is the model for token events
type NFTEvent struct {
	ID         string         `gorm:"primaryKey;size:255;default:uuid_generate_v4()"`
	Type       string         `gorm:"index:idx_event,unique"`
	Blockchain string         `gorm:"index:idx_event,unique"`
	Contract   string         `gorm:"index:idx_event,unique"`
	TokenID    string         `gorm:"index:idx_event,unique"`
	From       string         `gorm:"index:idx_event,unique"`
	To         string         `gorm:"index:idx_event,unique"`
	TXID       string         `gorm:"index:idx_event,unique"`
	EventIndex uint           `gorm:"index:idx_event,unique"`
	TXTime     time.Time      `gorm:"index:idx_event,unique"`
	Stage      string         `gorm:"index"`
	Status     NftEventStatus `gorm:"index"`
	CreatedAt  time.Time      `gorm:"default:now()"`
	UpdatedAt  time.Time      `gorm:"default:now()"`
}

func (NFTEvent) TableName() string {
	return "new_nft_events"
}

// SeriesRegistryEvent is the model for series registry events
type SeriesRegistryEvent struct {
	ID         string                    `gorm:"primaryKey;size:255;default:uuid_generate_v4()"`
	Type       string                    `gorm:"index:idx_event,unique"`
	Contract   string                    `gorm:"index:idx_event,unique"`
	TxID       string                    `gorm:"index:idx_event,unique"`
	EventIndex uint                      `gorm:"index:idx_event,unique"`
	TxTime     time.Time                 `gorm:"index:idx_event,unique"`
	Data       datatypes.JSON            `gorm:"type:jsonb;NOT NULL;default:'{}'"`
	Stage      string                    `gorm:"index"`
	Status     SeriesRegistryEventStatus `gorm:"index"`
	CreatedAt  time.Time                 `gorm:"default:now()"`
	UpdatedAt  time.Time                 `gorm:"default:now()"`
}

func (SeriesRegistryEvent) TableName() string {
	return "series_registry_events"
}

// NftEventTx is an transaction object with nft event values
type NftEventTx struct {
	*gorm.DB
	NftEvent NFTEvent
}

func NewNftEventTx(DB *gorm.DB, nftEvent NFTEvent) *NftEventTx {
	return &NftEventTx{
		DB:       DB,
		NftEvent: nftEvent,
	}
}

// UpdateNftEvent updates nft events by given stage or status
func (tx *NftEventTx) UpdateNftEvent(stage, status string) error {
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

	return tx.DB.Model(&NFTEvent{}).Where("id = ?", tx.NftEvent.ID).Updates(updates).Error
}

// ArchiveNFTEvent save an ArchiveNFTEvent and delete the NFTEvent
func (tx *NftEventTx) ArchiveNFTEvent() error {
	archivedEvent := ArchivedNFTEvent{
		Type:       tx.NftEvent.Type,
		Blockchain: tx.NftEvent.Blockchain,
		Contract:   tx.NftEvent.Contract,
		TokenID:    tx.NftEvent.TokenID,
		From:       tx.NftEvent.From,
		To:         tx.NftEvent.To,
		TXID:       tx.NftEvent.TXID,
		TXTime:     tx.NftEvent.TXTime,
		CreatedAt:  tx.NftEvent.CreatedAt,
		Status:     NftEventStatusProcessed,
	}

	if err := tx.DB.Save(&archivedEvent).Error; err != nil {
		return err
	}

	return tx.DB.Where("id = ?", tx.NftEvent.ID).Delete(&NFTEvent{}).Error
}

// SeriesRegistryEventTx is an transaction object with series registry event values
type SeriesRegistryEventTx struct {
	*gorm.DB
	Event SeriesRegistryEvent
}

func NewSeriesRegistryEventTx(DB *gorm.DB, evt SeriesRegistryEvent) *SeriesRegistryEventTx {
	return &SeriesRegistryEventTx{
		DB:    DB,
		Event: evt,
	}
}

// UpdateSeriesRegistryEvent updates series events by given stage or status
func (tx *SeriesRegistryEventTx) UpdateSeriesRegistryEvent(stage, status string) error {
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

	return tx.DB.Model(&SeriesRegistryEvent{}).Where("id = ?", tx.Event.ID).Updates(updates).Error
}

// FilterOption is an abstraction to help filtering events with
// specific conditions
type FilterOption struct {
	Statement string
	Arguments []interface{}
}

func (f FilterOption) Apply(tx *gorm.DB) *gorm.DB {
	return tx.Where(f.Statement, f.Arguments...)
}

func Filter(statement string, args ...interface{}) FilterOption {
	return FilterOption{
		Statement: statement,
		Arguments: args,
	}
}

type Pagination struct {
	Limit  int
	Offset int
}

func (p *Pagination) Apply(tx *gorm.DB) *gorm.DB {
	return tx.
		Limit(p.Limit).
		Offset(p.Offset)
}

type EventStore interface {
	CreateNftEvent(event NFTEvent) error
	GetNftEventTransaction(ctx context.Context, filters ...FilterOption) (*NftEventTx, error)
	DeleteNftEvents(duration time.Duration) error

	CreateSeriesRegistryEvent(event SeriesRegistryEvent) error
	GetSeriesRegistryEventTransaction(ctx context.Context, filters ...FilterOption) (*SeriesRegistryEventTx, error)
	DeleteSeriesRegistryEvents(duration time.Duration) error
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

// CreateNftEvent add a new event into nft event store.
func (s *PostgresEventStore) CreateNftEvent(event NFTEvent) error {
	err := s.db.Exec(`
	INSERT INTO new_nft_events("type","blockchain","contract","token_id","from","to","tx_id","event_index","tx_time","stage","status")
	SELECT @Type, @Blockchain, @Contract, @TokenID, @From, @To, @TXID, @EventIndex, @TXTime, @Stage, @Status
	WHERE NOT EXISTS (SELECT * FROM nft_events WHERE "type"=@Type AND "blockchain"=@Blockchain AND "contract"=@Contract
		AND "token_id"=@TokenID AND "from"=@From AND "to"=@To AND "tx_id"=@TXID AND "event_index"=@EventIndex)`, structs.Map(event)).Error

	var pgError *pgconn.PgError
	if err != nil && errors.As(err, &pgError) {
		if pgError.Code == "23505" { // Unique violation error code
			log.Warn("duplicated event", zap.Error(err))
			return nil
		}
	}

	return err
}

// GetNftEventTransaction returns an NftEventTx
func (s *PostgresEventStore) GetNftEventTransaction(ctx context.Context, filters ...FilterOption) (*NftEventTx, error) {
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

	return NewNftEventTx(tx, event), nil
}

func (s *PostgresEventStore) DeleteNftEvents(duration time.Duration) error {
	return s.db.Where("created_at < ?", time.Now().Add(-duration)).Delete(&ArchivedNFTEvent{}).Error
}

// CreateSeriesRegistryEvent add a new event into series registry event store.
func (s *PostgresEventStore) CreateSeriesRegistryEvent(event SeriesRegistryEvent) error {
	err := s.db.Exec(`
	INSERT INTO series_registry_events("type","contract","data","tx_id","event_index","tx_time","stage","status")
	SELECT @Type, @Contract, @Data, @TxID, @EventIndex, @TxTime, @Stage, @Status
	WHERE NOT EXISTS (SELECT * FROM series_registry_events WHERE "type"=@Type AND "contract"=@Contract
		AND "tx_id"=@TxID AND "event_index"=@EventIndex)`, structs.Map(event)).Error

	var pgError *pgconn.PgError
	if err != nil && errors.As(err, &pgError) {
		if pgError.Code == "23505" { // Unique violation error code
			log.Warn("duplicated event", zap.Error(err))
			return nil
		}
	}

	return err
}

// GetSeriesRegistryEventTransaction returns an SeriesRegistryEventTx
func (s *PostgresEventStore) GetSeriesRegistryEventTransaction(ctx context.Context, filters ...FilterOption) (*SeriesRegistryEventTx, error) {
	var event SeriesRegistryEvent

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

	return NewSeriesRegistryEventTx(tx, event), nil
}

// DeleteSeriesRegistryEvents deletes series registry events older than the given duration
func (s *PostgresEventStore) DeleteSeriesRegistryEvents(duration time.Duration) error {
	return s.db.Where("created_at < ?", time.Now().Add(-duration)).Delete(&SeriesRegistryEvent{}).Error
}

// AutoMigrate is a help function that update db when the schema changed.
func (s *PostgresEventStore) AutoMigrate() error {
	if err := s.db.AutoMigrate(&ArchivedNFTEvent{}); err != nil {
		return err
	}
	if err := s.db.AutoMigrate(&NFTEvent{}); err != nil {
		return err
	}
	if err := s.db.AutoMigrate(&SeriesRegistryEvent{}); err != nil {
		return err
	}
	return nil
}
