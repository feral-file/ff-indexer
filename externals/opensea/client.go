package opensea

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type OpenSeaTime struct {
	time.Time
}

func (t *OpenSeaTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
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

type AssetCreator struct {
	Address string `json:"address"`
	User    struct {
		Username string `json:"username"`
	} `json:"user"`
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

	Creator       AssetCreator  `json:"creator"`
	AssetContract AssetContract `json:"asset_contract"`
}

type OpenseaClient struct {
	apiKey  string
	network string
	client  *http.Client
}

func New(network, apiKey string) *OpenseaClient {
	return &OpenseaClient{
		apiKey:  apiKey,
		network: network,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
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
		Host:     "api.opensea.io",
		Path:     "/api/v1/assets",
		RawQuery: v.Encode(),
	}

	if c.network == "testnet" {
		u.Host = "rinkeby-api.opensea.io"
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	if c.apiKey != "" {
		req.Header.Add("X-API-KEY", c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var assetResp struct {
		Assets []Asset `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&assetResp); err != nil {
		return nil, err
	}

	return assetResp.Assets, nil
}
