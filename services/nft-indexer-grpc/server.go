package main

import (
	"context"
	"fmt"
	"net"
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/cache"
	indexerGRPCSDK "github.com/bitmark-inc/nft-indexer/sdk/nft-indexer-grpc"
	pb "github.com/bitmark-inc/nft-indexer/services/nft-indexer-grpc/grpc/indexer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
)

type IndexerServer struct {
	pb.UnimplementedIndexerServer

	grpcServer   *grpc.Server
	indexerStore *indexer.MongodbIndexerStore
	cacheStore   *cache.MongoDBCacheStore
	ethClient    *ethclient.Client

	network string
	port    int

	mapper *indexerGRPCSDK.Mapper
}

// NewIndexerGRPCServer creates a new IndexerServer
func NewIndexerGRPCServer(
	network string,
	port int,
	indexerStore *indexer.MongodbIndexerStore,
	cacheStore *cache.MongoDBCacheStore,
	ethClient *ethclient.Client,
) (*IndexerServer, error) {

	grpcServer := grpc.NewServer()
	mapper := &indexerGRPCSDK.Mapper{}

	return &IndexerServer{
		network:      network,
		port:         port,
		indexerStore: indexerStore,
		cacheStore:   cacheStore,
		ethClient:    ethClient,
		grpcServer:   grpcServer,
		mapper:       mapper,
	}, nil
}

// Run starts the IndexerServer
func (i *IndexerServer) Run(context.Context) error {
	listener, err := net.Listen(i.network, fmt.Sprintf("0.0.0.0:%d", i.port))
	if err != nil {
		return err
	}

	pb.RegisterIndexerServer(i.grpcServer, i)
	err = i.grpcServer.Serve(listener)
	if err != nil {
		return err
	}

	return nil
}

// GetTokenByIndexID returns a token by index ID
func (i *IndexerServer) GetTokenByIndexID(ctx context.Context, indexID *pb.IndexID) (*pb.Token, error) {
	token, err := i.indexerStore.GetTokenByIndexID(ctx, indexID.IndexID)
	if err != nil {
		return nil, err
	}

	if token == nil {
		return nil, fmt.Errorf("token does not exist")
	}

	pbToken := i.mapper.MapIndexerTokenToGrpcToken(token)

	return pbToken, nil
}

// PushProvenance pushes a provenance to the indexer
func (i *IndexerServer) PushProvenance(ctx context.Context, in *pb.PushProvenanceRequest) (*pb.EmptyMessage, error) {
	lockedTime, err := indexerGRPCSDK.ParseTime(in.LockedTime)
	if err != nil {
		return &pb.EmptyMessage{}, err
	}

	provenances, err := i.mapper.MapGRPCProvenancesToIndexerProvenances([]*pb.Provenance{in.Provenance})
	if err != nil {
		return &pb.EmptyMessage{}, err
	}
	if len(provenances) == 0 {
		return &pb.EmptyMessage{}, fmt.Errorf("invalid provenance")
	}

	err = i.indexerStore.PushProvenance(
		ctx,
		in.IndexID,
		lockedTime,
		provenances[0],
	)

	if err != nil {
		return &pb.EmptyMessage{}, err
	}

	return &pb.EmptyMessage{}, nil
}

// UpdateOwner updates the owner of a token
func (i *IndexerServer) UpdateOwner(ctx context.Context, in *pb.UpdateOwnerRequest) (*pb.EmptyMessage, error) {
	updatedAt, err := indexerGRPCSDK.ParseTime(in.UpdatedAt)
	if err != nil {
		return nil, err
	}

	err = i.indexerStore.UpdateOwner(ctx, in.IndexID, in.Owner, updatedAt)
	if err != nil {
		return nil, err
	}

	return &pb.EmptyMessage{}, nil
}

