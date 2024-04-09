package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	cadenceClient "go.uber.org/cadence/client"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"
	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
	"github.com/bitmark-inc/nft-indexer/traceutils"
)

type PendingTxParamsV1 struct {
	Blockchain      string `json:"blockchain"`
	ID              string `json:"id"`
	ContractAddress string `json:"contractAddress"`
	OwnerAccount    string `json:"ownerAccount"`
	PublicKey       string `json:"publicKey"`
	Timestamp       string `json:"timestamp"`
	Signature       string `json:"signature"`
	PendingTx       string `json:"pendingTx"`
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
	IDs          []string `json:"ids"`
	CollectionID string   `json:"collectionID"`

	// lastUpdatedAt
	LastUpdatedAt int64  `form:"lastUpdatedAt"`
	SortBy        string `form:"sortBy"`
}

type CollectionQueryParams struct {
	// global
	Offset int64 `form:"offset"`
	Size   int64 `form:"size"`

	// list by owners
	Owners string `form:"owners"`
}

type TokenFeedbackParams struct {
	Tokens    []indexer.TokenFeedbackUpdate `json:"tokens"`
	RequestID string                        `json:"requestID"`
}

type RequestedTokenFeedback struct {
	DID       string          `json:"did"`
	Timestamp int64           `json:"timestamp"`
	Tokens    map[string]bool `json:"tokens"`
}

