package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	indexer "github.com/feral-file/ff-indexer"
	"github.com/feral-file/ff-indexer/cache"
	sdk "github.com/feral-file/ff-indexer/sdk/grpc-gateway"
	pb "github.com/feral-file/ff-indexer/services/grpc-gateway/grpc"
	"google.golang.org/grpc"
)

type Server struct {
	pb.UnimplementedGrpcServer

	server       *grpc.Server
	indexerStore *indexer.MongodbIndexerStore
	cacheStore   *cache.MongoDBCacheStore
	ethClient    *ethclient.Client

	network string
	port    int

	mapper *sdk.Mapper
}

// NewServer creates a new server
func NewServer(
	network string,
	port int,
	store *indexer.MongodbIndexerStore,
	cache *cache.MongoDBCacheStore,
	ethClient *ethclient.Client,
) (*Server, error) {

	server := grpc.NewServer()
	mapper := &sdk.Mapper{}

	return &Server{
		network:      network,
		port:         port,
		indexerStore: store,
		cacheStore:   cache,
		ethClient:    ethClient,
		server:       server,
		mapper:       mapper,
	}, nil
}

// Run starts the IndexerServer
func (i *Server) Run(context.Context) error {
	listener, err := net.Listen(i.network, fmt.Sprintf("0.0.0.0:%d", i.port))
	if err != nil {
		return err
	}

	pb.RegisterGrpcServer(i.server, i)
	err = i.server.Serve(listener)
	if err != nil {
		return err
	}

	return nil
}

// GetTokenByIndexID returns a token by index ID
func (i *Server) GetTokenByIndexID(ctx context.Context, indexID *pb.IndexID) (*pb.Token, error) {
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
func (i *Server) PushProvenance(ctx context.Context, in *pb.PushProvenanceRequest) (*pb.EmptyMessage, error) {
	lockedTime, err := sdk.ParseTime(in.LockedTime)
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
func (i *Server) UpdateOwner(ctx context.Context, in *pb.UpdateOwnerRequest) (*pb.EmptyMessage, error) {
	updatedAt, err := sdk.ParseTime(in.UpdatedAt)
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
func (i *Server) UpdateOwnerForFungibleToken(ctx context.Context, in *pb.UpdateOwnerForFungibleTokenRequest) (*pb.EmptyMessage, error) {
	lockedTime, err := sdk.ParseTime(in.LockedTime)
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
func (i *Server) IndexAccountTokens(ctx context.Context, in *pb.IndexAccountTokensRequest) (*pb.EmptyMessage, error) {
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
func (i *Server) GetDetailedToken(ctx context.Context, in *pb.GetDetailedTokenRequest) (*pb.DetailedToken, error) {
	detailedToken, err := i.indexerStore.GetDetailedToken(ctx, in.IndexID, in.BurnedIncluded)
	if err != nil {
		return nil, err
	}

	pbDetailedToken := i.mapper.MapIndexerDetailedTokenToGRPCDetailedToken(detailedToken)

	return pbDetailedToken, nil
}

// GetTotalBalanceOfOwnerAccounts returns the total balance of owner accounts
func (i *Server) GetTotalBalanceOfOwnerAccounts(ctx context.Context, addresses *pb.Addresses) (*pb.TotalBalance, error) {
	totalBalance, err := i.indexerStore.GetTotalBalanceOfOwnerAccounts(ctx, addresses.Addresses)

	if err != nil {
		return nil, err
	}

	return &pb.TotalBalance{Count: int64(totalBalance)}, nil
}

// GetOwnerAccountsByIndexIDs get owner_accounts by indexIDs
func (i *Server) GetOwnerAccountsByIndexIDs(ctx context.Context, indexIDs *pb.IndexIDs) (*pb.Addresses, error) {
	owners, err := i.indexerStore.GetOwnerAccountsByIndexIDs(ctx, indexIDs.IndexIDs)
	if err != nil {
		return nil, err
	}

	return &pb.Addresses{Addresses: owners}, nil
}

// CheckAddressOwnTokenByCriteria checks if an address owns a token by criteria
func (i *Server) CheckAddressOwnTokenByCriteria(ctx context.Context, request *pb.CheckAddressOwnTokenByCriteriaRequest) (*pb.CheckAddressOwnTokenByCriteriaResponse, error) {
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
func (i *Server) GetOwnersByBlockchainContracts(ctx context.Context, request *pb.GetOwnersByBlockchainContractsRequest) (*pb.Addresses, error) {
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
func (i *Server) GetETHBlockTime(ctx context.Context, request *pb.GetETHBlockTimeRequest) (*pb.BlockTime, error) {
	blockTime, err := indexer.GetETHBlockTime(ctx, i.cacheStore, i.ethClient, common.HexToHash(request.BlockHash))

	if err != nil {
		return nil, err
	}

	return &pb.BlockTime{BlockTime: blockTime.Format(time.RFC3339Nano)}, nil
}

// GetIdentity returns account identity by address
func (i *Server) GetIdentity(ctx context.Context, request *pb.Address) (*pb.AccountIdentity, error) {
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
func (i *Server) SendTimeSeriesData(ctx context.Context, req *pb.SaleTimeSeriesRecords) (*pb.EmptyMessage, error) {
	return &pb.EmptyMessage{}, i.indexerStore.WriteTimeSeriesData(ctx, i.mapper.MapGrpcSaleTimeSeriesRecords(req))
}

func (i *Server) GetSaleTimeSeries(ctx context.Context, filter *pb.SaleTimeSeriesFilter) (*pb.SaleTimeSeriesListResponse, error) {
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
func (i *Server) GetSaleRevenues(ctx context.Context, filter *pb.SaleTimeSeriesFilter) (*pb.SaleRevenuesResponse, error) {
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

func (i *Server) GetHistoricalExchangeRate(ctx context.Context, filter *pb.HistoricalExchangeRateFilter) (*pb.ExchangeRateResponse, error) {
	result, err := i.indexerStore.GetHistoricalExchangeRate(ctx, indexer.HistoricalExchangeRateFilter{
		CurrencyPair: filter.CurrencyPair,
		Timestamp:    *i.mapper.MapGrpcTimestampToTime(filter.Timestamp),
	})
	if err != nil {
		return nil, err
	}

	return i.mapper.MapToExchangeRateResponse(result)
}

func (i *Server) UpdateAssetsConfiguration(ctx context.Context, in *pb.UpdateAssetsConfigurationRequest) (*pb.EmptyMessage, error) {
	configuration := i.mapper.MapGrpcAssetConfigurationToIndexerAssetConfiguration(in.Configuration)
	matchedCount, err := i.indexerStore.UpdateAssetsConfiguration(
		ctx,
		in.IDs,
		configuration,
	)
	if err != nil {
		return nil, err
	}

	if matchedCount == 0 {
		return nil, fmt.Errorf("no asset configuration updated")
	}

	return &pb.EmptyMessage{}, nil
}

func (i *Server) CheckAssetCreator(ctx context.Context, request *pb.CheckAssetCreatorRequest) (*pb.CheckAssetCreatorResponse, error) {
	result, err := i.indexerStore.CheckAssetCreator(ctx, request.IDs, request.CreatorAddresses)
	if err != nil {
		return nil, err
	}
	return &pb.CheckAssetCreatorResponse{Result: result}, nil
}
