package sdk

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	indexer "github.com/bitmark-inc/nft-indexer"
	pb "github.com/bitmark-inc/nft-indexer/services/nft-indexer-grpc/grpc/indexer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type IndexerGRPCClient struct {
	client pb.IndexerClient
	mapper *Mapper
}

// NewIndexerClient returns a new IndexerGRPCClient
func NewIndexerClient(ServerURL string) (*IndexerGRPCClient, error) {
	conn, err := grpc.Dial(ServerURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pb.NewIndexerClient(conn)

	return &IndexerGRPCClient{
		client: client,
		mapper: &Mapper{},
	}, nil
}

// GetTokenByIndexID returns a token by indexID
func (i *IndexerGRPCClient) GetTokenByIndexID(ctx context.Context, indexID string) (*indexer.Token, error) {
	token, err := i.client.GetTokenByIndexID(ctx, &pb.IndexID{IndexID: indexID})
	if err != nil {
		return nil, err
	}

	indexerToken, err := i.mapper.MapGrpcTokenToIndexerToken(token)
	if err != nil {
		return nil, err
	}

	return indexerToken, nil
}

// PushProvenance pushes provenance to indexer db
func (i *IndexerGRPCClient) PushProvenance(ctx context.Context, indexID string, lockedTime time.Time, provenance indexer.Provenance) error {
	_, err := i.client.PushProvenance(ctx, &pb.PushProvenanceRequest{
		IndexID:    indexID,
		LockedTime: lockedTime.Format(time.RFC3339Nano),
		Provenance: i.mapper.MapIndexerProvenancesToGRPCProvenances([]indexer.Provenance{provenance})[0],
	})

	return err
}

// UpdateOwner updates owner of a token
func (i *IndexerGRPCClient) UpdateOwner(ctx context.Context, indexID, owner string, updatedAt time.Time) error {
	_, err := i.client.UpdateOwner(ctx, &pb.UpdateOwnerRequest{
		IndexID:   indexID,
		Owner:     owner,
		UpdatedAt: updatedAt.Format(time.RFC3339Nano),
	})

	return err
}

// UpdateOwnerForFungibleToken updates owner of a fungible token
func (i *IndexerGRPCClient) UpdateOwnerForFungibleToken(ctx context.Context, indexID string, lockedTime time.Time, to string, total int64) error {
	_, err := i.client.UpdateOwnerForFungibleToken(ctx, &pb.UpdateOwnerForFungibleTokenRequest{
		IndexID:    indexID,
		LockedTime: lockedTime.Format(time.RFC3339Nano),
		To:         to,
		Total:      total,
	})

	return err
}

// IndexAccountTokens indexes account tokens
func (i *IndexerGRPCClient) IndexAccountTokens(ctx context.Context, owner string, accountTokens []indexer.AccountToken) error {
	_, err := i.client.IndexAccountTokens(ctx, &pb.IndexAccountTokensRequest{
		Owner:         owner,
		AccountTokens: i.mapper.MapIndexerAccountTokensToGRPCAccountTokens(accountTokens),
	})

	return err
}

// GetDetailedToken returns a detailed token by indexID
func (i *IndexerGRPCClient) GetDetailedToken(ctx context.Context, indexID string, burnedIncluded bool) (indexer.DetailedToken, error) {
	detailedToken, err := i.client.GetDetailedToken(
		ctx,
		&pb.GetDetailedTokenRequest{
			IndexID:        indexID,
			BurnedIncluded: burnedIncluded,
		},
	)
	if err != nil {
		return indexer.DetailedToken{}, err
	}

	indexerDetailedToken, err := i.mapper.MapGrpcDetailedTokenToIndexerDetailedToken(detailedToken)
	if err != nil {
		return indexer.DetailedToken{}, err
	}

	return *indexerDetailedToken, nil
}

// GetTotalBalanceOfOwnerAccounts returns total balance of owner accounts
func (i *IndexerGRPCClient) GetTotalBalanceOfOwnerAccounts(ctx context.Context, addresses []string) (int64, error) {
	totalBalance, err := i.client.GetTotalBalanceOfOwnerAccounts(ctx, &pb.Addresses{Addresses: addresses})
	if err != nil {
		return 0, err
	}

	return totalBalance.Count, nil
}

// GetOwnerAccountsByIndexIDs returns owner accounts by indexIDs
func (i *IndexerGRPCClient) GetOwnerAccountsByIndexIDs(ctx context.Context, indexIDs []string) ([]string, error) {
	addresses, err := i.client.GetOwnerAccountsByIndexIDs(ctx, &pb.IndexIDs{IndexIDs: indexIDs})
	if err != nil {
		return nil, err
	}

	return addresses.Addresses, nil
}

// GetOwnersByBlockchainContracts returns owners by blockchains and contracts
func (i *IndexerGRPCClient) GetOwnersByBlockchainContracts(ctx context.Context, blockchainContracts map[string][]string) ([]string, error) {
	var GRPCBlockchainContracts pb.GetOwnersByBlockchainContractsRequest
	GRPCBlockchainContracts.BlockchainContracts = make(map[string]*pb.Addresses)

	for k, v := range blockchainContracts {
		GRPCBlockchainContracts.BlockchainContracts[k] = &pb.Addresses{Addresses: v}
	}

	addresses, err := i.client.GetOwnersByBlockchainContracts(ctx, &GRPCBlockchainContracts)
	if err != nil {
		return nil, err
	}

	return addresses.Addresses, nil
}

// CheckAddressOwnTokenByCriteria checks if an address owns a token by criteria
func (i *IndexerGRPCClient) CheckAddressOwnTokenByCriteria(ctx context.Context, address string, criteria indexer.Criteria) (bool, error) {
	res, err := i.client.CheckAddressOwnTokenByCriteria(ctx, &pb.CheckAddressOwnTokenByCriteriaRequest{
		Address: address,
		Criteria: &pb.Criteria{
			Source:  criteria.Source,
			IndexID: criteria.IndexID,
		},
	})
	if err != nil {
		return false, err
	}

	return res.Result, nil
}

// GetETHBlockTime get block time for a blockHash
func (i *IndexerGRPCClient) GetETHBlockTime(ctx context.Context, blockHash string) (time.Time, error) {
	res, err := i.client.GetETHBlockTime(ctx, &pb.GetETHBlockTimeRequest{
		BlockHash: blockHash,
	})
	if err != nil {
		return time.Time{}, err
	}

	return ParseTime(res.BlockTime)
}

func (i *IndexerGRPCClient) GetIdentity(ctx context.Context, address string) (indexer.AccountIdentity, error) {
	identity, err := i.client.GetIdentity(ctx, &pb.Address{Address: address})

	if err != nil {
		return indexer.AccountIdentity{}, err
	}

	lastUpdatedTime, _ := ParseTime(identity.LastUpdatedTime)

	return indexer.AccountIdentity{
		AccountNumber:   identity.AccountNumber,
		Blockchain:      identity.Blockchain,
		Name:            identity.Name,
		LastUpdatedTime: lastUpdatedTime,
	}, nil
}

// SendTimeSeriesData send timestamped metadata and values
func (i *IndexerGRPCClient) SendTimeSeriesData(
	ctx context.Context,
	req []indexer.GenericSalesTimeSeries,
) error {
	_, err := i.client.SendTimeSeriesData(
		ctx,
		i.mapper.MapGenericSaleTimeSeries(req))
	return err
}

// GetTimeSeriesData get list of SaleTimeSeries base on the filter parameters
func (i *IndexerGRPCClient) GetTimeSeriesData(
	ctx context.Context,
	filter indexer.SalesFilterParameter,
) ([]indexer.SaleTimeSeries, error) {
	sales, err := i.client.GetSaleTimeSeries(ctx, &pb.SaleTimeSeriesFilter{
		Addresses:   filter.Addresses,
		Marketplace: filter.Marketplace,
		From:        i.mapper.MapTimeToGrpcTimestamp(filter.From),
		To:          i.mapper.MapTimeToGrpcTimestamp(filter.To),
		Offset:      aws.Int64(int64(filter.Offset)),
		Size:        aws.Int64(int64(filter.Limit)),
		SortASC:     aws.Bool(filter.SortASC),
	})

	if err != nil {
		return nil, err
	}

	return i.mapper.MapGrpcSaleTimeSeriesListResponseToIndexerSaleTimeSeries(sales)
}

// GetSaleRevenues get the revenues base on the filter parameters
func (i *IndexerGRPCClient) GetSaleRevenues(
	ctx context.Context,
	filter indexer.SalesFilterParameter,
) (map[string]string, error) {
	revenues, err := i.client.GetSaleRevenues(ctx, &pb.SaleTimeSeriesFilter{
		Addresses:   filter.Addresses,
		Marketplace: filter.Marketplace,
		From:        i.mapper.MapTimeToGrpcTimestamp(filter.From),
		To:          i.mapper.MapTimeToGrpcTimestamp(filter.To),
	})

	if err != nil {
		return nil, err
	}

	return revenues.Revenues, err
}

func (i *IndexerGRPCClient) GetExchangeRate(
	ctx context.Context,
	filter indexer.HistoricalExchangeRateFilter,
) (indexer.ExchangeRate, error) {
	result, err := i.client.GetHistoricalExchangeRate(ctx, &pb.HistoricalExchangeRateFilter{
		CurrencyPair: filter.CurrencyPair,
		Timestamp:    i.mapper.MapTimeToGrpcTimestamp(&filter.Timestamp),
	})

	if err != nil {
		return indexer.ExchangeRate{}, err
	}

	indexerExchangeRate, err := i.mapper.MapGrpcExchangeRateResponseToIndexerExchangeRate(result)
	if err != nil {
		return indexer.ExchangeRate{}, err
	}

	return indexerExchangeRate, nil
}

func (i *IndexerGRPCClient) UpdateAssetConfiguration(ctx context.Context, indexID string, configuration *indexer.AssetConfiguration) error {
	_, err := i.client.UpdateAssetConfiguration(ctx,
		&pb.UpdateAssetConfigurationRequest{
			IndexID:       indexID,
			Configuration: i.mapper.MapIndexerAssetConfigurationToGrpcAssetConfiguration(configuration),
		})

	return err
}
