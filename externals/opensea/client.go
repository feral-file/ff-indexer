package opensea

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/bitmark-inc/autonomy-logger"
	"github.com/feral-file/ff-indexer/traceutils"
	"go.uber.org/zap"
)

var ErrTooManyRequest = fmt.Errorf("too many requests")

const QueryPageSize = "50"

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

type AssetV2 struct {
	Identifier    string `json:"identifier"`
	Collection    string `json:"collection"`
	Contract      string `json:"contract"`
	TokenStandard string `json:"token_standard"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	ImageURL      string `json:"image_url"`
	MetadataURL   string `json:"metadata_url"`
	OpenseaURL    string `json:"opensea_url"`
	CreatedAt     Time   `json:"created_at"`
	UpdatedAt     Time   `json:"updated_at"`
	IsDisabled    bool   `json:"is_disabled"`
	IsNsfw        bool   `json:"is_nsfw"`
}

type AssetsResponse struct {
	NFTs []AssetV2 `json:"nfts"`
	Next string    `json:"next"`
}

type DetailedAssetV2 struct {
	Identifier      string  `json:"identifier"`
	Collection      string  `json:"collection"`
	Contract        string  `json:"contract"`
	TokenStandard   string  `json:"token_standard"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	ImageURL        string  `json:"image_url"`
	MetadataURL     string  `json:"metadata_url"`
	DisplayImageURL string  `json:"display_image_url"`
	AnimationURL    string  `json:"display_animation_url"`
	OpenseaURL      string  `json:"opensea_url"`
	CreatedAt       Time    `json:"created_at"`
	UpdatedAt       Time    `json:"updated_at"`
	Owners          []Owner `json:"owners"`
	Creator         string  `json:"creator"`
	IsDisabled      bool    `json:"is_disabled"`
	IsNsfw          bool    `json:"is_nsfw"`
}

type Owner struct {
	Address  string `json:"address"`
	Quantity int64  `json:"quantity"`
}

type Account struct {
	Address  string `json:"address"`
	Username string `json:"username"`
}

type CollectionsResponse struct {
	Collections []Collection `json:"collections"`
	Next        string       `json:"next"`
}

type Collection struct {
	ID             string `json:"collection"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	ImageURL       string `json:"image_url"`
	BannerImageURL string `json:"banner_image_url"`
	Owner          string `json:"owner"`
	IsDisabled     bool   `json:"is_disabled"`
	OpenseaURL     string `json:"opensea_url"`
	ProjectURL     string `json:"project_url"`
	Contracts      []struct {
		Address string `json:"address"`
		Chain   string `json:"chain"`
	} `json:"contracts"`
	TotalSupply int    `json:"total_supply"`
	CreatedDate string `json:"created_date"`
}

type Client struct {
	debug       bool
	apiKey      string
	apiEndpoint string
	chain       string //chain = ethereum, goerli, sepolia
	client      *http.Client

	limiter RateLimiter
}

func New(chain, apiKey string, rps int) *Client {
	if chain == "" {
		chain = "ethereum"
	}
	apiEndpoint := "api.opensea.io"
	if chain != "ethereum" {
		apiEndpoint = "testnets-api.opensea.io"
	}

	return &Client{
		apiKey:      apiKey,
		apiEndpoint: apiEndpoint,
		chain:       chain,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},

		limiter: *NewRateLimiter(rps),
	}
}

type RateLimiter struct {
	rps     int // Request per second
	reqChan chan struct{}
}

func (r *RateLimiter) Start() {
	if r.rps > 0 {
		go func() {
			for range time.Tick(time.Second) {
				for i := 0; i < r.rps; i++ {
					if len(r.reqChan) < r.rps {
						r.reqChan <- struct{}{}
					}
				}
			}
		}()
	}
}

func (r *RateLimiter) Request() struct{} {
	if r.rps > 0 {
		return <-r.reqChan
	}

	return struct{}{}
}

func NewRateLimiter(rps int) *RateLimiter {
	r := &RateLimiter{
		rps:     rps,
		reqChan: make(chan struct{}, rps),
	}

	r.Start()
	return r
}

func (c *Client) Debug(debug bool) {
	c.debug = debug
}

func (c *Client) makeRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-API-KEY", c.apiKey)

	if c.debug {
		log.Debug("debug request", zap.String("req_dump", traceutils.DumpRequest(req)))
	}

	c.limiter.Request()
	log.Debug("get a request from limiter")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		// close the body only when we return an error
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, ErrTooManyRequest
		}

		errResp, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf(
			"opensea api error: status: %d (%s)  body: '%s'",
			resp.StatusCode,
			http.StatusText(resp.StatusCode),
			errResp,
		)
	}

	return resp, nil
}

// RetrieveAsset returns the token information for a contract and a token id
func (c *Client) RetrieveAsset(ctx context.Context, contract, tokenID string) (*DetailedAssetV2, error) {
	u := url.URL{
		Scheme: "https",
		Host:   c.apiEndpoint,
		Path:   fmt.Sprintf("/api/v2/chain/%s/contract/%s/nfts/%s", c.chain, contract, tokenID),
	}

	resp, err := c.makeRequest(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var assetResp struct {
		Asset DetailedAssetV2 `json:"nft"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&assetResp); err != nil {
		return nil, err
	}

	return &assetResp.Asset, nil
}

func (c *Client) RetrieveAssets(ctx context.Context, owner string, next string) (*AssetsResponse, error) {
	// NOTE: query by offset is removed from the document but still support at this moment.
	v := url.Values{
		"limit": []string{QueryPageSize},
		"next":  []string{next},
	}

	u := url.URL{
		Scheme:   "https",
		Host:     c.apiEndpoint,
		Path:     fmt.Sprintf("/api/v2/chain/%s/account/%s/nfts", c.chain, owner),
		RawQuery: v.Encode(),
	}

	resp, err := c.makeRequest(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var assetResp AssetsResponse

	if err := json.NewDecoder(resp.Body).Decode(&assetResp); err != nil {
		return nil, err
	}

	return &assetResp, nil
}
