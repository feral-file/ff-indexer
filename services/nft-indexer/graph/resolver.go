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

	var attributes model.AssetAttributes
	if t.Asset.Attributes != nil {
		attributes = model.AssetAttributes{Scrollable: t.Asset.Attributes.Scrollable}
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
		Asset: &model.Asset{
			IndexID:           t.Asset.IndexID,
			ThumbnailID:       t.Asset.ThumbnailID,
			LastRefreshedTime: &t.Asset.LastRefreshedTime,
			Attributes:        &attributes,
			Metadata: &model.AssetMetadata{
				Project: &model.VersionedProjectMetadata{
					Origin: r.mapGraphQLProjectMetadata(t.Asset.Metadata.Project.Origin),
					Latest: r.mapGraphQLProjectMetadata(t.Asset.Metadata.Project.Latest),
				},
			},
		},
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
		ID:               c.ID,
		ExternalID:       c.ExternalID,
		Blockchain:       c.Blockchain,
		Owner:            c.Owner,
		Name:             c.Name,
		Description:      c.Description,
		ImageURL:         c.ImageURL,
		Items:            int64(c.Items),
		Source:           c.Source,
		Published:        c.Published,
		SourceURL:        c.SourceURL,
		LastActivityTime: &c.LastActivityTime,
		LastUpdatedTime:  &c.LastUpdatedTime,
	}
}
