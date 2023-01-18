package ens

import (
	log "github.com/bitmark-inc/nft-indexer/zapLog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/wealdtech/go-ens/v3"
	"go.uber.org/zap"
)

type ENS struct {
	rpcEndpoint string
	rpcClient   *ethclient.Client
}

func New(rpcEndpoint string) *ENS {
	client, err := ethclient.Dial(rpcEndpoint)
	if err != nil {
		log.Logger.Panic("fail to dial ethereum rpc", zap.Error(err))
	}

	return &ENS{
		rpcEndpoint: rpcEndpoint,
		rpcClient:   client,
	}
}

func (e *ENS) ResolveDomain(accountNumber string) (string, error) {
	resolver, err := ens.NewReverseResolver(e.rpcClient)
	if err != nil {
		return "", err
	}
	return resolver.Name(common.HexToAddress(accountNumber))
}
