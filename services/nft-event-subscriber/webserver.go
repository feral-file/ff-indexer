package main

import (
	"net/http"
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type EventSubscriberAPI struct {
	feedServer *FeedClient
	subscriber *NFTEventSubscriber
}

func NewEventSubscriberAPI(s *NFTEventSubscriber, feed *FeedClient) *EventSubscriberAPI {
	return &EventSubscriberAPI{
		feedServer: feed,
		subscriber: s,
	}
}

type RequestNewEvent struct {
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Contract  string    `json:"contract" binding:"required"`
	TokenID   string    `json:"tokenID" binding:"required"`
	From      string    `json:"from" binding:"required"`
	To        string    `json:"to" binding:"required"`
	IsTestnet bool      `json:"isTest"`
}

func (api *EventSubscriberAPI) ReceiveEvents(c *gin.Context) {
	var req RequestNewEvent

	if err := c.Bind(&req); err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	tokenBlockchain := indexer.GetBlockchainByAddress(req.Contract)

	if tokenBlockchain == indexer.UnknownBlockchain {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "unknown blockchain",
		})
		return
	}

	indexID := indexer.TokenIndexID(tokenBlockchain, req.Contract, req.TokenID)
	// Flow:
	// 1. Check if the destination address is followed
	// 2. Check if the token is indexed
	//   - if indexed:
	//	   - update provenance
	//     - send notification if there is any follower
	//   - if not indexed:
	//     - if there is any follower:
	// 		 - index the token
	//       - update provenance
	//       - send notification

	mintType := "transfer"
	// FIXME: we do not have mint event as this moment.

	go func() {
		if err := api.feedServer.SendEvent(tokenBlockchain, req.Contract, req.TokenID, req.To, mintType, req.IsTestnet); err != nil {
			log.Debug("fail to push event to feed server", zap.Error(err))
		}
	}()

	// TODO: do we need to move this account specific function out of this service
	accounts, err := api.subscriber.GetAccountIDByAddress(req.To)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	token, err := api.subscriber.GetTokensByIndexID(c, indexID)
	if err != nil {
		log.Error("fail to check token by index ID", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "fail to check token by index ID",
		})
		return
	}

	// check if a token is existent
	// if existent, update provenance
	// if not, index it by blockchain
	if token != nil {
		// ignore the indexing process since an indexed token found
		log.Debug("an indexed token found for a corresponded event", zap.String("indexID", indexID))
		if token.Fungible {
			indexerWorker.StartRefreshTokenOwnershipWorkflow(c, &api.subscriber.Worker, "subscriber", indexID, 0)
		} else {
			if err := api.subscriber.UpdateOwner(c, indexID, req.To, req.Timestamp); err != nil {
				log.Error("fail to update the token ownership",
					zap.String("indexID", indexID),
					zap.Error(err),
					zap.String("from", req.From),
					zap.String("to", req.To))
			}
			indexerWorker.StartRefreshTokenProvenanceWorkflow(c, &api.subscriber.Worker, "subscriber", indexID, 0)
		}
	} else {
		// index the new token since it is a new token for our indexer and watched by our user
		if len(accounts) > 0 {
			log.Info("start indexing a new token",
				zap.String("indexID", indexID),
				zap.String("from", req.From),
				zap.String("to", req.To))

			indexerWorker.StartIndexTokenWorkflow(c, &api.subscriber.Worker, req.To, req.Contract, req.TokenID, false, false)

			// ensure the token successfully indexed in the end
			// if token, err = api.subscriber.GetTokensByIndexID(c, indexID); err != nil || token == nil {
			// 	logrus.WithError(err).Error("token is not successfully indexed")
			// 	return
			// }
		}
	}

	for _, accountID := range accounts {
		log.Info("send notification for the new token to related accounts",
			zap.String("accountID", accountID), zap.String("indexID", indexID))
		if err := api.subscriber.notifyNewNFT(accountID, req.To, indexID); err != nil {
			log.Error("fail to send notification for the new token",
				zap.Error(err),
				zap.String("accountID", accountID), zap.String("indexID", indexID))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}

func (api *EventSubscriberAPI) Run() error {
	r := gin.New()
	r.POST("/events", api.ReceiveEvents)

	return r.Run(viper.GetString("server.port"))
}
