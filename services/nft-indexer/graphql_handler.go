package main

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/bitmark-inc/nft-indexer/services/nft-indexer/graph"
	"github.com/gin-gonic/gin"
)

// Defining the Graphql handler
func (s *NFTIndexerServer) graphqlHandler(c *gin.Context) {
	h := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: graph.NewResolver(s.indexerStore)}))

	h.ServeHTTP(c.Writer, c.Request)
}

// Defining the Playground handler
func (s *NFTIndexerServer) playgroundHandler(c *gin.Context) {
	h := playground.Handler("Token", "/v1/graphql/query")

	h.ServeHTTP(c.Writer, c.Request)
}
