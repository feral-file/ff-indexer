package fxhash

import (
	"context"
	"math/big"
	"net/http"
	"time"

	"github.com/hasura/go-graphql-client"
)

// GraphQL Explorer: https://api.fxhash.xyz/graphiql
// API: https://api.fxhash.xyz/graphql
type FxHashAPI struct {
	client   *graphql.Client
	endpoint string
}

type URLMetadata struct {
	Description  string `json:"description"`
	ArtifactURI  string `json:"artifactUri"`
	DisplayURI   string `json:"displayUri"`
	ThumbnailURI string `json:"thumbnailUri"`
}

func New(graphQLEndpoint string) *FxHashAPI {
	var c = &http.Client{
		Timeout: 10 * time.Second,
	}

	client := graphql.NewClient(graphQLEndpoint, c)

	return &FxHashAPI{
		client:   client,
		endpoint: graphQLEndpoint,
	}
}

// {
// 	objkt(id: 358743) {
// 	  name
// 	  createdAt
// 	  issuer {
// 		name
// 		slug
// 		author {
// 		  id
// 		  name
// 		}
// 	  }
// 	}
// }
type FxHashObjectDetail struct {
	Name      string
	CreatedAt time.Time
	Iteration int64
	Metadata  URLMetadata `scalar:"true"`
	Issuer    struct {
		Supply int64
		Author struct {
			ID   string
			Name string
		}
	}
}

// GetObjectDetail returns an object detail for fxhash nfts
func (api *FxHashAPI) GetObjectDetail(c context.Context, id big.Int) (FxHashObjectDetail, error) {
	var query struct {
		Object FxHashObjectDetail `graphql:"objkt(id: $id) "`
	}

	if err := api.client.Query(c, &query, map[string]interface{}{
		"id": graphql.Float(id.Int64()),
	}); err != nil {
		return FxHashObjectDetail{}, err
	}

	return query.Object, nil
}
