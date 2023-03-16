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
}

// NewIndexerServer creates a new IndexerServer
func NewIndexerServer(
	network string,
	port int,
	indexerStore *indexer.MongodbIndexerStore,
) (*IndexerServer, error) {

	grpcServer := grpc.NewServer()

	return &IndexerServer{
		grpcServer:   grpcServer,
		indexerStore: indexerStore,
		network:      network,
		port:         port,
	}, nil
}

// GetTokensByIndexID returns a token by index ID
func (i *IndexerServer) GetTokensByIndexID(ctx context.Context, indexID *pb.IndexID) (*pb.Token, error) {
	token, err := i.indexerStore.GetTokensByIndexID(ctx, indexID.IndexID)
	if err != nil {
		return nil, err
	}

	pbToken := indexerGRPCSDK.MapIndexerTokenToGrpcToken(token)

	return pbToken, nil
}

// Run starts the IndexerServer
func (i *IndexerServer) Run(context.Context) error {
	listener, err := net.Listen(i.network, fmt.Sprintf("0.0.0.0:%d", i.port))
	if err != nil {
		return err
	}

	pb.RegisterIndexerServer(i.grpcServer, i)
	i.grpcServer.Serve(listener)

	return nil
}
