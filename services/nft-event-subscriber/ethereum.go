package main

import (
	"context"
	"math/big"
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/log"
	goethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// FIXME: the process is not thread-safe ensure it run only once.
// WatchEthereumEvent start subscribing new transfer events from Ethereum blockchain
// It index new token and push provenances if necessary
func (s *NFTEventSubscriber) WatchEthereumEvent(ctx context.Context) error {
	s.ethLogChan = make(chan types.Log, 100)

	go func() {
		for {
			subscription, err := s.wsClient.SubscribeFilterLogs(ctx, goethereum.FilterQuery{Topics: [][]common.Hash{
				{common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")}, // transfer event
			}}, s.ethLogChan)
			if err != nil {
				log.Error("fail to start subscription connection", zap.Error(err), log.SourceETHClient)
				time.Sleep(time.Second)
				continue
			}

			s.ethSubscription = &subscription
			err = <-subscription.Err()
			log.Error("subscription stopped with failure", zap.Error(err), log.SourceETHClient)
		}
	}()

	log.Info("start watching blockchain events")
	go func() {
		for eLog := range s.ethLogChan {
			paringStartTime := time.Now()
			log.Debug("start processing ethereum log",
				zap.Any("txHash", eLog.TxHash),
				zap.Uint("logIndex", eLog.Index),
				zap.Time("time", paringStartTime))
			timestamp, err := indexer.GetETHBlockTime(ctx, s.wallet.RPCClient(), eLog.BlockHash)
			log.Debug("get block time",
				zap.Any("txHash", eLog.TxHash),
				zap.Uint("logIndex", eLog.Index),
				zap.Duration("delay", time.Since(paringStartTime)))

			if err != nil {
				log.Error("fail to get block time", zap.Error(err))
				continue
			}

			if topicLen := len(eLog.Topics); topicLen == 4 {

				fromAddress := indexer.EthereumChecksumAddress(eLog.Topics[1].Hex())
				toAddress := indexer.EthereumChecksumAddress(eLog.Topics[2].Hex())
				contractAddress := indexer.EthereumChecksumAddress(eLog.Address.String())
				tokenIDHash := eLog.Topics[3]

				log.Debug("receive transfer event on ethereum",
					zap.String("from", fromAddress),
					zap.String("to", toAddress),
					zap.String("contractAddress", contractAddress),
					zap.Any("tokenIDHash", tokenIDHash))

				if eLog.Topics[1].Big().Cmp(big.NewInt(0)) == 0 {
					// ignore minting events
					continue
				}

				indexID := indexer.TokenIndexID(indexer.EthereumBlockchain, contractAddress, tokenIDHash.Big().Text(10))
				// Flow:
				// 1. Check if the destination address is followed
				// 2. Check if the token is indexed
				//   - if indexed:
				//	   - update provenance
				//     - send notification if there is any follower
				//   - if not indexed:
				//     - if there is any follower:
				// 		 - index the token
				//       - update provenance
				//       - send notification

				mintType := "transfer"
				if fromAddress == indexer.EthereumZeroAddress {
					mintType = "mint"
				}

				go func() {
					if toAddress == indexer.EthereumZeroAddress {
						if err := s.feedServer.SendBurn(indexer.EthereumBlockchain, contractAddress, tokenIDHash.Big().Text(10)); err != nil {
							log.Debug("fail to push event to feed server", zap.Error(err))

						}
					} else {
						if err := s.feedServer.SendEvent(indexer.EthereumBlockchain, contractAddress, tokenIDHash.Big().Text(10), toAddress, mintType, viper.GetString("network.ethereum") == "testnet"); err != nil {
							log.Debug("fail to push event to feed server", zap.Error(err))
						}
					}
				}()

				log.Debug("feed event sent",
					zap.Any("txHash", eLog.TxHash),
					zap.Uint("logIndex", eLog.Index),
					zap.Duration("delay", time.Since(paringStartTime)))

				// TODO: do we need to move this account specific function out of this service
				accounts, err := s.accountStore.GetAccountIDByAddress(toAddress)
				if err != nil {
					log.Error("fail to get accounts that watches this address", zap.Error(err))
					continue
				}

				log.Debug("check related account",
					zap.Any("txHash", eLog.TxHash),
					zap.Uint("logIndex", eLog.Index),
					zap.Duration("delay", time.Since(paringStartTime)))

				tokens, err := s.store.GetTokensByIndexIDs(ctx, []string{indexID})
				if err != nil {
					log.Error("fail to get a token by index ID", zap.Error(err))
				}
				if len(tokens) != 0 {
					log.Info("a token found for a corresponded event", zap.String("indexID", indexID))

					// if the new owner is not existent in our system, index a new account_token
					if len(accounts) == 0 {
						accountToken := indexer.AccountToken{
							BaseTokenInfo:     tokens[0].BaseTokenInfo,
							IndexID:           indexID,
							OwnerAccount:      toAddress,
							Balance:           int64(1),
							LastActivityTime:  timestamp,
							LastRefreshedTime: tokens[0].LastActivityTime,
						}

						if err := s.store.IndexAccountTokens(ctx, toAddress, []indexer.AccountToken{accountToken}); err != nil {
							log.Error("cannot index a new account_token", zap.String("indexID", indexID), zap.String("owner", toAddress))
						}
					}
				} else {
					// index the new token since it is a new token for our indexer and watched by our user
					if len(accounts) > 0 {
						update, err := s.Engine.IndexETHToken(ctx, toAddress, contractAddress, tokenIDHash.Big().Text(10))
						if err != nil {
							log.Error("fail to generate index data", zap.Error(err))
							continue
						}

						if update == nil { // ignored updates
							continue
						}

						if err := s.store.IndexAsset(ctx, update.ID, *update); err != nil {
							log.Error("fail to index token in to db", zap.Error(err))
							continue
						}

						accountToken := indexer.AccountToken{
							BaseTokenInfo:     update.Tokens[0].BaseTokenInfo,
							IndexID:           update.Tokens[0].IndexID,
							OwnerAccount:      update.Tokens[0].Owner,
							Balance:           update.Tokens[0].Balance,
							LastActivityTime:  update.Tokens[0].LastActivityTime,
							LastRefreshedTime: update.Tokens[0].LastRefreshedTime,
						}

						if err := s.store.IndexAccountTokens(ctx, update.Tokens[0].Owner, []indexer.AccountToken{accountToken}); err != nil {
							log.Error("fail to index account token to db", zap.Error(err))
							continue
						}

						tokens, err = s.store.GetTokensByIndexIDs(ctx, []string{indexID})
						if err != nil || len(tokens) == 0 {
							log.Error("token is not successfully indexed", zap.Error(err))
							continue
						}
					} else {
						continue
					}
				}

				txID := eLog.TxHash.Hex()

				token := tokens[0]
				if !token.Fungible {
					if err := s.store.PushProvenance(ctx, indexID, token.LastRefreshedTime, indexer.Provenance{
						Type:        "transfer",
						FormerOwner: &fromAddress,
						Owner:       toAddress,
						Blockchain:  indexer.EthereumBlockchain,
						Timestamp:   timestamp,
						TxID:        txID,
						TxURL:       indexer.TxURL(indexer.EthereumBlockchain, s.environment, txID),
					}); err != nil {
						log.Warn("unable to push provenance, will trigger a full provenance refresh", zap.Error(err))
						if err := s.UpdateOwner(ctx, indexID, toAddress, timestamp); err != nil {
							log.Error("fail to update the token owner for the event",
								zap.String("indexID", indexID), zap.Error(err),
								zap.String("from", fromAddress), zap.String("to", toAddress))

						}

						go indexerWorker.StartRefreshTokenProvenanceWorkflow(ctx, &s.Worker, "subscriber", indexID, 0)
					}
				}

				// send notification in the end
				for _, accountID := range accounts {
					if err := s.notifyNewNFT(accountID, toAddress, indexID); err != nil {
						log.Error("fail to send notification for the new token",
							zap.Error(err),
							zap.String("accountID", accountID), zap.String("indexID", indexID))

					}
				}

			} else {
				logrus.WithField("topicLen", topicLen).
					WithField("log", eLog).
					Trace("not a valid nft transfer event, expect topic length to be 4")
			}
			log.Debug("end processing ethereum log",
				zap.Any("txHash", eLog.TxHash),
				zap.Uint("logIndex", eLog.Index),
				zap.Duration("delay", time.Since(paringStartTime)))
			log.Debug("channel counts", zap.Int("chanLen", len(s.ethLogChan)))
		}
	}()

	return nil
}
