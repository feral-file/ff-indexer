package tzkt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/bitmark-inc/nft-indexer/traceutils"
)

var ErrTooManyRequest = fmt.Errorf("too many requests")

type TZKT struct {
	debug    bool
	endpoint string

	client *http.Client
}

type NullableInt int64

func New(network string) *TZKT {
	endpoint := "api.mainnet.tzkt.io"
	if network == "testnet" {
		endpoint = "api.kathmandunet.tzkt.io"
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
	MIMEType   MIMEFormat       `json:"mimeType"`
	Dimensions FormatDimensions `json:"dimensions,omitempty"`
}

type MIMEFormat string

func (m *MIMEFormat) UnmarshalJSON(data []byte) error {
	type mime MIMEFormat

	if data[0] == 91 {
		data = bytes.Trim(data, "[]")
	}

	if err := json.Unmarshal(data, (*mime)(m)); err != nil {
		return err
	}
	return nil
}

type FileFormats []FileFormat

func (t *NullableInt) UnmarshalJSON(data []byte) error {
	var num int64

	err := json.Unmarshal(data, &num)
	if err != nil {
		*t = NullableInt(-1)

		return nil
	}

	*t = NullableInt(num)

	return nil
}

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
	TotalSupply NullableInt   `json:"totalSupply,string"`
	Timestamp   time.Time     `json:"firstTime"`
	Metadata    TokenMetadata `json:"metadata"`
}

type OwnedToken struct {
	Token    Token       `json:"token"`
	Balance  NullableInt `json:"balance,string"`
	LastTime time.Time   `json:"lastTime"`
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

func (c *TZKT) Debug() *TZKT {
	c.debug = true
	return c
}

func (c *TZKT) request(req *http.Request, responseData interface{}) error {
	if c.debug {
		logrus.WithField("req", traceutils.DumpRequest(req)).Debug("tzkt request")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if c.debug {
		logrus.WithField("resp", traceutils.DumpResponse(resp)).Debug("tzkt response")
	}

	if resp.StatusCode != 200 {
		// close the body only when we return an error
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			return ErrTooManyRequest
		}

		errResp, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("tzkt api error: %s", errResp)
	}

	err = json.NewDecoder(resp.Body).Decode(&responseData)
	if err != nil {
		logrus.
			WithField("req", traceutils.DumpRequest(req)).
			WithField("resp", traceutils.DumpResponse(resp)).
			Error("tzkt error response")
	}

	return err
}

func (c *TZKT) GetContractToken(contract, tokenID string) (Token, error) {
	var tokenResponse []Token

	u := url.URL{
		Scheme: "https",
		Host:   c.endpoint,
		Path:   "/v1/tokens",
		RawQuery: url.Values{
			"contract": []string{contract},
			"tokenId":  []string{tokenID},
		}.Encode(),
	}

	req, _ := http.NewRequest("GET", u.String(), nil)
	if err := c.request(req, &tokenResponse); err != nil {
		return Token{}, err
	}

	if len(tokenResponse) == 0 {
		return Token{}, fmt.Errorf("token not found")
	}

	return tokenResponse[0], nil
}

// RetrieveTokens returns OwnedToken for a specific token. The OwnedToken object includes
// both balance and token information
func (c *TZKT) RetrieveTokens(owner string, lastTime time.Time, offset int) ([]OwnedToken, error) {
	v := url.Values{
		"account":        []string{owner},
		"limit":          []string{"50"},
		"offset":         []string{fmt.Sprintf("%d", offset)},
		"balance.ge":     []string{"0"},
		"token.standard": []string{"fa2"},
		"sort":           []string{"lastTime"},
	}

	// prevent QueryEscape for colons in time
	rawQuery := v.Encode() // + "&lastTime.gt=" + lastTime.UTC().Format(time.RFC3339)

	u := url.URL{
		Scheme:   "https",
		Host:     c.endpoint,
		Path:     "/v1/tokens/balances",
		RawQuery: rawQuery,
	}
	var ownedTokens []OwnedToken

	req, _ := http.NewRequest("GET", u.String(), nil)
	if err := c.request(req, &ownedTokens); err != nil {
		return ownedTokens, err
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

	var transfers []TokenTransfer

	req, _ := http.NewRequest("GET", u.String(), nil)
	if err := c.request(req, &transfers); err != nil {
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

	var activityTime []time.Time

	req, _ := http.NewRequest("GET", u.String(), nil)
	if err := c.request(req, &activityTime); err != nil {
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

	var txs []Transaction

	req, _ := http.NewRequest("GET", u.String(), nil)
	if err := c.request(req, &txs); err != nil {
		return t, err
	}

	if len(txs) == 0 {
		return t, fmt.Errorf("transaction not found")
	}
	return txs[0], nil
}

type TokenOwner struct {
	Address  string    `json:"address"`
	Balance  int64     `json:"balance,string"`
	LastTime time.Time `json:"lastTime"`
}

// GetTokenOwners returns a list of TokenOwner for a specific token
func (c *TZKT) GetTokenOwners(contract, tokenID string, limit int, lastTime time.Time) ([]TokenOwner, error) {
	v := url.Values{
		"token.contract": []string{contract},
		"token.tokenId":  []string{tokenID},
		"balance.gt":     []string{"0"},
		"token.standard": []string{"fa2"},
		"sort.asc":       []string{"lastTime"},
		"limit":          []string{fmt.Sprintf("%d", limit)},
		"select":         []string{"account.address as address,balance,lastTime"},
	}

	rawQuery := v.Encode() + "&lastTime.ge=" + lastTime.UTC().Format(time.RFC3339)

	u := url.URL{
		Scheme:   "https",
		Host:     c.endpoint,
		Path:     "/v1/tokens/balances",
		RawQuery: rawQuery,
	}

	var owners []TokenOwner

	req, _ := http.NewRequest("GET", u.String(), nil)
	if err := c.request(req, &owners); err != nil {
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

	var owners []TokenOwner

	req, _ := http.NewRequest("GET", u.String(), nil)
	if err := c.request(req, &owners); err != nil {
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

type TransactionDetails struct {
	Block     string               `json:"block"`
	Parameter TransactionParameter `json:"parameter"`
	Target    Account              `json:"target"`
	Timestamp time.Time            `json:"timestamp" bson:"timestamp"`
}

type TransactionParameter struct {
	EntryPoint string            `json:"entrypoint"`
	Value      []ParametersValue `json:"value"`
}

type ParametersValue struct {
	From_ string      `json:"from_"`
	Txs   []TxsFormat `json:"txs`
}

type TxsFormat struct {
	To_     string `json:"to_"`
	Amount  string `json:"amount"`
	TokenID string `json:"token_id"`
}

// GetTransactionByTx gets transaction details from a specific Tx
func (c *TZKT) GetTransactionByTx(hash string) ([]TransactionDetails, error) {
	u := url.URL{
		Scheme: "https",
		Host:   c.endpoint,
		Path:   fmt.Sprintf("%s/%s", "/v1/operations/transactions", hash),
	}

	var transactionDetails []TransactionDetails

	resp, err := c.client.Get(u.String())
	if err != nil {
		return transactionDetails, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&transactionDetails); err != nil {
		return transactionDetails, err
	}

	return transactionDetails, nil
}