// UpdateOwnerForFungibleToken updates the owner of a fungible token
func (i *IndexerServer) UpdateOwnerForFungibleToken(ctx context.Context, in *pb.UpdateOwnerForFungibleTokenRequest) (*pb.EmptyMessage, error) {
	lockedTime, err := indexerGRPCSDK.ParseTime(in.LockedTime)
	if err != nil {
		return nil, err
	}

	err = i.indexerStore.UpdateOwnerForFungibleToken(ctx, in.IndexID, lockedTime, in.To, in.Total)
	if err != nil {
		return nil, err
	}

	return &pb.EmptyMessage{}, nil
}

// IndexAccountTokens indexes the Account tokens of an account
func (i *IndexerServer) IndexAccountTokens(ctx context.Context, in *pb.IndexAccountTokensRequest) (*pb.EmptyMessage, error) {
	accountTokens, err := i.mapper.MapGRPCAccountTokensToIndexerAccountTokens(in.AccountTokens)
	if err != nil {
		return nil, err
	}

	err = i.indexerStore.IndexAccountTokens(ctx, in.Owner, accountTokens)
	if err != nil {
		return nil, err
	}

	return &pb.EmptyMessage{}, nil
}

// GetDetailedToken returns a detailed token by index ID
func (i *IndexerServer) GetDetailedToken(ctx context.Context, in *pb.GetDetailedTokenRequest) (*pb.DetailedToken, error) {
	detailedToken, err := i.indexerStore.GetDetailedToken(ctx, in.IndexID, in.BurnedIncluded)
	if err != nil {
		return nil, err
	}

	pbDetailedToken := i.mapper.MapIndexerDetailedTokenToGRPCDetailedToken(detailedToken)

	return pbDetailedToken, nil
}

// GetTotalBalanceOfOwnerAccounts returns the total balance of owner accounts
func (i *IndexerServer) GetTotalBalanceOfOwnerAccounts(ctx context.Context, addresses *pb.Addresses) (*pb.TotalBalance, error) {
	totalBalance, err := i.indexerStore.GetTotalBalanceOfOwnerAccounts(ctx, addresses.Addresses)

	if err != nil {
		return nil, err
	}

	return &pb.TotalBalance{Count: int64(totalBalance)}, nil
}

// GetOwnerAccountsByIndexIDs get owner_accounts by indexIDs
func (i *IndexerServer) GetOwnerAccountsByIndexIDs(ctx context.Context, indexIDs *pb.IndexIDs) (*pb.Addresses, error) {
	owners, err := i.indexerStore.GetOwnerAccountsByIndexIDs(ctx, indexIDs.IndexIDs)
	if err != nil {
		return nil, err
	}

	return &pb.Addresses{Addresses: owners}, nil
}

// CheckAddressOwnTokenByCriteria checks if an address owns a token by criteria
func (i *IndexerServer) CheckAddressOwnTokenByCriteria(ctx context.Context, request *pb.CheckAddressOwnTokenByCriteriaRequest) (*pb.CheckAddressOwnTokenByCriteriaResponse, error) {
	result, err := i.indexerStore.CheckAddressOwnTokenByCriteria(ctx, request.Address, indexer.Criteria{
		IndexID: request.Criteria.IndexID,
		Source:  request.Criteria.Source,
	})
	if err != nil {
		return nil, err
	}

	return &pb.CheckAddressOwnTokenByCriteriaResponse{Result: result}, nil
}

// GetOwnersByBlockchainsAndContracts returns owners by blockchains and contracts
func (i *IndexerServer) GetOwnersByBlockchainContracts(ctx context.Context, request *pb.GetOwnersByBlockchainContractsRequest) (*pb.Addresses, error) {
	blockchainContract := make(map[string][]string)

	for k, v := range request.BlockchainContracts {
		blockchainContract[k] = v.Addresses
	}

	owners, err := i.indexerStore.GetOwnersByBlockchainContracts(ctx, blockchainContract)
	if err != nil {
		return nil, err
	}

	return &pb.Addresses{Addresses: owners}, nil
}

// GetETHBlockTime returns blockTime by blockHash
func (i *IndexerServer) GetETHBlockTime(ctx context.Context, request *pb.GetETHBlockTimeRequest) (*pb.BlockTime, error) {
	blockTime, err := indexer.GetETHBlockTime(ctx, i.cacheStore, i.ethClient, common.HexToHash(request.BlockHash))

	if err != nil {
		return nil, err
	}

	return &pb.BlockTime{BlockTime: blockTime.Format(time.RFC3339Nano)}, nil
}

