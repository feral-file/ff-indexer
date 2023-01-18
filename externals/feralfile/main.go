package feralfile

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bitmark-inc/nft-indexer/traceutils"
	log "github.com/bitmark-inc/nft-indexer/zapLog"
	"go.uber.org/zap"
)

type Feralfile struct {
	apiURL string
	client *http.Client
}

func New(apiURL string) *Feralfile {
	return &Feralfile{
		apiURL: apiURL,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type Account struct {
	Contract string `json:"contract"`
	Network  string `json:"network"`
	ID       int    `json:"token_id"`
	Name     string `json:"name"`
	Alias    string `json:"alias"`
}

func (c *Feralfile) GetAccountInfo(owner string) (Account, error) {
	var a Account

	resp, err := c.client.Get(c.apiURL + "/api/accounts/" + owner)
	if err != nil {
		return a, err
	}
	defer resp.Body.Close()

	var respBody struct {
		Result Account `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		log.Logger.Error("fail to decode response",
			zap.String("apiSource", log.FeralFile),
			zap.String("response", traceutils.DumpResponse(resp)))
		return a, err
	}

	return respBody.Result, nil
}
