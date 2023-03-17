package sdk

import (
	"context"
	"google.golang.org/grpc/credentials/insecure"
	"time"

	"google.golang.org/grpc"

	"github.com/bitmark-inc/nft-indexer"
	pb "github.com/bitmark-inc/nft-indexer/services/nft-indexer-grpc/grpc/indexer"
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

// GetTokensByIndexID returns a token by indexID
func (i *IndexerGRPCClient) GetTokensByIndexID(ctx context.Context, indexID string) (*indexer.Token, error) {
	token, err := i.client.GetTokensByIndexID(ctx, &pb.IndexID{IndexID: indexID})
	if err != nil {
		return nil, err
	}

	return i.mapper.MapGrpcTokenToIndexerToken(token), nil
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
func (i *IndexerGRPCClient) GetDetailedToken(ctx context.Context, indexID string) (indexer.DetailedToken, error) {
	detailedToken, err := i.client.GetDetailedToken(ctx, &pb.IndexID{IndexID: indexID})
	if err != nil {
		return indexer.DetailedToken{}, err
	}

	indexerDetailedToken, err := i.mapper.MapGrpcDetailedTokenToIndexerDetailedToken(detailedToken)
	if err != nil {
		return indexer.DetailedToken{}, err
	}

	return *indexerDetailedToken, nil
}
