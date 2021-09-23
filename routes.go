package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *NFTIndexerServer) SetupRoute() {
	s.route.POST("/nft/query", s.QueryNFTs)
	s.route.GET("/nft", s.ListNFTs)
	s.route.POST("/nft/query_price", s.QueryNFTPrices)

	s.route.Use(TokenAuthenticate("API-TOKEN", s.apiToken))
	s.route.PUT("/asset/:asset_id", s.IndexAsset)
}

// IndexAsset indexes the data of assets and tokens
func (s *NFTIndexerServer) IndexAsset(c *gin.Context) {
	assetID := c.Param("asset_id")
	var input AssetUpdates
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
