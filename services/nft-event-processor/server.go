package main

import (
	"context"

	"github.com/bitmark-inc/autonomy-account/storage"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"

	"time"

	notification "github.com/bitmark-inc/autonomy-notification"
	notificationSdk "github.com/bitmark-inc/autonomy-notification/sdk"
	"github.com/sirupsen/logrus"
)

type EventProcessor struct {
	grpcServer     *GRPCServer
	queueProcessor *EventQueueProcessor
	indexerStore   *indexer.MongodbIndexerStore
	worker         *cadence.CadenceWorkerClient
	accountStore   *storage.AccountInformationStorage
	indexerEngine  *indexer.IndexEngine
	notification   *notificationSdk.NotificationClient
	feedServer     *FeedClient
}

func NewEventProcessor(
	network string,
	address string,
	store EventStore,
	indexerStore *indexer.MongodbIndexerStore,
	worker *cadence.CadenceWorkerClient,
	accountStore *storage.AccountInformationStorage,
	indexerEngine *indexer.IndexEngine,
	notification *notificationSdk.NotificationClient,
	feedServer *FeedClient,
) *EventProcessor {
	queueProcessor := NewEventQueueProcessor(store)
	grpcServer := NewGRPCServer(network, address, queueProcessor)

	return &EventProcessor{
		grpcServer:     grpcServer,
		queueProcessor: queueProcessor,
		indexerStore:   indexerStore,
		worker:         worker,
		accountStore:   accountStore,
		indexerEngine:  indexerEngine,
		notification:   notification,
		feedServer:     feedServer,
	}
}

// UpdateOwner update owner for a specific non-fungible token
func (e *EventProcessor) UpdateOwner(c context.Context, id, owner string, updatedAt time.Time) error {
	return e.indexerStore.UpdateOwner(c, id, owner, updatedAt)
}

// notifyChangeOwner notifies the arrival of a new token
func (e *EventProcessor) notifyChangeOwner(accountID, toAddress, tokenID string) error {
	return e.notification.SendNotification("",
		notification.ISSUE_UPDATED,
		accountID,
		gin.H{
			"notification_type": "change_token_owner",
			"owner":             toAddress,
			"token_id":          tokenID,
		})
}

// Run starts event processor server. It spawns a queue processor in the
// background routine and starts up a gRPC server to wait new events.
func (e *EventProcessor) Run(ctx context.Context) {
	e.ProcessEvents(ctx)

	if err := e.grpcServer.Run(); err != nil {
		logrus.WithError(err).Error("gRPC stopped with error")
	}
}

// ProcessEvents start a loop to continuously consuming queud event
func (e *EventProcessor) ProcessEvents(ctx context.Context) {
	// run goroutines forever
	logrus.Trace("start event processing goroutines")

	//stage 1: update the latest owner into mongodb
	go e.UpdateLatestOwner(ctx)

	//stage 2: trigger full updates for the token
	go e.UpdateOwnerAndProvenance(ctx)

	//stage 3: send notificationSdk
	go e.NotifyChangeTokenOwner()

	//stage 4: send to feed server
	go e.SendEventToFeedServer()

}

// GetAccountIDByAddress get account IDS by address
func (e *EventProcessor) GetAccountIDByAddress(address string) ([]string, error) {
	return e.accountStore.GetAccountIDByAddress(address)
}

