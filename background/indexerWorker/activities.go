package indexerWorker

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"path/filepath"
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
	"github.com/bitmark-inc/nft-indexer/traceutils"
)

var (
	ErrMapKeyNotFound   = errors.New("key is not found")
	ErrValueNotString   = errors.New("value is not of string type")
	ErrInvalidEditionID = errors.New("invalid edition id")
)

func mediumByPreviewFileExtension(url string) string {
	ext := filepath.Ext(url)

	switch ext {
	case ".jpg", ".jpeg", ".png", ".svg":
		return "image"
	case ".mp4", ".mov":
		return "video"
	default:
		return "other"
	}
}

func getTokenSourceByContract(contractAddress string) string {
	switch indexer.DetectContractBlockchain(contractAddress) {
	case indexer.EthereumBlockchain:
		if _, ok := artblocksContracts[contractAddress]; ok {
			return "Art Blocks"

		} else if contractAddress == "0x70e6b3f9d99432fCF35274d6b24D83Ef5Ba3dE2D" {
			return "Crayon Codes"
		}

		return "OpenSea"
	case indexer.TezosBlockchain:
		// WIP
		return ""
	default:
		return ""
	}
}

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
	// The data source of the asset data. This field is not related to displaying
	dataSource := "opensea"

	assets, err := w.opensea.RetrieveAssets(owner, offset)
	if err != nil {
		return nil, err
	}

	tokenUpdates := make([]indexer.AssetUpdates, 0, len(assets))

	for _, a := range assets {
		var sourceURL string
		var artistURL string
		artistName := a.Creator.User.Username
		contractAddress := indexer.EthereumChecksumAddress(a.AssetContract.Address)
		switch contractAddress {
		case indexer.ENSContractAddress:
			continue
		}

		source := getTokenSourceByContract(contractAddress)

		switch source {
		case "Art Blocks":
			sourceURL = "https://www.artblocks.io/"
			artistURL = fmt.Sprintf("%s/%s", sourceURL, a.Creator.Address)
		case "Crayon Codes":
			sourceURL = "https://openprocessing.org/crayon/"
			artistURL = fmt.Sprintf("https://opensea.io/%s", a.Creator.Address)
		default:
			if viper.GetString("network") == "testnet" {
				sourceURL = "https://testnets.opensea.io"
			} else {
				sourceURL = "https://opensea.io"
			}
			artistURL = fmt.Sprintf("https://opensea.io/%s", a.Creator.Address)
		}

		log.WithField("source", source).WithField("assetID", a.ID).Debug("source debug")

		if a.Creator.Address != "" {
			if artistName == "" {
				artistName = a.Creator.Address
			}

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
			metadata.PreviewURL = a.AnimationURL

			if source == "Art Blocks" {
				metadata.Medium = "software"
			} else {
				medium := mediumByPreviewFileExtension(metadata.PreviewURL)
				log.WithField("previewURL", metadata.PreviewURL).WithField("medium", medium).Debug("fallback medium check")
				metadata.Medium = medium
			}
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
			Source:          dataSource,
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

// fxhashLink converts an IPFS link to a HTTP link by using fxhash ipfs gateway.
// If a link is failed to parse, it returns the original link
func fxhashLink(ipfsLink string) string {
	u, err := url.Parse(ipfsLink)
	if err != nil {
		return ipfsLink
	}

	u.Path = fmt.Sprintf("ipfs/%s/", u.Host)
	u.Host = "gateway.fxhash.xyz"
	u.Scheme = "https"

	return u.String()
}

// IndexTokenDataFromFromTezos indexes data from Tezos into the format of AssetUpdates
func (w *NFTIndexerWorker) IndexTokenDataFromFromTezos(ctx context.Context, owner string, offset int) ([]indexer.AssetUpdates, error) {
	// The data source of the asset data. This field is not related to displaying
	dataSource := "bcdhub"

	tokens, err := w.bettercall.RetrieveTokens(owner, offset)
	if err != nil {
		return nil, err
	}

	tokenUpdates := make([]indexer.AssetUpdates, 0, len(tokens))

	for _, t := range tokens {
		switch t.Contract {
		case indexer.KALAMContractAddress, indexer.TezosDNSContractAddress:
			continue
		}

		log.WithField("token", t).Debug("index tezos token")

		tokenBlockchainMetadata, err := w.bettercall.GetTokenMetadata(t.Contract, t.ID.String())
		if err != nil {
			log.WithError(err).Error("fail to get metadata for the token")
		}

		name := t.Name
		description := t.Description
		mintedAt := tokenBlockchainMetadata.Timestamp
		maxEdition := tokenBlockchainMetadata.Supply

		assetID := sha3.Sum256([]byte(fmt.Sprintf("%s-%s", t.Contract, t.ID.String())))
		assetIDString := hex.EncodeToString(assetID[:])

		var artistName, artistURL string
		if len(t.Creators) > 0 {
			artistName = t.Creators[0]
			artistURL = fmt.Sprintf("https://objkt.com/profile/%s", artistName)
		}

		// default display URI
		displayURI := "ipfs://QmV2cw5ytr3veNfAbJPpM5CeaST5vehT88XEmfdYY2wwiV"
		if t.DisplayURI != "" {
			displayURI = t.DisplayURI
		}

		previewURI := displayURI
		if t.ArtifactURI != "" {
			previewURI = t.ArtifactURI
		}

		var source, sourceURL, assetURL string
		var edition int64
		medium := "unknown"
		switch t.Contract {
		case indexer.FXHASHV2ContractAddress, indexer.FXHASHContractAddress, indexer.FXHASHOldContractAddress:
			detail, err := w.fxhash.GetObjectDetail(ctx, t.ID.Int)
			if err != nil {
				log.WithError(err).Error("fail to get token detail from fxhash")
			} else {
				name = detail.Name
				description = detail.Metadata.Description
				artistName = detail.Issuer.Author.ID
				mintedAt = detail.CreatedAt
				edition = detail.Iteration
				maxEdition = detail.Issuer.Supply
				artistURL = fmt.Sprintf("https://www.fxhash.xyz/u/%s", detail.Issuer.Author.Name)
				displayURI = detail.Metadata.DisplayURI
				previewURI = detail.Metadata.ArtifactURI
			}

			source = "fxhash"
			sourceURL = "https://www.fxhash.xyz"
			assetURL = fmt.Sprintf("https://www.fxhash.xyz/gentk/%s", t.ID.String())

			displayURI = fxhashLink(displayURI)
			previewURI = fxhashLink(previewURI)
			medium = "software"

		case indexer.VersumContractAddress:
			source = "versum"
			sourceURL = "https://versum.xyz"
			assetURL = fmt.Sprintf("https://versum.xyz/token/versum/%s", t.ID.String())
			displayURI = strings.ReplaceAll(displayURI, "ipfs://", "https://ipfs.io/ipfs/")
			previewURI = strings.ReplaceAll(previewURI, "ipfs://", "https://ipfs.io/ipfs/")
			artistURL = fmt.Sprintf("https://versum.xyz/user/%s", artistName)
		case indexer.HicEtNuncContractAddress:
			source = "hic et nunc"
			sourceURL = "https://objkt.com" // hicetnunc is down. We not fallback to objkt.com
			assetURL = fmt.Sprintf("https://objkt.com/asset/%s/%s", t.Contract, t.ID.String())
			displayURI = strings.ReplaceAll(displayURI, "ipfs://", "https://ipfs.io/ipfs/")
			previewURI = strings.ReplaceAll(previewURI, "ipfs://", "https://ipfs.io/ipfs/")
		default:
			detail, err := w.objkt.GetObjktDetailed(ctx, t.ID.Text(10), t.Contract)
			if err != nil {
				log.WithError(err).Error("fail to get token detail from objkt")
				source = "unknown"
			} else {
				name = detail.Name
				description = detail.Description
				mintedAt = detail.MintedAt
				maxEdition = detail.Supply
				artistName = detail.Contract.CreatorAddress
				artistURL = fmt.Sprintf("https://objkt.com/profile/%s", detail.Contract.CreatorAddress)

				displayURI = detail.DisplayURI
				previewURI = detail.ArtifactURI

				mimeItems := strings.Split(detail.MIMEType, "/")
				if len(mimeItems) > 0 {
					switch mimeItems[0] {
					case "image":
						medium = "image"
					case "video":
						medium = "other"
					}
				}
				source = "objkt.com"
				sourceURL = "https://objkt.com"
				assetURL = fmt.Sprintf("https://objkt.com/asset/%s/%s", t.Contract, t.ID.String())
				displayURI = strings.ReplaceAll(displayURI, "ipfs://", "https://ipfs.io/ipfs/")
				previewURI = strings.ReplaceAll(previewURI, "ipfs://", "https://ipfs.io/ipfs/")
			}
		}

		if medium == "unknown" {
			for _, f := range t.Formats {
				if f.URI == t.ArtifactURI {
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
			Title:               name,
			Description:         description,
			Medium:              medium,
			MaxEdition:          maxEdition,
			Source:              source,
			SourceURL:           sourceURL,
			PreviewURL:          previewURI,
			ThumbnailURL:        displayURI,
			GalleryThumbnailURL: displayURI,
			AssetURL:            assetURL,
		}

		tokenUpdate := indexer.AssetUpdates{
			ID:              assetIDString,
			Source:          dataSource,
			ProjectMetadata: metadata,
			Tokens: []indexer.Token{
				{
					BaseTokenInfo: indexer.BaseTokenInfo{
						ID:              t.ID.String(),
						Blockchain:      indexer.TezosBlockchain,
						ContractType:    "fa2",
						ContractAddress: t.Contract,
					},
					IndexID: fmt.Sprintf("%s-%s-%s", indexer.BlockchianAlias[indexer.TezosBlockchain], t.Contract, t.ID.String()),
					Edition: edition,
					Owner:   owner,
					MintAt:  mintedAt,
				},
			},
		}

		log.WithField("blockchain", indexer.TezosBlockchain).
			WithField("owner", owner).
			WithField("id", fmt.Sprintf("%s-%s-%s", indexer.BlockchianAlias[indexer.TezosBlockchain], t.Contract, t.ID.String())).
			WithField("metadata", metadata).
			Debug("asset updating data prepared")
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

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.WithError(err).WithField("respData", traceutils.DumpResponse(resp)).Error("fail to decode bitmark payload")
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
			TxURL:      indexer.TxURL(indexer.BitmarkBlockchain, w.Network, p.TxId),
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
			TxURL:      indexer.TxURL(indexer.EthereumBlockchain, w.Network, l.TxHash.Hex()),
		})
	}

	return provenances, nil
}

func (w *NFTIndexerWorker) GetOutdatedTokens(ctx context.Context, size int64) ([]indexer.Token, error) {
	return w.indexerStore.GetOutdatedTokens(ctx, size)
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

		log.WithField("indexID", token.IndexID).Trace("start refresh token provenance")

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
