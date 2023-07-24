package ens

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/wealdtech/go-ens/v3"
	"go.uber.org/zap"

	logger "github.com/bitmark-inc/autonomy-logger"
)

type ENS struct {
	rpcEndpoint string
	rpcClient   *ethclient.Client
}

func New(rpcEndpoint string) *ENS {
	client, err := ethclient.Dial(rpcEndpoint)
	if err != nil {
		logger.Fatal("fail to initiate ETH client", zap.Error(err))
	}

	return &ENS{
		rpcEndpoint: rpcEndpoint,
		rpcClient:   client,
	}
}

func (e *ENS) ResolveDomain(accountNumber string) (string, error) {
	return ens.ReverseResolve(e.rpcClient, common.HexToAddress(accountNumber))
}
