package tzkt

import (
	"bytes"
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

func New(network string) *TZKT {
	endpoint := "api.mainnet.tzkt.io"
	if network == "testnet" {
		endpoint = "api.ghostnet.tzkt.io"
	}

	return &TZKT{
		client: &http.Client{
			Timeout: time.Minute,
		},
		endpoint: endpoint,
	}
}

type FormatDimensions struct {
	Unit  string `json:"unit"`
	Value string `json:"value"`
}

type FileFormat struct {
	URI        string           `json:"uri"`
	FileName   string           `json:"fileName,omitempty"`
	FileSize   int              `json:"fileSize,string"`
	MIMEType   string           `json:"mimeType"`
	Dimensions FormatDimensions `json:"dimensions,omitempty"`
}

type FileFormats []FileFormat

func (f *FileFormats) UnmarshalJSON(data []byte) error {
	type formats FileFormats

	switch data[0] {
	case 34:
		d_ := bytes.ReplaceAll(bytes.Trim(data, `"`), []byte{92, 117, 48, 48, 50, 50}, []byte{34})
		d := bytes.ReplaceAll(d_, []byte{92, 34}, []byte{34})

		if err := json.Unmarshal(d, (*formats)(f)); err != nil {
			return err
		}
	case 123: // If the "formats" is not an array
		d := append([]byte{91}, data...)
		if data[len(data)-1] != 93 {
			d = append(d, []byte{93}...)
		}
		if err := json.Unmarshal(d, (*formats)(f)); err != nil {
			return err
		}
	default:
		if err := json.Unmarshal(data, (*formats)(f)); err != nil {
			return err
		}
	}

	return nil
}

type FileCreators []string

func (c *FileCreators) UnmarshalJSON(data []byte) error {
	type creators FileCreators

	switch data[0] {
	case 34:
		d_ := bytes.ReplaceAll(bytes.Trim(data, `"`), []byte{92, 117, 48, 48, 50, 50}, []byte{34})
		d := bytes.ReplaceAll(d_, []byte{92, 34}, []byte{34})

		if err := json.Unmarshal(d, (*creators)(c)); err != nil {
			return err
		}
	default:
		if err := json.Unmarshal(data, (*creators)(c)); err != nil {
			return err
		}
	}

	return nil
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

type OwnedToken struct {
	Token   Token `json:"token"`
	Balance int64 `json:"balance,string"`
}

type TokenMetadata struct {
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Symbol       string       `json:"symbol"`
	RightURI     string       `json:"rightUri"`
	ArtifactURI  string       `json:"artifactUri"`
	DisplayURI   string       `json:"displayUri"`
	ThumbnailURI string       `json:"thumbnailUri"`
	Creators     FileCreators `json:"creators"`
	Formats      FileFormats  `json:"formats"`
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

// RetrieveTokens returns OwnedToken for a specific token. The OwnedToken object includes
// both balance and token information
func (c *TZKT) RetrieveTokens(owner string, offset int) ([]OwnedToken, error) {
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

	var ownedTokens []OwnedToken
	if err := json.NewDecoder(resp.Body).Decode(&ownedTokens); err != nil {
		return nil, err
	}

	return ownedTokens, nil
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
		"token.standard": []string{"fa2"},
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

// GetTokenLastActivityTime returns the timestamp of the last activity for a token
func (c *TZKT) GetTokenLastActivityTime(contract, tokenID string) (time.Time, error) {
	v := url.Values{
		"token.contract": []string{contract},
		"token.tokenId":  []string{tokenID},
		"token.standard": []string{"fa2"},
		"sort.desc":      []string{"timestamp"},
		"limit":          []string{"1"},
		"select":         []string{"timestamp"},
	}

	u := url.URL{
		Scheme:   "https",
		Host:     c.endpoint,
		Path:     "/v1/tokens/transfers",
		RawQuery: v.Encode(),
	}

	resp, err := c.client.Get(u.String())
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	var activityTime []time.Time

	if err := json.NewDecoder(resp.Body).Decode(&activityTime); err != nil {
		return time.Time{}, err
	}

	if len(activityTime) == 0 {
		return time.Time{}, fmt.Errorf("no activities for this token")
	}

	return activityTime[0], nil
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

type TokenOwner struct {
	Address string `json:"address"`
	Balance int64  `json:"balance,string"`
}

// GetTokenOwners returns a list of TokenOwner for a specific token
func (c *TZKT) GetTokenOwners(contract, tokenID string) ([]TokenOwner, error) {
	v := url.Values{
		"token.contract": []string{contract},
		"token.tokenId":  []string{tokenID},
		"balance.gt":     []string{"0"},
		"token.standard": []string{"fa2"},
		"select":         []string{"account.address as address,balance"},
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

	var owners []TokenOwner

	if err := json.NewDecoder(resp.Body).Decode(&owners); err != nil {
		return nil, err
	}

	return owners, nil
}

// GetTokenBalanceForOwner returns a list of TokenOwner for a specific token
func (c *TZKT) GetTokenBalanceForOwner(contract, tokenID, owner string) (int64, error) {
	v := url.Values{
		"token.contract": []string{contract},
		"token.tokenId":  []string{tokenID},
		"balance.gt":     []string{"0"},
		"account":        []string{owner},
		"token.standard": []string{"fa2"},
		"select":         []string{"account.address as address,balance"},
	}

	u := url.URL{
		Scheme:   "https",
		Host:     c.endpoint,
		Path:     "/v1/tokens/balances",
		RawQuery: v.Encode(),
	}

	resp, err := c.client.Get(u.String())
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var owners []TokenOwner

	if err := json.NewDecoder(resp.Body).Decode(&owners); err != nil {
		return 0, err
	}

	if len(owners) == 0 {
		return 0, fmt.Errorf("token not found")
	}

	if len(owners) > 1 {
		return 0, fmt.Errorf("multiple token owners returned")
	}

	return owners[0].Balance, nil
}
