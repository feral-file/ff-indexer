package main

import (
	"context"
	"fmt"
	"strings"

	tx "github.com/bitmark-inc/bitmark-sdk-go/tx"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	log "github.com/bitmark-inc/nft-indexer/zapLog"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// WatchBitmarkEvent listens events from bitmark blockchain db for new transfer events.
// When a new event comes in, it requests the provenance update and pushes notification
func (s *NFTEventSubscriber) WatchBitmarkEvent(ctx context.Context) error {
	if s.bitmarkListener == nil {
		return fmt.Errorf("bitmark listener is not initialized")
	}

	s.bitmarkListener.Start()

	if err := s.bitmarkListener.Watch("new_transfers"); err != nil {
		return err
	}

	go func() {
		for n := range s.bitmarkListener.Notify {
			logrus.WithField("event", n.Channel).WithField("transfers", n.Extra).Info("new event")
			row := s.bitmarkListener.db.QueryRow("SELECT value FROM event WHERE id = $1", n.Extra)

			var txIDs string
			err := row.Scan(&txIDs)
			if err != nil {
				logrus.WithField("event_id", n.Extra).WithError(err).Error("fail to get transaction ids")
				continue
			}

			for _, txID := range strings.Split(txIDs, ",") {
				if t, err := tx.Get(txID); err == nil {
					action := "transfer"
					if t.BitmarkID == t.ID {
						action = "mint"
					}
					go func() {
						if t.Owner == indexer.LivenetZeroAddress || t.Owner == indexer.TestnetZeroAddress {
							if err := s.feedServer.SendBurn(indexer.BitmarkBlockchain, "", t.BitmarkID); err != nil {
								log.Logger.Debug("fail to push event to feed server", zap.Error(err))
							}
						} else {
							if err := s.feedServer.SendEvent(indexer.BitmarkBlockchain, "", t.BitmarkID, t.Owner, action, viper.GetString("network.bitmark") == "testnet"); err != nil {
								log.Logger.Debug("fail to push event to feed server", zap.Error(err))
							}
						}
					}()

					indexID := indexer.TokenIndexID(indexer.BitmarkBlockchain, "", t.BitmarkID)

					logrus.WithField("id", indexID).Debug("refresh provenance from subscriber")
					go indexerWorker.StartRefreshTokenProvenanceWorkflow(ctx, &s.Worker, "subscriber", indexID, 0)

					// TODO: do something for the feed
					toAddress := t.Owner
					accounts, err := s.accountStore.GetAccountIDByAddress(toAddress)
					if err != nil {
						logrus.WithField("toAddress", toAddress).WithError(err).Error("fail to get account address map")
						continue
					}

					// send notification in the end
					for _, accountID := range accounts {
						if err := s.notifyNewNFT(accountID, toAddress, indexID); err != nil {
							logrus.WithError(err).
								WithField("accountID", accountID).WithField("indexID", indexID).
								Error("fail to send notification for the new token")
						}
					}
				} else {
					logrus.WithError(err).
						WithField("txID", txID).
						Error("fail to get transaction detail")
				}
			}
		}
	}()

	return nil
}
