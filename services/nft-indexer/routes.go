package main

import (
	"net/http"
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
	s.route.GET("/nft/search", s.SearchNFTs)
	s.route.GET("/nft/owned", s.OwnedNFTIDs)
	s.route.GET("/nft", s.ListNFTs)

	s.route.POST("/nft/index_one", s.IndexOneNFT)

	s.route.GET("/identity/:account_number", s.GetIdentity)
	s.route.POST("/identity/", s.GetIdentities)

	s.route.POST("/nft/swap", TokenAuthenticate("API-TOKEN", s.apiToken), s.SwapNFT)
	s.route.PUT("/asset/:asset_id", TokenAuthenticate("API-TOKEN", s.apiToken), s.IndexAsset)

	s.route.GET("/eth/:block_hash/block_time", s.GetETHBlockTime)

	s.route.GET("/exchange_rate", s.GetExchangeRate)

	v1 := s.route.Group("/v1")
	v1NFT := v1.Group("/nft")
	v1NFT.GET("/owned", s.OwnedNFTIDs)

	v1.POST("/admin/demo-tokens/", TokenAuthenticate("API-TOKEN", s.adminAPIToken), s.CreateDemoTokens)
	v1.POST("/admin/force-reindex-nft/", TokenAuthenticate("API-TOKEN", s.adminAPIToken), s.ForceReindexNFT)

	feralfileAPI := v1.Group("/feralfile", TokenAuthenticate("API-TOKEN", s.apiToken))
	feralfileAPI.POST("/nft/:token_id/provenance", s.RefreshProvenanceWithOwner)
	feralfileAPI.POST("/nft/swap", s.SwapNFT)
	feralfileAPI.PUT("/asset/:asset_id", s.IndexAsset)

	// temp while gRPC is ported to FF
	feralfileAPI.POST("/salests", s.SalesTimeSeries)

	v2 := s.route.Group("/v2")
	v2NFT := v2.Group("/nft")
	v2NFT.GET("", s.GetAccountNFTsV2)
	v2NFT.GET("/count", s.CountAccountNFTsV2)
	v2NFT.POST("/query", s.QueryNFTsV2)
	v2NFT.POST("/index_one", s.IndexOneNFT)
	v2NFT.POST("/index", s.IndexNFTsV2)
	v2NFT.POST("/index_history", s.IndexHistory)

	v2Collections := v2.Group("/collections")
	v2Collections.POST("/index", s.IndexCollections)
	v2Collections.GET("", s.GetCollectionsByCreators)
	v2Collections.GET("/:collection_id", s.GetCollectionByID)

	v2.POST("/graphql", s.graphqlHandler)
	v2.GET("/graphiql", s.playgroundHandler)

	s.route.GET("/healthz", func(c *gin.Context) {
		if err := s.indexerStore.Healthz(c); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		if err := s.cacheStore.Healthz(c); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok": 1,
		})
	})

	s.route.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"message": "this is not what you are looking for",
		})
	})
}
