package main

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/status"

	log "github.com/bitmark-inc/autonomy-logger"
	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
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
	case string(EventTypeTransfer):
		token, err := e.indexerGRPC.GetTokenByIndexID(ctx, indexID)
		if err != nil {
			if grpcError, ok := status.FromError(err); !ok || grpcError.Message() != "token does not exist" {
				log.Error("fail to query token from indexer", zap.Error(err))
				return err
			}
		}

		if token != nil {
			if !token.Fungible {
				err := e.indexerGRPC.PushProvenance(ctx, indexID, token.LastRefreshedTime, indexer.Provenance{
					Type:        eventType,
					FormerOwner: &event.From,
					Owner:       to,
					Blockchain:  blockchain,
					Timestamp:   event.TXTime,
					TxID:        event.TXID,
					TxURL:       indexer.TxURL(event.Blockchain, e.environment, event.TXID),
				})

				if err != nil {
					log.Error("fail to push provenance", zap.Error(err))

					err = e.indexerGRPC.UpdateOwner(ctx, indexID, to, event.CreatedAt)
					if err != nil {
						log.Error("fail to update owner", zap.Error(err))
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

				if err := e.indexerGRPC.IndexAccountTokens(ctx, to, []indexer.AccountToken{accountToken}); err != nil {
					log.Error("fail to index account token", zap.Error(err))
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
	e.StartWorker(ctx,
		1, 2,
		[]EventType{EventTypeTransfer, EventTypeMint, EventTypeBurned},
		0, 0, e.updateLatestOwner,
	)
}

// updateOwnerAndProvenance checks if a token is related to Autonomy. If so, it updates its
// owner and refresh the provenance
func (e *EventProcessor) updateOwnerAndProvenance(ctx context.Context, event NFTEvent) error {
	from := event.From
	blockchain := event.Blockchain
	contract := event.Contract
	tokenID := event.TokenID
	to := event.To

	// FIXME: Switch to account server GRPC
	accounts, err := e.accountStore.GetAccountIDByAddress(to)
	if err != nil {
		log.Error("fail to check accounts by address", zap.Error(err))
		return err
	}

	indexID := indexer.TokenIndexID(blockchain, contract, tokenID)
	token, err := e.indexerGRPC.GetTokenByIndexID(ctx, indexID)
	if err != nil {
		if grpcError, ok := status.FromError(err); !ok || grpcError.Message() != "token does not exist" {
			log.Error("fail to query token from indexer", zap.Error(err))
			return err
		}
	}

	// check if a token is existent
	// if existent, update provenance
	// if not, index it by blockchain
	if token != nil {
		// ignore the indexing process since an indexed token found
		log.Debug("an indexed token found for a corresponded event", zap.String("indexID", indexID))

		if token.Fungible {
			indexerWorker.StartRefreshTokenOwnershipWorkflow(ctx, e.worker, "processor", indexID, 0)
		} else {
			// if the new owner is not existent in our system, index a new account_token
			if len(accounts) == 0 {
				accountToken := indexer.AccountToken{
					BaseTokenInfo:     token.BaseTokenInfo,
					IndexID:           indexID,
					OwnerAccount:      to,
					Balance:           int64(1),
					LastActivityTime:  event.CreatedAt,
					LastRefreshedTime: time.Now(),
				}

				if err := e.indexerGRPC.IndexAccountTokens(ctx, to, []indexer.AccountToken{accountToken}); err != nil {
					log.Error("cannot index a new account_token", zap.Error(err), zap.String("indexID", indexID), zap.String("owner", to))
					return err
				}
			}

			if err := e.indexerGRPC.UpdateOwner(ctx, indexID, to, event.CreatedAt); err != nil {
				log.Error("fail to update the token ownership",
					zap.String("indexID", indexID), zap.Error(err),
					zap.String("from", from), zap.String("to", to))
			}
			indexerWorker.StartRefreshTokenProvenanceWorkflow(ctx, e.worker, "processor", indexID, 0)
		}
	} else {
		// index the new token since it is a new token send to our watched user
		if len(accounts) > 0 {
			log.Info("start indexing a new token",
				zap.String("indexID", indexID),
				zap.String("from", from), zap.String("to", to))

			indexerWorker.StartIndexTokenWorkflow(ctx, e.worker, to, contract, tokenID, true, false)
		}
	}
	return nil
}

// UpdateOwnerAndProvenance is a stage 2 worker.
func (e *EventProcessor) UpdateOwnerAndProvenance(ctx context.Context) {
	e.StartWorker(ctx,
		2, 3,
		[]EventType{EventTypeTransfer, EventTypeMint},
		0, 0, e.updateOwnerAndProvenance,
	)
}

// UpdateOwnerAndProvenanceForBurnedToken is a variant stage 2 worker. It ignores sending notification
func (e *EventProcessor) UpdateOwnerAndProvenanceForBurnedToken(ctx context.Context) {
	e.StartWorker(ctx,
		2, 4,
		[]EventType{EventTypeBurned},
		0, 0, e.updateOwnerAndProvenance,
	)
}

// notifyChangeTokenOwner send notifications to related account ids.
func (e *EventProcessor) notifyChangeTokenOwner(_ context.Context, event NFTEvent) error {
	blockchain := event.Blockchain
	contract := event.Contract
	tokenID := event.TokenID
	to := event.To

	accounts, err := e.accountStore.GetAccountIDByAddress(to)
	if err != nil {
		log.Error("fail to check accounts by address", zap.Error(err))
		return err
	}
	indexID := indexer.TokenIndexID(blockchain, contract, tokenID)

	for _, accountID := range accounts {
		if err := e.notifyChangeOwner(accountID, to, indexID); err != nil {
			log.Error("fail to send notification for the new owner update",
				zap.Error(err),
				zap.String("accountID", accountID), zap.String("indexID", indexID))
			return err
		}
	}
	return nil
}

// NotifyChangeTokenOwner is a stage 3 worker.
func (e *EventProcessor) NotifyChangeTokenOwner(ctx context.Context) {
	e.StartWorker(ctx,
		3, 4,
		[]EventType{EventTypeTransfer, EventTypeMint},
		0, 0, e.notifyChangeTokenOwner,
	)
}

// sendEventToFeedServer sends the new processed event to feed server
func (e *EventProcessor) sendEventToFeedServer(ctx context.Context, event NFTEvent) error {
	blockchain := event.Blockchain
	contract := event.Contract
	tokenID := event.TokenID
	to := event.To
	eventType := event.Type

	return e.feedServer.SendEvent(ctx, blockchain, contract, tokenID, to, eventType,
		e.environment == indexer.DevelopmentEnvironment)
}

// SendEventToFeedServer is a stage 4 worker.
func (e *EventProcessor) SendEventToFeedServer(ctx context.Context) {
	e.StartWorker(ctx,
		4, 0,
		[]EventType{EventTypeTransfer, EventTypeMint, EventTypeBurned},
		0, 0, e.sendEventToFeedServer,
	)
}
