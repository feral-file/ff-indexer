package main

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/bitmark-inc/autonomy-account/storage"
	notification "github.com/bitmark-inc/autonomy-notification"
	notificationSdk "github.com/bitmark-inc/autonomy-notification/sdk"
	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/log"
)

type EventProcessor struct {
	checkInterval time.Duration

	grpcServer   *GRPCServer
	eventQueue   *EventQueue
	indexerStore *indexer.MongodbIndexerStore
	worker       *cadence.WorkerClient
	accountStore *storage.AccountInformationStorage
	notification *notificationSdk.NotificationClient
	feedServer   *FeedClient
}

func NewEventProcessor(
	checkInterval time.Duration,
	network string,
	address string,
	store EventStore,
	indexerStore *indexer.MongodbIndexerStore,
	worker *cadence.WorkerClient,
	accountStore *storage.AccountInformationStorage,
	notification *notificationSdk.NotificationClient,
	feedServer *FeedClient,
) *EventProcessor {
	queue := NewEventQueue(store)
	grpcServer := NewGRPCServer(network, address, queue)

	return &EventProcessor{
		checkInterval: checkInterval,

		grpcServer:   grpcServer,
		eventQueue:   queue,
		indexerStore: indexerStore,
		worker:       worker,
		accountStore: accountStore,
		notification: notification,
		feedServer:   feedServer,
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

func (e *EventProcessor) StartWorker(ctx context.Context, currentStage, nextStage int8,
	types []EventType, processor processorFunc) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info("process stopped")
				return
			default:
				e.logStageEvent(currentStage, "query event")
				eventTx, err := e.eventQueue.GetEventTransaction(ctx,
					Filter("type = ANY(?)", pq.Array(types)),
					Filter("status = ANY(?)", pq.Array([]EventStatus{EventStatusCreated, EventStatusProcessing})),
					Filter("stage = ?", EventStages[currentStage]),
				)
				if err != nil {
					if err == gorm.ErrRecordNotFound {
						log.Info("No new events")
					} else {
						log.Error("Fail to get a event db transaction", zap.Error(err))
					}
					time.Sleep(e.checkInterval)
					continue
				}
				e.logStartStage(eventTx.Event, currentStage)
				if err := processor(ctx, eventTx.Event); err != nil {
					log.Error("stage processing failed", zap.Error(err))
					if err := eventTx.UpdateEvent("", string(EventStatusFailed)); err != nil {
						log.Error("fail to update event", zap.Error(err))
						eventTx.Rollback()
					}
				}

				// stage starts from 1. stage zero means there is no next stage.
				var newStage, newStatus string
				if nextStage == 0 {
					newStatus = string(EventStatusProcessed)
				} else {
					newStage = EventStages[nextStage]
				}
				if err := eventTx.UpdateEvent(newStage, newStatus); err != nil {
					log.Error("fail to update event", zap.Error(err))
					eventTx.Rollback()
				}

				eventTx.Commit()
				e.logEndStage(eventTx.Event, currentStage)
			}
		}
	}()
}

// ProcessEvents start a loop to continuously consuming queud event
func (e *EventProcessor) ProcessEvents(ctx context.Context) {
	// run goroutines forever
	log.Debug("start event processing goroutines")

	// token update
	e.RefreshTokenData(ctx)

	//stage 1: update the latest owner into mongodb
	e.UpdateLatestOwner(ctx)

	//stage 2: trigger full updates for the token
	e.UpdateOwnerAndProvenance(ctx)

	//stage 3: send notificationSdk
	e.NotifyChangeTokenOwner(ctx)

	//stage 4: send to feed server
	e.SendEventToFeedServer(ctx)

}

// GetAccountIDByAddress get account IDS by address
func (e *EventProcessor) GetAccountIDByAddress(address string) ([]string, error) {
	return e.accountStore.GetAccountIDByAddress(address)
}

// logStageEvent logs events by stages
func (e *EventProcessor) logStageEvent(stage int8, message string, fields ...zap.Field) {
	fields = append(fields, zap.Int8("stage", stage))
	log.Info(message, fields...)
}

// logStartStage log when start a stage
func (e *EventProcessor) logStartStage(event NFTEvent, stage int8) {
	log.Info("start stage for event: ", zap.Int8("stage", stage), zap.Any("event", event.ID))
}

