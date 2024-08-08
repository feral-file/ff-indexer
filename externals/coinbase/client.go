package coinbase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	baseURL                   = "https://api.exchange.coinbase.com"
	requestTimeout            = 10 * time.Second
	httpStatusOK              = 200
	httpStatusMultipleChoices = 300
	contentTypeHeader         = "Content-Type"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient() *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: requestTimeout},
	}
}

func (c *Client) MakeRequest(
	method, contentType, endpoint string,
	query url.Values, body any) (any, error) {
	// Build the URL
	reqURL := fmt.Sprintf("%s%s?%s", c.BaseURL, endpoint, query.Encode())

	// Marshal the body if it exists
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	// Create the request
	req, err := http.NewRequest(method, reqURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	// Make the request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read and unmarshal the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode < httpStatusOK || resp.StatusCode >= httpStatusMultipleChoices {
		return nil, fmt.Errorf("request failed with status %s: %s", resp.Status, respBody)
	}

	// Check Content-Type header
	responseContentType := resp.Header.Get(contentTypeHeader)
	if !strings.HasPrefix(responseContentType, "application/json") {
		return nil, fmt.Errorf("unexpected content type %s from endpoint %s", resp.Header.Get(contentTypeHeader), endpoint)
	}

	var result any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return result, nil
}
