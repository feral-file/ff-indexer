package indexer

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/url"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"

	"github.com/bitmark-inc/nft-indexer/externals/bettercall"
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

// IndexTezosTokenByOwner indexes all tokens owned by a specific tezos address
func (e *IndexEngine) IndexTezosTokenByOwner(ctx context.Context, owner string, offset int) ([]AssetUpdates, error) {
	tokens, err := e.bettercall.RetrieveTokens(owner, offset)
	if err != nil {
		return nil, err
	}

	tokenUpdates := make([]AssetUpdates, 0, len(tokens))

	for _, t := range tokens {
		update, err := e.indexTezosToken(ctx, owner, t)
		if err != nil {
			log.WithError(err).Error("fail to index a tezos token")
			continue
		}

		if update != nil {
			tokenUpdates = append(tokenUpdates, *update)
		}
	}

	return tokenUpdates, nil
}

// IndexTezosToken indexes a Tezos token with a specific contract and ID
func (e *IndexEngine) IndexTezosToken(ctx context.Context, owner, contract, tokenID string) (*AssetUpdates, error) {
	t, err := e.bettercall.GetContractToken(contract, tokenID)
	if err != nil {
		return nil, err
	}

	return e.indexTezosToken(ctx, "", t)
}

// indexTezosToken prepares indexing data for a tezos token using the
// source API token object. It currently uses token objects from better-call.dev
func (e *IndexEngine) indexTezosToken(ctx context.Context, owner string, t bettercall.Token) (*AssetUpdates, error) {
	log.WithField("token", t).Debug("index tezos token")

	// FIXME: can be merged by tzkt api
	m, err := e.bettercall.GetTokenMetadata(t.Contract, t.ID.String())
	if err != nil {
		log.WithError(err).Error("can not index token: fail to get metadata for the token")
		return nil, err
	}

	assetIDBytes := sha3.Sum256([]byte(fmt.Sprintf("%s-%s", t.Contract, t.ID.String())))
	assetID := hex.EncodeToString(assetIDBytes[:])

	metadataDetail := NewAssetMetadataDetail(assetID)
	metadataDetail.FromBetterCallDev(t, m)

	tokenDetail := TokenDetail{
		MintedAt: m.Timestamp,
	}

	switch t.Contract {
	case KALAMContractAddress, TezosDNSContractAddress:
		return nil, nil

	case FXHASHV2ContractAddress, FXHASHContractAddress, FXHASHOldContractAddress:
		metadataDetail.SetMarketplace(
			MarketplaceProfile{
				"fxhash",
				"https://www.fxhash.xyz",
				fmt.Sprintf("https://www.fxhash.xyz/gentk/%s", t.ID.String()),
			},
		)
		metadataDetail.SetMedium(MediumSoftware)

		if detail, err := e.fxhash.GetObjectDetail(ctx, t.ID.Int); err != nil {
			log.WithError(err).Error("fail to get token detail from fxhash")
		} else {
			metadataDetail.FromFxhashObject(detail)
			tokenDetail.MintedAt = detail.CreatedAt
			tokenDetail.Edition = detail.Iteration
		}
	case VersumContractAddress:
		metadataDetail.SetMarketplace(MarketplaceProfile{"versum", "https://versum.xyz",
			fmt.Sprintf("https://versum.xyz/token/versum/%s", t.ID.String())},
		)

		metadataDetail.ArtistURL = fmt.Sprintf("https://versum.xyz/user/%s", metadataDetail.ArtistName)

	case HicEtNuncContractAddress:
		// hicetnunc is down. We now fallback the source url and asset url to objkt.com
		metadataDetail.SetMarketplace(MarketplaceProfile{"hic et nunc", "https://objkt.com",
			fmt.Sprintf("https://objkt.com/asset/%s/%s", t.Contract, t.ID.String())},
		)
	default:
		// fallback marketplace
		metadataDetail.SetMarketplace(MarketplaceProfile{"unknown", "https://objkt.com",
			fmt.Sprintf("https://objkt.com/asset/%s/%s", t.Contract, t.ID.String())},
		)

		if detail, err := e.objkt.GetObjktDetailed(ctx, t.ID.Text(10), t.Contract); err != nil {
			log.WithError(err).Error("fail to get token detail from objkt")
		} else {
			metadataDetail.SetMarketplace(MarketplaceProfile{"objkt", "https://objkt.com",
				fmt.Sprintf("https://objkt.com/asset/%s/%s", t.Contract, t.ID.String())},
			)
			metadataDetail.FromObjktObject(detail)
			tokenDetail.MintedAt = detail.MintedAt
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

		ArtistName: metadataDetail.ArtistName,
		ArtistURL:  metadataDetail.ArtistURL,
		MaxEdition: metadataDetail.MaxEdition,

		PreviewURL:          metadataDetail.PreviewURI,
		ThumbnailURL:        metadataDetail.DisplayURI,
		GalleryThumbnailURL: metadataDetail.DisplayURI,
	}

	tokenUpdate := AssetUpdates{
		ID:              assetID,
		Source:          "bcdhub", // asset data source which is different than project source
		ProjectMetadata: pm,
		Tokens: []Token{
			{
				BaseTokenInfo: BaseTokenInfo{
					ID:              t.ID.String(),
					Blockchain:      TezosBlockchain,
					ContractType:    "fa2",
					ContractAddress: t.Contract,
				},
				IndexID: TokenIndexID(TezosBlockchain, t.Contract, t.ID.String()),
				Owner:   owner,
				Edition: tokenDetail.Edition,
				MintAt:  tokenDetail.MintedAt,
			},
		},
	}

	log.WithField("blockchain", TezosBlockchain).
		WithField("id", TokenIndexID(TezosBlockchain, t.Contract, t.ID.String())).
		WithField("tokenUpdate", tokenUpdate).
		Trace("asset updating data prepared")

	return &tokenUpdate, nil
}
