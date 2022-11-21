package main

import (
	"context"
	"math/big"
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	goethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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
				logrus.WithError(err).Error("fail to start subscription connection")
				time.Sleep(time.Second)
				continue
			}

			s.ethSubscription = &subscription
			err = <-subscription.Err()
			logrus.WithError(err).Error("subscription stopped with failure")
		}
	}()

	logrus.Info("start watching blockchain events")
	go func() {
		for eLog := range s.ethLogChan {
			paringStartTime := time.Now()
			logrus.WithField("txHash", eLog.TxHash).
				WithField("logIndex", eLog.Index).
				WithField("time", paringStartTime).
				Debug("start processing ethereum log")
			timestamp, err := getBlockTime(ctx, s.wallet.RPCClient(), eLog.BlockHash)
			logrus.WithField("txHash", eLog.TxHash).
				WithField("logIndex", eLog.Index).
				WithField("delay", time.Since(paringStartTime)).Debug("get block time")

			if err != nil {
				logrus.WithError(err).Error("fail to get block time")
				continue
			}

			if topicLen := len(eLog.Topics); topicLen == 4 {

				fromAddress := indexer.EthereumChecksumAddress(eLog.Topics[1].Hex())
				toAddress := indexer.EthereumChecksumAddress(eLog.Topics[2].Hex())
				contractAddress := indexer.EthereumChecksumAddress(eLog.Address.String())
				tokenIDHash := eLog.Topics[3]

				logrus.WithField("from", fromAddress).
					WithField("to", toAddress).
					WithField("contractAddress", contractAddress).
					WithField("tokenIDHash", tokenIDHash).
					Debug("receive transfer event on ethereum")

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
							logrus.WithError(err).Trace("fail to push event to feed server")
						}
					} else {
						if err := s.feedServer.SendEvent(indexer.EthereumBlockchain, contractAddress, tokenIDHash.Big().Text(10), toAddress, mintType, viper.GetString("network.ethereum") == "testnet"); err != nil {
							logrus.WithError(err).Trace("fail to push event to feed server")
						}
					}
				}()

				logrus.WithField("txHash", eLog.TxHash).
					WithField("logIndex", eLog.Index).
					WithField("delay", time.Since(paringStartTime)).Debug("feed event sent")

				// TODO: do we need to move this account specific function out of this service
				accounts, err := s.accountStore.GetAccountIDByAddress(toAddress)
				if err != nil {
					logrus.WithError(err).Error("fail to get accounts that watches this address")
					continue
				}

				logrus.WithField("txHash", eLog.TxHash).
					WithField("logIndex", eLog.Index).
					WithField("delay", time.Since(paringStartTime)).Debug("check related account")

				tokens, err := s.store.GetTokensByIndexIDs(ctx, []string{indexID})
				if err != nil {
					logrus.WithError(err).Error("fail to get a token by index ID")
				}
				if len(tokens) != 0 {
					logrus.WithField("indexID", indexID).Info("a token found for a corresponded event")
				} else {
					// index the new token since it is a new token for our indexer and watched by our user
					if len(accounts) > 0 {
						update, err := s.Engine.IndexETHToken(ctx, toAddress, contractAddress, tokenIDHash.Big().Text(10))
						if err != nil {
							logrus.WithError(err).Error("fail to generate index data")
							continue
						}

						if update == nil { // ignored updates
							continue
						}

						if err := s.store.IndexAsset(ctx, update.ID, *update); err != nil {
							logrus.WithError(err).Error("fail to index token in to db")
							continue
						}

						tokens, err = s.store.GetTokensByIndexIDs(ctx, []string{indexID})
						if err != nil || len(tokens) == 0 {
							logrus.WithError(err).Error("token is not successfully indexed")
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
						logrus.WithError(err).Warn("unable to push provenance, will trigger a full provenance refresh")
						if err := s.UpdateOwner(ctx, indexID, toAddress, timestamp); err != nil {
							logrus.
								WithField("indexID", indexID).WithError(err).
								WithField("from", fromAddress).WithField("to", toAddress).
								Error("fail to update the token owner for the event")
						}

						go indexerWorker.StartRefreshTokenProvenanceWorkflow(ctx, &s.Worker, "subscriber", indexID, 0)
					}
				}

				// send notification in the end
				for _, accountID := range accounts {
					if err := s.notifyNewNFT(accountID, toAddress, indexID); err != nil {
						logrus.WithError(err).
							WithField("accountID", accountID).WithField("indexID", indexID).
							Error("fail to send notification for the new token")
					}
				}

			} else {
				logrus.WithField("topicLen", topicLen).
					WithField("log", eLog).
					Trace("not a valid nft transfer event, expect topic length to be 4")
			}
			logrus.WithField("txHash", eLog.TxHash).
				WithField("logIndex", eLog.Index).
				WithField("delay", time.Since(paringStartTime)).Debug("end processing ethereum log")

			logrus.WithField("chanLen", len(s.ethLogChan)).Debug("channel counts")
		}
	}()

	return nil
}
