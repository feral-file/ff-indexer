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
	network string

	wallet        *ethereum.Wallet
	wsClient      *ethclient.Client
	store         indexer.IndexerStore
	engine        *indexer.IndexEngine
	opensea       *opensea.OpenseaClient
	accountStore  *storage.AccountInformationStorage
	notification  *notification.NotificationClient
	cadenceWorker cadence.CadenceWorkerClient

	bitmarkListener *Listener

	ethLogChan      chan types.Log
	ethSubscription *goethereum.Subscription
}

func New(wallet *ethereum.Wallet,
	network string,
	wsClient *ethclient.Client,
	store indexer.IndexerStore,
	engine *indexer.IndexEngine,
	accountStore *storage.AccountInformationStorage,
	bitmarkListener *Listener,
	notification *notification.NotificationClient,
	cadenceWorker cadence.CadenceWorkerClient) *NFTEventSubscriber {
	return &NFTEventSubscriber{
		network:         network,
		wallet:          wallet,
		wsClient:        wsClient,
		store:           store,
		engine:          engine,
		accountStore:    accountStore,
		bitmarkListener: bitmarkListener,
		notification:    notification,
		cadenceWorker:   cadenceWorker,
	}
}

func (s *NFTEventSubscriber) Close() {
	if s.ethSubscription != nil {
		(*s.ethSubscription).Unsubscribe()
		close(s.ethLogChan)
	}
}
