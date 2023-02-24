package objkt

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hasura/go-graphql-client"
)

type Client struct {
	Client   *graphql.Client
	Endpoint string
}

func New(graphQLEndpoint string) *Client {
	var c = &http.Client{
		Timeout: 10 * time.Second,
	}

	client := graphql.NewClient(graphQLEndpoint, c)

	return &Client{
		Client:   client,
		Endpoint: graphQLEndpoint,
	}
}

type SliceToken []struct {
	Token
}

type Token struct {
	Name         string
	Description  string
	Mime         string
	DisplayURI   string `graphql:"display_uri"`
	ThumbnailURI string `graphql:"thumbnail_uri"`
	ArtifactURI  string `graphql:"artifact_uri"`
	TokenID      string `graphql:"token_id"`
	FaContract   string `graphql:"fa_contract"`
	Creators     []Creators
}

type Creators struct {
	Holder Holder
}

type Holder struct {
	Alias     string
	Website   string
	Address   string
	Facebook  string
	Github    string
	Gitlab    string
	Instagram string
	Medium    string
	Reddit    string
	Telegram  string
	Twitter   string
}

// GetObjectToken query Objkt Token object from Objkt API
func (g *Client) GetObjectToken(contract string, tokenID string) (Token, error) {
	var query struct {
		SliceToken `graphql:"token(where: {token_id: {_eq: $tokenID}, fa_contract: {_eq: $fa_contract}})"`
	}

	variables := map[string]interface{}{
		"tokenID":     graphql.String(tokenID),
		"fa_contract": graphql.String(contract),
	}

	err := g.Client.Query(context.Background(), &query, variables)
	if err != nil {
		return Token{}, err
	}

	if len(query.SliceToken) == 0 {
		return Token{}, fmt.Errorf("there is no token in objkt")
	}

	return query.SliceToken[0].Token, nil
}
