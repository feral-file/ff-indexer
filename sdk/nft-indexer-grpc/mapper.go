package sdk

import (
	"fmt"
	"strconv"
	"time"

	"encoding/json"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	log "github.com/bitmark-inc/autonomy-logger"
	indexer "github.com/bitmark-inc/nft-indexer"
	grpcIndexer "github.com/bitmark-inc/nft-indexer/services/nft-indexer-grpc/grpc/indexer"
)

type Mapper struct{}

// DerefString de-reference string
func DerefString(s *string) string {
	if s != nil {
		return *s
	}

	return ""
}

// ParseTime parses time string to time.Time
func ParseTime(timeString string) (time.Time, error) {
	timestamp, err := time.Parse(time.RFC3339Nano, timeString)
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
	mintedAt, err := ParseTime(tokenBuffer.MintedAt)
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
		MintedAt:        mintedAt,
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

// MapIndexerTokenToGrpcToken maps indexer token to grpc indexer token
func (m *Mapper) MapIndexerTokenToGrpcToken(token *indexer.Token) *grpcIndexer.Token {
	return &grpcIndexer.Token{
		ID:                token.ID,
		Blockchain:        token.Blockchain,
		Fungible:          token.Fungible,
		ContractType:      token.ContractType,
		ContractAddress:   token.ContractAddress,
		Edition:           token.Edition,
		EditionName:       token.EditionName,
		MintedAt:          token.MintedAt.Format(time.RFC3339Nano),
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
		LastActivityTime:  token.LastActivityTime.Format(time.RFC3339Nano),
		LastRefreshedTime: token.LastRefreshedTime.Format(time.RFC3339Nano),
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
			Timestamp:   v.Timestamp.Format(time.RFC3339Nano),
			TxID:        v.Timestamp.Format(time.RFC3339Nano),
			TxURL:       v.TxURL,
		}
	}

	return GRPCProvenances
}

func ConvertTimeStringsToTimes(timeStrings []string) ([]time.Time, error) {
	times := make([]time.Time, len(timeStrings))

	for i, k := range timeStrings {
		t, err := ParseTime(k)
		if err != nil {
			return nil, err
		}

		times[i] = t
	}

	return times, nil
}

// ConvertTimesToTimeStrings converts times to time strings
func ConvertTimesToTimeStrings(times []time.Time) (timeStrings []string) {
	for _, k := range times {
		timeStrings = append(timeStrings, k.Format(time.RFC3339Nano))
	}

	return
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

		lastPendingTime, err := ConvertTimeStringsToTimes(v.LastPendingTime)
		if err != nil {
			return nil, err
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

	var artists []*grpcIndexer.Artist
	for _, i := range projectMetadata.Artists {
		artists = append(artists, &grpcIndexer.Artist{
			ArtistID:   i.ID,
			ArtistName: i.Name,
			ArtistURL:  i.URL,
		})
	}

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

		Attributes:       attributes,
		ArtworkMetadata:  m.MapIndexerArtworkMetadataToGRPCArtworkMetadata(projectMetadata.ArtworkMetadata),
		LastUpdatedAt:    projectMetadata.LastUpdatedAt.Format(time.RFC3339Nano),
		InitialSaleModel: projectMetadata.InitialSaleModel,
		OriginalFileURL:  projectMetadata.OriginalFileURL,
		Artists:          artists,
	}
}

