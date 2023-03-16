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

// PushProvenance pushes a provenance to the indexer
func (i *IndexerServer) PushProvenance(ctx context.Context, in *pb.PushProvenanceRequest) (*pb.Error, error) {
	lockedTime, err := indexerGRPCSDK.ParseTime(in.LockedTime)
	if err != nil {
		return &pb.Error{Exist: true, Message: err.Error()}, err
	}

	provenance := indexerGRPCSDK.MapGRPCProvenancesToIndexerProvenances([]*pb.Provenance{in.Provenance})[0]

	err = i.indexerStore.PushProvenance(
		ctx,
		in.IndexID,
		lockedTime,
		provenance,
	)

	if err != nil {
		return &pb.Error{Exist: true, Message: err.Error()}, err
	}

	return nil, nil
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
