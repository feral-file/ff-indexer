package main

import (
	"context"
	"time"

	"github.com/bitmark-inc/autonomy-account/storage"
	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	notification "github.com/bitmark-inc/autonomy-notification"
	notificationSdk "github.com/bitmark-inc/autonomy-notification/sdk"
)

type EventProcessor struct {
	grpcServer     *GRPCServer
	queueProcessor *EventQueueProcessor
	indexerStore   *indexer.MongodbIndexerStore
	worker         *cadence.WorkerClient
	accountStore   *storage.AccountInformationStorage
	notification   *notificationSdk.NotificationClient
	feedServer     *FeedClient
}

func NewEventProcessor(
	network string,
	address string,
	store EventStore,
	indexerStore *indexer.MongodbIndexerStore,
	worker *cadence.WorkerClient,
	accountStore *storage.AccountInformationStorage,
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
		notification.NEW_NFT_ARRIVED,
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
		log.Error("gRPC stopped with error", zap.Error(err))
	}
}

// ProcessEvents start a loop to continuously consuming queud event
func (e *EventProcessor) ProcessEvents(ctx context.Context) {
	// run goroutines forever
	log.Debug("start event processing goroutines")

	// token update
	go e.RefreshTokenData(ctx)

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
			log.Error("Have error when try to get queue event", zap.Error(err))
		}

		if event == nil {
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

		accounts, err := e.accountStore.GetAccountIDByAddress(to)
		if err != nil {
			log.Error("fail to check accounts by address", zap.Error(err))
			return
		}

		indexID := indexer.TokenIndexID(blockchain, contract, tokenID)

		token, err := e.indexerStore.GetTokensByIndexID(ctx, indexID)
		if err != nil {
			log.Error("fail to check token by index ID", zap.Error(err))
			return
		}

		// check if a token is existent
		// if existent, update provenance
		// if not, index it by blockchain
		if token != nil {
			// ignore the indexing process since an indexed token found
			log.Debug("an indexed token found for a corresponded event", zap.String("indexID", indexID))

			// if the new owner is not existent in our system, index a new account_token
			if len(accounts) == 0 {
				accountToken := indexer.AccountToken{
					BaseTokenInfo:     token.BaseTokenInfo,
					IndexID:           indexID,
					OwnerAccount:      to,
					Balance:           int64(1),
					LastActivityTime:  event.CreatedAt,
					LastRefreshedTime: time.Now(),
				}

				if err := e.indexerStore.IndexAccountTokens(ctx, to, []indexer.AccountToken{accountToken}); err != nil {
					log.Error("cannot index a new account_token", zap.Error(err), zap.String("indexID", indexID), zap.String("owner", to))
				}
			}

			if token.Fungible {
				indexerWorker.StartRefreshTokenOwnershipWorkflow(ctx, e.worker, "processor", indexID, 0)
			} else {
				if err := e.indexerStore.UpdateOwner(ctx, indexID, to, event.CreatedAt); err != nil {
					log.Error("fail to update the token ownership",
						zap.String("indexID", indexID), zap.Error(err),
						zap.String("from", from), zap.String("to", to))

				}
				indexerWorker.StartRefreshTokenProvenanceWorkflow(ctx, e.worker, "processor", indexID, 0)
			}
		} else {
			// index the new token since it is a new token for our indexer and watched by our user
			if len(accounts) > 0 {
				log.Info("start indexing a new token",
					zap.String("indexID", indexID),
					zap.String("from", from), zap.String("to", to))

				indexerWorker.StartIndexTokenWorkflow(ctx, e.worker, to, contract, tokenID, true, false)
			}
		}

		if err := e.UpdateEvent(eventID, map[string]interface{}{
			"stage": EventStages[stage+1],
		}); err != nil {
			log.Error("fail to update event", zap.Error(err))
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

// logStageEvent logs events by stages
func (e *EventProcessor) logStageEvent(stage int8, message string, fields ...zap.Field) {
	fields = append(fields, zap.Int8("stage", stage))
	log.Info(message, fields...)
}

// logStartStage log when start a stage
func (e *EventProcessor) logStartStage(event *NFTEvent, stage int8) {
	log.Info("start stage for event: ", zap.Int8("stage", stage), zap.Any("event", event.ID))
}

// logEndStage log when end a stage
func (e *EventProcessor) logEndStage(event *NFTEvent, stage int8) {
	log.Info("finished stage for event: ", zap.Int8("stage", stage), zap.Any("event", event.ID))
}

// UpdateLatestOwner [stage 1] update owner for nft and ft by event information
func (e *EventProcessor) UpdateLatestOwner(ctx context.Context) {
	var stage int8 = 1

	for {
		event, err := e.GetQueueEventByStage(stage)
		if err != nil {
			log.Error("Have error when try to get queue event", zap.Error(err))
		}

		if event == nil {
			time.Sleep(WaitingTime)
			continue
		}

		eventType := event.Type
		eventID := event.ID
		blockchain := event.Blockchain
		contract := event.Contract
		tokenID := event.TokenID
		to := event.To
		indexID := indexer.TokenIndexID(blockchain, contract, tokenID)

		e.logStartStage(event, stage)

		switch event.Type {
		case "mint":
			// do nothing here.
		default:
			token, err := e.indexerStore.GetTokensByIndexID(ctx, indexID)
			if err != nil {
				log.Error("fail to get token by index id", zap.Error(err))
			}

			if token != nil {
				if !token.Fungible {
					err := e.indexerStore.PushProvenance(ctx, indexID, token.LastRefreshedTime, indexer.Provenance{
						Type:        eventType,
						FormerOwner: &event.From,
						Owner:       to,
						Blockchain:  blockchain,
						Timestamp:   event.CreatedAt,
						TxID:        "",
						TxURL:       "",
					})

					if err != nil {
						log.Error("fail to push provenance", zap.Error(err))

						err = e.indexerStore.UpdateOwner(ctx, indexID, to, event.CreatedAt)
						if err != nil {
							log.Error("fail to update owner", zap.Error(err))
						}
					}
				} else {
					err := e.indexerStore.UpdateOwnerForFungibleToken(ctx, indexID, token.LastRefreshedTime, event.To, 1)
					if err != nil {
						log.Error("fail to update owner for fungible token", zap.Error(err))
					}
				}

				accountToken := indexer.AccountToken{
					BaseTokenInfo:     token.BaseTokenInfo,
					IndexID:           indexID,
					OwnerAccount:      to,
					Balance:           int64(1),
					LastActivityTime:  event.CreatedAt,
					LastRefreshedTime: time.Now(),
				}

				if err := e.indexerStore.IndexAccountTokens(ctx, to, []indexer.AccountToken{accountToken}); err != nil {
					log.Error("fail to index account token", zap.Error(err))
					continue
				}
			} else {
				log.Debug("token not found", zap.String("indexID", indexID))
			}

		}

		if err := e.UpdateEvent(eventID, map[string]interface{}{
			"stage": EventStages[stage+1],
		}); err != nil {
			log.Error("fail to update event", zap.Error(err))
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
			log.Error("have error when try to get queue event", zap.Error(err))
		}

		if event == nil {
			time.Sleep(WaitingTime)

			continue
		}

		eventID := event.ID
		blockchain := event.Blockchain
		contract := event.Contract
		tokenID := event.TokenID
		to := event.To

		e.logStartStage(event, stage)

		accounts, err := e.accountStore.GetAccountIDByAddress(to)
		if err != nil {
			log.Error("fail to check accounts by address", zap.Error(err))
			return
		}
		indexID := indexer.TokenIndexID(blockchain, contract, tokenID)

		for _, accountID := range accounts {
			if err := e.notifyChangeOwner(accountID, to, indexID); err != nil {
				log.Error("fail to send notificationSdk for the new update",
					zap.Error(err),
					zap.String("accountID", accountID), zap.String("indexID", indexID))

			}
		}

		if err := e.UpdateEvent(eventID, map[string]interface{}{
			"stage": EventStages[stage+1],
		}); err != nil {
			log.Error("fail to update event", zap.Error(err))
		}

		e.logEndStage(event, stage)
	}
}

// SendEventToFeedServer send event to feed server
func (e *EventProcessor) SendEventToFeedServer() {
	var stage int8 = 4

	for {
		e.logStageEvent(stage, "query events")
		event, err := e.GetQueueEventByStage(stage)
		if err != nil {
			log.Error("Have error when try to get queue event", zap.Error(err))
		}

		if event == nil {
			time.Sleep(WaitingTime)
			e.logStageEvent(stage, "no event found")
			continue
		}
		e.logStageEvent(stage, "get an event", zap.String("eventID", event.ID), zap.Any("event", event))

		eventID := event.ID
		blockchain := event.Blockchain
		contract := event.Contract
		tokenID := event.TokenID
		to := event.To
		eventType := event.Type

		e.logStageEvent(stage, "start sending an event to feed", zap.String("eventID", event.ID))
		if err := e.feedServer.SendEvent(blockchain, contract, tokenID, to, eventType, viper.GetString("network.ethereum") == "testnet"); err != nil {
			log.Debug("fail to push event to feed server", zap.Error(err))
		}
		e.logStageEvent(stage, "event has sent to feed", zap.String("eventID", event.ID))

		// finish all stage of event processing
		if err := e.queueProcessor.store.CompleteEvent(eventID); err != nil {
			log.Error("fail to mark an event completed", zap.Error(err))
		}
		e.logStageEvent(stage, "mark an event completed", zap.String("eventID", event.ID))
	}
}
