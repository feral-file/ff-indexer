package main

import (
	"net/http"
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/traceutils"
	"github.com/gin-gonic/gin"
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

	if err := s.indexerStore.IndexAsset(c, assetID, input); err != nil {
		abortWithError(c, http.StatusInternalServerError, "unable to update asset data", err)
		return
	}

	accountToken := indexer.AccountToken{
		BaseTokenInfo:     input.Tokens[0].BaseTokenInfo,
		IndexID:           input.Tokens[0].IndexID,
		OwnerAccount:      input.Tokens[0].Owner,
		Balance:           input.Tokens[0].Balance,
		LastActivityTime:  input.Tokens[0].LastActivityTime,
		LastRefreshedTime: input.Tokens[0].LastRefreshedTime,
	}

	if err := s.indexerStore.IndexAccountTokens(c, input.Tokens[0].Owner, []indexer.AccountToken{accountToken}); err != nil {
		abortWithError(c, http.StatusInternalServerError, "unable to update account token data", err)
		return
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