// GetIdentity returns account identity by address
func (i *IndexerServer) GetIdentity(ctx context.Context, request *pb.Address) (*pb.AccountIdentity, error) {
	identity, err := i.indexerStore.GetIdentity(ctx, request.Address)

	if err != nil {
		return nil, err
	}

	return &pb.AccountIdentity{
		AccountNumber:   identity.AccountNumber,
		Blockchain:      identity.Blockchain,
		Name:            identity.Name,
		LastUpdatedTime: identity.LastUpdatedTime.Format(time.RFC3339Nano),
	}, nil
}

// SendTimeSeriesData send timestamped metadata and values
func (i *IndexerServer) SendTimeSeriesData(ctx context.Context, req *pb.SaleTimeSeriesRecords) (*pb.EmptyMessage, error) {
	return &pb.EmptyMessage{}, i.indexerStore.WriteTimeSeriesData(ctx, i.mapper.MapGrpcSaleTimeSeriesRecords(req))
}

func (i *IndexerServer) GetSaleTimeSeries(ctx context.Context, filter *pb.SaleTimeSeriesFilter) (*pb.SaleTimeSeriesListResponse, error) {
	offset := int64(0)
	size := int64(50)
	sortASC := false
	if filter.Offset != nil {
		offset = *filter.Offset
	}
	if filter.Size != nil {
		size = *filter.Size
	}
	if filter.SortASC != nil {
		sortASC = *filter.SortASC
	}

	sales, err := i.indexerStore.GetSaleTimeSeriesData(ctx, indexer.SalesFilterParameter{
		Addresses:   filter.Addresses,
		Marketplace: filter.Marketplace,
		From:        i.mapper.MapGrpcTimestampToTime(filter.From),
		To:          i.mapper.MapGrpcTimestampToTime(filter.To),
		Offset:      offset,
		Limit:       size,
		SortASC:     sortASC,
	})

	if err != nil {
		return nil, err
	}

	return i.mapper.MapToGrpcSaleTimeSeriesListResponse(sales)
}
func (i *IndexerServer) GetSaleRevenues(ctx context.Context, filter *pb.SaleTimeSeriesFilter) (*pb.SaleRevenuesResponse, error) {
	revenues, err := i.indexerStore.AggregateSaleRevenues(ctx, indexer.SalesFilterParameter{
		Addresses:   filter.Addresses,
		Marketplace: filter.Marketplace,
		From:        i.mapper.MapGrpcTimestampToTime(filter.From),
		To:          i.mapper.MapGrpcTimestampToTime(filter.To),
	})

	if err != nil {
		return nil, err
	}

	return i.mapper.MapToGrpcSaleRevenuesResponse(revenues)
}

func (i *IndexerServer) GetHistoricalExchangeRate(ctx context.Context, filter *pb.HistoricalExchangeRateFilter) (*pb.ExchangeRateResponse, error) {
	result, err := i.indexerStore.GetHistoricalExchangeRate(ctx, indexer.HistoricalExchangeRateFilter{
		CurrencyPair: filter.CurrencyPair,
		Timestamp:    *i.mapper.MapGrpcTimestampToTime(filter.Timestamp),
	})
	if err != nil {
		return nil, err
	}

	return i.mapper.MapToExchangeRateResponse(result)
}

func (i *IndexerServer) UpdateAssetsConfiguration(ctx context.Context, in *pb.UpdateAssetsConfigurationRequest) (*pb.EmptyMessage, error) {
	configuration := i.mapper.MapGrpcAssetConfigurationToIndexerAssetConfiguration(in.Configuration)
	modifiedCount, err := i.indexerStore.UpdateAssetsConfiguration(
		ctx,
		in.IndexIDs,
		configuration,
	)
	if err != nil {
		return nil, err
	}

	if modifiedCount == 0 {
		return nil, fmt.Errorf("no asset configuration updated")
	}

	return &pb.EmptyMessage{}, nil
}
