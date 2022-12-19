package indexer

import (
	"context"
	"fmt"

	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
)

func PushGRPCEvent(ctx context.Context, grpcClient processor.EventProcessorClient, eventType, fromAddress, toAddress, contractAddress, blockchain, tokenID string) error {
	eventInput := processor.EventInput{
		EventType:  eventType,
		Blockchain: blockchain,
		Contract:   contractAddress,
		From:       fromAddress,
		To:         toAddress,
		TokenID:    tokenID,
	}

	r, err := grpcClient.PushEvent(ctx, &eventInput)
	if err != nil {
		panic(err)
	}

	if r.Status != 200 {
		return fmt.Errorf("gRPC response status not 200 ")
	}

	return nil
}
