package main

import (
	"context"
	"time"

	"github.com/bitmark-inc/autonomy-account/storage"
	log "github.com/bitmark-inc/autonomy-logger"
	notificationConst "github.com/bitmark-inc/autonomy-notification"
	notificationSdk "github.com/bitmark-inc/autonomy-notification/sdk"
	"github.com/bitmark-inc/nft-indexer/cadence"
	indexerGRPCSDK "github.com/bitmark-inc/nft-indexer/sdk/nft-indexer-grpc"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type EventProcessor struct {
	environment          string
	defaultCheckInterval time.Duration
	eventExpiryDuration  time.Duration

	grpcServer   *GRPCServer
	eventQueue   *EventQueue
	indexerGRPC  *indexerGRPCSDK.IndexerGRPCClient
	worker       *cadence.WorkerClient
	accountStore *storage.AccountInformationStorage
	notification *notificationSdk.Client
	feedServer   *FeedClient
}

func NewEventProcessor(
	environment string,
	defaultCheckInterval time.Duration,
	eventExpiryDuration time.Duration,
	network string,
	address string,
	store EventStore,
	indexerGRPC *indexerGRPCSDK.IndexerGRPCClient,
	worker *cadence.WorkerClient,
	accountStore *storage.AccountInformationStorage,
	notification *notificationSdk.Client,
	feedServer *FeedClient,
) *EventProcessor {
	queue := NewEventQueue(store)
	grpcServer := NewGRPCServer(network, address, queue)

	return &EventProcessor{
		environment:          environment,
		defaultCheckInterval: defaultCheckInterval,
		eventExpiryDuration:  eventExpiryDuration,

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
		notificationConst.NewNFTArrived,
		accountID,
		[]any{},
		gin.H{
			"notification_type": "change_token_owner",
			"owner":             toAddress,
			"token_id":          tokenID,
		})
}

// removeDeprecatedEvents removes expired archived events
func (e *EventProcessor) removeDeprecatedEvents() error {
	return e.eventQueue.store.DeleteEvents(e.eventExpiryDuration)
}

// PruneDeprecatedEventsCrobjob runs the cron job in a goroutine to remove expired archived events
func (e *EventProcessor) PruneDeprecatedEventsCrobjob() {
	go func() {
		for {
			now := time.Now()

			// Calculate the next run time at midnight
			nextRun := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
			durationUntilNextRun := nextRun.Sub(now)

			// Wait until the next run time
			time.Sleep(durationUntilNextRun)

			// Remove expired events
			if err := e.removeDeprecatedEvents(); err != nil {
				log.Error("error delete archived events", zap.Error(err))
			}
		}
	}()
}

// Run starts event processor server. It spawns a queue processor in the
// background routine and starts up a gRPC server to wait new events.
func (e *EventProcessor) Run(ctx context.Context) {
	log.Info("start event processor")

	e.PruneDeprecatedEventsCrobjob()

	e.ProcessEvents(ctx)

	if err := e.grpcServer.Run(); err != nil {
		log.Error("gRPC stopped with error", zap.Error(err))
	}
}

type processorFunc func(ctx context.Context, event NFTEvent) error

func (e *EventProcessor) StartWorker(ctx context.Context, currentStage, nextStage Stage,
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
	e.DoubleSyncTokenData(ctx)

	//stage 1: update the latest owner into mongodb
	e.UpdateLatestOwner(ctx)

	//stage 2-1: trigger full updates for the token
	e.UpdateOwnerAndProvenance(ctx)

	//stage 2-2: trigger full updates for the token for burned token
	e.UpdateOwnerAndProvenanceForBurnedToken(ctx)

	//stage 3-1: send notificationSdk for transfer tokens
	e.NotifyChangeTokenOwnerForTransferToken(ctx)

	//stage 3-2: send notificationSdk for minted tokens
	e.NotifyChangeTokenOwnerForMintToken(ctx)

	//stage 4: send to feed server
	// e.SendEventToFeedServer(ctx)

	//stage 5: index token sale
	e.IndexTokenSale(ctx)
}

// GetAccountIDByAddress get account IDS by address
func (e *EventProcessor) GetAccountIDByAddress(address string) ([]string, error) {
	return e.accountStore.GetAccountIDByAddress(address)
}
