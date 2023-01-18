package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/bitmark-inc/bitmark-sdk-go/tx"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/emitter"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
	log "github.com/bitmark-inc/nft-indexer/zapLog"
	"go.uber.org/zap"
)

type BitmarkEventsEmitter struct {
	emitter.EventsEmitter
	bitmarkListener *Listener
	grpcClient      processor.EventProcessorClient
}

func New(bitmarkListener *Listener, grpcClient processor.EventProcessorClient) *BitmarkEventsEmitter {
	return &BitmarkEventsEmitter{
		EventsEmitter:   emitter.New(grpcClient),
		bitmarkListener: bitmarkListener,
		grpcClient:      grpcClient,
	}
}

// Watch listens events from bitmark blockchain db for new transfer events.
func (e *BitmarkEventsEmitter) Watch() error {
	if e.bitmarkListener == nil {
		return fmt.Errorf("bitmark listener is not initialized")
	}

	e.bitmarkListener.Start()

	if err := e.bitmarkListener.Watch("new_transfers"); err != nil {
		return err
	}

	return nil
}

func (e *BitmarkEventsEmitter) Run(ctx context.Context) {
	for n := range e.bitmarkListener.Notify {
		log.Logger.Info("new event", zap.String("event", n.Channel), zap.String("transfers", n.Extra))
		row := e.bitmarkListener.db.QueryRow("SELECT value FROM event WHERE id = $1", n.Extra)

		var txIDs string
		err := row.Scan(&txIDs)
		if err != nil {
			log.Logger.Error("fail to get transaction ids", zap.String("event_id", n.Extra), zap.Error(err))
			continue
		}

		for _, txID := range strings.Split(txIDs, ",") {
			if t, err := tx.Get(txID); err == nil {
				eventType := "transfer"
				if t.BitmarkID == t.ID {
					eventType = "mint"
				}

				if err = e.PushEvent(ctx, eventType, t.PreviousOwner, t.Owner, "", indexer.BitmarkBlockchain, t.BitmarkID); err != nil {
					log.Logger.Error("gRPC request failed", zap.Error(err), zap.String("txID", txID), zap.String("apiSource", log.GRPC))
				}

			} else {
				log.Logger.Error("fail to get transaction detail", zap.Error(err), zap.String("txID", txID))
			}
		}
	}
}
