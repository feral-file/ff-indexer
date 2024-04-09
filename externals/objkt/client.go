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

func New(network string) *Client {
	endpoint := "https://data.objkt.com/v3/graphql"
	if network == "testnet" {
		endpoint = "https://data.ghostnet.objkt.com/v3/graphql"
	}

	var c = &http.Client{
		Timeout: 10 * time.Second,
	}

	client := graphql.NewClient(endpoint, c)

	return &Client{
		Client:   client,
		Endpoint: endpoint,
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

type SliceGallery []struct {
	Gallery
}

type Gallery struct {
	Name        string
	Slug        string
	Editions    int
	Description string
	Items       int
	Logo        string
	GalleryID   string `graphql:"gallery_id"`
	PK          int64  `graphql:"pk"`
	Registry    Registry
	Published   bool
}

type Registry struct {
	ID   int
	Name string
	Slug string
}

type SliceGalleryToken []struct {
	GalleryToken
}

type GalleryToken struct {
	Token Token
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

// GetGalleries query Objkt galleries from Objkt API
func (g *Client) GetGalleries(address string, offset, limit int) (SliceGallery, error) {
	var query struct {
		SliceGallery `graphql:"gallery(where: {curators: {curator_address: {_eq: $address}}}, offset: $offset, limit: $limit)"`
	}

	variables := map[string]interface{}{
		"address": graphql.String(address),
		"offset":  graphql.Int(offset),
		"limit":   graphql.Int(limit),
	}

	err := g.Client.Query(context.Background(), &query, variables)
	if err != nil {
		return SliceGallery{}, err
	}

	return query.SliceGallery, nil
}

// GetGalleryToken query Objkt gallery tokens from Objkt API
func (g *Client) GetGalleryTokens(galleryPK int64, offset, limit int) (SliceGalleryToken, error) {
	var query struct {
		SliceGalleryToken `graphql:"gallery_token(where: {gallery_pk: {_eq: $gallery_pk}}, offset: $offset, limit: $limit)"`
	}

	variables := map[string]interface{}{
		"gallery_pk": graphql.Int(galleryPK),
		"offset":     graphql.Int(offset),
		"limit":      graphql.Int(limit),
	}

	err := g.Client.Query(context.Background(), &query, variables)
	if err != nil {
		return SliceGalleryToken{}, err
	}

	return query.SliceGalleryToken, nil
}
