package indexer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/sha3"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"
	"github.com/bitmark-inc/tzkt-go"
	"github.com/feral-file/ff-indexer/externals/objkt"
)

type HexString string

func (s *HexString) UnmarshalJSON(data []byte) error {
	var hexString string

	if err := json.Unmarshal(data, &hexString); err != nil {
		return err
	}

	b, err := hex.DecodeString(hexString)
	if err != nil {
		return err
	}
	*s = HexString(b)

	return nil
}

func (s HexString) MarshalJSON() ([]byte, error) {
	hexString := hex.EncodeToString([]byte(s))
	return json.Marshal(hexString)
}

type TezosTokenMetadata struct {
	TokenID   string               `json:"token_id"`
	TokenInfo map[string]HexString `json:"token_info"`
}

func (e *IndexEngine) GetTezosTokenByOwner(owner string, lastTime time.Time, offset int) ([]tzkt.OwnedToken, error) {
	if _, excluded := TezosIndexExcludedOwners[owner]; excluded {
		return nil, nil
	}

	tokens, err := e.tzkt.RetrieveTokens(owner, lastTime, offset)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// IndexTezosTokenByOwner indexes all tokens owned by a specific tezos address
func (e *IndexEngine) IndexTezosTokenByOwner(ctx context.Context, owner string, lastTime time.Time, offset int) ([]AssetUpdates, time.Time, error) {
	var newLastTime = time.Time{}
	ownedTokens, err := e.GetTezosTokenByOwner(owner, lastTime, offset)
	if err != nil {
		return nil, newLastTime, err
	}

	log.Debug("retrieve tokens for owner", zap.Any("tokens", ownedTokens), zap.String("owner", owner))

	tokenUpdates := make([]AssetUpdates, 0, len(ownedTokens))

	for _, t := range ownedTokens {

		update, err := e.indexTezosToken(ctx, t.Token, owner, int64(t.Balance), t.LastTime)
		if err != nil {
			log.WarnWithContext(ctx, "fail to index a tezos token", zap.Error(err))
			return nil, newLastTime, err
		}

		if update != nil {
			tokenUpdates = append(tokenUpdates, *update)
		}
	}

	if len(ownedTokens) > 0 {
		newLastTime = ownedTokens[len(ownedTokens)-1].LastTime
	}

	return tokenUpdates, newLastTime, nil
}

// IndexTezosToken indexes a Tezos token with a specific contract and ID
func (e *IndexEngine) IndexTezosToken(ctx context.Context, contract, tokenID string) (*AssetUpdates, error) {
	tzktToken, err := e.tzkt.GetContractToken(contract, tokenID)
	if err != nil {
		log.Debug("GetContractToken",
			log.SourceTZKT,
			zap.Error(err),
			zap.String("contract", contract), zap.String("tokenID", tokenID))
		return nil, err
	}

	return e.indexTezosToken(ctx, tzktToken, "", 0, tzktToken.LastTime)
}

// searchMetadataFromIPFS searches token metadata from a list of preferred ipfs gateway
func (e *IndexEngine) searchMetadataFromIPFS(ipfsURI string) (*tzkt.TokenMetadata, error) {
	if strings.HasPrefix(ipfsURI, "https://") {
		return e.fetchMetadataByLink(ipfsURI)
	}

	if !strings.HasPrefix(ipfsURI, "ipfs://") {
		return nil, fmt.Errorf("invalid ipfs link")
	}

	for _, gateway := range e.ipfsGateways {
		u := ipfsURLToGatewayURL(gateway, ipfsURI)
		return e.fetchMetadataByLink(u)
	}

	return nil, fmt.Errorf("fail to get metadata from the preferred gateways")
}

// fetchMetadataByLink reads tezos metadata by a given link
func (e *IndexEngine) fetchMetadataByLink(url string) (*tzkt.TokenMetadata, error) {
	resp, err := e.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var metadata tzkt.TokenMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// getTokenMetadataURL fetches token metadata URL from blockchain
func (e *IndexEngine) getTokenMetadataURL(contractAddress, tokenID string) (string, error) {
	p, err := e.tzkt.GetBigMapPointerForContractTokenMetadata(contractAddress)
	if err != nil {
		return "", err
	}

	b, err := e.tzkt.GetBigMapValueByPointer(p, tokenID)
	if err != nil {
		return "", err
	}

	var tokenMetadata TezosTokenMetadata
	if err := json.Unmarshal(b, &tokenMetadata); err != nil {
		return "", err
	}

	return string(tokenMetadata.TokenInfo[""]), nil
}

// getTezosTokenBalanceOfOwner returns current balance of a token that the owner owns
func (e *IndexEngine) getTezosTokenBalanceOfOwner(_ context.Context, contract, tokenID, owner string) (int64, error) {
	if _, excluded := TezosIndexExcludedOwners[owner]; excluded {
		return 0, nil
	}

	return e.tzkt.GetTokenBalanceOfOwner(contract, tokenID, owner)
}

// indexTezosToken prepares indexing data for a tezos token using the
// source API token object. It currently uses token objects from tzkt api
func (e *IndexEngine) indexTezosToken(ctx context.Context, tzktToken tzkt.Token, owner string, balance int64, lastActivityTime time.Time) (*AssetUpdates, error) {
	log.Debug("index tezos token", zap.Any("token", tzktToken))
	assetIDBytes := sha3.Sum256([]byte(fmt.Sprintf("%s-%s", tzktToken.Contract.Address, tzktToken.ID.String())))
	assetID := hex.EncodeToString(assetIDBytes[:])

	metadataDetail := NewAssetMetadataDetail(assetID)
	metadataDetail.FromTZKT(tzktToken)
	gateway := DefaultIPFSGateway
	tokenDetail := TokenDetail{
		MintedAt: tzktToken.Timestamp,
	}

	var nonCustomizedMarketplace bool
	if e.environment != DevelopmentEnvironment { // production indexing process
		if tzktToken.Metadata == nil || time.Since(lastActivityTime) < 14*24*time.Hour {
			tokenMetadataURL, err := e.getTokenMetadataURL(tzktToken.Contract.Address, tzktToken.ID.String())
			if err != nil {
				log.WarnWithContext(ctx, "fail to get token metadata url from blockchain", zap.Error(err), log.SourceTZKT)
			} else {
				metadata, err := e.searchMetadataFromIPFS(tokenMetadataURL)
				if err != nil {
					log.WarnWithContext(ctx, "fail to search token metadata from ipfs", zap.String("tokenMetadataURL", tokenMetadataURL), zap.Error(err), log.SourceTZKT)
				} else {
					metadataDetail.FromTZIP21TokenMetadata(*metadata)
				}
			}
		}

		tokenDetail.Fungible = metadataDetail.MaxEdition > 1 || !metadataDetail.IsBooleanAmount

		switch tzktToken.Contract.Address {
		case KALAMContractAddress, TezDaoContractAddress, TezosDNSContractAddress:
			return nil, nil

		case FXHASHContractAddressFX0_0, FXHASHContractAddressFX0_1, FXHASHContractAddressFX0_2, FXHASHContractAddressFX1:
			tokenDetail.Fungible = false
			fxObjktID := fmt.Sprintf("%s-%s", tzktToken.Contract.Address, tzktToken.ID.String())
			e.indexTokenFromFXHASH(ctx, fxObjktID, metadataDetail, &tokenDetail)

		case VersumContractAddress:
			tokenDetail.Fungible = true
			metadataDetail.SetMarketplace(MarketplaceProfile{"versum", "https://versum.xyz",
				fmt.Sprintf("https://versum.xyz/token/versum/%s", tzktToken.ID.String())},
			)

			metadataDetail.ArtistURL = fmt.Sprintf("https://versum.xyz/user/%s", metadataDetail.ArtistName)
			for i, artist := range metadataDetail.Artists {
				metadataDetail.Artists[i].URL = fmt.Sprintf("https://versum.xyz/user/%s", artist.Name)
			}

		default:
			nonCustomizedMarketplace = true
		}
	} else { // development indexing process
		tokenMetadataURL, err := e.getTokenMetadataURL(tzktToken.Contract.Address, tzktToken.ID.String())
		if err != nil {
			log.WarnWithContext(ctx, "fail to get token metadata url from blockchain", zap.Error(err), log.SourceTZKT)
		} else {
			var metadata *tzkt.TokenMetadata
			if gateway != DefaultIPFSGateway {
				var err error
				tokenMetadataURL = ipfsURLToGatewayURL(gateway, tokenMetadataURL)
				metadata, err = e.fetchMetadataByLink(tokenMetadataURL)
				if err != nil {
					log.WarnWithContext(ctx, "fail to read token metadata from ipfs",
						zap.Error(err), zap.String("gateway", gateway), log.SourceTZKT)
				}
			} else {
				var err error
				metadata, err = e.searchMetadataFromIPFS(tokenMetadataURL)
				if err != nil {
					log.WarnWithContext(ctx, "fail to search token metadata from ipfs",
						zap.String("tokenMetadataURL", tokenMetadataURL), zap.Error(err), log.SourceTZKT)
				}
			}

			if metadata != nil {
				metadataDetail.FromTZIP21TokenMetadata(*metadata)
			}
		}

		tokenDetail.Fungible = metadataDetail.MaxEdition > 1 || !metadataDetail.IsBooleanAmount

		switch tzktToken.Contract.Address {
		case FXHASHContractAddressDev0_0, FXHASHContractAddressDev0_1:
			gateway = FxhashDevIPFSGateway
			metadataDetail.SetMarketplace(
				MarketplaceProfile{
					"fxhash-dev",
					"https://dev.fxhash-dev.xyz",
					"",
				},
			)
			metadataDetail.SetMedium(MediumSoftware)
		default:
			nonCustomizedMarketplace = true
		}
	}

	// for non-customized marketplace, we detect the source and urls by metadata and objkt.com
	if nonCustomizedMarketplace {
		var source, sourceURL string

		if metadataDetail.Source != "" {
			source = metadataDetail.Source
		} else {
			if tzktToken.Metadata != nil {
				switch tzktToken.Metadata.Symbol {
				case "OBJKTCOM":
					source = "objkt"
				case "OBJKT":
					source = "hic et nunc"
				default:
					source = "unknown"
				}
			}
		}

		if _, ok := inhouseMinter[metadataDetail.Minter]; !ok {
			// fallback to objkt marketplace if the minter is not autonomy inhouse minter
			objktToken, err := e.GetObjktToken(ctx, tzktToken.Contract.Address, tzktToken.ID.String())
			if err != nil {
				log.WarnWithContext(ctx, "fail to get token detail from objkt", zap.Error(err), log.SourceObjkt)
			} else {
				metadataDetail.FromObjkt(objktToken)
			}
			sourceURL = "https://objkt.com"
		} else {
			sourceURL = "https://autonomy.io"
		}

		// always use objkt for the fallback asset url
		objktHost := "objkt.com"
		if e.environment == DevelopmentEnvironment {
			objktHost = "ghostnet.objkt.com"
		}
		assetURL := fmt.Sprintf("https://%s/asset/%s/%s", objktHost, tzktToken.Contract.Address, tzktToken.ID.String())

		metadataDetail.SetMarketplace(MarketplaceProfile{source, sourceURL, assetURL})
	}

	if g, ok := e.minterGateways[metadataDetail.Minter]; ok {
		log.Debug("override ipfs gateway by minter", zap.String("new_gateway", g))
		gateway = g
	}

	// ensure ipfs urls are converted to http links
	metadataDetail.ThumbnailURI = ipfsURLToGatewayURL(gateway, metadataDetail.ThumbnailURI)
	metadataDetail.DisplayURI = ipfsURLToGatewayURL(gateway, metadataDetail.DisplayURI)
	metadataDetail.PreviewURI = ipfsURLToGatewayURL(gateway, metadataDetail.PreviewURI)

	pm := ProjectMetadata{
		AssetID:   metadataDetail.AssetID,
		Source:    metadataDetail.Source,
		SourceURL: metadataDetail.SourceURL,
		AssetURL:  metadataDetail.AssetURL,

		Title:       metadataDetail.Name,
		Description: metadataDetail.Description,
		MIMEType:    metadataDetail.MIMEType,
		Medium:      metadataDetail.Medium,

		ArtistID:   metadataDetail.ArtistID,
		ArtistName: metadataDetail.ArtistName,
		ArtistURL:  metadataDetail.ArtistURL,
		Artists:    metadataDetail.Artists,
		MaxEdition: metadataDetail.MaxEdition,

		PreviewURL: metadataDetail.PreviewURI,
		// use the thumbnail in metadata for ThumbnailURL
		ThumbnailURL: metadataDetail.ThumbnailURI,
		// use the high quality image for GalleryThumbnailURL
		GalleryThumbnailURL: metadataDetail.DisplayURI,

		ArtworkMetadata: metadataDetail.ArtworkMetadata,

		LastUpdatedAt: time.Now(),
	}

	tokenUpdate := AssetUpdates{
		ID:              assetID,
		Source:          SourceTZKT, // asset data source which is different than project source
		ProjectMetadata: pm,
		Tokens: []Token{
			{
				BaseTokenInfo: BaseTokenInfo{
					ID:              tzktToken.ID.String(),
					Blockchain:      utils.TezosBlockchain,
					Fungible:        tokenDetail.Fungible,
					ContractType:    tzktToken.Standard,
					ContractAddress: tzktToken.Contract.Address,
				},
				IndexID:           TokenIndexID(utils.TezosBlockchain, tzktToken.Contract.Address, tzktToken.ID.String()),
				Owner:             owner,
				Balance:           balance,
				Owners:            map[string]int64{owner: balance},
				Edition:           tokenDetail.Edition,
				MintedAt:          tokenDetail.MintedAt,
				LastRefreshedTime: time.Now(),
				LastActivityTime:  lastActivityTime,
			},
		},
	}

	log.Debug("asset updating data prepared",
		zap.String("blockchain", utils.TezosBlockchain),
		zap.String("id", TokenIndexID(utils.TezosBlockchain, tzktToken.Contract.Address, tzktToken.ID.String())),
		zap.Any("tokenUpdate", tokenUpdate))

	return &tokenUpdate, nil
}

// IndexTezosTokenProvenance indexes provenance of a specific token
func (e *IndexEngine) IndexTezosTokenProvenance(contract, tokenID string) ([]Provenance, error) {
	log.Debug("index tezos token provenance",
		zap.String("blockchain", utils.TezosBlockchain),
		zap.String("contract", contract), zap.String("tokenID", tokenID))

	count, err := e.tzkt.GetTokenTransfersCount(contract, tokenID)
	if err != nil {
		return nil, err
	}

	transfers, err := e.tzkt.GetTokenTransfers(contract, tokenID, count)
	if err != nil {
		return nil, err
	}

	provenances := make([]Provenance, 0, len(transfers))
	for i := len(transfers) - 1; i >= 0; i-- {
		t := transfers[i]

		tx, err := e.tzkt.GetTransaction(t.TransactionID)
		if err != nil {
			return nil, err
		}

		txType := "transfer"
		if t.From == nil {
			txType = "mint"
		}

		provenances = append(provenances, Provenance{
			Type:        txType,
			Owner:       t.To.Address,
			Blockchain:  utils.TezosBlockchain,
			BlockNumber: &t.Level,
			Timestamp:   t.Timestamp,
			TxID:        tx.Hash,
			TxURL:       fmt.Sprintf("https://tzkt.io/%s", tx.Hash),
		})
	}

	return provenances, nil
}

// IndexTezosTokenLastActivityTime indexes the last activity timestamp of a given token
func (e *IndexEngine) IndexTezosTokenLastActivityTime(contract, tokenID string) (time.Time, error) {
	return e.tzkt.GetTokenLastActivityTime(contract, tokenID)
}

// IndexTezosTokenOwners indexes owners of a given token
func (e *IndexEngine) IndexTezosTokenOwners(contract, tokenID string) ([]OwnerBalance, error) {
	log.Debug("index tezos token owners",
		zap.String("blockchain", utils.TezosBlockchain),
		zap.String("contract", contract), zap.String("tokenID", tokenID))

	var lastTime time.Time
	var querLimit = 50
	ownerBalances := []OwnerBalance{}
	for {
		owners, err := e.tzkt.GetTokenOwners(contract, tokenID, querLimit, lastTime)
		if err != nil {
			return nil, err
		}

		ownersLen := len(owners)

		for i, o := range owners {
			//ignore index token owners if total supply > 100.000
			if o.TotalSupply > 100000 {
				return []OwnerBalance{}, nil
			}

			ownerBalances = append(ownerBalances, OwnerBalance{
				Address:  o.Address,
				Balance:  o.Balance,
				LastTime: o.LastTime,
			})

			if i == ownersLen-1 {
				lastTime = o.LastTime
			}
		}

		if ownersLen < querLimit {
			break
		}
	}

	return ownerBalances, nil
}

func (e *IndexEngine) GetObjktToken(ctx context.Context, contract, tokenID string) (objkt.Token, error) {
	return e.objkt.GetObjectToken(ctx, contract, tokenID)
}

// GetTezosTxTimestamp returns the timestamp of an transaction if it exists
func (e *IndexEngine) GetTezosTxTimestamp(_ context.Context, txHashString string) (time.Time, error) {
	detailedTransactions, err := e.tzkt.GetTransactionByTx(txHashString)
	if err != nil {
		return time.Time{}, err
	}

	if len(detailedTransactions) == 0 {
		return time.Time{}, ErrTXNotFound
	}

	return detailedTransactions[0].Timestamp, nil
}

// GetTzktTransactionByID get tzkt transaction by transaction id
func (e *IndexEngine) GetTzktTransactionByID(id uint64) (tzkt.Transaction, error) {
	return e.tzkt.GetTransaction(id)
}

// GetTzktTransactionByID get tzkt transactions by transaction id
func (e *IndexEngine) GetTzktTransactionByContractsAndEntrypoint(contracts []string, entrypoints []string, lastTime *time.Time, offset, limit int) ([]tzkt.Transaction, error) {
	return e.tzkt.GetTransactions(contracts, entrypoints, lastTime, offset, limit)
}

// GetTzktTransactionByHash get tzkt transactions by transaction hash
func (e *IndexEngine) GetTzktTransactionsByHash(hash string) ([]tzkt.DetailedTransaction, error) {
	return e.tzkt.GetTransactionByTx(hash)
}
