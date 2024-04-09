package objkt

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hasura/go-graphql-client"
)

type Time struct {
	time.Time
}

func (t *Time) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	s = strings.Split(s, "+")[0]
	tt, err := time.Parse("2006-01-02T15:04:05.999999", s)
	if err != nil {
		return err
	}

	t.Time = tt
	return nil
}

func (t *Time) MarshalJSON() ([]byte, error) {
	return t.Time.MarshalJSON()
}

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
	UpdatedAt   Time `graphql:"updated_at"`
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
func (g *Client) GetGalleryTokens(galleryPK string, offset, limit int) (SliceGalleryToken, error) {
	// NOTE: use `graphql.Client.Exec` to query since normal query doesn't support bigint varable
	query := fmt.Sprintf(`query{
		gallery_token(where: {gallery_pk: {_eq: %s}}, offset: %d, limit: %d) {
			token {
				fa_contract
				token_id
			}
		}
	}`, galleryPK, offset, limit)
	res := struct {
		GalleryToken SliceGalleryToken `graphql:"gallery_token"`
	}{}

	err := g.Client.Exec(context.Background(), query, &res, map[string]any{})
	if err != nil {
		return SliceGalleryToken{}, err
	}

	return res.GalleryToken, nil
}