// FIXME: remove this and merge with background / helpers
func (s *NFTIndexerServer) startIndexWorkflow(c context.Context, owner, blockchain string, workflowFunc interface{}) {
	workflowContext := buildIndexNFTsContext(owner, blockchain)

	workflow, err := s.cadenceWorker.StartWorkflow(c, indexerWorker.ClientName, workflowContext, workflowFunc, owner, false)
	if err != nil {
		log.Error("fail to start indexing workflow", zap.Error(err), zap.String("owner", owner), zap.String("blockchain", blockchain))
	} else {
		log.Info("start workflow for indexing tokens", zap.String("owner", owner), zap.String("blockchain", blockchain), zap.String("workflow_id", workflow.ID))
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
	go indexerWorker.StartRefreshTokenProvenanceWorkflow(c, s.cadenceWorker, "api-swap-nft", swappedTokenIndexID, 0)

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

	go indexerWorker.StartRefreshTokenProvenanceWorkflow(c, s.cadenceWorker, "api-refresh", tokenID, 0)

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}

func (s *NFTIndexerServer) IndexNFTs(c *gin.Context) {
	traceutils.SetHandlerTag(c, "IndexNFTs")
	var req struct {
		Owner      indexer.BlockchainAddress `json:"owner"`
		Blockchain string                    `json:"blockchain"`
	}

	if err := c.Bind(&req); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	// FIXME: remove this addition step by replace the input of startIndexWorkflow by indexer.BlockchainAddress
	owner := string(req.Owner)

	var w indexerWorker.NFTIndexerWorker

	switch req.Blockchain {
	case "eth":
		go s.startIndexWorkflow(c, owner, req.Blockchain, w.IndexETHTokenWorkflow)
	case "tezos":
		go s.startIndexWorkflow(c, owner, req.Blockchain, w.IndexTezosTokenWorkflow)
	default:
		if strings.HasPrefix(owner, "0x") {
			go s.startIndexWorkflow(c, owner, indexer.BlockchainAlias[utils.EthereumBlockchain], w.IndexETHTokenWorkflow)
		} else if strings.HasPrefix(owner, "tz") {
			go s.startIndexWorkflow(c, owner, indexer.BlockchainAlias[utils.TezosBlockchain], w.IndexTezosTokenWorkflow)
		} else {
			abortWithError(c, http.StatusInternalServerError, "owner address with unsupported blockchain", fmt.Errorf("owner address with unsupported blockchain"))
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}

func (s *NFTIndexerServer) IndexNFTsV2(c *gin.Context) {
	traceutils.SetHandlerTag(c, "IndexNFTsV2")
	var req struct {
		Owner          indexer.BlockchainAddress `json:"owner"`
		IncludeHistory bool                      `json:"history"`
	}

	if err := c.Bind(&req); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	owner := req.Owner.String()
	blockchain := utils.GetBlockchainByAddress(owner)

	switch blockchain {
	case utils.EthereumBlockchain:
		indexerWorker.StartIndexETHTokenWorkflow(c, s.cadenceWorker, "indexer", owner, req.IncludeHistory)
	case utils.TezosBlockchain:
		indexerWorker.StartIndexTezosTokenWorkflow(c, s.cadenceWorker, "indexer", owner, req.IncludeHistory)
	default:
		abortWithError(c, http.StatusInternalServerError, "owner address with unsupported blockchain", fmt.Errorf("owner address with unsupported blockchain"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}

func (s *NFTIndexerServer) IndexOneNFT(c *gin.Context) {
	traceutils.SetHandlerTag(c, "IndexOneNFT")
	var req struct {
		Owner    indexer.BlockchainAddress `json:"owner"`
		Contract indexer.BlockchainAddress `json:"contract" binding:"required"`
		TokenID  string                    `json:"tokenID" binding:"required"`
		DryRun   bool                      `json:"dryrun"`
		Preview  bool                      `json:"preview"`
	}

	if err := c.Bind(&req); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	contract := req.Contract.String()

	if req.DryRun {
		u, err := s.indexerEngine.IndexToken(c, contract, req.TokenID)
		if err != nil {
			abortWithError(c, http.StatusInternalServerError, "fail to index token", err)
			return
		}

		c.JSON(200, gin.H{
			"update": u,
		})
	} else {
		owner := req.Owner.String()
		indexerWorker.StartIndexTokenWorkflow(c, s.cadenceWorker, owner, contract, req.TokenID, false, req.Preview)
		c.JSON(200, gin.H{
			"ok": 1,
		})
	}
}

func (s *NFTIndexerServer) IndexHistory(c *gin.Context) {
	traceutils.SetHandlerTag(c, "IndexHistory")

	var reqParams struct {
		IndexID string `json:"indexID" binding:"required"`
	}

	if err := c.Bind(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	token, err := s.indexerStore.GetTokenByIndexID(c, reqParams.IndexID)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "failed to get token", err)
		return
	}

	if token == nil {
		abortWithError(c, http.StatusBadRequest, "token does not exist", fmt.Errorf("token does not exist"))
		return
	}

	if token.Fungible {
		indexerWorker.StartRefreshTokenOwnershipWorkflow(c, s.cadenceWorker, "indexer", reqParams.IndexID, 0)
	} else {
		indexerWorker.StartRefreshTokenProvenanceWorkflow(c, s.cadenceWorker, "indexer", reqParams.IndexID, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}

func (s *NFTIndexerServer) SetTokenPendingV1(c *gin.Context) {
	s.SetTokenPending(c, false)
}

func (s *NFTIndexerServer) SetTokenPendingV2(c *gin.Context) {
	s.SetTokenPending(c, true)
}

func (s *NFTIndexerServer) SetTokenPending(c *gin.Context, withPrefix bool) {
	traceutils.SetHandlerTag(c, "TokenPending")

	var reqParams PendingTxParamsV1

	if err := c.BindQuery(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if err := c.Bind(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if reqParams.PendingTx == "" {
		abortWithError(c, http.StatusBadRequest, "invalid parameter", fmt.Errorf("pendingTx is required"))
		return
	}

	createdAt, err := utils.EpochStringToTime(reqParams.Timestamp)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameter", err)
		return
	}

	now := time.Now()
	if !utils.IsTimeInRange(createdAt, now, 5) {
		abortWithError(c, http.StatusBadRequest, "invalid parameter", fmt.Errorf("request time too skewed"))
		return
	}

	message := reqParams.Timestamp
	if withPrefix {
		jsonMessage, err := json.Marshal(struct {
			Blockchain      string `json:"blockchain"`
			ID              string `json:"id"`
			ContractAddress string `json:"contractAddress"`
			OwnerAccount    string `json:"ownerAccount"`
			Timestamp       string `json:"timestamp"`
		}{
			Blockchain:      reqParams.Blockchain,
			ID:              reqParams.ID,
			ContractAddress: reqParams.ContractAddress,
			OwnerAccount:    reqParams.OwnerAccount,
			Timestamp:       reqParams.Timestamp,
		})
		if err != nil {
			abortWithError(c, http.StatusInternalServerError, "error marshall json message", err)
			return
		}
		message = indexer.GetPrefixedSigningMessage(string(jsonMessage))
	}

	isValidAddress, err := s.verifyAddressOwner(
		reqParams.Blockchain,
		message,
		reqParams.Signature,
		reqParams.OwnerAccount,
		reqParams.PublicKey,
	)

	if err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if !isValidAddress {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", fmt.Errorf("invalid signature for ownerAddress"))
		return
	}

	indexID := indexer.TokenIndexID(reqParams.Blockchain, reqParams.ContractAddress, reqParams.ID)

	if err := s.indexerStore.AddPendingTxToAccountToken(c, reqParams.OwnerAccount, indexID, reqParams.PendingTx, reqParams.Blockchain, reqParams.ID); err != nil {
		log.Warn("error while adding pending accountToken", zap.Error(err), zap.String("owner", reqParams.OwnerAccount), zap.String("pendingTx", reqParams.PendingTx))
		return
	}
	log.Info("a pending account token is added", zap.String("owner", reqParams.OwnerAccount), zap.String("pendingTx", reqParams.PendingTx))

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}

func (s *NFTIndexerServer) IndexCollections(c *gin.Context) {
	traceutils.SetHandlerTag(c, "IndexCollections")
	var req struct {
		Addresses []indexer.BlockchainAddress `json:"addresses"`
	}

	if err := c.Bind(&req); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	for _, addr := range req.Addresses {
		owner := addr.String()
		blockchain := utils.GetBlockchainByAddress(owner)

		switch blockchain {
		case utils.EthereumBlockchain:
			log.Debug("Not implemented")
		case utils.TezosBlockchain:
			indexerWorker.StartIndexTezosCollectionWorkflow(c, s.cadenceWorker, "indexer", owner)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}
