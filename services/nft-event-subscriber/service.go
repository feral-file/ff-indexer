package main

import (
	goethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	ethereum "github.com/bitmark-inc/account-vault-ethereum"
	"github.com/bitmark-inc/autonomy-account/storage"
	notification "github.com/bitmark-inc/autonomy-notification/sdk"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
)

type NFTEventSubscriber struct {
	environment string

	wallet       *ethereum.Wallet
	wsClient     *ethclient.Client
	store        indexer.IndexerStore
	Engine       *indexer.IndexEngine
	opensea      *opensea.OpenseaClient
	accountStore *storage.AccountInformationStorage
	notification *notification.NotificationClient
	feedServer   *FeedClient
	Worker       cadence.CadenceWorkerClient

	bitmarkListener *Listener

	ethLogChan      chan types.Log
	ethSubscription *goethereum.Subscription
}

func New(wallet *ethereum.Wallet,
	environment string,
	wsClient *ethclient.Client,
	store indexer.IndexerStore,
	engine *indexer.IndexEngine,
	accountStore *storage.AccountInformationStorage,
	bitmarkListener *Listener,
	notification *notification.NotificationClient,
	feedServer *FeedClient,
	cadenceWorker cadence.CadenceWorkerClient) *NFTEventSubscriber {
	return &NFTEventSubscriber{
		environment:     environment,
		wallet:          wallet,
		wsClient:        wsClient,
		store:           store,
		Engine:          engine,
		accountStore:    accountStore,
		bitmarkListener: bitmarkListener,
		notification:    notification,
		feedServer:      feedServer,
		Worker:          cadenceWorker,
	}
}

func (s *NFTEventSubscriber) Close() {
	if s.ethSubscription != nil {
		(*s.ethSubscription).Unsubscribe()
		close(s.ethLogChan)
	}
}
