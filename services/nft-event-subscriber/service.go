package main

import (
	"context"
	"fmt"
	"math/big"
	"time"

	goethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	cadenceClient "go.uber.org/cadence/client"

	ethereum "github.com/bitmark-inc/account-vault-ethereum"
	"github.com/bitmark-inc/autonomy-account/storage"
	notification "github.com/bitmark-inc/autonomy-notification/sdk"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
)

type NFTEventSubscriber struct {
	network string

	wallet        *ethereum.Wallet
	wsClient      *ethclient.Client
	store         indexer.IndexerStore
	opensea       *opensea.OpenseaClient
	accountStore  *storage.AccountInformationStorage
	notification  *notification.NotificationClient
	cadenceWorker cadence.CadenceWorkerClient

	ethLogChan      chan types.Log
	ethSubscription *goethereum.Subscription
}

func New(wallet *ethereum.Wallet,
	network string,
	wsClient *ethclient.Client,
	store indexer.IndexerStore,
	accountStore *storage.AccountInformationStorage,
	opensea *opensea.OpenseaClient,
	notification *notification.NotificationClient,
	cadenceWorker cadence.CadenceWorkerClient) *NFTEventSubscriber {
	return &NFTEventSubscriber{
		network:       network,
		wallet:        wallet,
		wsClient:      wsClient,
		store:         store,
		opensea:       opensea,
		accountStore:  accountStore,
		notification:  notification,
		cadenceWorker: cadenceWorker,
	}
}

func (s *NFTEventSubscriber) startRefreshProvenanceWorkflow(c context.Context, refreshProvenanceTaskID string, indexIDs []string, delay time.Duration) {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           fmt.Sprintf("index-token-%s-provenance", refreshProvenanceTaskID),
		TaskList:                     indexerWorker.TaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w indexerWorker.NFTIndexerWorker

	workflow, err := s.cadenceWorker.StartWorkflow(c, indexerWorker.ClientName, workflowContext, w.RefreshTokenProvenanceWorkflow, indexIDs, delay)
	if err != nil {
		logrus.WithError(err).WithField("refreshProvenanceTaskID", refreshProvenanceTaskID).Error("fail to start refreshing provenance workflow")
	} else {
		logrus.WithField("refreshProvenanceTaskID", refreshProvenanceTaskID).WithField("workflow_id", workflow.ID).Info("start workflow for refreshing provenance")
	}
}

func (s *NFTEventSubscriber) Subscribe(ctx context.Context) error {
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

			indexID := fmt.Sprintf("%s-%s-%s", indexer.BlockchianAlias[indexer.EthereumBlockchain], contractAddress, tokenIDHash.Big().Text(16))

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

			// TODO: do we need to move this account specific function out of this service
			accounts, err := s.accountStore.GetAccountIDByAddress(toAddress)
			if err != nil {
				return err
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
					a, err := s.opensea.RetrieveAsset(contractAddress, tokenIDHash.Big().Text(10))
					if err != nil {
						logrus.WithError(err).Error("fail to get the token data from opensea")
						continue
					}

					update, err := indexer.IndexETHToken(a)
					if err != nil {
						logrus.WithError(err).Error("fail to generate index data")
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
			if err := s.store.PushProvenance(ctx, indexID, token.LastRefreshedTime, indexer.Provenance{
				Type:        "transfer",
				FormerOwner: &fromAddress,
				Owner:       toAddress,
				Blockchain:  indexer.EthereumBlockchain,
				Timestamp:   timestamp,
				TxID:        txID,
				TxURL:       indexer.TxURL(indexer.EthereumBlockchain, s.network, txID),
			}); err != nil {
				logrus.WithError(err).Warn("unable to push provenance, will trigger a full provenance refresh")
				go s.startRefreshProvenanceWorkflow(ctx, fmt.Sprintf("subscriber-%s", indexID), []string{indexID}, 0)
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
				Debug("not a valid nft transfer event, expect topic length to be 4")
		}
	}

	return nil
}

func (s *NFTEventSubscriber) Close() {
	if s.ethSubscription != nil {
		(*s.ethSubscription).Unsubscribe()
		close(s.ethLogChan)
	}
}
