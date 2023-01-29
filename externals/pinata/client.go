package pinata

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/bitmark-inc/nft-indexer/traceutils"
)

type Client struct {
	endpoint  string
	authToken string
	client    *http.Client
}

func New(endpoint, authToken string, timeout time.Duration) *Client {
	return &Client{
		endpoint:  endpoint,
		authToken: authToken,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *Client) makeRequest(method, url string, body io.Reader) *http.Request {
	req, _ := http.NewRequest(method, url, body)

	req.Header.Add("Content-Type", "application/json")

	if p.authToken != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.authToken))
	}

	return req
}

type PinnedFile struct {
	ID       string   `json:"id"`
	CID      string   `json:"ipfs_pin_hash"`
	Size     int64    `json:"size"`
	Metadata Metadata `json:"metadata"`
}

type Metadata struct {
	Name string            `json:"name"`
	KV   map[string]string `json:"keyvalues"`
}

type PinHashRequest struct {
	CID      string    `json:"hashToPin"`
	Metadata *Metadata `json:"pinataMetadata,omitempty"`
}

// PinnedFile returns the pinned file for a given CID
func (p *Client) PinnedFile(cid string) (*PinnedFile, error) {
	u := url.URL{
		Scheme:   "https",
		Host:     p.endpoint,
		Path:     "/data/pinList",
		RawQuery: fmt.Sprintf("hashContains=%s", cid),
	}

	req := p.makeRequest("GET", u.String(), nil)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println(traceutils.DumpRequest(req))
		fmt.Println(traceutils.DumpResponse(resp))
		return nil, errors.New("http status not 200")
	}

	var result struct {
		Count int64        `json:"count"`
		Rows  []PinnedFile `json:"rows"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Count == 0 {
		return nil, nil
	}

	return &result.Rows[0], nil
}

// PinJobs gets all pinning jobs
func (p *Client) PinJobs(cid string) (*PinByHashResp, error) {
	u := url.URL{
		Scheme:   "https",
		Host:     p.endpoint,
		Path:     "/pinning/pinJobs",
		RawQuery: fmt.Sprintf("ipfs_pin_hash=%s", cid),
	}

	req := p.makeRequest("GET", u.String(), nil)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println(traceutils.DumpRequest(req))
		fmt.Println(traceutils.DumpResponse(resp))
		return nil, errors.New("http status not 200")
	}

	var result struct {
		Count int64           `json:"count"`
		Rows  []PinByHashResp `json:"rows"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Count == 0 {
		return nil, nil
	}

	return &result.Rows[0], nil
}

type PinByHashResp struct {
	ID     string `json:"id"`
	CID    string `json:"ipfsHash"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// PinByHash makes a pin request to pinata
func (p *Client) PinByHash(cid string, metadata *Metadata) (*PinByHashResp, error) {
	u := url.URL{
		Scheme: "https",
		Host:   p.endpoint,
		Path:   "/pinning/pinByHash",
	}

	reqBody := PinHashRequest{
		CID:      cid,
		Metadata: metadata,
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(reqBody); err != nil {
		return nil, err
	}

	req := p.makeRequest("POST", u.String(), &body)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println(traceutils.DumpRequest(req))
		fmt.Println(traceutils.DumpResponse(resp))
		return nil, errors.New("http status not 200")
	}

	var result PinByHashResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
