package graph

import (
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/cache"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/services/nft-indexer/graph/model"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Resolver struct {
	indexerStore  indexer.Store
	cacheStore    cache.Store
	ethClient     *ethclient.Client
	cadenceWorker *cadence.WorkerClient
}

func NewResolver(indexerStore indexer.Store, cacheStore cache.Store, ethClient *ethclient.Client, cadenceWorker *cadence.WorkerClient) *Resolver {
	return &Resolver{
		indexerStore:  indexerStore,
		cacheStore:    cacheStore,
		ethClient:     ethClient,
		cadenceWorker: cadenceWorker,
	}
}

func (r *Resolver) mapGraphQLToken(t indexer.DetailedTokenV2) *model.Token {
	provenances := []*model.Provenance{}
	for _, t := range t.Provenances {
		provenances = append(provenances, r.mapGraphQLProvenance(t))
	}

	originalTokenInfo := []*model.BaseTokenInfo{}
	for _, t := range t.OriginTokenInfo {
		originalTokenInfo = append(originalTokenInfo, r.mapGraphQLBaseTokenInfo(t))
	}

	var owners []*model.Owner
	for address, balance := range t.Owners {
		owners = append(owners, &model.Owner{
			Address: address,
			Balance: balance,
		})
	}

	return &model.Token{
		ID:                t.ID,
		Blockchain:        t.Blockchain,
		ContractType:      t.ContractType,
		ContractAddress:   t.ContractAddress,
		IndexID:           t.IndexID,
		Owner:             t.Owner,
		Owners:            owners,
		OriginTokenInfo:   originalTokenInfo,
		Balance:           t.Balance,
		Fungible:          t.Fungible,
		Burned:            t.Burned,
		Edition:           t.Edition,
		EditionName:       t.EditionName,
		Source:            t.Source,
		MintAt:            &t.MintedAt, // FIXME: deprecated this after a month
		MintedAt:          &t.MintedAt,
		Swapped:           t.Swapped,
		Provenance:        provenances,
		LastActivityTime:  &t.LastActivityTime,
		LastRefreshedTime: &t.LastRefreshedTime,
		Asset:             r.mapGraphQLAsset(t.Asset),
	}
}

func (r *Resolver) mapGraphQLAsset(a indexer.AssetV2) *model.Asset {
	var attributes *model.AssetAttributes
	if a.Attributes != nil && a.Attributes.Configuration != nil {
		c := a.Attributes.Configuration
		attributes = &model.AssetAttributes{
			Configuration: &model.AssetConfiguration{
				Orientation:     c.Orientation,
				Scaling:         c.Scaling,
				BackgroundColor: c.BackgroundColor,
				MarginLeft:      c.MarginLeft,
				MarginRight:     c.MarginRight,
				MarginTop:       c.MarginTop,
				MarginBottom:    c.MarginBottom,
				AutoPlay:        c.AutoPlay,
				Looping:         c.Looping,
				Interactable:    c.Interactable,
				Overridable:     c.Overridable,
			},
		}
	}

	return &model.Asset{
		IndexID:           a.IndexID,
		ThumbnailID:       a.ThumbnailID,
		LastRefreshedTime: &a.LastRefreshedTime,
		Attributes:        attributes,
		Metadata: &model.AssetMetadata{
			Project: &model.VersionedProjectMetadata{
				Origin: r.mapGraphQLProjectMetadata(a.Metadata.Project.Origin),
				Latest: r.mapGraphQLProjectMetadata(a.Metadata.Project.Latest),
			},
		},
		StaticPreviewURLLandscape: a.StaticPreviewURLLandscape,
		StaticPreviewURLPortrait:  a.StaticPreviewURLPortrait,
	}
}

func (r *Resolver) mapGraphQLProjectMetadata(p indexer.ProjectMetadata) *model.ProjectMetadata {
	var artists []*model.Artist

	for _, a := range p.Artists {
		artists = append(artists, &model.Artist{
			ID:   a.ID,
			Name: a.Name,
			URL:  a.URL,
		})
	}

	return &model.ProjectMetadata{
		ArtistID:            p.ArtistID,
		ArtistName:          p.ArtistName,
		ArtistURL:           p.ArtistURL,
		Artists:             artists,
		AssetID:             p.AssetID,
		Title:               p.Title,
		Description:         p.Description,
		MimeType:            p.MIMEType,
		Medium:              string(p.Medium),
		MaxEdition:          p.MaxEdition,
		BaseCurrency:        p.BaseCurrency,
		BasePrice:           p.BasePrice,
		Source:              p.Source,
		SourceURL:           p.SourceURL,
		PreviewURL:          p.PreviewURL,
		ThumbnailURL:        p.ThumbnailURL,
		GalleryThumbnailURL: p.GalleryThumbnailURL,
		AssetData:           p.AssetData,
		AssetURL:            p.AssetURL,
		ArtworkMetadata:     p.ArtworkMetadata,
	}
}

func (r *Resolver) mapGraphQLProvenance(p indexer.Provenance) *model.Provenance {
	var b int64
	if p.BlockNumber != nil {
		b = int64(*p.BlockNumber)
	}

	return &model.Provenance{
		Type:        p.Type,
		Owner:       p.Owner,
		Blockchain:  p.Blockchain,
		BlockNumber: &b,
		Timestamp:   &p.Timestamp,
		TxID:        p.TxID,
		TxURL:       p.TxURL,
	}
}

func (r *Resolver) mapGraphQLIdentity(a indexer.AccountIdentity) *model.Identity {
	return &model.Identity{
		AccountNumber: a.AccountNumber,
		Blockchain:    a.Blockchain,
		Name:          a.Name,
	}
}

func (r *Resolver) mapGraphQLBaseTokenInfo(t indexer.BaseTokenInfo) *model.BaseTokenInfo {
	return &model.BaseTokenInfo{
		ID:              t.ID,
		Blockchain:      t.Blockchain,
		Fungible:        t.Fungible,
		ContractType:    t.ContractType,
		ContractAddress: t.ContractAddress,
	}
}

func (r *Resolver) mapGraphQLCollection(c indexer.Collection) *model.Collection {
	return &model.Collection{
		ID:          c.ID,
		ExternalID:  c.ExternalID,
		Creators:    c.Creators,
		Name:        c.Name,
		Description: c.Description,
		Items:       int64(c.Items),
		ImageURL:    c.ImageURL,
		Contracts: &model.ContractAddresses{
			Ethereum: &model.EthereumContractAddresses{
				Erc721:  c.Contracts.Ethereum.ERC721,
				Erc1155: c.Contracts.Ethereum.ERC1155,
			},
			Tezos: &model.TezosContractAddresses{
				Fa2: c.Contracts.Tezos.FA2,
			},
		},
		Source:          c.Source,
		Published:       c.Published,
		ExternalURL:     c.ExternalURL,
		Metadata:        c.Metadata,
		LastUpdatedTime: &c.LastUpdatedTime,
		CreatedAt:       &c.CreatedAt,
	}
}
