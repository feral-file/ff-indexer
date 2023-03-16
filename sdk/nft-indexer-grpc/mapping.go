package sdk

import (
	"time"

	"go.uber.org/zap"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/log"
	grpcIndexer "github.com/bitmark-inc/nft-indexer/services/nft-indexer-grpc/grpc/indexer"
)

const timeLayout = "2006-01-02T15:04:05Z07:00"

// MapGRPCTokenInforToIndexerTokenInfor maps grpc token info to indexer token info
func MapGRPCTokenInforToIndexerTokenInfor(token []*grpcIndexer.BaseTokenInfo) []indexer.BaseTokenInfo {
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

func ParseTime(timeString string) (time.Time, error) {
	timestamp, err := time.Parse(timeLayout, timeString)
	if err != nil {
		return time.Time{}, err
	}

	return timestamp, nil
}

// MapGRPCProvenancesToIndexerProvenances maps grpc provenance to indexer provenance
func MapGRPCProvenancesToIndexerProvenances(provenance []*grpcIndexer.Provenance) []indexer.Provenance {
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
func MapGrpcTokenToIndexerToken(tokenBuffer grpcIndexer.Token) indexer.Token {
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

	return indexer.Token{
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
		OriginTokenInfo: MapGRPCTokenInforToIndexerTokenInfor(tokenBuffer.OriginTokenInfo),
		IsDemo:          tokenBuffer.IsDemo,

		IndexID:           tokenBuffer.IndexID,
		Source:            tokenBuffer.Source,
		Swapped:           tokenBuffer.Swapped,
		SwappedFrom:       &tokenBuffer.SwappedFrom,
		SwappedTo:         &tokenBuffer.SwappedTo,
		Burned:            tokenBuffer.Burned,
		Provenances:       MapGRPCProvenancesToIndexerProvenances(tokenBuffer.Provenances),
		LastActivityTime:  lastActivityTime,
		LastRefreshedTime: lastRefreshedTime,
	}
}

func MapIndexerTokenToGrpcToken(token *indexer.Token) *grpcIndexer.Token {
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
		OriginTokenInfo:   MapIndexerTokenInforToGRPCTokenInfor(token.OriginTokenInfo),
		IsDemo:            token.IsDemo,
		IndexID:           token.IndexID,
		Source:            token.Source,
		Swapped:           token.Swapped,
		SwappedFrom:       DerefString(token.SwappedFrom),
		SwappedTo:         DerefString(token.SwappedTo),
		Burned:            token.Burned,
		Provenances:       MapIndexerProvenancesToGRPCProvenances(token.Provenances),
		LastActivityTime:  token.LastActivityTime.String(),
		LastRefreshedTime: token.LastRefreshedTime.String(),
	}
}

// MapIndexerTokenInforToGRPCTokenInfor maps indexer token info to grpc token info
func MapIndexerTokenInforToGRPCTokenInfor(token []indexer.BaseTokenInfo) []*grpcIndexer.BaseTokenInfo {
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

func DerefString(s *string) string {
	if s != nil {
		return *s
	}

	return ""
}

// MapIndexerProvenancesToGRPCProvenances maps indexer provenance to grpc provenance
func MapIndexerProvenancesToGRPCProvenances(provenance []indexer.Provenance) []*grpcIndexer.Provenance {
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
