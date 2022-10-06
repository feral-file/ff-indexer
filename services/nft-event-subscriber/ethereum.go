package main

import (
	"context"
	"fmt"
	"math/big"

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

	subscription, err := s.wsClient.SubscribeFilterLogs(ctx, goethereum.FilterQuery{Topics: [][]common.Hash{
		{common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")}, // transfer event
	}}, s.ethLogChan)
	if err != nil {
		return err
	}

	s.ethSubscription = &subscription

	go func() {
		err := <-subscription.Err()
		logrus.WithError(err).Error("subscription failed")
		close(s.ethLogChan)
	}()

	logrus.Info("start watching blockchain events")
	go func() {
		for log := range s.ethLogChan {
			timestamp, err := getBlockTime(ctx, s.wallet.RPCClient(), log.BlockHash)
			if err != nil {
				logrus.WithError(err).Error("fail to get block time")
				continue
			}

			if topicLen := len(log.Topics); topicLen == 4 {
				if log.Topics[1].Big().Cmp(big.NewInt(0)) == 0 {
					// ignore minting events
					continue
				}

				fromAddress := indexer.EthereumChecksumAddress(log.Topics[1].Hex())
				toAddress := indexer.EthereumChecksumAddress(log.Topics[2].Hex())
				contractAddress := indexer.EthereumChecksumAddress(log.Address.String())
				tokenIDHash := log.Topics[3]

				indexID := indexer.TokenIndexID(indexer.EthereumBlockchain, contractAddress, tokenIDHash.Big().Text(16))
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
				//       - send notificiation

				mintType := "transfer"
				if fromAddress == indexer.EthereumZeroAddress {
					mintType = "mint"
				}

				if toAddress == indexer.EthereumZeroAddress {
					if err := s.feedServer.SendBurn(indexer.EthereumBlockchain, contractAddress, tokenIDHash.Big().Text(10)); err != nil {
						logrus.WithError(err).Error("fail to push event to feed server")
					}
				} else {
					if err := s.feedServer.SendEvent(indexer.EthereumBlockchain, contractAddress, tokenIDHash.Big().Text(10), toAddress, mintType, viper.GetString("network.ethereum") == "testnet"); err != nil {
						logrus.WithError(err).Error("fail to push event to feed server")
					}
				}

				// TODO: do we need to move this account specific function out of this service
				accounts, err := s.accountStore.GetAccountIDByAddress(toAddress)
				if err != nil {
					logrus.WithError(err).Error("fail to get accounts that watches this address")
					continue
				}

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

				txID := log.TxHash.Hex()

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
						go indexerWorker.StartRefreshTokenProvenanceWorkflow(ctx, &s.Worker,
							fmt.Sprintf("subscriber-%s", indexID), indexID, 0)
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
					WithField("log", log).
					Trace("not a valid nft transfer event, expect topic length to be 4")
			}
		}
	}()

	return nil
}
