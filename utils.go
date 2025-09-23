package indexer

import (
	"bytes"
	"context"
	"crypto/sha1" // #nosec G505 -- FIXME: SHA1 used for non-cryptographic purposes (hashing, deduplication)
	"encoding/hex"
	"fmt"
	"io"
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
	"go.mongodb.org/mongo-driver/bson"
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

	defer func() {
		_ = res.Body.Close()
	}()

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

// GetCIDFromIPFSLink searches and returns the CID for a give url
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
	h := sha1.New() // #nosec G401 -- FIXME: SHA1 used for non-cryptographic purposes (hashing, deduplication)
	h.Write([]byte(
		str,
	))
	hashedBytes := h.Sum(nil)
	return hex.EncodeToString(hashedBytes)
}

// IsBurnAddress returns true if the address is a burn address
func IsBurnAddress(address string, environment string) bool {
	return address == EthereumZeroAddress ||
		address == TezosBurnAddress ||
		(environment == DevelopmentEnvironment && address == TestnetZeroAddress) ||
		(environment == ProductionEnvironment && address == LivenetZeroAddress)
}

// ResolveIPFSURI converts an IPFS URI (ipfs://CID/path) to a HTTP URL using the specified gateway.
// If the URI is already an HTTPS URL, it returns it unchanged. For IPFS URIs, it constructs
// a URL in the format https://gateway/ipfs/CID/path to enable content retrieval through the gateway.
func ResolveIPFSURI(gateway, uri string) string {
	// If the URI is not an IPFS URI, return it as is
	if !IsIPFSURI(uri) {
		return uri
	}

	// Parse the URI
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	// Clean the path and construct gateway URL
	cid := parsedURL.Host // CID is typically the host in ipfs://CID/path
	path := strings.TrimLeft(parsedURL.Path, "/")
	gatewayPath := fmt.Sprintf("ipfs/%s", cid)
	if path != "" {
		gatewayPath = fmt.Sprintf("%s/%s", gatewayPath, path)
	}

	return fmt.Sprintf("https://%s/%s", gateway, gatewayPath)
}

// ReadFromURL reads the data from given URL within a timeout
func ReadFromURL(ctx context.Context, url string, timeout time.Duration) ([]byte, error) {
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d for URL: %s", resp.StatusCode, url)
	}

	// Check content length to avoid extremely large responses
	const maxContentLength = 10 * 1024 * 1024 // 10MB limit
	if resp.ContentLength > maxContentLength {
		return nil, fmt.Errorf("response too large: %d bytes", resp.ContentLength)
	}

	// Use LimitReader to prevent reading extremely large bodies
	bodyReader := io.LimitReader(resp.Body, maxContentLength)
	data, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty response body from URL: %s", url)
	}

	return data, nil
}

// IsIPFSURI returns true if the URI is an IPFS URI
func IsIPFSURI(uri string) bool {
	return strings.HasPrefix(uri, "ipfs://")
}

// IsHTTPSURI returns true if the URI is an HTTPS URI
func IsHTTPSURI(uri string) bool {
	return strings.HasPrefix(uri, "https://")
}

// flattenMap recursively processes nested maps to create dot notation paths for MongoDB
func flattenMap(input map[string]interface{}, prefix string, result bson.M) {
	for k, v := range input {
		key := prefix + "." + k

		// If value is a nested map, process it recursively
		if nestedMap, ok := v.(map[string]interface{}); ok {
			flattenMap(nestedMap, key, result)
		} else {
			// Otherwise add the value with the full path
			result[key] = v
		}
	}
}
