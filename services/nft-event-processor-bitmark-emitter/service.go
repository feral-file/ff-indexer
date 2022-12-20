package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/bitmark-inc/bitmark-sdk-go/tx"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
	"github.com/sirupsen/logrus"
)

type BitmarkEventsEmitter struct {
	bitmarkListener *Listener
	grpcClient      processor.EventProcessorClient
}

func New(bitmarkListener *Listener, grpcClient processor.EventProcessorClient) *BitmarkEventsEmitter {
	return &BitmarkEventsEmitter{
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
			logrus.WithField("event", n.Channel).WithField("transfers", n.Extra).Info("new event")
			row := e.bitmarkListener.db.QueryRow("SELECT value FROM event WHERE id = $1", n.Extra)

			var txIDs string
			err := row.Scan(&txIDs)
			if err != nil {
				logrus.WithField("event_id", n.Extra).WithError(err).Error("fail to get transaction ids")
				continue
			}

			for _, txID := range strings.Split(txIDs, ",") {
				if t, err := tx.Get(txID); err == nil {
					eventType := "transfer"
					if t.BitmarkID == t.ID {
						eventType = "mint"
					}

					if err = indexer.PushGRPCEvent(ctx, e.grpcClient, eventType, t.PreviousOwner, t.Owner, "", indexer.BitmarkBlockchain, t.BitmarkID); err != nil {
						logrus.WithError(err).WithField("txID", txID).Error("gRPC request failed")
					}

				} else {
					logrus.WithError(err).WithField("txID", txID).Error("fail to get transaction detail")
				}
			}
		}
	}

}
