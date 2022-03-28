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
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/cadence"
)

type NFTEventSubscriber struct {
	wallet        *ethereum.Wallet
	wsClient      *ethclient.Client
	store         indexer.IndexerStore
	cadenceWorker cadence.CadenceWorkerClient

	ethLogChan      chan types.Log
	ethSubscription *goethereum.Subscription
}

func New(wallet *ethereum.Wallet,
	wsClient *ethclient.Client,
	store indexer.IndexerStore,
	cadenceWorker cadence.CadenceWorkerClient) *NFTEventSubscriber {
	return &NFTEventSubscriber{
		wallet:        wallet,
		wsClient:      wsClient,
		store:         store,
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

			indexID := fmt.Sprintf("%s-%s-%s", indexer.BlockchianAlias[indexer.EthereumBlockchain], contractAddress, log.Topics[3].Big().Text(16))

			tokens, err := s.store.GetTokensByIndexIDs(ctx, []string{indexID})
			if err != nil {
				logrus.WithError(err).Error("fail to get a token by index ID")
			}
			if len(tokens) != 0 {
				logrus.WithField("indexID", indexID).Info("a token found for a corresponded event")
			} else {
				continue
			}

			token := tokens[0]
			if err := s.store.PushProvenance(ctx, indexID, token.LastRefreshedTime, indexer.Provenance{
				Type:       "transfer",
				FromOwner:  fromAddress,
				Owner:      toAddress,
				Blockchain: indexer.EthereumBlockchain,
				Timestamp:  timestamp,
				TxID:       log.TxHash.Hex(),
			}); err != nil {
				logrus.WithError(err).Warn("unable to push provenance, will trigger a full provenance refresh")
				go s.startRefreshProvenanceWorkflow(ctx, fmt.Sprintf("subscriber-%s", indexID), []string{indexID}, 0)
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
