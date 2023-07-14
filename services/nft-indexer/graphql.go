package main

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"

	"github.com/bitmark-inc/nft-indexer/services/nft-indexer/graph"
)

// Defining the Graphql handler
func (s *NFTIndexerServer) graphqlHandler(c *gin.Context) {
	h := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: graph.NewResolver(s.indexerStore, s.cacheClient, s.cadenceWorker)}))

	h.ServeHTTP(c.Writer, c.Request)
}

// Defining the Playground handler
func (s *NFTIndexerServer) playgroundHandler(c *gin.Context) {
	h := playground.Handler("Token", "/v2/graphql")

	h.ServeHTTP(c.Writer, c.Request)
}
