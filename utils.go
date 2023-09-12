package indexer

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	utils "github.com/bitmark-inc/autonomy-utils"
	"github.com/ethereum/go-ethereum/common"
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

// AssetIndexID returns a source-based unique asset id. It is constructed by
// source of the asset data and the asset id from the source site.
func AssetIndexID(source, id string) string {
	return fmt.Sprintf("%s-%s", source, id)
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

func AESSeal(message []byte, passphrase string) (string, error) {
	key := []byte(passphrase)

	c, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	bytes := gcm.Seal(nonce, nonce, message, nil)

	return hex.EncodeToString(bytes), nil
}

func AESOpen(hexString string, passphrase string) ([]byte, error) {
	ciphertext, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, err
	}

	key := []byte(passphrase)

	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, err
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	message, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return message, nil
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

// PreprocessTokens takes an array of token ids and return an array formatted token ids
// which includes formatting ethereum address and converting token id from hex to decimal if
// isConvertToDecimal is set to true. NOTE: There is no error return in this call.
func PreprocessTokens(indexIDs []string, isConvertToDecimal bool) []string {
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
