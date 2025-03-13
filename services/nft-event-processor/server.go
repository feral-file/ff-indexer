package main

import (
	"context"
	"time"

	"github.com/bitmark-inc/autonomy-account/storage"
	log "github.com/bitmark-inc/autonomy-logger"
	notificationConst "github.com/bitmark-inc/autonomy-notification"
	notificationSdk "github.com/bitmark-inc/autonomy-notification/sdk"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/cadence"
	indexerGRPCSDK "github.com/bitmark-inc/nft-indexer/sdk/nft-indexer-grpc"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type EventProcessor struct {
	environment            string
	seriesRegistryContract string
	ipfsGateways           []string
	defaultCheckInterval   time.Duration
	eventExpiryDuration    time.Duration

	grpcServer   *GRPCServer
	eventQueue   *EventQueue
	indexerGRPC  *indexerGRPCSDK.IndexerGRPCClient
	worker       *cadence.WorkerClient
	accountStore *storage.AccountInformationStorage
	indexerStore indexer.Store
	notification *notificationSdk.Client
	feedServer   *FeedClient
	rpcClient    *ethclient.Client
}

func NewEventProcessor(
	environment string,
	seriesRegistryContract string,
	ipfsGateways []string,
	defaultCheckInterval time.Duration,
	eventExpiryDuration time.Duration,
	network string,
	address string,
	store EventStore,
	indexerGRPC *indexerGRPCSDK.IndexerGRPCClient,
	worker *cadence.WorkerClient,
	accountStore *storage.AccountInformationStorage,
	indexerStore indexer.Store,
	notification *notificationSdk.Client,
	feedServer *FeedClient,
	rpcClient *ethclient.Client,
) *EventProcessor {
	queue := NewEventQueue(store)
	grpcServer := NewGRPCServer(network, address, queue)

	return &EventProcessor{
		environment:            environment,
		seriesRegistryContract: seriesRegistryContract,
		ipfsGateways:           ipfsGateways,
		defaultCheckInterval:   defaultCheckInterval,
		eventExpiryDuration:    eventExpiryDuration,

		grpcServer:   grpcServer,
		eventQueue:   queue,
		indexerGRPC:  indexerGRPC,
		worker:       worker,
		accountStore: accountStore,
		indexerStore: indexerStore,
		notification: notification,
		feedServer:   feedServer,
		rpcClient:    rpcClient,
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

// removeDeprecatedNftEvents removes expired archived events
func (e *EventProcessor) removeDeprecatedNftEvents() error {
	return e.eventQueue.store.DeleteNftEvents(e.eventExpiryDuration)
}

// removeDeprecatedSeriesRegistryEvents removes expired archived series registry events
func (e *EventProcessor) removeDeprecatedSeriesRegistryEvents() error {
	return e.eventQueue.store.DeleteSeriesRegistryEvents(e.eventExpiryDuration)
}

// PruneDeprecatedEventsCronjob runs the cron job in a goroutine to remove expired events
func (e *EventProcessor) PruneDeprecatedEventsCronjob() {
	go func() {
		for {
			now := time.Now()

			// Calculate the next run time at midnight
			nextRun := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
			durationUntilNextRun := nextRun.Sub(now)

			// Wait until the next run time
			time.Sleep(durationUntilNextRun)

			// Remove expired events
			if err := e.removeDeprecatedNftEvents(); err != nil {
				log.Error("error delete archived events", zap.Error(err))
			}

			if err := e.removeDeprecatedSeriesRegistryEvents(); err != nil {
				log.Error("error delete archived series registry events", zap.Error(err))
			}
		}
	}()
}

// Run starts event processor server. It spawns a queue processor in the
// background routine and starts up a gRPC server to wait new events.
func (e *EventProcessor) Run(ctx context.Context) {
	log.Info("start event processor")

	e.PruneDeprecatedEventsCronjob()

	e.ProcessEvents(ctx)

	if err := e.grpcServer.Run(); err != nil {
		log.Error("gRPC stopped with error", zap.Error(err))
	}
}

type nftEventProcessorFunc func(ctx context.Context, event NFTEvent) error

func (e *EventProcessor) StartNftEventWorker(ctx context.Context, currentStage, nextStage Stage,
	types []NftEventType, checkIntervalSecond, deferSecond int64, processor nftEventProcessorFunc) {

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
					Filter("status = ANY(?)", pq.Array([]NftEventStatus{NftEventStatusCreated, NftEventStatusProcessing})),
					Filter("stage = ?", NftEventStages[currentStage]),
				}

				if deferSecond > 0 {
					filters = append(filters, Filter("created_at < ?", time.Now().Add(-time.Duration(deferSecond)*time.Second)))
				}

				eventTx, err := e.eventQueue.GetNftEventTransaction(ctx, filters...)
				if err != nil {
					if err == gorm.ErrRecordNotFound {
						log.Info("No new events")
					} else {
						log.Error("Fail to get a event db transaction", zap.Error(err))
					}
					time.Sleep(checkInterval)
					continue
				}
				e.logStartStage(eventTx.NftEvent.ID, currentStage)
				if err := processor(ctx, eventTx.NftEvent); err != nil {
					log.Error("stage processing failed", zap.Error(err))
					if err := eventTx.UpdateNftEvent("", string(NftEventStatusFailed)); err != nil {
						log.Error("fail to update event", zap.Error(err))
						eventTx.Rollback()
					}
				}

				// stage starts from 1. stage zero means there is no next stage.
				if nextStage == NftEventStageDone {
					if err := eventTx.ArchiveNFTEvent(); err != nil {
						log.Error("fail to archive event", zap.Error(err))
						eventTx.Rollback()
					}
				} else {
					if err := eventTx.UpdateNftEvent(NftEventStages[nextStage], ""); err != nil {
						log.Error("fail to update event", zap.Error(err))
						eventTx.Rollback()
					}
				}

				eventTx.Commit()
				e.logEndStage(eventTx.NftEvent.ID, currentStage)
			}
		}
	}()
}

