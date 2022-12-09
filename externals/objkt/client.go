package objkt

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hasura/go-graphql-client"
)

type ObjktAPI struct {
	Client   *graphql.Client
	Endpoint string
}

func New(graphQLEndpoint string) *ObjktAPI {
	var c = &http.Client{
		Timeout: 10 * time.Second,
	}

	client := graphql.NewClient(graphQLEndpoint, c)

	return &ObjktAPI{
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
	DisplayUri   string `graphql:"display_uri"`
	ThumbnailUri string `graphql:"thumbnail_uri"`
	ArtifactUri  string `graphql:"artifact_uri"`
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
func (g *ObjktAPI) GetObjectToken(contract string, token_id string) (Token, error) {
	var query struct {
		SliceToken `graphql:"token(where: {token_id: {_eq: $token_id}, fa_contract: {_eq: $fa_contract}})"`
	}

	variables := map[string]interface{}{
		"token_id":    graphql.String(token_id),
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