// UpdateOwnerAndProvenance trigger cadence to update owner and provenance of token
func (e *EventProcessor) UpdateOwnerAndProvenance(ctx context.Context) {
	var stage int8 = 2

	for {
		event, err := e.GetQueueEventByStage(stage)
		if err != nil {
			logrus.WithError(err).Error("Have error when try to get queue event")
		}

		if event == nil {
			logrus.Info("event queue empty")
			time.Sleep(WaitingTime)

			continue
		}

		eventID := event.ID
		from := event.From
		blockchain := event.Blockchain
		contract := event.Contract
		tokenID := event.TokenID
		to := event.To

		e.logStartStage(event, stage)

		accounts, _ := e.accountStore.GetAccountIDByAddress(to)

		indexID := indexer.TokenIndexID(blockchain, contract, tokenID)

		token, err := e.indexerStore.GetTokensByIndexID(ctx, indexID)
		if err != nil {
			logrus.WithError(err).Error("fail to check token by index ID")
			return
		}

		// check if a token is existent
		// if existent, update provenance
		// if not, index it by blockchain
		if token != nil {
			// ignore the indexing process since an indexed token found
			logrus.WithField("indexID", indexID).Debug("an indexed token found for a corresponded event")

			// if the new owner is not existent in our system, index a new account_token
			if len(accounts) == 0 {
				accountToken := indexer.AccountToken{
					BaseTokenInfo:     token.BaseTokenInfo,
					IndexID:           indexID,
					OwnerAccount:      to,
					Balance:           int64(1),
					LastRefreshedTime: token.LastActivityTime,
				}

				if err := e.indexerStore.IndexAccountTokens(ctx, to, []indexer.AccountToken{accountToken}); err != nil {
					logrus.WithField("indexID", indexID).WithField("owner", to).Error("cannot index a new account_token")
				}
			}

			if token.Fungible {
				indexerWorker.StartRefreshTokenOwnershipWorkflow(ctx, e.worker, "processor", indexID, 0)
			} else {
				if err := e.indexerStore.UpdateOwner(ctx, indexID, to, time.Now()); err != nil {
					logrus.
						WithField("indexID", indexID).WithError(err).
						WithField("from", from).WithField("to", to).
						Error("fail to update the token ownership")
				}
				indexerWorker.StartRefreshTokenProvenanceWorkflow(ctx, e.worker, "processor", indexID, 0)
			}
		} else {
			// index the new token since it is a new token for our indexer and watched by our user
			if len(accounts) > 0 {
				logrus.WithField("indexID", indexID).
					WithField("from", from).WithField("to", to).
					Info("start indexing a new token")

				indexerWorker.StartIndexTokenWorkflow(ctx, e.worker, to, contract, tokenID, false)
			}
		}

		if err := e.UpdateEvent(eventID, map[string]interface{}{
			"stage": EventStages[stage+1],
		}); err != nil {
			logrus.WithError(err).Error("fail to update event")
		}

		e.logEndStage(event, stage)
	}
}

// UpdateEvent update event by map
func (e *EventProcessor) UpdateEvent(id string, updates map[string]interface{}) error {
	err := e.queueProcessor.store.UpdateEvent(id, updates)

	return err
}

// GetQueueEventByStage get event by stage
func (e *EventProcessor) GetQueueEventByStage(stage int8) (*NFTEvent, error) {
	event, err := e.queueProcessor.store.GetQueueEventByStage(stage)
	if err != nil {
		return nil, err
	}

	if event == nil {
		return nil, nil
	}

	return event, nil
}

// logStartStage log when start a stage
func (e *EventProcessor) logStartStage(event *NFTEvent, stage int8) {
	logrus.WithFields(logrus.Fields{
		"event": event,
	}).Info("start stage ", stage, " for event: ")
}

// logEndStage log when end a stage
func (e *EventProcessor) logEndStage(event *NFTEvent, stage int8) {
	logrus.WithFields(logrus.Fields{
		"event": event,
	}).Info("Finished stage ", stage, " for event: ")
}

