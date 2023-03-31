package indexer

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/sha3"

	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
	"github.com/bitmark-inc/nft-indexer/log"
)

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

func (e *IndexEngine) GetTezosTokenByOwner(ctx context.Context, owner string, lastTime time.Time, offset int) ([]tzkt.OwnedToken, error) {
	tokens, err := e.tzkt.RetrieveTokens(owner, lastTime, offset)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// IndexTezosTokenByOwner indexes all tokens owned by a specific tezos address
func (e *IndexEngine) IndexTezosTokenByOwner(ctx context.Context, owner string, lastTime time.Time, offset int) ([]AssetUpdates, time.Time, error) {
	var newLastTime = time.Time{}
	ownedTokens, err := e.GetTezosTokenByOwner(ctx, owner, lastTime, offset)
	if err != nil {
		return nil, newLastTime, err
	}

	log.Debug("retrieve tokens for owner", zap.Any("tokens", ownedTokens), zap.String("owner", owner))

	tokenUpdates := make([]AssetUpdates, 0, len(ownedTokens))

	for _, t := range ownedTokens {

		update, err := e.indexTezosToken(ctx, t.Token, owner, int64(t.Balance), t.LastTime)
		if err != nil {
			log.Error("fail to index a tezos token", zap.Error(err))
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
func (e *IndexEngine) IndexTezosToken(ctx context.Context, owner, contract, tokenID string) (*AssetUpdates, error) {
	tzktToken, err := e.tzkt.GetContractToken(contract, tokenID)
	if err != nil {
		log.Debug("GetContractToken",
			log.SourceTZKT,
			zap.Error(err),
			zap.String("contract", contract), zap.String("tokenID", tokenID))
		return nil, err
	}

	balance, lastTime, err := e.tzkt.GetTokenBalanceAndLastTimeForOwner(contract, tokenID, owner)
	if err != nil {
		log.Debug("GetTokenBalanceAndLastTimeForOwner",
			log.SourceTZKT,
			zap.Error(err),
			zap.String("contract", contract), zap.String("tokenID", tokenID))
		return nil, err
	}

	return e.indexTezosToken(ctx, tzktToken, owner, balance, lastTime)
}

// indexTezosToken prepares indexing data for a tezos token using the
// source API token object. It currently uses token objects from tzkt api
func (e *IndexEngine) indexTezosToken(ctx context.Context, tzktToken tzkt.Token, owner string, balance int64, lastActivityTime time.Time) (*AssetUpdates, error) {
	log.Debug("index tezos token", zap.Any("token", tzktToken))

	assetIDBytes := sha3.Sum256([]byte(fmt.Sprintf("%s-%s", tzktToken.Contract.Address, tzktToken.ID.String())))
	assetID := hex.EncodeToString(assetIDBytes[:])

	metadataDetail := NewAssetMetadataDetail(assetID)
	metadataDetail.FromTZKT(tzktToken)

	tokenDetail := TokenDetail{
		MintedAt: tzktToken.Timestamp,
	}

	if e.environment != DevelopmentEnvironment {
		switch tzktToken.Contract.Address {
		case KALAMContractAddress, TezDaoContractAddress, TezosDNSContractAddress:
			return nil, nil

		case FXHASHV2ContractAddress, FXHASHContractAddress, FXHASHOldContractAddress:
			metadataDetail.SetMarketplace(
				MarketplaceProfile{
					"fxhash",
					"https://www.fxhash.xyz",
					fmt.Sprintf("https://www.fxhash.xyz/gentk/%s", tzktToken.ID.String()),
				},
			)
			metadataDetail.SetMedium(MediumSoftware)

			if detail, err := e.fxhash.GetObjectDetail(ctx, tzktToken.ID.Int); err != nil {
				log.Error("fail to get token detail from fxhash", zap.Error(err), log.SourceFXHASH)
			} else {
				metadataDetail.FromFxhashObject(detail)
				tokenDetail.MintedAt = detail.CreatedAt
				tokenDetail.Edition = detail.Iteration
			}
		case VersumContractAddress:
			tokenDetail.Fungible = true
			metadataDetail.SetMarketplace(MarketplaceProfile{"versum", "https://versum.xyz",
				fmt.Sprintf("https://versum.xyz/token/versum/%s", tzktToken.ID.String())},
			)

			metadataDetail.ArtistURL = fmt.Sprintf("https://versum.xyz/user/%s", metadataDetail.ArtistName)

		default:
			// fallback marketplace
			tokenDetail.Fungible = true
			objktToken, err := e.GetObjktToken(tzktToken.Contract.Address, tzktToken.ID.String())
			if err != nil {
				log.Error("fail to get token detail from objkt", zap.Error(err), log.SourceObjkt)
			} else {
				metadataDetail.FromObjkt(objktToken)
			}

			assetURL := fmt.Sprintf("https://objkt.com/asset/%s/%s", tzktToken.Contract.Address, tzktToken.ID.String())
			switch tzktToken.Metadata.Symbol {
			case "OBJKTCOM":
				metadataDetail.SetMarketplace(MarketplaceProfile{"objkt", "https://objkt.com", assetURL})
			case "OBJKT":
				metadataDetail.SetMarketplace(MarketplaceProfile{"hic et nunc", "https://objkt.com", assetURL})
			default:
				metadataDetail.SetMarketplace(MarketplaceProfile{"unknown", "https://objkt.com", assetURL})
			}
		}
	}

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
		MaxEdition: metadataDetail.MaxEdition,

		PreviewURL:          metadataDetail.PreviewURI,
		ThumbnailURL:        metadataDetail.DisplayURI,
		GalleryThumbnailURL: metadataDetail.DisplayURI,

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
					Blockchain:      TezosBlockchain,
					Fungible:        tokenDetail.Fungible,
					ContractType:    tzktToken.Standard,
					ContractAddress: tzktToken.Contract.Address,
				},
				IndexID:           TokenIndexID(TezosBlockchain, tzktToken.Contract.Address, tzktToken.ID.String()),
				Owner:             owner,
				Balance:           balance,
				Owners:            map[string]int64{owner: balance},
				Edition:           tokenDetail.Edition,
				MintAt:            tokenDetail.MintedAt,
				LastRefreshedTime: time.Now(),
				LastActivityTime:  lastActivityTime,
			},
		},
	}

	log.Debug("asset updating data prepared",
		zap.String("blockchain", TezosBlockchain),
		zap.String("id", TokenIndexID(TezosBlockchain, tzktToken.Contract.Address, tzktToken.ID.String())),
		zap.Any("tokenUpdate", tokenUpdate))

	return &tokenUpdate, nil
}

