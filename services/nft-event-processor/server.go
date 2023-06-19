package main

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/bitmark-inc/autonomy-account/storage"
	notification "github.com/bitmark-inc/autonomy-notification"
	notificationSdk "github.com/bitmark-inc/autonomy-notification/sdk"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/log"
	indexerGRPCSDK "github.com/bitmark-inc/nft-indexer/sdk/nft-indexer-grpc"
)

type EventProcessor struct {
	environment   string
	checkInterval time.Duration

	grpcServer   *GRPCServer
	eventQueue   *EventQueue
	indexerGRPC  *indexerGRPCSDK.IndexerGRPCClient
	worker       *cadence.WorkerClient
	accountStore *storage.AccountInformationStorage
	notification *notificationSdk.NotificationClient
	feedServer   *FeedClient
}

func NewEventProcessor(
	environment string,
	checkInterval time.Duration,
	network string,
	address string,
	store EventStore,
	indexerGRPC *indexerGRPCSDK.IndexerGRPCClient,
	worker *cadence.WorkerClient,
	accountStore *storage.AccountInformationStorage,
	notification *notificationSdk.NotificationClient,
	feedServer *FeedClient,
) *EventProcessor {
	queue := NewEventQueue(store)
	grpcServer := NewGRPCServer(network, address, queue)

	return &EventProcessor{
		environment:   environment,
		checkInterval: checkInterval,

		grpcServer:   grpcServer,
		eventQueue:   queue,
		indexerGRPC:  indexerGRPC,
		worker:       worker,
		accountStore: accountStore,
		notification: notification,
		feedServer:   feedServer,
	}
}

// notifyChangeOwner send change_token_owner notification to notification server
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

type processorFunc func(ctx context.Context, event NewNFTEvent) error

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
					if err := e.eventQueue.SaveProcessedEvent(eventTx.Event.toProcessedNFTEvent(EventStatusFailed)); err != nil {
						log.Error("fail to update failed event", zap.Error(err))
					}

					if err := eventTx.DeleteEvent(); err != nil {
						log.Error("fail to delete event", zap.Error(err))
						eventTx.Rollback()
					}
				}

				// stage starts from 1. stage zero means there is no next stage.
				if nextStage == 0 {
					if err := e.eventQueue.SaveProcessedEvent(eventTx.Event.toProcessedNFTEvent(EventStatusProcessed)); err != nil {
						log.Error("fail to update processed event", zap.Error(err))
					}

					if err := eventTx.DeleteEvent(); err != nil {
						log.Error("fail to delete event", zap.Error(err))
						eventTx.Rollback()
					}
				} else {
					if err := eventTx.UpdateEvent(EventStages[nextStage]); err != nil {
						log.Error("fail to update event", zap.Error(err))
						eventTx.Rollback()
					}
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
