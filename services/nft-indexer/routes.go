package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	cadenceClient "go.uber.org/cadence/client"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
)

func (s *NFTIndexerServer) SetupRoute() {
	s.route.POST("/nft/query", s.QueryNFTs)
	s.route.POST("/nft/index", s.IndexNFTs)

	s.route.GET("/nft", s.ListNFTs)

	s.route.POST("/nft/query_price", s.QueryNFTPrices)

	s.route.Use(TokenAuthenticate("API-TOKEN", s.apiToken))
	s.route.PUT("/asset/:asset_id", s.IndexAsset)
}

// IndexAsset indexes the data of assets and tokens
func (s *NFTIndexerServer) IndexAsset(c *gin.Context) {
	assetID := c.Param("asset_id")
	var input indexer.AssetUpdates
	if err := c.Bind(&input); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if err := s.indexerStore.IndexAsset(c, assetID, input); err != nil {
		abortWithError(c, http.StatusInternalServerError, "unable to update asset data", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": 1})
}

// QueryNFTs queries NFTs based on given criteria
func (s *NFTIndexerServer) QueryNFTs(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}

	if err := c.Bind(&req); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	tokenInfo, err := s.indexerStore.GetTokens(c, req.IDs)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query tokens from indexer store", err)
		return
	}

	c.JSON(http.StatusOK, tokenInfo)
}

// ListNFTs returns information for a list of NFTs with some criterias.
// It currently only supports listing by owners.
func (s *NFTIndexerServer) ListNFTs(c *gin.Context) {
	var params struct {
		Owner string `form:"owner" binding:"required"`
	}

	if err := c.BindQuery(&params); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	tokenInfo, err := s.indexerStore.GetTokensByOwner(c, params.Owner)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query tokens from indexer store", err)
		return
	}

	c.JSON(http.StatusOK, tokenInfo)
}

// QueryNFTPrices returns prices information for NFTs
func (s *NFTIndexerServer) QueryNFTPrices(c *gin.Context) {
	abortWithError(c, http.StatusInternalServerError, "not implemented", nil)
}

func buildIndexNFTsContext(owner, blockchain string) cadenceClient.StartWorkflowOptions {
	return cadenceClient.StartWorkflowOptions{
		ID:                           fmt.Sprintf("index-%s-nft-by-owner-%s", blockchain, owner),
		TaskList:                     indexerWorker.TaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}
}

func (s *NFTIndexerServer) startIndexWorkflow(c context.Context, owner, blockchain string, workflowFunc interface{}) {
	workflowContext := buildIndexNFTsContext(owner, blockchain)

	workflow, err := s.cadenceWorker.StartWorkflow(c, indexerWorker.ClientName, workflowContext, workflowFunc, owner)
	if err != nil {
		log.WithError(err).WithField("owner", owner).Error("fail to start bitmark opensea indexing workflow")
	} else {
		log.WithField("owner", owner).WithField("workflow_id", workflow.ID).Info("start workflow for indexing opensea")
	}
}

func (s *NFTIndexerServer) IndexNFTs(c *gin.Context) {
	var req struct {
		Owner      string `json:"owner"`
		Blockchain string `json:"blockchain"`
	}

	if err := c.Bind(&req); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if req.Owner == "" {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", fmt.Errorf("missing parameters"))
		return
	}

	var w indexerWorker.NFTIndexerWorker

	blockchain := "eth"
	if req.Blockchain != "" {
		blockchain = req.Blockchain
	}

	switch blockchain {
	case "eth":
		go s.startIndexWorkflow(c, req.Owner, blockchain, w.IndexOpenseaTokenWorkflow)
	case "tezos":
		go s.startIndexWorkflow(c, req.Owner, blockchain, w.IndexTezosTokenWorkflow)
	default:
		abortWithError(c, http.StatusInternalServerError, "unsupported blockchain", fmt.Errorf("unsupported blockchain"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}
