package tzkt

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TZKT struct {
	endpoint string

	client *http.Client
}

func New(endpoint string) *TZKT {
	return &TZKT{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		endpoint: endpoint,
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

	z, ok := big.NewInt(0).SetString(strings.Trim(s, `"`), 0)
	if !ok {
		return fmt.Errorf("invalid token id: %s", p)
	}

	b.Int = *z
	return nil
}

type TokenInfo struct {
	MimeType string `json:"mimeType"`
}

type Account struct {
	Alias   string `json:"alias"`
	Address string `json:"address"`
}

type Token struct {
	Contract    Account       `json:"contract"`
	ID          TokenID       `json:"tokenId"`
	Standard    string        `json:"standard"`
	TotalSupply int64         `json:"totalSupply,string"`
	Timestamp   time.Time     `json:"firstTime"`
	Metadata    TokenMetadata `json:"metadata"`
}

type TokenMetadata struct {
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Symbol       string       `json:"symbol"`
	MIMEType     string       `json:"type"`
	RightURI     string       `json:"rightUri"`
	ArtifactURI  string       `json:"artifactUri"`
	DisplayURI   string       `json:"displayUri"`
	ThumbnailURI string       `json:"thumbnailUri"`
	Creators     []string     `json:"creators"`
	Formats      []FileFormat `json:"formats"`
}

func (c *TZKT) GetContractToken(contract, tokenID string) (Token, error) {
	var t Token
	v := url.Values{
		"contract": []string{contract},
		"tokenId":  []string{tokenID},
	}

	u := url.URL{
		Scheme:   "https",
		Host:     c.endpoint,
		Path:     "/v1/tokens",
		RawQuery: v.Encode(),
	}
	resp, err := c.client.Get(u.String())
	if err != nil {
		return t, err
	}
	defer resp.Body.Close()

	var tokenResponse []Token
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return t, err
	}

	if len(tokenResponse) == 0 {
		return t, fmt.Errorf("token not found")
	}

	return tokenResponse[0], nil
}

func (c *TZKT) RetrieveTokens(owner string, offset int) ([]Token, error) {
	v := url.Values{
		"account":        []string{owner},
		"limit":          []string{"50"},
		"offset":         []string{fmt.Sprintf("%d", offset)},
		"balance.gt":     []string{"0"},
		"token.standard": []string{"fa2"},
	}

	u := url.URL{
		Scheme:   "https",
		Host:     c.endpoint,
		Path:     "/v1/tokens/balances",
		RawQuery: v.Encode(),
	}

	resp, err := c.client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResponse []struct {
		Token Token `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, err
	}

	tokens := make([]Token, 0, len(tokenResponse))
	for _, resp := range tokenResponse {
		tokens = append(tokens, resp.Token)
	}

	return tokens, nil
}

type TokenTransfer struct {
	Timestamp     time.Time `json:"timestamp"`
	TransactionID uint64    `json:"transactionId"`
	From          *Account  `json:"from"`
	To            Account   `json:"to"`
}

func (c *TZKT) GetTokenTransfers(contract, tokenID string) ([]TokenTransfer, error) {
	v := url.Values{
		"token.contract": []string{contract},
		"token.tokenId":  []string{tokenID},
		"select":         []string{"timestamp,from,to,transactionId"},
	}

	u := url.URL{
		Scheme:   "https",
		Host:     c.endpoint,
		Path:     "/v1/tokens/transfers",
		RawQuery: v.Encode(),
	}

	resp, err := c.client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var transfers []TokenTransfer

	if err := json.NewDecoder(resp.Body).Decode(&transfers); err != nil {
		return nil, err
	}

	return transfers, nil
}

type Transaction struct {
	ID   uint64 `json:"id"`
	Hash string `json:"hash"`
}

func (c *TZKT) GetTransaction(id uint64) (Transaction, error) {
	var t Transaction
	v := url.Values{
		"id": []string{fmt.Sprintf("%d", id)},
	}

	u := url.URL{
		Scheme:   "https",
		Host:     c.endpoint,
		Path:     "/v1/operations/transactions",
		RawQuery: v.Encode(),
	}

	resp, err := c.client.Get(u.String())
	if err != nil {
		return t, err
	}
	defer resp.Body.Close()

	var txs []Transaction

	if err := json.NewDecoder(resp.Body).Decode(&txs); err != nil {
		return t, err
	}

	if len(txs) == 0 {
		return t, fmt.Errorf("transaction not found")
	}
	return txs[0], nil
}
