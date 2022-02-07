package indexerWorker

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http/httputil"
	"strings"
	"time"

	goethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/crypto/sha3"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/contracts"
)

var (
	ErrMapKeyNotFound   = errors.New("key is not found")
	ErrValueNotString   = errors.New("value is not of string type")
	ErrInvalidEditionID = errors.New("invalid edition id")
)

// GetOwnedERC721TokenIDByContract returns a list of token id belongs to an owner for a specific contract
func (w *NFTIndexerWorker) GetOwnedERC721TokenIDByContract(ctx context.Context, contractAddress, ownerAddress string) ([]*big.Int, error) {
	rpcClient, err := ethclient.Dial(viper.GetString("ethereum.rpc_url"))
	if err != nil {
		return nil, err
	}

	contractAddr := common.HexToAddress(contractAddress)
	instance, err := contracts.NewERC721Enumerable(contractAddr, rpcClient)
	if err != nil {
		return nil, err
	}

	var tokenIDs []*big.Int

	ownerAddr := common.HexToAddress(ownerAddress)
	n, err := instance.BalanceOf(nil, ownerAddr)
	if err != nil {
		return nil, err
	}
	for i := int64(0); i < n.Int64(); i++ {
		id, err := instance.TokenOfOwnerByIndex(nil, ownerAddr, big.NewInt(i))
		if err != nil {
			return nil, err
		}

		tokenIDs = append(tokenIDs, id)
	}

	return tokenIDs, nil
}

// IndexTokenDataFromFromOpensea indexes data from OpenSea into the format of AssetUpdates
func (w *NFTIndexerWorker) IndexTokenDataFromFromOpensea(ctx context.Context, owner string, offset int) ([]indexer.AssetUpdates, error) {
	assets, err := w.opensea.RetrieveAssets(owner, offset)
	if err != nil {
		return nil, err
	}

	tokenUpdates := make([]indexer.AssetUpdates, 0, len(assets))

	for _, a := range assets {
		var source string
		var sourceURL string
		var artistURL string

		contractAddress := indexer.EthereumChecksumAddress(a.AssetContract.Address)
		switch contractAddress {
		case indexer.ENSContractAddress:
			continue
		}

		if _, ok := artblocksContracts[contractAddress]; ok {
			source = "ArtBlocks"
			sourceURL = "https://www.artblocks.io/"
		} else {
			source = "OpenSea"
			if viper.GetString("network") == "testnet" {
				sourceURL = "https://testnets.opensea.io"
			} else {
				sourceURL = "https://opensea.io"
			}
		}

		artistName := a.Creator.User.Username
		if a.Creator.Address != "" {
			if artistName == "" {
				artistName = a.Creator.Address
			}
			artistURL = fmt.Sprintf("%s/%s", sourceURL, a.Creator.Address)
		}

		metadata := indexer.ProjectMetadata{
			ArtistName:          artistName,
			ArtistURL:           artistURL,
			AssetID:             contractAddress,
			Title:               a.Name,
			Description:         a.Description,
			Medium:              "unknown",
			Source:              source,
			SourceURL:           sourceURL,
			PreviewURL:          a.ImageURL,
			ThumbnailURL:        a.ImageURL,
			GalleryThumbnailURL: a.ImagePreviewURL,
			AssetURL:            a.Permalink,
		}

		if a.AnimationURL != "" {
			metadata.Medium = "other"
			metadata.PreviewURL = a.AnimationURL
		} else if a.ImageURL != "" {
			metadata.Medium = "image"
		}

		// token id from opensea is a decimal integer string
		tokenID, ok := big.NewInt(0).SetString(a.TokenID, 10)
		if !ok {
			return nil, fmt.Errorf("fail to read token id from opensea")
		}

		tokenUpdate := indexer.AssetUpdates{
			ID:              fmt.Sprintf("%d", a.ID),
			Source:          "opensea",
			ProjectMetadata: metadata,
			Tokens: []indexer.Token{
				{
					BaseTokenInfo: indexer.BaseTokenInfo{
						ID:              a.TokenID,
						Blockchain:      indexer.EthereumBlockchain,
						ContractType:    strings.ToLower(a.AssetContract.SchemaName),
						ContractAddress: contractAddress,
					},
					IndexID: fmt.Sprintf("%s-%s-%s", indexer.BlockchianAlias[indexer.EthereumBlockchain], contractAddress, tokenID.Text(16)),
					Edition: 0,
					Owner:   owner,
					MintAt:  a.AssetContract.CreatedDate.Time,
				},
			},
		}

		log.WithField("asset update", tokenUpdate).Debug("asset updating data prepared")
		tokenUpdates = append(tokenUpdates, tokenUpdate)
	}

	return tokenUpdates, nil
}

