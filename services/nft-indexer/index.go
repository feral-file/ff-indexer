package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	cadenceClient "go.uber.org/cadence/client"
	"go.uber.org/zap"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/log"
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
	IDs []string `json:"ids"`

	// lastUpdatedAt
	LastUpdatedAt int64 `form:"lastUpdatedAt"`
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

	workflow, err := s.cadenceWorker.StartWorkflow(c, indexerWorker.ClientName, workflowContext, workflowFunc, owner)
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
		go s.startIndexWorkflow(c, owner, req.Blockchain, w.IndexOpenseaTokenWorkflow)
	case "tezos":
		go s.startIndexWorkflow(c, owner, req.Blockchain, w.IndexTezosTokenWorkflow)
	default:
		if strings.HasPrefix(owner, "0x") {
			go s.startIndexWorkflow(c, owner, indexer.BlockchainAlias[indexer.EthereumBlockchain], w.IndexOpenseaTokenWorkflow)
		} else if strings.HasPrefix(owner, "tz") {
			go s.startIndexWorkflow(c, owner, indexer.BlockchainAlias[indexer.TezosBlockchain], w.IndexTezosTokenWorkflow)
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
		Owner indexer.BlockchainAddress `json:"owner" binding:"required"`
	}

	if err := c.Bind(&req); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	// FIXME: remove this addition step by replace the input of startIndexWorkflow by indexer.BlockchainAddress
	owner := string(req.Owner)

	var ownerIndexFunc func(ctx context.Context, owner string, offset int) ([]indexer.AssetUpdates, error)
	switch indexer.GetBlockchainByAddress(owner) {
	case indexer.EthereumBlockchain:
		ownerIndexFunc = func(ctx context.Context, owner string, offset int) ([]indexer.AssetUpdates, error) {
			return s.indexerEngine.IndexETHTokenByOwner(c, owner, offset)
		}
	case indexer.TezosBlockchain:
		ownerIndexFunc = func(ctx context.Context, owner string, offset int) ([]indexer.AssetUpdates, error) {
			assetUpdates, _, err := s.indexerEngine.IndexTezosTokenByOwner(c, owner, time.Time{}, offset)
			return assetUpdates, err
		}
	default:
		abortWithError(c, http.StatusBadRequest, "unsupported blockchain", nil)
		return
	}

	var updates []indexer.AssetUpdates
	offset := 0
	for {
		u, err := ownerIndexFunc(c, owner, offset)
		if err != nil {
			abortWithError(c, http.StatusInternalServerError, "fail to index token", err)
			return
		}

		if len(u) == 0 {
			break
		}
		offset += len(u)

		updates = append(updates, u...)
	}

	c.JSON(200, gin.H{
		"updates": updates,
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

	// FIXME: remove this addition step by replace the input of startIndexWorkflow by indexer.BlockchainAddress
	owner := string(req.Owner)
	contract := string(req.Contract)

	if req.DryRun {
		u, err := s.indexerEngine.IndexToken(c, owner, contract, req.TokenID)
		if err != nil {
			abortWithError(c, http.StatusInternalServerError, "fail to index token", err)
			return
		}

		c.JSON(200, gin.H{
			"update": u,
		})
	} else {
		indexerWorker.StartIndexTokenWorkflow(c, s.cadenceWorker, owner, contract, req.TokenID, false, req.Preview)
		c.JSON(200, gin.H{
			"ok": 1,
		})
	}
}
