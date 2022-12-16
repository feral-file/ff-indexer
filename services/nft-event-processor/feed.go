package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bitmark-inc/nft-indexer/traceutils"
	"github.com/sirupsen/logrus"
)

type FeedClient struct {
	client   *http.Client
	endpoint string
	apiToken string
	isDebug  bool
}

func NewFeedClient(endpoint, apiToken string, isDebug bool) *FeedClient {
	return &FeedClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		endpoint: strings.TrimSuffix(endpoint, "/"),
		apiToken: apiToken,
		isDebug:  isDebug,
	}
}

type EventRequest struct {
	Blockchain string    `json:"chain"`
	Contract   string    `json:"contract"`
	TokenID    string    `json:"token"`
	Recipient  string    `json:"recipient"`
	Action     string    `json:"action"`
	IsTest     bool      `json:"testnet"`
	Timestamp  time.Time `json:"timestamp"`
}

func (f *FeedClient) SendEvent(blockchain, contract, tokenID, owner, action string, isTestnet bool) error {
	body := bytes.Buffer{}

	if err := json.NewEncoder(&body).Encode(EventRequest{
		Blockchain: blockchain,
		Contract:   contract,
		TokenID:    tokenID,
		Recipient:  owner,
		Action:     action,
		IsTest:     isTestnet,
		Timestamp:  time.Now(),
	}); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/hook/event", f.endpoint), &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", f.apiToken))

	resp, err := f.client.Do(req)
	if err != nil {
		if f.isDebug {
			logrus.
				WithError(err).
				WithField("req_dump", traceutils.DumpRequest(req)).
				Debug("fail to submit event to feed server")
		}
		return err
	}

	if resp.StatusCode != 200 {
		if f.isDebug {
			logrus.
				WithField("req_dump", traceutils.DumpRequest(req)).
				WithField("resp_dump", traceutils.DumpResponse(resp)).
				Debug("fail to submit event to feed server")
		}
		return fmt.Errorf("fail to submit event to feed server")
	}

	return nil
}
