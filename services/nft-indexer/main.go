package main

import (
	"context"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/bitmark-inc/config-loader"
	"github.com/bitmark-inc/config-loader/external/aws/ssm"
	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/ens"
	"github.com/bitmark-inc/nft-indexer/externals/feralfile"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	tezosDomain "github.com/bitmark-inc/nft-indexer/externals/tezos-domain"
	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/bitmark-inc/tzkt-go"
)

func main() {
	ctx := context.Background()

	config.LoadConfig("NFT_INDEXER")

	environment := viper.GetString("environment")
	if err := log.Initialize(viper.GetString("log.level"), viper.GetBool("debug")); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         viper.GetString("sentry.dsn"),
		Environment: environment,
	}); err != nil {
		log.Panic("Sentry initialization failed", zap.Error(err))
	}

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"))
	if err != nil {
		log.Panic("fail to initiate indexer store", zap.Error(err))
	}

	cadenceClient := cadence.NewWorkerClient(viper.GetString("cadence.domain"))
	cadenceClient.AddService(indexerWorker.ClientName)

	ensClient := ens.New(viper.GetString("ens.rpc_url"))
	tezosDomain := tezosDomain.New(viper.GetString("tezos.domain_api_url"))
	feralfileClient := feralfile.New(viper.GetString("feralfile.api_url"))

	var minterGateways map[string]string
	if err := yaml.Unmarshal([]byte(viper.GetString("ipfs.minter_gateways")), &minterGateways); err != nil {
		log.Panic("fail to initiate indexer store", zap.Error(err))
	}

	ethClient, err := ethclient.Dial(viper.GetString("ethereum.rpc_url"))
	if err != nil {
		log.Panic("fail to initiate eth client", zap.Error(err))
	}

	engine := indexer.New(
		environment,
		viper.GetStringSlice("ipfs.preferred_gateways"),
		minterGateways,
		opensea.New(viper.GetString("network.ethereum"), viper.GetString("opensea.api_key"), viper.GetInt("opensea.ratelimit")),
		tzkt.New(viper.GetString("network.tezos")),
		fxhash.New(viper.GetString("fxhash.api_endpoint")),
		objkt.New(viper.GetString("objkt.api_endpoint")),
		ethClient,
	)

	parameterStore, err := ssm.NewParameterStore(ctx)
	if err != nil {
		log.Panic("can not create new parameter store", zap.Error(err))
	}

	jwtPublicKeyString, err := parameterStore.GetString(ctx, viper.GetString("jwt.public_key_name"))
	if err != nil {
		log.Panic("get jwt public key failed", zap.Error(err))
	}

	jwtPubkey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(jwtPublicKeyString))
	if err != nil {
		log.Panic("jwt public key parsing failed", zap.Error(err))
	}

	s := NewNFTIndexerServer(cadenceClient, ensClient, tezosDomain, feralfileClient, indexerStore, engine, jwtPubkey, viper.GetString("server.api_token"), viper.GetString("server.admin_api_token"), viper.GetString("server.secret_symmetric_key"))
	s.SetupRoute()
	if err := s.Run(viper.GetString("server.port")); err != nil {
		log.Panic("server interrupted", zap.Error(err))
	}
}