type seriesRegistryEventProcessorFunc func(ctx context.Context, event SeriesRegistryEvent) error

func (e *EventProcessor) StartSeriesRegistryEventWorker(ctx context.Context, currentStage, nextStage Stage,
	types []SeriesRegistryEventType, checkIntervalSecond, deferSecond int64, processor seriesRegistryEventProcessorFunc) {

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
					Filter("status = ANY(?)", pq.Array([]SeriesRegistryEventStatus{SeriesRegistryEventStatusCreated, SeriesRegistryEventStatusProcessing})),
					Filter("stage = ?", SeriesEventStages[currentStage]),
				}

				if deferSecond > 0 {
					filters = append(filters, Filter("created_at < ?", time.Now().Add(-time.Duration(deferSecond)*time.Second)))
				}

				eventTx, err := e.eventQueue.GetSeriesRegistryEventTransaction(ctx, filters...)
				if err != nil {
					if err == gorm.ErrRecordNotFound {
						log.Info("No new events")
					} else {
						log.Error("Fail to get a event db transaction", zap.Error(err))
					}
					time.Sleep(checkInterval)
					continue
				}
				e.logStartStage(eventTx.Event.ID, currentStage)
				if err := eventTx.UpdateSeriesRegistryEvent("", string(SeriesRegistryEventStatusProcessing)); err != nil {
					log.Error("fail to update series event status processing", zap.Error(err))
					eventTx.Rollback()
				}
				if err := processor(ctx, eventTx.Event); err != nil {
					log.Error("stage processing failed", zap.Error(err))
					if err := eventTx.UpdateSeriesRegistryEvent("", string(SeriesRegistryEventStatusFailed)); err != nil {
						log.Error("fail to update series event status failed", zap.Error(err))
						eventTx.Rollback()
					}
					continue
				}

				// stage starts from 1. stage zero means there is no next stage.
				if nextStage == SeriesRegistryEventStageDone {
					if err := eventTx.UpdateSeriesRegistryEvent("", string(SeriesRegistryEventStatusProcessed)); err != nil {
						log.Error("fail to set event to handled", zap.Error(err))
						eventTx.Rollback()
					}
				} else {
					if err := eventTx.UpdateSeriesRegistryEvent(NftEventStages[nextStage], ""); err != nil {
						log.Error("fail to update event", zap.Error(err))
						eventTx.Rollback()
					}
				}

				eventTx.Commit()
				e.logEndStage(eventTx.Event.ID, currentStage)
			}
		}
	}()
}

// ProcessEvents start a loop to continuously consuming queud event
func (e *EventProcessor) ProcessEvents(ctx context.Context) {
	// run goroutines forever
	log.Debug("start nft event processing goroutines")

	//--------------------------------
	//------NFT Event Processing------
	//--------------------------------

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

	//--------------------------------------
	//---Series Registry Event Processing---
	//--------------------------------------

	log.Debug("start series event processing goroutines")

	e.IndexCollection(ctx)
	e.DeleteCollection(ctx)
	e.ReplaceCollectionCreator(ctx)
	e.UpdateCollectionCreators(ctx)
}

// GetAccountIDByAddress get account IDS by address
func (e *EventProcessor) GetAccountIDByAddress(address string) ([]string, error) {
	return e.accountStore.GetAccountIDByAddress(address)
}
