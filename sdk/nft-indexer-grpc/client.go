package sdk

import (
	"context"
	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc"

	"github.com/bitmark-inc/nft-indexer"
	pb "github.com/bitmark-inc/nft-indexer/services/nft-indexer-grpc/grpc/indexer"
)

type IndexerGRPCClient struct {
	client pb.IndexerClient
	mapper *Mapper
}

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

func (i *IndexerGRPCClient) GetTokensByIndexID(ctx context.Context, indexID string) (*indexer.Token, error) {
	token, err := i.client.GetTokensByIndexID(ctx, &pb.IndexID{IndexID: indexID})
	if err != nil {
		return nil, err
	}

	return i.mapper.MapGrpcTokenToIndexerToken(token), nil
}