// IndexTezosTokenProvenance indexes provenance of a specific token
func (e *IndexEngine) IndexTezosTokenProvenance(ctx context.Context, contract, tokenID string) ([]Provenance, error) {
	log.Debug("index tezos token provenance",
		zap.String("blockchain", TezosBlockchain),
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
			log.Error("fail to get transaction",
				log.SourceTZKT,
				zap.String("blockchain", TezosBlockchain),
				zap.Uint64("txID", t.TransactionID),
				zap.Any("transfer", t),
			)
			return nil, err
		}

		txType := "transfer"
		if t.From == nil {
			txType = "mint"
		}

		provenances = append(provenances, Provenance{
			Type:        txType,
			Owner:       t.To.Address,
			Blockchain:  TezosBlockchain,
			BlockNumber: &t.Level,
			Timestamp:   t.Timestamp,
			TxID:        tx.Hash,
			TxURL:       fmt.Sprintf("https://tzkt.io/%s", tx.Hash),
		})
	}

	return provenances, nil
}

// IndexTezosTokenLastActivityTime indexes the last activity timestamp of a given token
func (e *IndexEngine) IndexTezosTokenLastActivityTime(ctx context.Context, contract, tokenID string) (time.Time, error) {
	return e.tzkt.GetTokenLastActivityTime(contract, tokenID)
}

// IndexTezosTokenOwners indexes owners of a given token
func (e *IndexEngine) IndexTezosTokenOwners(ctx context.Context, contract, tokenID string) (map[string]int64, error) {
	log.Debug("index tezos token owners",
		zap.String("blockchain", TezosBlockchain),
		zap.String("contract", contract), zap.String("tokenID", tokenID))

	var lastTime time.Time
	var querLimit = 50
	ownersMap := map[string]int64{}
	for {
		owners, err := e.tzkt.GetTokenOwners(contract, tokenID, querLimit, lastTime)
		if err != nil {
			return nil, err
		}

		ownersLen := len(owners)

		for i, o := range owners {
			ownersMap[o.Address] = o.Balance
			if i == ownersLen-1 {
				lastTime = o.LastTime
			}
		}

		if ownersLen < querLimit {
			break
		}
	}

	return ownersMap, nil
}

func (e *IndexEngine) GetObjktToken(contract, tokenID string) (objkt.Token, error) {
	if e.environment == DevelopmentEnvironment {
		return objkt.Token{}, nil
	}

	objktToken, err := e.objkt.GetObjectToken(contract, tokenID)
	if err != nil {
		return objkt.Token{}, err
	}

	return objktToken, nil
}
