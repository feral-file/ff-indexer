package indexer

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strings"
	"time"

	utils "github.com/bitmark-inc/autonomy-utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/structs"
)

func EthereumChecksumAddress(address string) string {
	return common.HexToAddress(address).Hex()
}

func OpenseaTokenIDToHex(tokenID string) (string, error) {
	tokenIDBig, ok := big.NewInt(0).SetString(tokenID, 10)
	if !ok {
		return "", fmt.Errorf("fail to parse token id")
	}

	return tokenIDBig.Text(16), nil
}

// TokenIndexID returns blockchain-based unique token id. It is constructed by
// blockchain alias, token contract and token id.
func TokenIndexID(blockchainType, contractAddress, id string) string {
	blockchainAlias, ok := BlockchainAlias[blockchainType]
	if !ok {
		blockchainAlias = "undefined"
	}

	if blockchainType == utils.EthereumBlockchain {
		contractAddress = EthereumChecksumAddress(contractAddress)
	}

	return fmt.Sprintf("%s-%s-%s", blockchainAlias, contractAddress, id)
}

// ParseTokenIndexID return blockchainType, contractAddress and token id for
// a given indexID
func ParseTokenIndexID(indexID string) (string, string, string, error) {
	v := strings.Split(indexID, "-")
	if len(v) != 3 {
		return "", "", "", fmt.Errorf("error while parsing indexID: %v", indexID)
	}

	if v[0] == BlockchainAlias[utils.EthereumBlockchain] {
		v[1] = EthereumChecksumAddress(v[1])
	}

	return v[0], v[1], v[2], nil
}

// TxURL returns corresponded blockchain transaction URL
func TxURL(blockchain, environment, txID string) string {
	switch blockchain {
	case utils.BitmarkBlockchain:
		if environment == DevelopmentEnvironment {
			return fmt.Sprintf("https://registry.test.bitmark.com/transaction/%s", txID)
		}
		return fmt.Sprintf("https://registry.bitmark.com/transaction/%s", txID)
	case utils.EthereumBlockchain:
		if environment == DevelopmentEnvironment {
			return fmt.Sprintf("https://goerli.etherscan.io/tx/%s", txID)
		}
		return fmt.Sprintf("https://etherscan.io/tx/%s", txID)
	default:
		return ""
	}
}

// SleepWithContext will return whenever the slept time reached or context done
// It returns true if the context is done
func SleepWithContext(ctx context.Context, d time.Duration) bool {
	select {
	case <-time.After(d):
		return false
	case <-ctx.Done():
		return true
	}
}

func DemoTokenPrefix(indexID string) string {
	return fmt.Sprintf("demo%s", indexID)
}

// CheckCDNURLIsExist check whether CDN URL exist or not
func CheckCDNURLIsExist(url string) bool {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	res, err := client.Head(url)
	if err != nil {
		return false
	}

	defer res.Body.Close()

	if res.StatusCode >= 200 && res.StatusCode < 400 {
		return true
	}

	return false
}

// GetMIMETypeByURL returns mimeType of a file based on the extension of the url
func GetMIMETypeByURL(urlString string) string {
	u, err := url.Parse(urlString)
	if err != nil {
		return ""
	}
	ext := path.Ext(u.Path)

	switch ext {
	case ".svg":
		return fmt.Sprintf("%s/%s", MediumImage, "svg+xml")
	case ".jpg", ".jpeg":
		return fmt.Sprintf("%s/%s", MediumImage, "jpeg")
	case ".png", ".gif":
		return fmt.Sprintf("%s/%s", MediumImage, strings.Split(ext, ".")[1])
	case ".mp4":
		return fmt.Sprintf("%s/%s", MediumVideo, "mp4")
	case ".mov":
		return fmt.Sprintf("%s/%s", MediumVideo, "quicktime")
	default:
		return ""
	}
}

// GetCIDFromIPFSLink seaches and returns the CID for a give url
func GetCIDFromIPFSLink(link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", err
	}

	switch u.Scheme {
	case "ipfs":
		return u.Host, nil
	case "http", "https":
		paths := strings.Split(path.Clean(u.Path), "/")

		for i := 0; i < len(paths); i++ {
			if paths[i] == "ipfs" {
				if i+1 < len(paths) {
					return paths[i+1], nil
				}
			}
		}
	default:
		return "", fmt.Errorf("unsupported ipfs link")
	}

	return "", fmt.Errorf("cid not found")
}

// NormalizeIndexIDs takes an array of token ids and return an array formatted token ids
// which includes formatting ethereum address and converting token id from hex to decimal if
// isConvertToDecimal is set to true. NOTE: There is no error return in this call.
func NormalizeIndexIDs(indexIDs []string, isConvertToDecimal bool) []string {
	var processedAddresses = []string{}
	for _, indexID := range indexIDs {
		blockchain, contractAddress, tokenID, err := ParseTokenIndexID(indexID)
		if err != nil {
			continue
		}

		if blockchain == BlockchainAlias[utils.EthereumBlockchain] {
			if isConvertToDecimal {
				decimalTokenID, ok := big.NewInt(0).SetString(tokenID, 16)
				if !ok {
					continue
				}
				tokenID = decimalTokenID.String()
			}
			indexID = fmt.Sprintf("%s-%s-%s", blockchain, contractAddress, tokenID)
		}

		processedAddresses = append(processedAddresses, indexID)
	}
	return processedAddresses
}

func BuildQueryParams(params interface{}) string {
	// 1. Validate input type
	t := reflect.TypeOf(params)
	if t.Kind() != reflect.Struct {
		panic("The params has to be a struct")
	}

	// 2. Iterate over the map to build the query params
	m := structs.Map(params)
	var buf bytes.Buffer
	for k, v := range m {
		buf.WriteString(getQueryParams(k, reflect.ValueOf(v)))
		buf.WriteString("&")
	}

	return strings.TrimSuffix(buf.String(), "&")
}

func getQueryParams(key string, val reflect.Value) string {
	urlKey := url.PathEscape(key)
	var buf bytes.Buffer
	switch val.Kind() {
	case reflect.Array, reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			buf.WriteString(getQueryParams(urlKey, val.Index(i)))
			buf.WriteString("&")
		}
	case reflect.Pointer:
		v := url.PathEscape(fmt.Sprintf("%v", val.Elem().Interface()))
		buf.WriteString(fmt.Sprintf("%s=%s", urlKey, v))
	default:
		v := url.PathEscape(fmt.Sprintf("%v", val.Interface()))
		buf.WriteString(fmt.Sprintf("%s=%s", urlKey, v))
	}

	return strings.TrimSuffix(strings.ReplaceAll(buf.String(), "+", "%2B"), "&")
}

// convert HEX token to DEC format
func HexToDec(hex string) string {
	n, ok := big.NewInt(0).SetString(hex, 16)
	if !ok {
		return ""
	}

	return n.Text(10)
}

func HexSha1(str string) string {
	h := sha1.New()
	h.Write([]byte(
		str,
	))
	hashedBytes := h.Sum(nil)
	return hex.EncodeToString(hashedBytes)
}