// UpdateLatestOwner [stage 1] update owner for nft and ft by event information
func (e *EventProcessor) UpdateLatestOwner(ctx context.Context) {
	var stage int8 = 1

	for {
		event, err := e.GetQueueEventByStage(stage)
		if err != nil {
			logrus.WithError(err).Error("Have error when try to get queue event")
		}

		if event == nil {
			logrus.Info("event queue empty")
			time.Sleep(WaitingTime)

			continue
		}

		eventID := event.ID
		blockchain := event.Blockchain
		contract := event.Contract
		tokenID := event.TokenID
		to := event.To

		e.logStartStage(event, stage)

		indexID := indexer.TokenIndexID(blockchain, contract, tokenID)

		token, err := e.indexerStore.GetTokensByIndexID(ctx, indexID)
		if err != nil {
			logrus.WithError(err).Error("fail to get token by index id")
		}

		if token == nil {
			continue
		}

		if !token.Fungible {
			err := e.indexerStore.PushProvenance(ctx, indexID, token.LastRefreshedTime, indexer.Provenance{
				Type:        "transfer",
				FormerOwner: &event.From,
				Owner:       to,
				Blockchain:  blockchain,
				Timestamp:   time.Now(),
				TxID:        "",
				TxURL:       "",
			})

			if err != nil {
				err = e.indexerStore.UpdateOwner(ctx, indexID, to, time.Now())
				if err != nil {
					logrus.WithError(err).Error("fail to update owner")
				}
			}
		} else {
			err := e.indexerStore.UpdateOwnerForFungibleToken(ctx, indexID, token.LastRefreshedTime, event.To, 1)
			if err != nil {
				logrus.WithError(err).Error("fail to Update owner for fungible token")
			}
		}

		var accountTokens []indexer.AccountToken
		var accountToken indexer.AccountToken

		accountToken.IndexID = indexID
		accountToken.OwnerAccount = to
		accountToken.Balance = 1
		accountToken.ContractAddress = contract
		accountToken.ID = tokenID

		accountTokens = append(accountTokens, accountToken)
		e.indexerStore.IndexAccountTokens(ctx, to, accountTokens)

		if err := e.UpdateEvent(eventID, map[string]interface{}{
			"stage": EventStages[stage+1],
		}); err != nil {
			logrus.WithError(err).Error("fail to update event")
		}

		e.logEndStage(event, stage)
	}
}

// NotifyChangeTokenOwner send notification to notificationSdk
func (e *EventProcessor) NotifyChangeTokenOwner() {
	var stage int8 = 3

	for {
		event, err := e.GetQueueEventByStage(stage)
		if err != nil {
			logrus.WithError(err).Error("Have error when try to get queue event")
		}

		if event == nil {
			logrus.Info("event queue empty")
			time.Sleep(WaitingTime)

			continue
		}

		eventID := event.ID
		blockchain := event.Blockchain
		contract := event.Contract
		tokenID := event.TokenID
		to := event.To

		e.logStartStage(event, stage)

		accounts, _ := e.accountStore.GetAccountIDByAddress(to)
		indexID := indexer.TokenIndexID(blockchain, contract, tokenID)

		for _, accountID := range accounts {
			if err := e.notifyChangeOwner(accountID, to, indexID); err != nil {
				logrus.WithError(err).
					WithField("accountID", accountID).WithField("indexID", indexID).
					Error("fail to send notificationSdk for the new update")
			}
		}

		if err := e.UpdateEvent(eventID, map[string]interface{}{
			"stage": EventStages[stage+1],
		}); err != nil {
			logrus.WithError(err).Error("fail to update event")
		}

		e.logEndStage(event, stage)
	}
}

// SendEventToFeedServer send event to feed server
func (e *EventProcessor) SendEventToFeedServer() {
	var stage int8 = 4

	for {
		event, err := e.GetQueueEventByStage(stage)
		if err != nil {
			logrus.WithError(err).Error("Have error when try to get queue event")
		}

		if event == nil {
			logrus.Info("event queue empty")
			time.Sleep(WaitingTime)

			continue
		}

		eventID := event.ID
		blockchain := event.Blockchain
		contract := event.Contract
		tokenID := event.TokenID
		to := event.To
		eventType := event.EventType

		e.logStartStage(event, stage)

		if err := e.feedServer.SendEvent(blockchain, contract, tokenID, to, eventType, viper.GetString("network.ethereum") == "testnet"); err != nil {
			logrus.WithError(err).Trace("fail to push event to feed server")
		}

		// finish all stage of event processing
		if err := e.queueProcessor.store.CompleteEvent(eventID); err != nil {
			logrus.WithError(err).Error("fail to mark an event completed")
		}

		e.logEndStage(event, stage)
	}
}
