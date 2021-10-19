package artblocks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ArtblocksClient struct {
	client *http.Client
}

func New() *ArtblocksClient {
	return &ArtblocksClient{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *ArtblocksClient) GetTokenData(tokenID string) (map[string]interface{}, error) {
	resp, err := c.client.Get(fmt.Sprintf("https://token.artblocks.io/%s", tokenID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenData); err != nil {
		return nil, err
	}

	return tokenData, nil
}