// IndexTokenDataFromFromTezos indexes data from Tezos into the format of AssetUpdates
func (w *NFTIndexerWorker) IndexTokenDataFromFromTezos(ctx context.Context, owner string, offset int) ([]indexer.AssetUpdates, error) {
	tokens, err := w.bettercall.RetrieveTokens(owner, offset)
	if err != nil {
		return nil, err
	}

	tokenUpdates := make([]indexer.AssetUpdates, 0, len(tokens))

	for _, t := range tokens {
		log.WithField("token", t).Debug("index tezos token")

		tokenBlockchainMetadata, err := w.bettercall.GetTokenMetadata(t.Contract, t.ID)
		if err != nil {
			return nil, err
		}

		switch t.Contract {
		case indexer.KALAMContractAddress:
			continue
		}

		assetID := sha3.Sum256([]byte(fmt.Sprintf("%s-%d", t.Contract, t.ID)))
		assetIDString := hex.EncodeToString(assetID[:])

		var artistName, artistURL string
		if len(t.Creators) > 0 {
			artistName = t.Creators[0]
			artistURL = fmt.Sprintf("https://objkt.com/profile/%s", artistName)
		}

		// default display URI
		displayURI := "ipfs://QmV2cw5ytr3veNfAbJPpM5CeaST5vehT88XEmfdYY2wwiV"
		if t.DisplayUri != "" {
			displayURI = t.DisplayUri
		}

		previewURL := displayURI
		if t.ArtifactUri != "" {
			previewURL = t.ArtifactUri
		}

		var source, sourceURL, assetURL string
		medium := "unknown"
		switch t.Symbol {
		case "GENTK", "FXGEN":
			source = "FXHASH"
			sourceURL = "https://www.fxhash.xyz"
			assetURL = fmt.Sprintf("https://www.fxhash.xyz/gentk/%d", t.ID)
			displayURI = strings.ReplaceAll(displayURI, "ipfs://", "https://gateway.fxhash.xyz/ipfs/")
			previewURL = strings.ReplaceAll(previewURL, "ipfs://", "https://gateway.fxhash.xyz/ipfs/")
			medium = "software"
		case "OBJKT":
			source = "hicetnunc"
			sourceURL = "https://hicetnunc.art"
			assetURL = fmt.Sprintf("https://hicetnunc.art/objkt/%d", t.ID)
			displayURI = strings.ReplaceAll(displayURI, "ipfs://", "https://ipfs.io/ipfs/")
			previewURL = strings.ReplaceAll(previewURL, "ipfs://", "https://ipfs.io/ipfs/")
		default:
			source = "OBJKT.COM"
			sourceURL = "https://objkt.com"
			assetURL = fmt.Sprintf("https://objkt.com/asset/%s/%d", t.Contract, t.ID)
			displayURI = strings.ReplaceAll(displayURI, "ipfs://", "https://ipfs.io/ipfs/")
			previewURL = strings.ReplaceAll(previewURL, "ipfs://", "https://ipfs.io/ipfs/")
		}

		if medium == "unknown" {
			for _, f := range t.Formats {
				if f.URI == t.ArtifactUri {
					mimeItems := strings.Split(f.MIMEType, "/")
					if len(mimeItems) > 0 {
						switch mimeItems[0] {
						case "image":
							medium = "image"
						case "video":
							medium = "other"
						}
					}
				}
			}
		}

		metadata := indexer.ProjectMetadata{
			ArtistName:          artistName,
			ArtistURL:           artistURL,
			AssetID:             assetIDString,
			Title:               t.Name,
			Description:         t.Description,
			Medium:              medium,
			Source:              source,
			SourceURL:           sourceURL,
			PreviewURL:          previewURL,
			ThumbnailURL:        displayURI,
			GalleryThumbnailURL: displayURI,
			AssetURL:            assetURL,
		}

		tokenUpdate := indexer.AssetUpdates{
			ID:              assetIDString,
			Source:          "bcdhub",
			ProjectMetadata: metadata,
			Tokens: []indexer.Token{
				{
					BaseTokenInfo: indexer.BaseTokenInfo{
						ID:              fmt.Sprint(t.ID),
						Blockchain:      indexer.TezosBlockchain,
						ContractType:    "fa2",
						ContractAddress: t.Contract,
					},
					IndexID: fmt.Sprintf("%s-%s-%d", indexer.BlockchianAlias[indexer.TezosBlockchain], t.Contract, t.ID),
					Edition: 0,
					Owner:   owner,
					MintAt:  tokenBlockchainMetadata.Timestamp,
				},
			},
		}

		log.WithField("asset update", tokenUpdate).Debug("asset updating data prepared")
		tokenUpdates = append(tokenUpdates, tokenUpdate)
	}

	return tokenUpdates, nil
}

