package sdk

import (
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/log"
	grpcIndexer "github.com/bitmark-inc/nft-indexer/services/nft-indexer-grpc/grpc/indexer"
)

const timeLayout = "2006-01-02T15:04:05Z07:00"

type Mapper struct{}

type GRPCIndexerMapper interface {
	MapGRPCTokenInforToIndexerTokenInfor(token []*grpcIndexer.BaseTokenInfo) []indexer.BaseTokenInfo
	MapGRPCProvenancesToIndexerProvenances(provenance []*grpcIndexer.Provenance) []indexer.Provenance
	MapGrpcTokenToIndexerToken(tokenBuffer *grpcIndexer.Token) *indexer.Token
	MapIndexerTokenToGrpcToken(token *indexer.Token) *grpcIndexer.Token
	MapIndexerTokenInforToGRPCTokenInfor(token []indexer.BaseTokenInfo) []*grpcIndexer.BaseTokenInfo

	MapIndexerProvenancesToGRPCProvenances(provenance []indexer.Provenance) []*grpcIndexer.Provenance
	MapGRPCAccountTokensToIndexerAccountTokens(accountTokens []*grpcIndexer.AccountToken) ([]indexer.AccountToken, error)
	MapIndexerAttributesToGRPCAttributes(attributes *indexer.AssetAttributes) *grpcIndexer.AssetAttributes
	MapIndexerProjectMetadataToGRPCProjectMetadata(projectMetadata *indexer.ProjectMetadata) *grpcIndexer.ProjectMetadata
	MapIndexerDetailedTokenToGRPCDetailedToken(token indexer.DetailedToken) *grpcIndexer.DetailedToken
}

// DerefString de-reference string
func DerefString(s *string) string {
	if s != nil {
		return *s
	}

	return ""
}

// ParseTime parses time string to time.Time
func ParseTime(timeString string) (time.Time, error) {
	timestamp, err := time.Parse(timeLayout, timeString)
	if err != nil {
		return time.Time{}, err
	}

	return timestamp, nil
}

// MapGRPCTokenInforToIndexerTokenInfor maps grpc token info to indexer token info
func (m *Mapper) MapGRPCTokenInforToIndexerTokenInfor(token []*grpcIndexer.BaseTokenInfo) []indexer.BaseTokenInfo {
	baseTokenInfors := make([]indexer.BaseTokenInfo, len(token))

	for i, v := range token {
		baseTokenInfors[i] = indexer.BaseTokenInfo{
			ID:              v.ID,
			Blockchain:      v.Blockchain,
			Fungible:        v.Fungible,
			ContractType:    v.ContractType,
			ContractAddress: v.ContractAddress,
		}
	}

	return baseTokenInfors
}

// MapGRPCProvenancesToIndexerProvenances maps grpc provenance to indexer provenance
func (m *Mapper) MapGRPCProvenancesToIndexerProvenances(provenance []*grpcIndexer.Provenance) []indexer.Provenance {
	provenances := make([]indexer.Provenance, len(provenance))

	for i, v := range provenance {
		timestamp, err := ParseTime(v.Timestamp)
		if err != nil {
			log.Error("fail when parse provenance timestamp time", zap.Error(err))
		}

		provenances[i] = indexer.Provenance{
			FormerOwner: &v.FormerOwner,
			Type:        v.Type,
			Owner:       v.Owner,
			Blockchain:  v.Blockchain,
			Timestamp:   timestamp,
			TxID:        v.Timestamp,
			TxURL:       v.TxURL,
		}
	}

	return provenances
}

