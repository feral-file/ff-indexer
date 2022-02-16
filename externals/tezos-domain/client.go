package tezosDomain

import (
	"context"
	"net/http"
	"time"

	"github.com/hasura/go-graphql-client"
)

type TezosDomainAPI struct {
	client   *graphql.Client
	endpoint string
}

func New(graphQLEndpoint string) *TezosDomainAPI {
	var c = &http.Client{
		Timeout: 10 * time.Second,
	}

	client := graphql.NewClient(graphQLEndpoint, c)

	return &TezosDomainAPI{
		client:   client,
		endpoint: graphQLEndpoint,
	}
}

// {
//   reverseRecord(address: "") {
//     domain{
//       name
//     }
//   }
// }
func (api *TezosDomainAPI) ResolveDomain(c context.Context, address string) (string, error) {
	var query struct {
		Record struct {
			Domain struct {
				Name string
			}
		} `graphql:"reverseRecord(address: $address) "`
	}

	if err := api.client.Query(c, &query, map[string]interface{}{
		"address": graphql.String(address),
	}); err != nil {
		return "", err
	}

	return query.Record.Domain.Name, nil
}
