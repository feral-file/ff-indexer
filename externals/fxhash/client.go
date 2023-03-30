package fxhash

import (
	"context"
	"math/big"
	"net/http"
	"time"

	"github.com/hasura/go-graphql-client"
)

// GraphQL Explorer: https://api.fxhash.xyz/graphiql
// Client: https://api.fxhash.xyz/graphql
type Client struct {
	client   *graphql.Client
	endpoint string
}

type URLMetadata struct {
	Description  string `json:"description"`
	ArtifactURI  string `json:"artifactUri"`
	DisplayURI   string `json:"displayUri"`
	ThumbnailURI string `json:"thumbnailUri"`
}

func New(graphQLEndpoint string) *Client {
	var c = &http.Client{
		Timeout: 10 * time.Second,
	}

	client := graphql.NewClient(graphQLEndpoint, c)

	return &Client{
		client:   client,
		endpoint: graphQLEndpoint,
	}
}

//	{
//		objkt(id: 358743) {
//		  name
//		  createdAt
//		  issuer {
//			name
//			slug
//			author {
//			  id
//			  name
//			}
//		  }
//		}
//	}
type ObjectDetail struct {
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

// This is a special type of fxhash graphql
type ObjktId int64 //nolint

// GetObjectDetail returns an object detail for fxhash nfts
func (api *Client) GetObjectDetail(c context.Context, id big.Int) (ObjectDetail, error) {
	var query struct {
		Object ObjectDetail `graphql:"objkt(id: $id) "`
	}

	if err := api.client.Query(c, &query, map[string]interface{}{
		"id": ObjktId(id.Int64()),
	}); err != nil {
		return ObjectDetail{}, err
	}

	return query.Object, nil
}
