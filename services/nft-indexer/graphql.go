package main

import (
	"context"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/bitmark-inc/nft-indexer/services/nft-indexer/graph"
)

// Defining the Graphql handler
func (s *NFTIndexerServer) graphqlHandler(c *gin.Context) {
	server := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: graph.NewResolver(s.indexerStore, s.cacheStore, s.ethClient, s.cadenceWorker)}))
	server.SetErrorPresenter(graphqlErrorPresenter)

	filteredHandler := GraphQLMiddleware(server)
	filteredHandler.ServeHTTP(c.Writer, c.Request)
}

// Defining the Playground handler
func (s *NFTIndexerServer) playgroundHandler(c *gin.Context) {
	h := playground.Handler("Token", "/v2/graphql")

	h.ServeHTTP(c.Writer, c.Request)
}

// graphqlErrorPresenter modifies the error before sending it to the client.
func graphqlErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	gqlErr := graphql.DefaultErrorPresenter(ctx, err)

	// Remove the hint from the error message.
	parts := strings.Split(gqlErr.Message, "Did you mean")
	gqlErr.Message = strings.TrimSpace(parts[0])

	return gqlErr
}
