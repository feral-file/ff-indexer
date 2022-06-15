package main

import (
	"context"
	"net/http"
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type EventSubscriberAPI struct {
	subscriber *NFTEventSubscriber
}

func NewEventSubscriberAPI(s *NFTEventSubscriber) *EventSubscriberAPI {
	return &EventSubscriberAPI{
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
}

func (api *EventSubscriberAPI) ReceiveEvents(c *gin.Context) {
	var req RequestNewEvent

	if err := c.Bind(&req); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	tokenBlockchain := indexer.DetectContractBlockchain(req.Contract)

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
	//       - send notificiation

	// TODO: do we need to move this account specific function out of this service
	accounts, err := api.subscriber.GetAccountIDByAddress(req.To)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	token, err := api.subscriber.GetTokensByIndexID(c, indexID)
	if err != nil {
		logrus.WithError(err).Error("fail to check token by index ID")
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
		logrus.WithField("indexID", indexID).Info("an indexed token found for a corresponded event")
		if err := api.subscriber.UpdateOwner(c, indexID, req.To, req.Timestamp); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "fail to update owner for the token",
			})
		}
	} else {
		// index the new token since it is a new token for our indexer and watched by our user
		if len(accounts) > 0 {
			logrus.WithField("indexID", indexID).
				WithField("from", req.From).WithField("to", req.To).
				Info("start indexing a new token")

			indexerWorker.StartIndexTokenWorkflow(c, &api.subscriber.Worker, req.To, req.Contract, req.TokenID)

			// ensure the token successfully indexed in the end
			// if token, err = api.subscriber.GetTokensByIndexID(c, indexID); err != nil || token == nil {
			// 	logrus.WithError(err).Error("token is not successfully indexed")
			// 	return
			// }
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}

func (api *EventSubscriberAPI) Run(ctx context.Context) error {
	r := gin.New()
	r.POST("/events", api.ReceiveEvents)

	return r.Run(viper.GetString("server.port"))
}