// MapGrpcTokenToIndexerToken maps grpc indexer token to indexer token
func (m *Mapper) MapGrpcTokenToIndexerToken(tokenBuffer *grpcIndexer.Token) *indexer.Token {
	mintAt, err := ParseTime(tokenBuffer.MintAt)
	if err != nil {
		log.Error("fail when parse mintAt time", zap.Error(err))
	}

	lastActivityTime, err := ParseTime(tokenBuffer.LastRefreshedTime)
	if err != nil {
		log.Error("fail when parse lastActivityTime", zap.Error(err))
	}

	lastRefreshedTime, err := ParseTime(tokenBuffer.LastRefreshedTime)
	if err != nil {
		log.Error("fail when parse lastRefreshedTime", zap.Error(err))
	}

	return &indexer.Token{
		BaseTokenInfo: indexer.BaseTokenInfo{
			ID:              tokenBuffer.ID,
			Blockchain:      tokenBuffer.Blockchain,
			Fungible:        tokenBuffer.Fungible,
			ContractType:    tokenBuffer.ContractType,
			ContractAddress: tokenBuffer.ContractAddress,
		},
		Edition:         tokenBuffer.Edition,
		EditionName:     tokenBuffer.EditionName,
		MintAt:          mintAt,
		Balance:         tokenBuffer.Balance,
		Owner:           tokenBuffer.Owner,
		Owners:          tokenBuffer.Owners,
		OwnersArray:     tokenBuffer.OwnersArray,
		AssetID:         tokenBuffer.AssetID,
		OriginTokenInfo: m.MapGRPCTokenInforToIndexerTokenInfor(tokenBuffer.OriginTokenInfo),
		IsDemo:          tokenBuffer.IsDemo,

		IndexID:           tokenBuffer.IndexID,
		Source:            tokenBuffer.Source,
		Swapped:           tokenBuffer.Swapped,
		SwappedFrom:       &tokenBuffer.SwappedFrom,
		SwappedTo:         &tokenBuffer.SwappedTo,
		Burned:            tokenBuffer.Burned,
		Provenances:       m.MapGRPCProvenancesToIndexerProvenances(tokenBuffer.Provenances),
		LastActivityTime:  lastActivityTime,
		LastRefreshedTime: lastRefreshedTime,
	}
}

func (m *Mapper) MapIndexerTokenToGrpcToken(token *indexer.Token) *grpcIndexer.Token {
	return &grpcIndexer.Token{
		ID:                token.ID,
		Blockchain:        token.Blockchain,
		Fungible:          token.Fungible,
		ContractType:      token.ContractType,
		ContractAddress:   token.ContractAddress,
		Edition:           token.Edition,
		EditionName:       token.EditionName,
		MintAt:            token.MintAt.String(),
		Balance:           token.Balance,
		Owner:             token.Owner,
		Owners:            token.Owners,
		OwnersArray:       token.OwnersArray,
		AssetID:           token.AssetID,
		OriginTokenInfo:   m.MapIndexerTokenInforToGRPCTokenInfor(token.OriginTokenInfo),
		IsDemo:            token.IsDemo,
		IndexID:           token.IndexID,
		Source:            token.Source,
		Swapped:           token.Swapped,
		SwappedFrom:       DerefString(token.SwappedFrom),
		SwappedTo:         DerefString(token.SwappedTo),
		Burned:            token.Burned,
		Provenances:       m.MapIndexerProvenancesToGRPCProvenances(token.Provenances),
		LastActivityTime:  token.LastActivityTime.String(),
		LastRefreshedTime: token.LastRefreshedTime.String(),
	}
}

// MapIndexerTokenInforToGRPCTokenInfor maps indexer token info to grpc token info
func (m *Mapper) MapIndexerTokenInforToGRPCTokenInfor(token []indexer.BaseTokenInfo) []*grpcIndexer.BaseTokenInfo {
	GRPCBaseTokenInfors := make([]*grpcIndexer.BaseTokenInfo, len(token))

	for i, v := range token {
		GRPCBaseTokenInfors[i] = &grpcIndexer.BaseTokenInfo{
			ID:              v.ID,
			Blockchain:      v.Blockchain,
			Fungible:        v.Fungible,
			ContractType:    v.ContractType,
			ContractAddress: v.ContractAddress,
		}
	}

	return GRPCBaseTokenInfors
}

// MapIndexerProvenancesToGRPCProvenances maps indexer provenance to grpc provenance
func (m *Mapper) MapIndexerProvenancesToGRPCProvenances(provenance []indexer.Provenance) []*grpcIndexer.Provenance {
	GRPCProvenances := make([]*grpcIndexer.Provenance, len(provenance))

	for i, v := range provenance {
		GRPCProvenances[i] = &grpcIndexer.Provenance{
			FormerOwner: DerefString(v.FormerOwner),
			Type:        v.Type,
			Owner:       v.Owner,
			Blockchain:  v.Blockchain,
			Timestamp:   v.Timestamp.String(),
			TxID:        v.Timestamp.String(),
			TxURL:       v.TxURL,
		}
	}

	return GRPCProvenances
}