// logEndStage log when end a stage
func (e *EventProcessor) logEndStage(event NFTEvent, stage int8) {
	log.Info("finished stage for event: ", zap.Int8("stage", stage), zap.Any("event", event.ID))
}

func (e *EventProcessor) updateLatestOwner(ctx context.Context, event NFTEvent) error {
	eventType := event.Type
	blockchain := event.Blockchain
	contract := event.Contract
	tokenID := event.TokenID
	to := event.To
	indexID := indexer.TokenIndexID(blockchain, contract, tokenID)

	switch event.Type {
	case "mint":
		// do nothing here.
	default:
		token, err := e.indexerStore.GetTokensByIndexID(ctx, indexID)
		if err != nil {
			log.Error("fail to get token by index id", zap.Error(err))
			return err
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
						return err
					}
				}
			} else {
				err := e.indexerStore.UpdateOwnerForFungibleToken(ctx, indexID, token.LastRefreshedTime, event.To, 1)
				if err != nil {
					log.Error("fail to update owner for fungible token", zap.Error(err))
					return err
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
				return err
			}
		} else {
			log.Debug("token not found", zap.String("indexID", indexID))
		}

	}

	return nil
}

// UpdateLatestOwner [stage 1] update owner for nft and ft by event information
func (e *EventProcessor) UpdateLatestOwner(ctx context.Context) {
	e.StartWorker(ctx,
		1, 2,
		[]EventType{EventTypeTransfer, EventTypeMint},
		e.updateLatestOwner,
	)
}

// updateOwnerAndProvenance checks if a token is related to Autonomy. If so, it updates its
// owner and refresh the provenance
func (e *EventProcessor) updateOwnerAndProvenance(ctx context.Context, event NFTEvent) error {
	from := event.From
	blockchain := event.Blockchain
	contract := event.Contract
	tokenID := event.TokenID
	to := event.To

	accounts, err := e.accountStore.GetAccountIDByAddress(to)
	if err != nil {
		log.Error("fail to check accounts by address", zap.Error(err))
		return err
	}

	indexID := indexer.TokenIndexID(blockchain, contract, tokenID)
	token, err := e.indexerStore.GetTokensByIndexID(ctx, indexID)
	if err != nil {
		log.Error("fail to check token by index ID", zap.Error(err))
		return err
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
				return err
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
	return nil
}

// UpdateOwnerAndProvenance trigger cadence to update owner and provenance of token
func (e *EventProcessor) UpdateOwnerAndProvenance(ctx context.Context) {
	e.StartWorker(ctx,
		2, 3,
		[]EventType{EventTypeTransfer, EventTypeMint},
		e.updateOwnerAndProvenance,
	)
}

// NotifyChangeTokenOwner send notification to notificationSdk
func (e *EventProcessor) notifyChangeTokenOwner(ctx context.Context, event NFTEvent) error {
	blockchain := event.Blockchain
	contract := event.Contract
	tokenID := event.TokenID
	to := event.To

	accounts, err := e.accountStore.GetAccountIDByAddress(to)
	if err != nil {
		log.Error("fail to check accounts by address", zap.Error(err))
		return err
	}
	indexID := indexer.TokenIndexID(blockchain, contract, tokenID)

	for _, accountID := range accounts {
		if err := e.notifyChangeOwner(accountID, to, indexID); err != nil {
			log.Error("fail to send notificationSdk for the new update",
				zap.Error(err),
				zap.String("accountID", accountID), zap.String("indexID", indexID))
			return err
		}
	}
	return nil
}

// NotifyChangeTokenOwner send notification to notificationSdk
func (e *EventProcessor) NotifyChangeTokenOwner(ctx context.Context) {
	e.StartWorker(ctx,
		3, 4,
		[]EventType{EventTypeTransfer, EventTypeMint},
		e.notifyChangeTokenOwner,
	)
}

func (e *EventProcessor) sendEventToFeedServer(ctx context.Context, event NFTEvent) error {
	blockchain := event.Blockchain
	contract := event.Contract
	tokenID := event.TokenID
	to := event.To
	eventType := event.Type

	return e.feedServer.SendEvent(blockchain, contract, tokenID, to, eventType,
		viper.GetString("network.ethereum") == "testnet")
}

// SendEventToFeedServer send event to feed server
func (e *EventProcessor) SendEventToFeedServer(ctx context.Context) {
	e.StartWorker(ctx,
		4, 0,
		[]EventType{EventTypeTransfer, EventTypeMint},
		e.sendEventToFeedServer,
	)
}
