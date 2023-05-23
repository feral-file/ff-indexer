package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	client      *http.Client
	apiEndpoint string
}

// New create an indexer client connection
func New(apiEndpoint string, client *http.Client) *Client {
	if client == nil {
		client = &http.Client{
			Timeout: 15 * time.Second,
		}
	}

	return &Client{
		client:      client,
		apiEndpoint: apiEndpoint,
	}
}

// IndexOne index a single NFT
func (c *Client) IndexOne(contract string, tokenID string, dryRun bool, preview bool) error {
	var body bytes.Buffer

	if err := json.NewEncoder(&body).Encode(map[string]interface{}{
		"contract": contract,
		"tokenID":  tokenID,
		"dryrun":   dryRun,
		"preview":  preview,
	}); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v2/nft/index_one", c.apiEndpoint), &body)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("index_one failed with status code %d", resp.StatusCode)
	}

	return nil
}