func (m *Mapper) MapGRPCAccountTokensToIndexerAccountTokens(accountTokens []*grpcIndexer.AccountToken) ([]indexer.AccountToken, error) {
	accountTokensIndexer := make([]indexer.AccountToken, len(accountTokens))

	for i, v := range accountTokens {
		lastActivityTime, err := ParseTime(v.LastActivityTime)
		if err != nil {
			return nil, err
		}

		lastRefreshedTime, err := ParseTime(v.LastRefreshedTime)
		if err != nil {
			return nil, err
		}

		lastPendingTime := make([]time.Time, len(v.LastPendingTime))

		lastUpdatedAt, err := ParseTime(v.LastUpdatedAt)
		if err != nil {
			return nil, err
		}

		for _, k := range v.LastPendingTime {
			t, err := ParseTime(k)
			if err != nil {
				return nil, err
			}

			lastPendingTime = append(lastPendingTime, t)
		}

		accountTokensIndexer[i] = indexer.AccountToken{
			BaseTokenInfo: indexer.BaseTokenInfo{
				ID:              v.ID,
				Blockchain:      v.Blockchain,
				Fungible:        v.Fungible,
				ContractType:    v.ContractType,
				ContractAddress: v.ContractAddress,
			},
			IndexID:           v.IndexID,
			OwnerAccount:      v.OwnerAccount,
			Balance:           v.Balance,
			LastActivityTime:  lastActivityTime,
			LastRefreshedTime: lastRefreshedTime,
			LastPendingTime:   lastPendingTime,
			LastUpdatedAt:     lastUpdatedAt,
			PendingTxs:        v.PendingTxs,
		}
	}

	return accountTokensIndexer, nil
}

// MapIndexerAttributesToGRPCAttributes maps indexer attributes to grpc attributes
func (m *Mapper) MapIndexerAttributesToGRPCAttributes(attributes *indexer.AssetAttributes) *grpcIndexer.AssetAttributes {
	var GRPCattributes *grpcIndexer.AssetAttributes

	if attributes == nil {
		GRPCattributes = nil
	} else {
		GRPCattributes = &grpcIndexer.AssetAttributes{Scrollable: attributes.Scrollable}
	}

	return GRPCattributes
}

// MapIndexerProjectMetadataToGRPCProjectMetadata maps indexer project metadata to grpc project metadata
func (m *Mapper) MapIndexerProjectMetadataToGRPCProjectMetadata(projectMetadata *indexer.ProjectMetadata) *grpcIndexer.ProjectMetadata {
	attributes := m.MapIndexerAttributesToGRPCAttributes(projectMetadata.Attributes)

	return &grpcIndexer.ProjectMetadata{
		ArtistID:            projectMetadata.ArtistID,
		ArtistName:          projectMetadata.ArtistName,
		ArtistURL:           projectMetadata.ArtistURL,
		AssetID:             projectMetadata.AssetID,
		Title:               projectMetadata.Title,
		Description:         projectMetadata.Description,
		MIMEType:            projectMetadata.MIMEType,
		Medium:              string(projectMetadata.Medium),
		MaxEdition:          projectMetadata.MaxEdition,
		BaseCurrency:        projectMetadata.BaseCurrency,
		BasePrice:           projectMetadata.BasePrice,
		Source:              projectMetadata.Source,
		SourceURL:           projectMetadata.SourceURL,
		PreviewURL:          projectMetadata.PreviewURL,
		ThumbnailURL:        projectMetadata.ThumbnailURL,
		GalleryThumbnailURL: projectMetadata.GalleryThumbnailURL,
		AssetData:           projectMetadata.AssetData,
		AssetURL:            projectMetadata.AssetURL,

		Attributes: attributes,
		// FIXME: convert ArtworkMetadata to protobuf
		ArtworkMetadata:  map[string]*anypb.Any{},
		LastUpdatedAt:    projectMetadata.LastUpdatedAt.String(),
		InitialSaleModel: projectMetadata.InitialSaleModel,
		OriginalFileURL:  projectMetadata.OriginalFileURL,
	}
}

// MapIndexerDetailedTokenToGRPCDetailedToken maps indexer detailed token to grpc detailed token
func (m *Mapper) MapIndexerDetailedTokenToGRPCDetailedToken(token indexer.DetailedToken) *grpcIndexer.DetailedToken {
	origin := m.MapIndexerProjectMetadataToGRPCProjectMetadata(&token.ProjectMetadata.Origin)
	latest := m.MapIndexerProjectMetadataToGRPCProjectMetadata(&token.ProjectMetadata.Latest)
	attributes := m.MapIndexerAttributesToGRPCAttributes(token.Attributes)

	return &grpcIndexer.DetailedToken{
		Token:       m.MapIndexerTokenToGrpcToken(&token.Token),
		ThumbnailID: token.ThumbnailID,
		IPFSPinned:  token.IPFSPinned,
		Attributes:  attributes,
		ProjectMetadata: &grpcIndexer.VersionedProjectMetadata{
			Origin: origin,
			Latest: latest,
		},
	}
}
