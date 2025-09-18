package sdk

import (
	"time"

	"encoding/json"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/types/known/structpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	indexer "github.com/feral-file/ff-indexer"
	"github.com/feral-file/ff-indexer/services/grpc-gateway/grpc"
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

// MapGRPCTokenInfoToIndexerTokenInfo maps grpc token info to indexer token info
func (m *Mapper) MapGRPCTokenInfoToIndexerTokenInfo(token []*grpc.BaseTokenInfo) []indexer.BaseTokenInfo {
	ti := make([]indexer.BaseTokenInfo, len(token))

	for i, v := range token {
		ti[i] = indexer.BaseTokenInfo{
			ID:              v.ID,
			Blockchain:      v.Blockchain,
			Fungible:        v.Fungible,
			ContractType:    v.ContractType,
			ContractAddress: v.ContractAddress,
		}
	}

	return ti
}

// MapGRPCProvenancesToIndexerProvenances maps grpc provenance to indexer provenance
func (m *Mapper) MapGRPCProvenancesToIndexerProvenances(provenance []*grpc.Provenance) ([]indexer.Provenance, error) {
	provenances := make([]indexer.Provenance, len(provenance))

	for i, v := range provenance {
		timestamp, err := ParseTime(v.Timestamp)
		if err != nil {
			return nil, err
		}

		provenances[i] = indexer.Provenance{
			FormerOwner: &v.FormerOwner,
			Type:        v.Type,
			Owner:       v.Owner,
			Blockchain:  v.Blockchain,
			BlockNumber: v.BlockNumber,
			Timestamp:   timestamp,
			TxID:        v.TxID,
			TxURL:       v.TxURL,
		}
	}

	return provenances, nil
}

// MapGrpcTokenToIndexerToken maps grpc indexer token to indexer token
func (m *Mapper) MapGrpcTokenToIndexerToken(tokenBuffer *grpc.Token) (*indexer.Token, error) {
	mintedAt, err := ParseTime(tokenBuffer.MintedAt)
	if err != nil {
		return nil, err
	}

	lastActivityTime, err := ParseTime(tokenBuffer.LastRefreshedTime)
	if err != nil {
		return nil, err
	}

	lastRefreshedTime, err := ParseTime(tokenBuffer.LastRefreshedTime)
	if err != nil {
		return nil, err
	}

	provenances, err := m.MapGRPCProvenancesToIndexerProvenances(tokenBuffer.Provenances)
	if err != nil {
		return nil, err
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
		OriginTokenInfo: m.MapGRPCTokenInfoToIndexerTokenInfo(tokenBuffer.OriginTokenInfo),
		IsDemo:          tokenBuffer.IsDemo,

		IndexID:           tokenBuffer.IndexID,
		Source:            tokenBuffer.Source,
		Swapped:           tokenBuffer.Swapped,
		SwappedFrom:       &tokenBuffer.SwappedFrom,
		SwappedTo:         &tokenBuffer.SwappedTo,
		Burned:            tokenBuffer.Burned,
		Provenances:       provenances,
		LastActivityTime:  lastActivityTime,
		LastRefreshedTime: lastRefreshedTime,
	}, nil
}

// MapIndexerTokenToGrpcToken maps indexer token to grpc indexer token
func (m *Mapper) MapIndexerTokenToGrpcToken(token *indexer.Token) *grpc.Token {
	return &grpc.Token{
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
		OriginTokenInfo:   m.MapIndexerTokenInfoToGRPCTokenInfo(token.OriginTokenInfo),
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

// MapIndexerTokenInfoToGRPCTokenInfo maps indexer token info to grpc token info
func (m *Mapper) MapIndexerTokenInfoToGRPCTokenInfo(token []indexer.BaseTokenInfo) []*grpc.BaseTokenInfo {
	gti := make([]*grpc.BaseTokenInfo, len(token))

	for i, v := range token {
		gti[i] = &grpc.BaseTokenInfo{
			ID:              v.ID,
			Blockchain:      v.Blockchain,
			Fungible:        v.Fungible,
			ContractType:    v.ContractType,
			ContractAddress: v.ContractAddress,
		}
	}

	return gti
}

// MapIndexerProvenancesToGRPCProvenances maps indexer provenance to grpc provenance
func (m *Mapper) MapIndexerProvenancesToGRPCProvenances(provenance []indexer.Provenance) []*grpc.Provenance {
	gtp := make([]*grpc.Provenance, len(provenance))

	for i, v := range provenance {
		gtp[i] = &grpc.Provenance{
			FormerOwner: DerefString(v.FormerOwner),
			Type:        v.Type,
			Owner:       v.Owner,
			Blockchain:  v.Blockchain,
			BlockNumber: v.BlockNumber,
			Timestamp:   v.Timestamp.Format(time.RFC3339Nano),
			TxID:        v.TxID,
			TxURL:       v.TxURL,
		}
	}

	return gtp
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

func (m *Mapper) MapGRPCAccountTokensToIndexerAccountTokens(accountTokens []*grpc.AccountToken) ([]indexer.AccountToken, error) {
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
func (m *Mapper) MapIndexerAttributesToGRPCAttributes(attributes *indexer.AssetAttributes) *grpc.AssetAttributes {
	if attributes == nil {
		return nil
	}

	return &grpc.AssetAttributes{
		Configuration: m.MapIndexerAssetConfigurationToGrpcAssetConfiguration(attributes.Configuration),
	}
}

// MapGrpcAttributesToIndexerAttributes maps grpc attributes to indexer attributes
func (m *Mapper) MapGrpcAttributesToIndexerAttributes(attributes *grpc.AssetAttributes) *indexer.AssetAttributes {
	if attributes == nil {
		return nil
	}

	return &indexer.AssetAttributes{
		Configuration: m.MapGrpcAssetConfigurationToIndexerAssetConfiguration(attributes.Configuration),
	}
}

// MapIndexerProjectMetadataToGRPCProjectMetadata maps indexer project metadata to grpc project metadata
func (m *Mapper) MapIndexerProjectMetadataToGRPCProjectMetadata(projectMetadata *indexer.ProjectMetadata) *grpc.ProjectMetadata {
	attributes := m.MapIndexerAttributesToGRPCAttributes(projectMetadata.Attributes)

	var artists []*grpc.Artist
	for _, i := range projectMetadata.Artists {
		artists = append(artists, &grpc.Artist{
			ArtistID:   i.ID,
			ArtistName: i.Name,
			ArtistURL:  i.URL,
		})
	}

	return &grpc.ProjectMetadata{
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
func (m *Mapper) MapIndexerDetailedTokenToGRPCDetailedToken(token indexer.DetailedToken) *grpc.DetailedToken {
	origin := m.MapIndexerProjectMetadataToGRPCProjectMetadata(&token.ProjectMetadata.Origin)
	latest := m.MapIndexerProjectMetadataToGRPCProjectMetadata(&token.ProjectMetadata.Latest)
	attributes := m.MapIndexerAttributesToGRPCAttributes(token.Attributes)

	return &grpc.DetailedToken{
		Token:       m.MapIndexerTokenToGrpcToken(&token.Token),
		ThumbnailID: token.ThumbnailID,
		IPFSPinned:  token.IPFSPinned,
		Attributes:  attributes,
		ProjectMetadata: &grpc.VersionedProjectMetadata{
			Origin: origin,
			Latest: latest,
		},
	}
}

// MapIndexerAccountTokensToGRPCAccountTokens maps indexer account tokens to grpc account tokens
func (m *Mapper) MapIndexerAccountTokensToGRPCAccountTokens(accountTokens []indexer.AccountToken) []*grpc.AccountToken {
	GRPCAccountTokens := make([]*grpc.AccountToken, len(accountTokens))

	for i, v := range accountTokens {
		GRPCAccountTokens[i] = &grpc.AccountToken{
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
func (m *Mapper) MapGrpcDetailedTokenToIndexerDetailedToken(token *grpc.DetailedToken) (*indexer.DetailedToken, error) {
	origin, err := m.MapGrpcProjectMetadataToIndexerProjectMetadata(token.ProjectMetadata.Origin)
	if err != nil {
		return nil, err
	}

	latest, err := m.MapGrpcProjectMetadataToIndexerProjectMetadata(token.ProjectMetadata.Latest)
	if err != nil {
		return nil, err
	}

	indexerToken, err := m.MapGrpcTokenToIndexerToken(token.Token)
	if err != nil {
		return nil, err
	}

	attributes := m.MapGrpcAttributesToIndexerAttributes(token.Attributes)
	return &indexer.DetailedToken{
		Token:       *indexerToken,
		ThumbnailID: token.ThumbnailID,
		IPFSPinned:  token.IPFSPinned,
		Attributes:  attributes,
		ProjectMetadata: indexer.VersionedProjectMetadata{
			Origin: *origin,
			Latest: *latest,
		},
	}, nil
}

// MapGrpcAssetConfigurationToIndexerAssetConfiguration maps grpc asset configuration to indexer asset configuration
func (m *Mapper) MapGrpcAssetConfigurationToIndexerAssetConfiguration(configuration *grpc.AssetConfiguration) *indexer.AssetConfiguration {
	if configuration == nil {
		return nil
	}

	return &indexer.AssetConfiguration{
		Scaling:         configuration.Scaling,
		BackgroundColor: configuration.BackgroundColor,
		MarginLeft:      configuration.MarginLeft,
		MarginRight:     configuration.MarginRight,
		MarginTop:       configuration.MarginTop,
		MarginBottom:    configuration.MarginBottom,
		AutoPlay:        configuration.AutoPlay,
		Looping:         configuration.Looping,
		Interactable:    configuration.Interactable,
		Overridable:     configuration.Overridable,
	}
}

// MapIndexerAssetConfigurationToGrpcAssetConfiguration maps indexer asset configuration to grpc asset configuration
func (m *Mapper) MapIndexerAssetConfigurationToGrpcAssetConfiguration(configuration *indexer.AssetConfiguration) *grpc.AssetConfiguration {
	if configuration == nil {
		return nil
	}

	return &grpc.AssetConfiguration{
		Scaling:         configuration.Scaling,
		BackgroundColor: configuration.BackgroundColor,
		MarginLeft:      configuration.MarginLeft,
		MarginRight:     configuration.MarginRight,
		MarginTop:       configuration.MarginTop,
		MarginBottom:    configuration.MarginBottom,
		AutoPlay:        configuration.AutoPlay,
		Looping:         configuration.Looping,
		Interactable:    configuration.Interactable,
		Overridable:     configuration.Overridable,
	}
}

// MapGrpcProjectMetadataToIndexerProjectMetadata maps grpc project metadata to indexer project metadata
func (m *Mapper) MapGrpcProjectMetadataToIndexerProjectMetadata(projectMetadata *grpc.ProjectMetadata) (*indexer.ProjectMetadata, error) {
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

func (m *Mapper) MapGenericSaleTimeSeries(s []indexer.GenericSalesTimeSeries) *grpc.SaleTimeSeriesRecords {
	if nil == s {
		return nil
	}

	records := make([]*grpc.SaleTimeSeriesRecord, len(s))
	for i, v := range s {
		metadata, err := structpb.NewStruct(v.Metadata)
		if err != nil {
			return nil
		}
		records[i] = &grpc.SaleTimeSeriesRecord{
			Timestamp: v.Timestamp,
			Metadata:  metadata,
			Values:    v.Values,
			Shares:    v.Shares,
		}
	}

	return &grpc.SaleTimeSeriesRecords{
		Sales: records,
	}
}

func (m *Mapper) MapGrpcSaleTimeSeriesRecords(s *grpc.SaleTimeSeriesRecords) []indexer.GenericSalesTimeSeries {
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

func (m *Mapper) MapToGrpcSaleTimeSeriesListResponse(sales []indexer.SaleTimeSeries) (*grpc.SaleTimeSeriesListResponse, error) {
	results := make([]*grpc.SaleTimeSeries, len(sales))
	for i, s := range sales {
		s := s // Create a local copy to avoid memory aliasing
		metadata, err := m.MapToJson(s.Metadata)
		if err != nil {
			return nil, err
		}

		shares, err := m.MapToJson(s.Shares)
		if err != nil {
			return nil, err
		}

		results[i] = &grpc.SaleTimeSeries{
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

	return &grpc.SaleTimeSeriesListResponse{
		Sales: results,
	}, nil
}

func (m *Mapper) MapToGrpcSaleRevenuesResponse(revenues map[string]primitive.Decimal128) (*grpc.SaleRevenuesResponse, error) {
	results := make(map[string]string)

	for k, v := range revenues {
		results[k] = v.String()
	}

	return &grpc.SaleRevenuesResponse{
		Revenues: results,
	}, nil
}

func (m *Mapper) MapGrpcSaleTimeSeriesListResponseToIndexerSaleTimeSeries(sales *grpc.SaleTimeSeriesListResponse) ([]indexer.SaleTimeSeries, error) {
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

func (m *Mapper) MapToExchangeRateResponse(exchangeRate indexer.ExchangeRate) (*grpc.ExchangeRateResponse, error) {
	return &grpc.ExchangeRateResponse{
		Timestamp:    timestamppb.New(exchangeRate.Timestamp),
		Price:        exchangeRate.Price,
		CurrencyPair: exchangeRate.CurrencyPair,
	}, nil
}

func (m *Mapper) MapGrpcExchangeRateResponseToIndexerExchangeRate(exchangeRate *grpc.ExchangeRateResponse) (indexer.ExchangeRate, error) {
	return indexer.ExchangeRate{
		Timestamp:    exchangeRate.Timestamp.AsTime(),
		Price:        exchangeRate.Price,
		CurrencyPair: exchangeRate.CurrencyPair,
	}, nil
}
