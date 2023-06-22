package indexer

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"blockwatch.cc/tzgo/tezos"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

	if blockchainType == EthereumBlockchain {
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

	if v[0] == BlockchainAlias[EthereumBlockchain] {
		v[1] = EthereumChecksumAddress(v[1])
	}

	return v[0], v[1], v[2], nil
}

// GetBlockchainByAddress returns underlying blockchain of a given address
func GetBlockchainByAddress(address string) string {
	if strings.HasPrefix(address, "0x") {
		return EthereumBlockchain
	} else if len(address) == 50 {
		return BitmarkBlockchain
	} else if strings.HasPrefix(address, "tz") || strings.HasPrefix(address, "KT1") {
		return TezosBlockchain
	}

	return UnknownBlockchain
}

// TxURL returns corresponded blockchain transaction URL
func TxURL(blockchain, environment, txID string) string {
	switch blockchain {
	case BitmarkBlockchain:
		if environment == DevelopmentEnvironment {
			return fmt.Sprintf("https://registry.test.bitmark.com/transaction/%s", txID)
		}
		return fmt.Sprintf("https://registry.bitmark.com/transaction/%s", txID)
	case EthereumBlockchain:
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

// EpochStringToTime returns the time object of a milliseconds epoch time string
func EpochStringToTime(ts string) (time.Time, error) {
	t, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(0, t*1000000), nil
}

// IsTimeInRange ensures a given timestamp is within a range of a target time
func IsTimeInRange(actual, target time.Time, deviationInMinutes float64) bool {
	duration := target.Sub(actual)
	return math.Abs(duration.Minutes()) < deviationInMinutes
}

// VerifyETHSignature verifies a signature with a given message and address
func VerifyETHSignature(message, signature, address string) (bool, error) {
	hash := accounts.TextHash([]byte(message))
	signatureBytes := common.FromHex(signature)

	if len(signatureBytes) != 65 {
		return false, fmt.Errorf("signature must be 65 bytes long")
	}

	// see crypto.Ecrecover description
	if signatureBytes[64] == 27 || signatureBytes[64] == 28 {
		signatureBytes[64] -= 27
	}

	// get ecdsa public key
	sigPublicKeyECDSA, err := crypto.SigToPub(hash, signatureBytes)
	if err != nil {
		return false, err
	}

	// check for address match
	sigAddress := crypto.PubkeyToAddress(*sigPublicKeyECDSA)
	if sigAddress.String() != address {
		return false, fmt.Errorf("address doesn't match with signature's")
	}

	return true, nil
}

// VerifyTezosSignature verifies a signature with a given message and address
func VerifyTezosSignature(message, signature, address, publicKey string) (bool, error) {
	ta, err := tezos.ParseAddress(address)
	if err != nil {
		return false, err
	}
	pk, err := tezos.ParseKey(publicKey)
	if err != nil {
		return false, err
	}
	if pk.Address().String() != ta.String() {
		return false, errors.New("publicKey address is different from provided address")
	}
	sig, err := tezos.ParseSignature(signature)
	if err != nil {
		return false, err
	}
	dmp := tezos.Digest([]byte(message))
	err = pk.Verify(dmp[:], sig)
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetMIMEType returns mimeType of a file based on the extension of the url
func GetMIMEType(urlString string) string {
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
