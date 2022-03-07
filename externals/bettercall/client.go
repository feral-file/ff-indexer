package bettercall

import (
	"encoding/json"
	"fmt"
	"math/big"
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

type TokenID struct {
	big.Int
}

func (b TokenID) MarshalJSON() ([]byte, error) {
	return []byte(b.String()), nil
}

func (b *TokenID) UnmarshalJSON(p []byte) error {
	s := string(p)
	if s == "null" {
		return fmt.Errorf("invalid token id: %s", p)
	}

	z, ok := big.NewInt(0).SetString(s, 0)
	if !ok {
		return fmt.Errorf("invalid token id: %s", p)
	}

	b.Int = *z
	return nil
}

type Token struct {
	Contract     string       `json:"contract"`
	Network      string       `json:"network"`
	ID           TokenID      `json:"token_id"`
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

type TokenMetadata struct {
	Contract  string    `json:"contract"`
	Timestamp time.Time `json:"timestamp"`
	TokenID   TokenID   `json:"token_id"`
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

func (c *BetterCall) GetTokenMetadata(contract string, tokenID string) (TokenMetadata, error) {
	v := url.Values{
		"contract": []string{contract},
		"token_id": []string{tokenID},
	}

	u := url.URL{
		Scheme:   "https",
		Host:     "api.better-call.dev",
		Path:     "/v1/tokens/mainnet/metadata",
		RawQuery: v.Encode(),
	}

	resp, err := c.client.Get(u.String())
	if err != nil {
		return TokenMetadata{}, err
	}
	defer resp.Body.Close()

	var metadata []TokenMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return TokenMetadata{}, err
	}

	switch len(metadata) {
	case 0:
		return TokenMetadata{}, fmt.Errorf("no metadata found")
	case 1:
		return metadata[0], nil
	default:
		return TokenMetadata{}, fmt.Errorf("more than one metadata a the response")
	}
}
