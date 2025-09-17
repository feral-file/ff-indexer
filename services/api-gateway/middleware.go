package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	indexer "github.com/feral-file/ff-indexer"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

// TokenAuthenticate is the simplest authentication method based on a fixed key/value pair.
func TokenAuthenticate(tokenKey, tokenValue string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader(tokenKey)
		if token != tokenValue {
			abortWithError(c, http.StatusForbidden, "invalid api token", fmt.Errorf("invalid api token"))
			return
		}
		c.Next()
	}
}

// isIntrospectionQuery checks if the given query string is an introspection query.
func isIntrospectionQuery(query string) bool {
	lowerQuery := strings.ToLower(query)
	return strings.Contains(lowerQuery, "__schema") ||
		(strings.Contains(lowerQuery, "__type") && !strings.Contains(lowerQuery, "__typename"))
}

// validateSingleRequestPerOperation returns an error if any operation in the query
// contains more than one top-level field.
func validateSingleRequestPerOperation(query string) error {
	// Parse the query to get an AST
	parsedQuery, err := parser.ParseQuery(&ast.Source{Input: query})
	if err != nil {
		return err
	}

	// Iterate through all operations in the parsed query
	for _, operation := range parsedQuery.Operations {
		// Count top-level fields in the operation
		queryCount := 0
		for _, set := range operation.SelectionSet {
			// Workaround to ignore __typename query count for graphiql multiple queries
			if strings.Contains(fmt.Sprint(set), "__typename") {
				continue
			}
			queryCount++
		}
		if queryCount > 1 {
			return fmt.Errorf("multiple requests in a single operation are not allowed")
		}
	}

	return nil
}

func graphqlError(message string) string {
	return fmt.Sprintf(`{"errors":"%s"}`, message)
}

// GraphQLMiddleware blocks introspection and batch queries.
func GraphQLMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			next.ServeHTTP(w, r)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, graphqlError("Could not read request body"), http.StatusBadRequest)
			return
		}

		// Reset r.Body so it can be read again later.
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Parse the body query
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(bodyBytes, &params); err != nil {
			http.Error(w, graphqlError("Error parsing request body"), http.StatusBadRequest)
			return
		}

		// Validate the query to ensure it contains no more than one top-level field per operation
		if err := validateSingleRequestPerOperation(params.Query); err != nil {
			http.Error(w, graphqlError(err.Error()), http.StatusBadRequest)
			return
		}

		// Disable introspection in production environment
		environment := viper.GetString("environment")
		if environment != indexer.DevelopmentEnvironment {
			if isIntrospectionQuery(params.Query) {
				http.Error(w, graphqlError("Introspection queries are disabled"), http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