// IndexAsset saves asset data into indexer's storage
func (w *NFTIndexerWorker) GetTokenIDsByOwner(ctx context.Context, owner string) ([]string, error) {
	return w.indexerStore.GetTokenIDsByOwner(ctx, owner)
}

// IndexAsset saves asset data into indexer's storage
func (w *NFTIndexerWorker) IndexAsset(ctx context.Context, updates indexer.AssetUpdates) error {
	return w.indexerStore.IndexAsset(ctx, updates.ID, updates)
}

type Provenance struct {
	TxId      string    `json:"tx_id"`
	Owner     string    `json:"owner"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Bitmark is the response structure of bitmark registry
type Bitmark struct {
	Id         string       `json:"id"`
	HeadId     string       `json:"head_id"`
	Owner      string       `json:"owner"`
	AssetId    string       `json:"asset_id"`
	Issuer     string       `json:"issuer"`
	Head       string       `json:"head"`
	Status     string       `json:"status"`
	Provenance []Provenance `json:"provenance"`
}

// fetchBitmarkProvenance reads bitmark provenances through bitmark api
func (w *NFTIndexerWorker) fetchBitmarkProvenance(bitmarkID string) ([]indexer.Provenance, error) {
	provenances := []indexer.Provenance{}

	var data struct {
		Bitmark Bitmark `json:"bitmark"`
	}

	resp, err := w.http.Get(fmt.Sprintf("%s/v1/bitmarks/%s?provenance=true", w.bitmarkAPIEndpoint, bitmarkID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.WithError(err).Error("fail to dump http response")
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.WithError(err).WithField("httpdump", dump).Error("fail to decode bitmark payload")
		return nil, err
	}

	for i, p := range data.Bitmark.Provenance {
		txType := "transfer"

		if i == len(data.Bitmark.Provenance)-1 {
			txType = "issue"
		} else if p.Owner == w.bitmarkZeroAddress {
			txType = "burn"
		}

		provenances = append(provenances, indexer.Provenance{
			Type:       txType,
			Owner:      p.Owner,
			Blockchain: indexer.BitmarkBlockchain,
			Timestamp:  p.CreatedAt,
			TxID:       p.TxId,
		})
	}

	return provenances, nil
}

// fetchEthereumProvenance reads ethereum provenance through filterLogs
func (w *NFTIndexerWorker) fetchEthereumProvenance(ctx context.Context, tokenID, contractAddress string) ([]indexer.Provenance, error) {
	transferLogs, err := w.wallet.RPCClient().FilterLogs(ctx, goethereum.FilterQuery{
		Addresses: []common.Address{common.HexToAddress(contractAddress)},
		Topics: [][]common.Hash{
			{common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")},
			nil, nil,
			{common.HexToHash(tokenID)},
		},
	})
	if err != nil {
		return nil, err
	}

	log.WithField("tokenID", tokenID).WithField("logs", transferLogs).Debug("token provenance")

	totalTransferLogs := len(transferLogs)

	provenances := make([]indexer.Provenance, 0, totalTransferLogs)

	for i := range transferLogs {
		l := transferLogs[totalTransferLogs-i-1]

		fromAccountHash := l.Topics[1]
		toAccountHash := l.Topics[2]
		txType := "transfer"
		if fromAccountHash.Big().Cmp(big.NewInt(0)) == 0 {
			txType = "mint"
		}

		block, err := w.wallet.RPCClient().BlockByHash(ctx, l.BlockHash)
		if err != nil {
			return nil, err
		}
		txTime := time.Unix(int64(block.Time()), 0)

		provenances = append(provenances, indexer.Provenance{
			Timestamp:  txTime,
			Type:       txType,
			Owner:      indexer.EthereumChecksumAddress(toAccountHash.Hex()),
			Blockchain: indexer.EthereumBlockchain,
			TxID:       l.TxHash.Hex(),
		})
	}

	return provenances, nil
}

func (w *NFTIndexerWorker) GetOutdatedTokens(ctx context.Context) ([]indexer.Token, error) {
	return w.indexerStore.GetOutdatedTokens(ctx)
}

// RefreshTokenProvenance refresh provenance. This is a heavy task
func (w *NFTIndexerWorker) RefreshTokenProvenance(ctx context.Context, indexIDs []string, delay time.Duration) error {
	tokens, err := w.indexerStore.GetTokensByIndexIDs(ctx, indexIDs)
	if err != nil {
		return err
	}

	for _, token := range tokens {

		if token.LastRefreshedTime.Unix() > time.Now().Add(-delay).Unix() {
			log.WithField("indexID", token.IndexID).Debug("refresh too frequently")
			continue
		}

		log.WithField("indexID", token.IndexID).Debug("start refresh token provenance")

		totalProvenances := []indexer.Provenance{}
		switch token.Blockchain {
		case indexer.BitmarkBlockchain:
			provenance, err := w.fetchBitmarkProvenance(token.ID)
			if err != nil {
				return err
			}

			totalProvenances = append(totalProvenances, provenance...)
		case indexer.EthereumBlockchain:
			hexID, err := indexer.OpenseaTokenIDToHex(token.ID)
			if err != nil {
				return err
			}
			provenance, err := w.fetchEthereumProvenance(ctx, hexID, token.ContractAddress)
			if err != nil {
				return err
			}
			totalProvenances = append(totalProvenances, provenance...)
		}

		for _, tokenInfo := range token.OriginTokenInfo {
			switch tokenInfo.Blockchain {
			case indexer.BitmarkBlockchain:
				provenance, err := w.fetchBitmarkProvenance(tokenInfo.ID)
				if err != nil {
					return err
				}

				totalProvenances = append(totalProvenances, provenance...)
			case indexer.EthereumBlockchain:
				hexID, err := indexer.OpenseaTokenIDToHex(tokenInfo.ID)
				if err != nil {
					return err
				}
				provenance, err := w.fetchEthereumProvenance(ctx, hexID, token.ContractAddress)
				if err != nil {
					return err
				}
				totalProvenances = append(totalProvenances, provenance...)
			}
		}

		if err := w.indexerStore.UpdateTokenProvenance(ctx, token.IndexID, totalProvenances); err != nil {
			return err
		}
	}

	return nil
}
