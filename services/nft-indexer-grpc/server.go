package main

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"

	indexer "github.com/bitmark-inc/nft-indexer"
	indexerGRPCSDK "github.com/bitmark-inc/nft-indexer/sdk/nft-indexer-grpc"
	pb "github.com/bitmark-inc/nft-indexer/services/nft-indexer-grpc/grpc/indexer"
)

type IndexerServer struct {
	pb.UnimplementedIndexerServer

	grpcServer   *grpc.Server
	indexerStore *indexer.MongodbIndexerStore

	network string
	port    int

	mapper *indexerGRPCSDK.Mapper
}

// NewIndexerGRPCServer creates a new IndexerServer
func NewIndexerGRPCServer(
	network string,
	port int,
	indexerStore *indexer.MongodbIndexerStore,
) (*IndexerServer, error) {

	grpcServer := grpc.NewServer()
	mapper := &indexerGRPCSDK.Mapper{}

	return &IndexerServer{
		network:      network,
		port:         port,
		indexerStore: indexerStore,
		grpcServer:   grpcServer,
		mapper:       mapper,
	}, nil
}

// GetTokenByIndexID returns a token by index ID
func (i *IndexerServer) GetTokenByIndexID(ctx context.Context, indexID *pb.IndexID) (*pb.Token, error) {
	token, err := i.indexerStore.GetTokensByIndexID(ctx, indexID.IndexID)
	if err != nil {
		return nil, err
	}

	pbToken := i.mapper.MapIndexerTokenToGrpcToken(token)

	return pbToken, nil
}

// PushProvenance pushes a provenance to the indexer
func (i *IndexerServer) PushProvenance(ctx context.Context, in *pb.PushProvenanceRequest) (*pb.Empty, error) {
	lockedTime, err := indexerGRPCSDK.ParseTime(in.LockedTime)
	if err != nil {
		return &pb.Empty{}, err
	}

	provenance := i.mapper.MapGRPCProvenancesToIndexerProvenances([]*pb.Provenance{in.Provenance})[0]

	err = i.indexerStore.PushProvenance(
		ctx,
		in.IndexID,
		lockedTime,
		provenance,
	)

	if err != nil {
		return &pb.Empty{}, err
	}

	return &pb.Empty{}, nil
}

// UpdateOwner updates the owner of a token
func (i *IndexerServer) UpdateOwner(ctx context.Context, in *pb.UpdateOwnerRequest) (*pb.Empty, error) {
	updatedAt, err := indexerGRPCSDK.ParseTime(in.UpdatedAt)
	if err != nil {
		return nil, err
	}

	err = i.indexerStore.UpdateOwner(ctx, in.IndexID, in.Owner, updatedAt)
	if err != nil {
		return nil, err
	}

	return &pb.Empty{}, nil
}

// UpdateOwnerForFungibleToken updates the owner of a fungible token
func (i *IndexerServer) UpdateOwnerForFungibleToken(ctx context.Context, in *pb.UpdateOwnerForFungibleTokenRequest) (*pb.Empty, error) {
	lockedTime, err := indexerGRPCSDK.ParseTime(in.LockedTime)
	if err != nil {
		return nil, err
	}

	err = i.indexerStore.UpdateOwnerForFungibleToken(ctx, in.IndexID, lockedTime, in.To, in.Total)
	if err != nil {
		return nil, err
	}

	return &pb.Empty{}, nil
}

// IndexAccountTokens indexes the Account tokens of an account
func (i *IndexerServer) IndexAccountTokens(ctx context.Context, in *pb.IndexAccountTokensRequest) (*pb.Empty, error) {
	accountTokens, err := i.mapper.MapGRPCAccountTokensToIndexerAccountTokens(in.AccountTokens)
	if err != nil {
		return nil, err
	}

	err = i.indexerStore.IndexAccountTokens(ctx, in.Owner, accountTokens)
	if err != nil {
		return nil, err
	}

	return &pb.Empty{}, nil
}

// GetDetailedToken returns a detailed token by index ID
func (i *IndexerServer) GetDetailedToken(ctx context.Context, indexID *pb.IndexID) (*pb.DetailedToken, error) {
	detailedToken, err := i.indexerStore.GetDetailedToken(ctx, indexID.IndexID)
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

// GetOwnersByBlockchainsAndContracts returns owners by blockchains and contracts
func (i *IndexerServer) GetOwnersByBlockchainsAndContracts(ctx context.Context, request *pb.GetOwnersByBlockchainsAndContractsRequest) (*pb.Addresses, error) {
	blockchainContract := make(map[string][]string)

	for k, v := range request.BlockchainContracts {
		blockchainContract[k] = v.Addresses
	}

	owners, err := i.indexerStore.GetOwnersByBlockchainAndContract(ctx, blockchainContract)
	if err != nil {
		return nil, err
	}

	return &pb.Addresses{Addresses: owners}, nil
}
