package graph

import (
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/services/nft-indexer/graph/model"
)

type Resolver struct {
	indexerStore indexer.Store
}

func NewResolver(indexerStore indexer.Store) *Resolver {
	return &Resolver{
		indexerStore: indexerStore,
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
	if t.Attributes != nil {
		attributes = model.AssetAttributes{Scrollable: t.Attributes.Scrollable}
	}

	return &model.Token{
		ID:                t.ID,
		Blockchain:        t.Blockchain,
		ContractType:      t.ContractType,
		ContractAddress:   t.ContractAddress,
		IndexID:           t.IndexID,
		Owner:             t.Owner,
		OriginTokenInfo:   originalTokenInfo,
		Balance:           t.Balance,
		Fungible:          t.Fungible,
		Burned:            t.Burned,
		Edition:           t.Edition,
		EditionName:       t.EditionName,
		Source:            t.Source,
		MintAt:            &t.MintAt,
		Swapped:           t.Swapped,
		Provenance:        provenances,
		Attributes:        &attributes,
		LastActivityTime:  &t.LastActivityTime,
		LastRefreshedTime: &t.LastRefreshedTime,
		Asset: &model.Asset{
			IndexID:           t.Asset.IndexID,
			ThumbnailID:       t.Asset.ThumbnailID,
			LastRefreshedTime: &t.Asset.LastRefreshedTime,
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
	return &model.ProjectMetadata{
		ArtistID:            p.ArtistID,
		ArtistName:          p.ArtistName,
		ArtistURL:           p.ArtistURL,
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
	}
}

func (r *Resolver) mapGraphQLProvenance(p indexer.Provenance) *model.Provenance {
	var b int64
	if p.BlockNumber != nil {
		b = int64(*p.BlockNumber)
	}

	return &model.Provenance{
		Type:        p.Type,
		Owner:       p.Type,
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
