package opensea

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bitmark-inc/nft-indexer/traceutils"
	"github.com/sirupsen/logrus"
)

var ErrTooManyRequest = fmt.Errorf("too many requests")

type OpenSeaTime struct {
	time.Time
}

func (t *OpenSeaTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	s = strings.Split(s, "+")[0]
	tt, err := time.Parse("2006-01-02T15:04:05.999999", s)
	if err != nil {
		return err
	}

	t.Time = tt
	return nil
}

func (t *OpenSeaTime) MarshalJSON() ([]byte, error) {
	return t.Time.MarshalJSON()
}

type AssetContract struct {
	Address     string      `json:"address"`
	SchemaName  string      `json:"schema_name"`
	CreatedDate OpenSeaTime `json:"created_date"`
}

type User struct {
	Address string `json:"address"`
	User    struct {
		Username string `json:"username"`
	} `json:"user"`
}

type Ownership struct {
	Owner    User  `json:"owner"`
	Quantity int64 `json:"quantity,string"`
}

type Asset struct {
	ID                 int64  `json:"id"`
	TokenID            string `json:"token_id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	ExternalLink       string `json:"external_link"`
	ImageURL           string `json:"image_url"`
	ImagePreviewURL    string `json:"image_preview_url"`
	ImageThumbnailURL  string `json:"image_thumbnail_url"`
	ImageOriginURL     string `json:"image_original_url"`
	AnimationURL       string `json:"animation_url"`
	AnimationOriginURL string `json:"animation_original_url"`
	Permalink          string `json:"permalink"`
	TokenMetadata      string `json:"token_metadata"`

	Owner         User          `json:"owner"`
	Creator       User          `json:"creator"`
	AssetContract AssetContract `json:"asset_contract"`
	Ownership     *Ownership    `json:"ownership"`
}

type OpenseaClient struct {
	apiKey      string
	apiEndpoint string
	network     string
	client      *http.Client

	limiter RateLimiter
}

func New(network, apiKey string, rps int) *OpenseaClient {
	apiEndpoint := "api.opensea.io"
	if network == "testnet" {
		apiEndpoint = "testnets-api.opensea.io"
	}

	return &OpenseaClient{
		apiKey:      apiKey,
		apiEndpoint: apiEndpoint,
		network:     network,
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
						logrus.Trace("increase the request count")
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

func (c *OpenseaClient) makeRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if c.network != "testnet" {
		req.Header.Add("X-API-KEY", c.apiKey)
	}

	c.limiter.Request()
	logrus.Trace("get a request from limiter")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		// close the body only when we return an error
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, ErrTooManyRequest
		}

		errResp, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("opensea api error: %s", errResp)
	}

	return resp, nil
}

// RetrieveAsset returns the token information for a contract and a token id
func (c *OpenseaClient) RetrieveAsset(contract, tokenID string) (*Asset, error) {
	v := url.Values{
		"asset_contract_addresses": []string{contract},
		"token_ids":                []string{tokenID},
	}

	u := url.URL{
		Scheme:   "https",
		Host:     c.apiEndpoint,
		Path:     "/api/v1/assets",
		RawQuery: v.Encode(),
	}

	resp, err := c.makeRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var assetResp struct {
		Assets []Asset `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&assetResp); err != nil {
		logrus.WithError(err).WithField("resp_dump", traceutils.DumpResponse(resp)).Error("fail to read opensea response")
		return nil, err
	}

	if len(assetResp.Assets) > 0 {
		return &assetResp.Assets[0], nil
	}

	return nil, fmt.Errorf("asset not found")
}

func (c *OpenseaClient) RetrieveAssets(owner string, offset int) ([]Asset, error) {
	v := url.Values{
		"owner":           []string{owner},
		"limit":           []string{"50"},
		"order_direction": []string{"desc"},
		"offset":          []string{fmt.Sprintf("%d", offset)},
	}

	u := url.URL{
		Scheme:   "https",
		Host:     c.apiEndpoint,
		Path:     "/api/v1/assets",
		RawQuery: v.Encode(),
	}

	resp, err := c.makeRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var assetResp struct {
		Assets []Asset `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&assetResp); err != nil {
		logrus.WithError(err).WithField("resp_dump", traceutils.DumpResponse(resp)).Error("fail to read opensea response")
		return nil, err
	}

	return assetResp.Assets, nil
}

type TokenOwner struct {
	Owner       User        `json:"owner"`
	Quantity    int64       `json:"quantity,string"`
	CreatedDate OpenSeaTime `json:"created_date"`
}

type AssetOwners struct {
	Next   *string      `json:"next"`
	Owners []TokenOwner `json:"owners"`
}

func (c *OpenseaClient) GetTokenBalanceForOwner(contract, tokenID, owner string) (int64, error) {
	v := url.Values{
		"account_address": []string{owner},
	}

	u := url.URL{
		Scheme:   "https",
		Host:     c.apiEndpoint,
		Path:     fmt.Sprintf("/api/v1/asset/%s/%s", contract, tokenID),
		RawQuery: v.Encode(),
	}

	resp, err := c.makeRequest("GET", u.String(), nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var asset Asset
	if err := json.NewDecoder(resp.Body).Decode(&asset); err != nil {
		return 0, err
	}

	ownership := asset.Ownership
	if ownership == nil {
		return 0, fmt.Errorf("not the owner of this token")
	}

	if ownership.Quantity == 0 {
		return 0, fmt.Errorf("not the owner of this token")
	}

	return ownership.Quantity, nil
}

func (c *OpenseaClient) RetrieveTokenOwners(contract, tokenID string, cursor *string) ([]TokenOwner, *string, error) {
	v := url.Values{
		"limit":           []string{"50"},
		"order_direction": []string{"desc"},
	}

	if cursor != nil {
		v["cursor"] = []string{*cursor}
	}

	u := url.URL{
		Scheme:   "https",
		Host:     c.apiEndpoint,
		Path:     fmt.Sprintf("/api/v1/asset/%s/%s/owners", contract, tokenID),
		RawQuery: v.Encode(),
	}

	resp, err := c.makeRequest("GET", u.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	var ownersResp AssetOwners
	if err := json.NewDecoder(resp.Body).Decode(&ownersResp); err != nil {
		return nil, nil, err
	}

	return ownersResp.Owners, ownersResp.Next, nil
}

// GetTokenLastActivityTime returns the timestamp of the last activity for a token
func (c *OpenseaClient) GetTokenLastActivityTime(contract, tokenID string) (time.Time, error) {
	v := url.Values{
		"limit":           []string{"1"},
		"order_by":        []string{"created_date"},
		"order_direction": []string{"desc"},
	}

	u := url.URL{
		Scheme:   "https",
		Host:     c.apiEndpoint,
		Path:     fmt.Sprintf("/api/v1/asset/%s/%s/owners", contract, tokenID),
		RawQuery: v.Encode(),
	}

	resp, err := c.makeRequest("GET", u.String(), nil)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	var ownersResp AssetOwners
	if err := json.NewDecoder(resp.Body).Decode(&ownersResp); err != nil {
		return time.Time{}, err
	}

	if len(ownersResp.Owners) == 0 {
		return time.Time{}, fmt.Errorf("no activities for this token")
	}

	return ownersResp.Owners[0].CreatedDate.Time, nil
}