// MapIndexerArtworkMetadataToGRPCArtworkMetadata maps indexer artwork metadata to grpc artwork metadata
func (m *Mapper) MapIndexerArtworkMetadataToGRPCArtworkMetadata(artworkMetadata map[string]interface{}) string {
	b, err := json.Marshal(artworkMetadata)
	if err != nil {
		return ""
	}

	return string(b)
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

// MapIndexerAccountTokensToGRPCAccountTokens maps indexer account tokens to grpc account tokens
func (m *Mapper) MapIndexerAccountTokensToGRPCAccountTokens(accountTokens []indexer.AccountToken) []*grpcIndexer.AccountToken {
	GRPCAccountTokens := make([]*grpcIndexer.AccountToken, len(accountTokens))

	for i, v := range accountTokens {
		GRPCAccountTokens[i] = &grpcIndexer.AccountToken{
			ID:                v.ID,
			Blockchain:        v.Blockchain,
			Fungible:          v.Fungible,
			ContractType:      v.ContractType,
			ContractAddress:   v.ContractAddress,
			IndexID:           v.IndexID,
			OwnerAccount:      v.OwnerAccount,
			Balance:           v.Balance,
			LastActivityTime:  v.LastActivityTime.Format(time.RFC3339Nano),
			LastRefreshedTime: v.LastRefreshedTime.Format(time.RFC3339Nano),
			LastPendingTime:   ConvertTimesToTimeStrings(v.LastPendingTime),
			PendingTxs:        v.PendingTxs,
		}
	}

	return GRPCAccountTokens
}

// MapGrpcDetailedTokenToIndexerDetailedToken maps grpc detailed token to indexer detailed token
func (m *Mapper) MapGrpcDetailedTokenToIndexerDetailedToken(token *grpcIndexer.DetailedToken) (*indexer.DetailedToken, error) {
	origin, err := m.MapGrpcProjectMetadataToIndexerProjectMetadata(token.ProjectMetadata.Origin)
	if err != nil {
		return nil, err
	}
	latest, err := m.MapGrpcProjectMetadataToIndexerProjectMetadata(token.ProjectMetadata.Latest)
	if err != nil {
		return nil, err
	}

	attributes := m.MapGrpcAttributesToIndexerAttributes(token.Attributes)

	return &indexer.DetailedToken{
		Token:       *m.MapGrpcTokenToIndexerToken(token.Token),
		ThumbnailID: token.ThumbnailID,
		IPFSPinned:  token.IPFSPinned,
		Attributes:  attributes,
		ProjectMetadata: indexer.VersionedProjectMetadata{
			Origin: *origin,
			Latest: *latest,
		},
	}, nil
}

// MapGrpcAttributesToIndexerAttributes maps grpc attributes to indexer attributes
func (m *Mapper) MapGrpcAttributesToIndexerAttributes(attributes *grpcIndexer.AssetAttributes) *indexer.AssetAttributes {
	if attributes == nil {
		return nil
	}

	return &indexer.AssetAttributes{
		Scrollable: attributes.Scrollable,
	}
}

// MapGrpcProjectMetadataToIndexerProjectMetadata maps grpc project metadata to indexer project metadata
func (m *Mapper) MapGrpcProjectMetadataToIndexerProjectMetadata(projectMetadata *grpcIndexer.ProjectMetadata) (*indexer.ProjectMetadata, error) {
	attributes := m.MapGrpcAttributesToIndexerAttributes(projectMetadata.Attributes)
	lastUpdatedAt, err := time.Parse(time.RFC3339Nano, projectMetadata.LastUpdatedAt)
	if err != nil {
		return nil, err
	}

	var artists []indexer.Artist

	for _, a := range projectMetadata.Artists {
		artists = append(artists, indexer.Artist{
			ID:   a.ArtistID,
			Name: a.ArtistName,
			URL:  a.ArtistURL,
		})
	}

	return &indexer.ProjectMetadata{
		ArtistID:            projectMetadata.ArtistID,
		ArtistName:          projectMetadata.ArtistName,
		ArtistURL:           projectMetadata.ArtistURL,
		AssetID:             projectMetadata.AssetID,
		Title:               projectMetadata.Title,
		Description:         projectMetadata.Description,
		MIMEType:            projectMetadata.MIMEType,
		Medium:              indexer.Medium(projectMetadata.Medium),
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

		Attributes:       attributes,
		ArtworkMetadata:  m.MapGrpcArtworkMetadataToIndexerArtworkMetadata(projectMetadata.ArtworkMetadata),
		LastUpdatedAt:    lastUpdatedAt,
		InitialSaleModel: projectMetadata.InitialSaleModel,
		OriginalFileURL:  projectMetadata.OriginalFileURL,
		Artists:          artists,
	}, nil
}

// MapGrpcArtworkMetadataToIndexerArtworkMetadata maps grpc artwork metadata to indexer artwork metadata
func (m *Mapper) MapGrpcArtworkMetadataToIndexerArtworkMetadata(artworkMetadata string) map[string]interface{} {
	var b map[string]interface{}
	err := json.Unmarshal([]byte(artworkMetadata), &b)
	if err != nil {
		return nil
	}

	return b
}

func (m *Mapper) MapGenericSaleTimeSeries(s []indexer.GenericSalesTimeSeries) *grpcIndexer.SaleTimeSeriesRecords {
	if nil == s {
		return nil
	}

	records := make([]*grpcIndexer.SaleTimeSeriesRecord, len(s))
	for i, v := range s {
		metadata, err := structpb.NewStruct(v.Metadata)
		if err != nil {
			return nil
		}
		records[i] = &grpcIndexer.SaleTimeSeriesRecord{
			Timestamp: v.Timestamp,
			Metadata:  metadata,
			Values:    v.Values,
			Shares:    v.Shares,
		}
	}

	return &grpcIndexer.SaleTimeSeriesRecords{
		Sales: records,
	}
}

func (m *Mapper) MapGrpcSaleTimeSeriesRecords(s *grpcIndexer.SaleTimeSeriesRecords) []indexer.GenericSalesTimeSeries {
	if nil == s {
		return nil
	}

	records := make([]indexer.GenericSalesTimeSeries, len(s.Sales))
	for i, v := range s.Sales {
		records[i] = indexer.GenericSalesTimeSeries{
			Timestamp: v.Timestamp,
			Metadata:  v.Metadata.AsMap(),
			Values:    v.Values,
			Shares:    v.Shares,
		}
	}

	return records
}

func (m *Mapper) MapGrpcTimestampToTime(timestamp *timestamppb.Timestamp) *time.Time {
	if timestamp == nil {
		return nil
	}

	time := timestamp.AsTime()
	return &time
}

func (m *Mapper) MapTimeToGrpcTimestamp(time *time.Time) *timestamppb.Timestamp {
	if time == nil {
		return nil
	}

	return timestamppb.New(*time)
}

func (m *Mapper) MapToJson(input map[string]interface{}) (string, error) {
	b, err := json.Marshal(input)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (m *Mapper) MapToGrpcSaleTimeSeriesListResponse(sales []indexer.SaleTimeSeries) (*grpcIndexer.SaleTimeSeriesListResponse, error) {
	results := make([]*grpcIndexer.SaleTimeSeries, len(sales))
	for i, s := range sales {
		metadata, err := m.MapToJson(s.Metadata)
		if err != nil {
			return nil, err
		}

		shares, err := m.MapToJson(s.Shares)
		if err != nil {
			return nil, err
		}

		results[i] = &grpcIndexer.SaleTimeSeries{
			Timestamp:     m.MapTimeToGrpcTimestamp(&s.Timestamp),
			Metadata:      metadata,
			Shares:        shares,
			NetValue:      s.NetValue.String(),
			PaymentAmount: s.PaymentAmount.String(),
			PlatformFee:   s.PlatformFee.String(),
			UsdQuote:      s.USDQuote.String(),
			Price:         s.Price.String(),
		}
	}

	return &grpcIndexer.SaleTimeSeriesListResponse{
		Sales: results,
	}, nil
}

func (m *Mapper) MapToGrpcSaleRevenuesResponse(revenues map[string]primitive.Decimal128) (*grpcIndexer.SaleRevenuesResponse, error) {
	results := make(map[string]string)

	for k, v := range revenues {
		results[k] = v.String()
	}

	return &grpcIndexer.SaleRevenuesResponse{
		Revenues: results,
	}, nil
}

func (m *Mapper) MapGrpcSaleTimeSeriesListResponseToIndexerSaleTimeSeries(sales *grpcIndexer.SaleTimeSeriesListResponse) ([]indexer.SaleTimeSeries, error) {
	if sales == nil {
		return []indexer.SaleTimeSeries{}, nil
	}

	var results []indexer.SaleTimeSeries
	for _, s := range sales.Sales {
		netValue, err := primitive.ParseDecimal128(s.NetValue)
		if err != nil {
			return nil, err
		}

		paymentAmount, err := primitive.ParseDecimal128(s.PaymentAmount)
		if err != nil {
			return nil, err
		}

		platformFee, err := primitive.ParseDecimal128(s.PaymentAmount)
		if err != nil {
			return nil, err
		}

		usdQuote, err := primitive.ParseDecimal128(s.PaymentAmount)
		if err != nil {
			return nil, err
		}

		price, err := primitive.ParseDecimal128(s.PaymentAmount)
		if err != nil {
			return nil, err
		}

		var metadata, shares map[string]interface{}
		if err := json.Unmarshal([]byte(s.Metadata), &metadata); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(s.Shares), &shares); err != nil {
			return nil, err
		}

		results = append(results, indexer.SaleTimeSeries{
			Timestamp:     *m.MapGrpcTimestampToTime(s.Timestamp),
			Metadata:      metadata,
			Shares:        shares,
			NetValue:      netValue,
			PaymentAmount: paymentAmount,
			PlatformFee:   platformFee,
			USDQuote:      usdQuote,
			Price:         price,
		})
	}

	return results, nil
}

func (m *Mapper) MapToExchangeRateResponse(exchangeRate indexer.ExchangeRate) (*grpcIndexer.ExchangeRateResponse, error) {
	return &grpcIndexer.ExchangeRateResponse{
		Timestamp:    timestamppb.New(exchangeRate.Timestamp),
		Open:         fmt.Sprintf("%f", exchangeRate.Open),
		Close:        fmt.Sprintf("%f", exchangeRate.Close),
		CurrencyPair: exchangeRate.CurrencyPair,
	}, nil
}

func (m *Mapper) MapGrpcExchangeRateResponseToIndexerExchangeRate(exchangeRate *grpcIndexer.ExchangeRateResponse) (indexer.ExchangeRate, error) {
	open, err := primitive.ParseDecimal128(exchangeRate.Open)
	if err != nil {
		return indexer.ExchangeRate{}, err
	}

	close, err := primitive.ParseDecimal128(exchangeRate.Close)
	if err != nil {
		return indexer.ExchangeRate{}, err
	}

	openFloat, err := strconv.ParseFloat(open.String(), 64)
	if err != nil {
		return indexer.ExchangeRate{}, err
	}

	closeFloat, err := strconv.ParseFloat(close.String(), 64)
	if err != nil {
		return indexer.ExchangeRate{}, err
	}

	return indexer.ExchangeRate{
		Timestamp:    exchangeRate.Timestamp.AsTime(),
		Open:         openFloat,
		Close:        closeFloat,
		CurrencyPair: exchangeRate.CurrencyPair,
	}, nil
}
