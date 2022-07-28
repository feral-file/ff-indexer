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
	Offset int64  `form:"offset"`
	Size   int64  `form:"size"`
	Source string `form:"source"`

	// list by owner
	Owner string `form:"owner"`
	// text search
	Text string `form:"text"`

	// query tokens
	IDs []string `json:"ids"`
}

// FIXME: remove this and merge with background / helpers
func (s *NFTIndexerServer) startIndexWorkflow(c context.Context, owner, blockchain string, workflowFunc interface{}) {
	workflowContext := buildIndexNFTsContext(owner, blockchain)

	workflow, err := s.cadenceWorker.StartWorkflow(c, indexerWorker.ClientName, workflowContext, workflowFunc, owner)
	if err != nil {
		log.WithError(err).WithField("owner", owner).WithField("blockchain", blockchain).Error("fail to start indexing workflow")
	} else {
		log.WithField("owner", owner).WithField("blockchain", blockchain).WithField("workflow_id", workflow.ID).Info("start workflow for indexing tokens")
	}
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
	go indexerWorker.StartRefreshTokenProvenanceWorkflow(c, s.cadenceWorker, fmt.Sprintf("swap-%s", input.NewTokenID), swappedTokenIndexID, 0)

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

func (s *NFTIndexerServer) RefreshProvenance(c *gin.Context) {
	traceutils.SetHandlerTag(c, "RefreshProvenance")
	tokenID := c.Param("token_id")

	go indexerWorker.StartRefreshTokenProvenanceWorkflow(c, s.cadenceWorker, fmt.Sprintf("api-%s", tokenID), tokenID, 0)

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

func (s *NFTIndexerServer) IndexNFTByOwner(c *gin.Context) {
	traceutils.SetHandlerTag(c, "IndexNFTByOwner")
	var req struct {
		Owner string `json:"owner" binding:"required"`
	}

	if err := c.Bind(&req); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	var ownerIndexFunc func(ctx context.Context, owner string, offset int) ([]indexer.AssetUpdates, error)
	switch indexer.DetectAccountBlockchain(req.Owner) {
	case indexer.EthereumBlockchain:
		ownerIndexFunc = func(ctx context.Context, owner string, offset int) ([]indexer.AssetUpdates, error) {
			return s.indexerEngine.IndexETHTokenByOwner(c, owner, offset)
		}
	case indexer.TezosBlockchain:
		ownerIndexFunc = func(ctx context.Context, owner string, offset int) ([]indexer.AssetUpdates, error) {
			return s.indexerEngine.IndexTezosTokenByOwner(c, owner, offset)
		}
	default:
		abortWithError(c, http.StatusBadRequest, "unsupported blockchain", nil)
		return
	}

	var updates []indexer.AssetUpdates
	offset := 0
	for {
		u, err := ownerIndexFunc(c, req.Owner, offset)
		if err != nil {
			abortWithError(c, http.StatusInternalServerError, "fail to index token", err)
			return
		}

		if len(u) == 0 {
			break
		} else {
			offset += len(u)
		}

		updates = append(updates, u...)
	}

	c.JSON(200, gin.H{
		"updates": updates,
	})
}

func (s *NFTIndexerServer) IndexOneNFT(c *gin.Context) {
	traceutils.SetHandlerTag(c, "IndexOneNFT")
	var req struct {
		Owner    string `json:"owner"`
		Contract string `json:"contract" binding:"required"`
		TokenID  string `json:"tokenID" binding:"required"`
		DryRun   bool   `json:"dryrun"`
	}

	if err := c.Bind(&req); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}
	if req.DryRun {
		u, err := s.indexerEngine.IndexToken(c, req.Owner, req.Contract, req.TokenID)
		if err != nil {
			abortWithError(c, http.StatusInternalServerError, "fail to index token", err)
			return
		}

		c.JSON(200, gin.H{
			"update": u,
		})
	} else {
		indexerWorker.StartIndexTokenWorkflow(c, s.cadenceWorker, req.Owner, req.Contract, req.TokenID)
		c.JSON(200, gin.H{
			"ok": 1,
		})
	}
}
