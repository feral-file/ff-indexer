package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bitmark-inc/autonomy-logger"
	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
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
		input.Source = indexer.SourceFeralFile
	}

	tokenIndexIDs := []string{}
	for _, token := range input.Tokens {
		if token.ID == "" {
			abortWithError(c, http.StatusBadRequest, "invalid parameters", fmt.Errorf("invalid token id"))
			return
		}

		if token.IndexID == "" {
			token.IndexID = indexer.TokenIndexID(token.Blockchain, token.ContractAddress, token.ID)
		}

		tokenIndexIDs = append(tokenIndexIDs, token.IndexID)
	}

	if err := s.indexerStore.IndexAsset(c, assetID, input); err != nil {
		abortWithError(c, http.StatusInternalServerError, "unable to update asset data", err)
		return
	}

	if err := s.indexerStore.MarkAccountTokenChanged(c, tokenIndexIDs); err != nil {
		abortWithError(c, http.StatusInternalServerError, "unable to mark account token changed", err)
		return
	}

	nullProvenanceIDs, err := s.indexerStore.GetNullProvenanceTokensByIndexIDs(c, tokenIndexIDs)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "unable to find null provenance tokens", err)
		return
	}

	log.Info("start refresh null provenance tokens", zap.Any("tokenIDs", nullProvenanceIDs))
	for _, nullProvenanceID := range nullProvenanceIDs {
		go indexerWorker.StartRefreshTokenProvenanceWorkflow(c, s.cadenceWorker, "api-indexAsset", nullProvenanceID, 0)
	}

	c.JSON(http.StatusOK, gin.H{"ok": 1})
}

type RequestRefreshProvenanceWithOwner struct {
	Owner string `json:"owner" binding:"required"`
}

func (s *NFTIndexerServer) RefreshProvenanceWithOwner(c *gin.Context) {
	traceutils.SetHandlerTag(c, "RefreshProvenanceWithOwner")
	tokenID := c.Param("token_id")

	var input RequestRefreshProvenanceWithOwner

	if err := c.Bind(&input); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if err := s.indexerStore.UpdateOwner(c, tokenID, input.Owner, time.Now()); err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to update provenance", err)
		return
	}

	go indexerWorker.StartRefreshTokenProvenanceWorkflow(c, s.cadenceWorker, "api-refresh", tokenID, 0)

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}
