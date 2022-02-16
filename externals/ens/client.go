package ens

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	"github.com/wealdtech/go-ens/v3"
)

type ENS struct {
	rpcEndpoint string
	rpcClient   *ethclient.Client
}

func New(rpcEndpoint string) *ENS {
	client, err := ethclient.Dial(rpcEndpoint)
	if err != nil {
		logrus.WithError(err).Panic("fail to dial ethereum rpc")
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
