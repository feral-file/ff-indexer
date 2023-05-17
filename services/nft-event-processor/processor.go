package main

import (
	"context"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
	"github.com/bitmark-inc/nft-indexer/log"
)

func (e *EventProcessor) updateLatestOwner(ctx context.Context, event NFTEvent) error {
	eventType := event.Type
	blockchain := event.Blockchain
	contract := event.Contract
	tokenID := event.TokenID
	to := event.To
	indexID := indexer.TokenIndexID(blockchain, contract, tokenID)

	switch event.Type {
	case "mint":
		// do nothing here.
	default:
		token, err := e.indexerStore.GetTokensByIndexID(ctx, indexID)
		if err != nil {
			log.Error("fail to get token by index id", zap.Error(err))
			return err
		}

		if token != nil {
			if !token.Fungible {
				err := e.indexerStore.PushProvenance(ctx, indexID, token.LastRefreshedTime, indexer.Provenance{
					Type:        eventType,
					FormerOwner: &event.From,
					Owner:       to,
					Blockchain:  blockchain,
					Timestamp:   event.CreatedAt,
					TxID:        "",
					TxURL:       "",
				})

				if err != nil {
					log.Error("fail to push provenance", zap.Error(err))

					err = e.indexerStore.UpdateOwner(ctx, indexID, to, event.CreatedAt)
					if err != nil {
						log.Error("fail to update owner", zap.Error(err))
						return err
					}
				}
			} else {
				err := e.indexerStore.UpdateOwnerForFungibleToken(ctx, indexID, token.LastRefreshedTime, event.To, 1)
				if err != nil {
					log.Error("fail to update owner for fungible token", zap.Error(err))
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

			if err := e.indexerStore.IndexAccountTokens(ctx, to, []indexer.AccountToken{accountToken}); err != nil {
				log.Error("fail to index account token", zap.Error(err))
				return err
			}
		} else {
			log.Debug("token not found", zap.String("indexID", indexID))
		}

	}

	return nil
}

// UpdateLatestOwner [stage 1] update owner for nft and ft by event information
func (e *EventProcessor) UpdateLatestOwner(ctx context.Context) {
	e.StartWorker(ctx,
		1, 2,
		[]EventType{EventTypeTransfer, EventTypeMint},
		e.updateLatestOwner,
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

	accounts, err := e.accountStore.GetAccountIDByAddress(to)
	if err != nil {
		log.Error("fail to check accounts by address", zap.Error(err))
		return err
	}

	indexID := indexer.TokenIndexID(blockchain, contract, tokenID)
	token, err := e.indexerStore.GetTokensByIndexID(ctx, indexID)
	if err != nil {
		log.Error("fail to check token by index ID", zap.Error(err))
		return err
	}

	// check if a token is existent
	// if existent, update provenance
	// if not, index it by blockchain
	if token != nil {
		// ignore the indexing process since an indexed token found
		log.Debug("an indexed token found for a corresponded event", zap.String("indexID", indexID))

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

			if err := e.indexerStore.IndexAccountTokens(ctx, to, []indexer.AccountToken{accountToken}); err != nil {
				log.Error("cannot index a new account_token", zap.Error(err), zap.String("indexID", indexID), zap.String("owner", to))
				return err
			}
		}

		if token.Fungible {
			indexerWorker.StartRefreshTokenOwnershipWorkflow(ctx, e.worker, "processor", indexID, 0)
		} else {
			if err := e.indexerStore.UpdateOwner(ctx, indexID, to, event.CreatedAt); err != nil {
				log.Error("fail to update the token ownership",
					zap.String("indexID", indexID), zap.Error(err),
					zap.String("from", from), zap.String("to", to))

			}
			indexerWorker.StartRefreshTokenProvenanceWorkflow(ctx, e.worker, "processor", indexID, 0)
		}
	} else {
		// index the new token since it is a new token for our indexer and watched by our user
		if len(accounts) > 0 {
			log.Info("start indexing a new token",
				zap.String("indexID", indexID),
				zap.String("from", from), zap.String("to", to))

			indexerWorker.StartIndexTokenWorkflow(ctx, e.worker, to, contract, tokenID, true, false)
		}
	}
	return nil
}

// UpdateOwnerAndProvenance trigger cadence to update owner and provenance of token
func (e *EventProcessor) UpdateOwnerAndProvenance(ctx context.Context) {
	e.StartWorker(ctx,
		2, 3,
		[]EventType{EventTypeTransfer, EventTypeMint},
		e.updateOwnerAndProvenance,
	)
}

// NotifyChangeTokenOwner send notification to notificationSdk
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
			log.Error("fail to send notificationSdk for the new update",
				zap.Error(err),
				zap.String("accountID", accountID), zap.String("indexID", indexID))
			return err
		}
	}
	return nil
}

// NotifyChangeTokenOwner send notification to notificationSdk
func (e *EventProcessor) NotifyChangeTokenOwner(ctx context.Context) {
	e.StartWorker(ctx,
		3, 4,
		[]EventType{EventTypeTransfer, EventTypeMint},
		e.notifyChangeTokenOwner,
	)
}

func (e *EventProcessor) sendEventToFeedServer(ctx context.Context, event NFTEvent) error {
	blockchain := event.Blockchain
	contract := event.Contract
	tokenID := event.TokenID
	to := event.To
	eventType := event.Type

	return e.feedServer.SendEvent(ctx, blockchain, contract, tokenID, to, eventType,
		viper.GetString("network.ethereum") == "testnet")
}

// SendEventToFeedServer send event to feed server
func (e *EventProcessor) SendEventToFeedServer(ctx context.Context) {
	e.StartWorker(ctx,
		4, 0,
		[]EventType{EventTypeTransfer, EventTypeMint},
		e.sendEventToFeedServer,
	)
}
