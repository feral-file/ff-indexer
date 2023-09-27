package main

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/bitmark-inc/autonomy-account/storage"
	log "github.com/bitmark-inc/autonomy-logger"
	notification "github.com/bitmark-inc/autonomy-notification"
	notificationSdk "github.com/bitmark-inc/autonomy-notification/sdk"
	"github.com/bitmark-inc/nft-indexer/cadence"
	indexerGRPCSDK "github.com/bitmark-inc/nft-indexer/sdk/nft-indexer-grpc"
)

type EventProcessor struct {
	environment          string
	defaultCheckInterval time.Duration

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
	defaultCheckInterval time.Duration,
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
		environment:          environment,
		defaultCheckInterval: defaultCheckInterval,

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

type processorFunc func(ctx context.Context, event NFTEvent) error

func (e *EventProcessor) StartWorker(ctx context.Context, currentStage, nextStage int8,
	types []EventType, checkIntervalSecond, deferSecond int64, processor processorFunc) {

	checkInterval := e.defaultCheckInterval
	if checkIntervalSecond != 0 {
		checkInterval = time.Second * time.Duration(checkIntervalSecond)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info("process stopped")
				return
			default:
				e.logStageEvent(currentStage, "query event")

				filters := []FilterOption{
					Filter("type = ANY(?)", pq.Array(types)),
					Filter("status = ANY(?)", pq.Array([]EventStatus{EventStatusCreated, EventStatusProcessing})),
					Filter("stage = ?", EventStages[currentStage]),
				}

				if deferSecond > 0 {
					filters = append(filters, Filter("created_at < ?", time.Now().Add(-time.Duration(deferSecond)*time.Second)))
				}

				eventTx, err := e.eventQueue.GetEventTransaction(ctx, filters...)
				if err != nil {
					if err == gorm.ErrRecordNotFound {
						log.Info("No new events")
					} else {
						log.Error("Fail to get a event db transaction", zap.Error(err))
					}
					time.Sleep(checkInterval)
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
				if nextStage == 0 {
					if err := eventTx.ArchiveNFTEvent(); err != nil {
						log.Error("fail to archive event", zap.Error(err))
						eventTx.Rollback()
					}
				} else {
					if err := eventTx.UpdateEvent(EventStages[nextStage], ""); err != nil {
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

	//stage 2-1: trigger full updates for the token
	e.UpdateOwnerAndProvenance(ctx)

	//stage 2-2: trigger full updates for the token for burned token
	e.UpdateOwnerAndProvenanceForBurnedToken(ctx)

	//stage 3: send notificationSdk
	e.NotifyChangeTokenOwner(ctx)

	//stage 4: send to feed server
	e.SendEventToFeedServer(ctx)

}

// GetAccountIDByAddress get account IDS by address
func (e *EventProcessor) GetAccountIDByAddress(address string) ([]string, error) {
	return e.accountStore.GetAccountIDByAddress(address)
}
