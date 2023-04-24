package infura

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/bitmark-inc/nft-indexer/traceutils"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

var ErrTooManyRequest = fmt.Errorf("too many requests")

type Client struct {
	apiKey       string
	apiKeySecret string
	chainID      string
	client       *http.Client
}

func New(network, apiKey, apiKeySecret string) *Client {
	chainID := "1" // mainnet
	if network == "testnet" {
		chainID = "5" // sepolia: 11155111, goerli: 5
	}

	return &Client{
		apiKey:       apiKey,
		apiKeySecret: apiKeySecret,
		chainID:      chainID,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) makeRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth("e9ea28c448404bc1a38f1e543a7150ed", "749e1b0f2b644de38e1ffaf96576730e")

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

		errResp, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("opensea api error: %s", errResp)
	}

	return resp, nil
}

type OwnerBalance struct {
	Amount string `json:"amount"`
	Owner  string `json:"ownerOf"`
}

// RetrieveAsset returns the token information for a contract and a token id
func (c *Client) GetOwnersAndBalancesByToken(contract, tokenID string) (map[string]int64, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "nft.api.infura.io",
		Path:   fmt.Sprintf("/networks/%s/nfts/%s/%s/owners", c.chainID, contract, tokenID),
	}

	resp, err := c.makeRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var balanceResp struct {
		OwnerBalances []OwnerBalance `json:"owners"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&balanceResp); err != nil {
		log.Error("fail to read opensea response", zap.Error(err),
			log.SourceOpensea,
			zap.String("resp_dump", traceutils.DumpResponse(resp)))
		return nil, err
	}

	ownersMap := map[string]int64{}
	if len(balanceResp.OwnerBalances) > 0 {
		for _, ownerBalance := range balanceResp.OwnerBalances {
			balance, err := strconv.ParseInt(ownerBalance.Amount, 10, 64)
			if err != nil {
				continue
			}

			ownersMap[common.HexToAddress(ownerBalance.Owner).Hex()] = balance
		}

		return ownersMap, nil
	}

	return nil, fmt.Errorf("token not found")
}
