package bettercall

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type BetterCall struct {
	client *http.Client
}

func New() *BetterCall {
	return &BetterCall{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type FileFormat struct {
	MIMEType string `json:"mimeType"`
	URI      string `json:"uri"`
}

type Token struct {
	Contract     string       `json:"contract"`
	Network      string       `json:"network"`
	ID           int          `json:"token_id"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Symbol       string       `json:"symbol"`
	ArtifactUri  string       `json:"artifact_uri"`
	DisplayUri   string       `json:"display_uri"`
	ThumbnailUri string       `json:"thumbnail_uri"`
	Creators     []string     `json:"creators"`
	Formats      []FileFormat `json:"formats"`
}

type TokenResponse struct {
	Tokens []Token `json:"balances"`
	Total  int64   `json:"total"`
}

func (c *BetterCall) RetrieveTokens(owner string, offset int) ([]Token, error) {
	v := url.Values{
		"size":       []string{"50"},
		"offset":     []string{fmt.Sprintf("%d", offset)},
		"hide_empty": []string{"true"},
	}

	u := url.URL{
		Scheme:   "https",
		Host:     "api.better-call.dev",
		Path:     fmt.Sprintf("/v1/account/mainnet/%s/token_balances", owner),
		RawQuery: v.Encode(),
	}

	resp, err := c.client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResponse TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, err
	}

	return tokenResponse.Tokens, nil
}
