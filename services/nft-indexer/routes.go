package main

import (
	"time"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (s *NFTIndexerServer) SetupRoute() {
	s.route.Use(sentrygin.New(sentrygin.Options{
		Repanic: true,
	}))

	s.route.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           24 * time.Hour,
	}))

	s.route.POST("/nft/index", s.IndexNFTs)
	s.route.POST("/nft/:token_id/provenance", s.RefreshProvenance)

	s.route.POST("/nft/query", s.QueryNFTs)
	s.route.POST("/v1/nft/query", s.QueryNFTsV1)
	s.route.GET("/nft/search", s.SearchNFTs)
	s.route.GET("/nft/owned", s.OwnedNFTIDs)
	s.route.GET("/nft", s.ListNFTs)
	s.route.GET("/v1/nft", s.GetAccountNFTs)

	s.route.POST("/nft/index_one", s.IndexOneNFT)
	s.route.POST("/nft/index_owner", s.IndexNFTByOwner)

	s.route.GET("/identity/:account_number", s.GetIdentity)
	s.route.POST("/identity/", s.GetIdentities)

	s.route.POST("/nft/swap", TokenAuthenticate("API-TOKEN", s.apiToken), s.SwapNFT)
	s.route.PUT("/asset/:asset_id", TokenAuthenticate("API-TOKEN", s.apiToken), s.IndexAsset)

	s.route.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"message": "this is not what you are looking for",
		})
	})
}

// QueryNFTPrices returns prices information for NFTs
// func (s *NFTIndexerServer) QueryNFTPrices(c *gin.Context) {
// 	abortWithError(c, http.StatusInternalServerError, "not implemented", nil)
// }

// // PushNFTPrice returns push an trade price information to a specific NFT
// func (s *NFTIndexerServer) PushNFTPrice(c *gin.Context) {
// 	tokenID := c.Param("token_id")
// 	var input indexer.PriceUpdate
// 	if err := c.Bind(&input); err != nil {
// 		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
// 		return
// 	}

// 	if err := s.indexerStore.UpdateTokenPrice(c, tokenID, input); err != nil {
// 		abortWithError(c, http.StatusInternalServerError, "unable to push token price", err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"ok": 1,
// 	})
// }
