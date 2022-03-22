package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	cadenceClient "go.uber.org/cadence/client"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/traceutils"
)

// IndexAsset indexes the data of assets and tokens
func (s *NFTIndexerServer) IndexAsset(c *gin.Context) {
	traceutils.SetHandlerTag(c, "IndexAsset")
	assetID := c.Param("asset_id")
	var input indexer.AssetUpdates
	if err := c.Bind(&input); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if input.Source == "" {
		input.Source = "feralfile"
	}

	if err := s.indexerStore.IndexAsset(c, assetID, input); err != nil {
		abortWithError(c, http.StatusInternalServerError, "unable to update asset data", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": 1})
}

type NFTQueryParams struct {
	// global
	Offset int64 `form:"offset"`
	Size   int64 `form:"size"`

	// list by owner
	Owner string `form:"owner"`
	// text search
	Text string `form:"text"`

	// query tokens
	IDs []string `json:"ids"`
}

// SwapNFT migrate existent nft from a blockchain to another
func (s *NFTIndexerServer) SwapNFT(c *gin.Context) {
	traceutils.SetHandlerTag(c, "SwapNFT")

	var input indexer.SwapUpdate
	if err := c.Bind(&input); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	swappedTokenIndexID, err := s.indexerStore.SwapToken(c, input)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, fmt.Sprintf("unable to swap token. error: %s", err.Error()), err)
		return
	}

	// trigger refreshing provenance to merge two blockchain provenance
	go s.startRefreshProvenanceWorkflow(context.Background(), fmt.Sprintf("swap-%s", input.NewTokenID), []string{swappedTokenIndexID}, 0)

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
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
		log.WithError(err).WithField("owner", owner).WithField("blockchain", blockchain).Error("fail to start indexing workflow")
	} else {
		log.WithField("owner", owner).WithField("workflow_id", workflow.ID).Info("start workflow for indexing opensea")
	}
}

func (s *NFTIndexerServer) startRefreshProvenanceWorkflow(c context.Context, refreshProvenanceTaskID string, indexIDs []string, delay time.Duration) {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           fmt.Sprintf("index-token-%s-provenance", refreshProvenanceTaskID),
		TaskList:                     indexerWorker.TaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w indexerWorker.NFTIndexerWorker

	workflow, err := s.cadenceWorker.StartWorkflow(c, indexerWorker.ClientName, workflowContext, w.RefreshTokenProvenanceWorkflow, indexIDs, delay)
	if err != nil {
		log.WithError(err).WithField("refreshProvenanceTaskID", refreshProvenanceTaskID).Error("fail to start refreshing provenance workflow")
	} else {
		log.WithField("refreshProvenanceTaskID", refreshProvenanceTaskID).WithField("workflow_id", workflow.ID).Info("start workflow for refreshing provenance")
	}
}

func (s *NFTIndexerServer) RefreshProvenance(c *gin.Context) {
	traceutils.SetHandlerTag(c, "RefreshProvenance")
	tokenID := c.Param("token_id")

	go s.startRefreshProvenanceWorkflow(context.Background(), tokenID, []string{tokenID}, 0)

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}

func (s *NFTIndexerServer) IndexNFTs(c *gin.Context) {
	traceutils.SetHandlerTag(c, "IndexNFTs")
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

	switch req.Blockchain {
	case "eth":
		go s.startIndexWorkflow(c, req.Owner, req.Blockchain, w.IndexOpenseaTokenWorkflow)
	case "tezos":
		go s.startIndexWorkflow(c, req.Owner, req.Blockchain, w.IndexTezosTokenWorkflow)
	default:
		if strings.HasPrefix(req.Owner, "0x") {
			go s.startIndexWorkflow(c, req.Owner, indexer.BlockchianAlias[indexer.EthereumBlockchain], w.IndexOpenseaTokenWorkflow)
		} else if strings.HasPrefix(req.Owner, "tz") {
			go s.startIndexWorkflow(c, req.Owner, indexer.BlockchianAlias[indexer.TezosBlockchain], w.IndexTezosTokenWorkflow)
		} else {
			abortWithError(c, http.StatusInternalServerError, "owner address with unsupported blockchain", fmt.Errorf("owner address with unsupported blockchain"))
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}
