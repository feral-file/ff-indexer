package main

import (
	"context"
	"errors"
	"time"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"

	indexer "github.com/feral-file/ff-indexer"
	indexerWorker "github.com/feral-file/ff-indexer/background/worker"
)

// updateLatestOwner updates the latest owner of an existent token
func (e *EventProcessor) updateLatestOwner(ctx context.Context, event NFTEvent) error {
	eventType := event.Type
	blockchain := event.Blockchain
	contract := event.Contract
	tokenID := event.TokenID
	to := event.To
	indexID := indexer.TokenIndexID(blockchain, contract, tokenID)

	switch event.Type {
	case string(NftEventTypeTransfer):
		token, err := e.grpcGateway.GetTokenByIndexID(ctx, indexID)
		if err != nil {
			if grpcError, ok := status.FromError(err); !ok || grpcError.Message() != "token does not exist" {
				log.ErrorWithContext(ctx, errors.New("fail to query token from indexer"), zap.Error(err))
				return err
			}
		}

		if token != nil {
			if !token.Fungible {
				err := e.grpcGateway.PushProvenance(ctx, indexID, token.LastRefreshedTime, indexer.Provenance{
					Type:        eventType,
					FormerOwner: &event.From,
					Owner:       to,
					Blockchain:  blockchain,
					Timestamp:   event.TXTime,
					TxID:        event.TXID,
					TxURL:       indexer.TxURL(event.Blockchain, e.environment, event.TXID),
				})

				if err != nil {
					log.ErrorWithContext(ctx, errors.New("fail to push provenance"), zap.Error(err))

					err = e.grpcGateway.UpdateOwner(ctx, indexID, to, event.CreatedAt)
					if err != nil {
						log.ErrorWithContext(ctx, errors.New("fail to update owner"), zap.Error(err))
						return err
					}
				}

				accountToken := indexer.AccountToken{
					BaseTokenInfo:     token.BaseTokenInfo,
					IndexID:           indexID,
					OwnerAccount:      to,
					Balance:           int64(1),
					LastActivityTime:  event.CreatedAt,
					LastRefreshedTime: time.Now(),
				}

				if err := e.grpcGateway.IndexAccountTokens(ctx, to, []indexer.AccountToken{accountToken}); err != nil {
					log.ErrorWithContext(ctx, errors.New("fail to index account token"), zap.Error(err))
					return err
				}
			} else {
				// err := e.indexerGRPC.UpdateOwnerForFungibleToken(ctx, indexID, token.LastRefreshedTime, event.To, 1)
				// if err != nil {
				// 	log.Error("fail to update owner for fungible token", zap.Error(err))
				// 	return err
				// }
				log.Debug("ignore instant updates for fungible tokens", zap.String("indexID", indexID))
			}
		} else {
			log.Debug("token not found", zap.String("indexID", indexID))
		}
	default:
		// do nothing here.
	}

	return nil
}

// UpdateLatestOwner is a stage 1 worker.
func (e *EventProcessor) UpdateLatestOwner(ctx context.Context) {
	e.StartNftEventWorker(ctx,
		NftEventStageInit, NftEventStageFullSync,
		[]NftEventType{NftEventTypeTransfer, NftEventTypeMint, NftEventTypeBurned},
		0, 0, e.updateLatestOwner,
	)
}

// updateOwnerAndProvenance updates the owner and provenance of a token.
func (e *EventProcessor) updateOwnerAndProvenance(ctx context.Context, event NFTEvent) error {
	from := event.From
	blockchain := event.Blockchain
	contract := event.Contract
	tokenID := event.TokenID
	to := event.To

	indexID := indexer.TokenIndexID(blockchain, contract, tokenID)
	token, err := e.grpcGateway.GetTokenByIndexID(ctx, indexID)
	if err != nil {
		if grpcError, ok := status.FromError(err); !ok || grpcError.Message() != "token does not exist" {
			log.ErrorWithContext(ctx, errors.New("fail to query token from indexer"), zap.Error(err))
			return err
		}
	}

	if token != nil {
		log.Debug("An indexed token found for a corresponded event. Start refreshing the token ownership and provenance", zap.String("indexID", indexID))

		if token.Fungible {
			indexerWorker.StartRefreshTokenOwnershipWorkflow(ctx, e.worker, "processor", indexID, 0)
		} else {
			if err := e.grpcGateway.UpdateOwner(ctx, indexID, to, event.CreatedAt); err != nil {
				log.ErrorWithContext(ctx, errors.New("fail to update the token ownership"),
					zap.String("indexID", indexID), zap.Error(err),
					zap.String("from", from), zap.String("to", to))
			}
			indexerWorker.StartRefreshTokenProvenanceWorkflow(ctx, e.worker, "processor", indexID, 0)
		}
	} else {
		log.InfoWithContext(ctx, "token has not been indexed yet, skipped.", zap.String("indexID", indexID))
		// Do nothing here.
	}
	return nil
}

// UpdateOwnerAndProvenance is a stage 2 worker.
func (e *EventProcessor) UpdateOwnerAndProvenance(ctx context.Context) {
	e.StartNftEventWorker(ctx,
		NftEventStageFullSync, NftEventStageTokenSaleIndexing,
		[]NftEventType{NftEventTypeTransfer, NftEventTypeMint},
		0, 0, e.updateOwnerAndProvenance,
	)
}

// UpdateOwnerAndProvenanceForBurnedToken is a variant stage 2 worker. It ignores sending notification
func (e *EventProcessor) UpdateOwnerAndProvenanceForBurnedToken(ctx context.Context) {
	e.StartNftEventWorker(ctx,
		NftEventStageFullSync, NftEventStageDone,
		[]NftEventType{NftEventTypeBurned},
		0, 0, e.updateOwnerAndProvenance,
	)
}

func (e *EventProcessor) IndexTokenSale(ctx context.Context) {
	e.StartNftEventWorker(
		ctx,
		NftEventStageTokenSaleIndexing, NftEventStageDone,
		[]NftEventType{NftEventTypeTransfer},
		0, 0, e.indexTokenSale,
	)
}

func (e *EventProcessor) indexTokenSale(ctx context.Context, event NFTEvent) error {
	if event.Type != "transfer" {
		log.InfoWithContext(ctx, "ignore non-transfer event", zap.String("type", event.Type))
		return nil
	}
	if event.Blockchain == utils.TezosBlockchain &&
		event.Contract != indexer.TezosOBJKTMarketplaceAddress &&
		event.Contract != indexer.TezosOBJKTMarketplaceAddressV2 {
		log.InfoWithContext(ctx, "ignore non-objkt sale event", zap.String("contract", event.Contract), zap.String("txID", event.TXID))
		return nil
	}

	err := indexerWorker.StartIndexingTokenSale(
		ctx,
		e.worker,
		event.Blockchain,
		event.TXID)
	if nil != err {
		log.ErrorWithContext(ctx, errors.New("fail to start indexing token sale"), zap.Error(err))
	}

	return nil
}
