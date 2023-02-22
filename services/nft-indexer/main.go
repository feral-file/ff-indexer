package main

import (
	"context"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/aws/ssm"
	"github.com/bitmark-inc/nft-indexer/externals/ens"
	"github.com/bitmark-inc/nft-indexer/externals/feralfile"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	tezosDomain "github.com/bitmark-inc/nft-indexer/externals/tezos-domain"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
	"github.com/bitmark-inc/nft-indexer/log"
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

	engine := indexer.New(
		environment,
		opensea.New(viper.GetString("network.ethereum"), viper.GetString("opensea.api_key"), viper.GetInt("opensea.ratelimit")),
		tzkt.New(viper.GetString("network.tezos")),
		fxhash.New(viper.GetString("fxhash.api_endpoint")),
		objkt.New(viper.GetString("objkt.api_endpoint")),
	)

	awsSystemManager, err := ssm.New(ctx)
	if err != nil {
		log.Fatal("fail to create aws System Manager", zap.Error(err))
	}

	jwtPublicKey, err := awsSystemManager.GetRSAPublishKeyFromParameterStore(ctx, viper.GetString("jwt.pubkey_name"))
	if err != nil {
		log.Fatal("fail to read jwt publish key file", zap.Error(err))
	}

	s := NewNFTIndexerServer(cadenceClient, ensClient, tezosDomain, feralfileClient, indexerStore, engine, jwtPublicKey, viper.GetString("server.api_token"), viper.GetString("server.admin_api_token"), viper.GetString("server.secret_symmetric_key"))
	s.SetupRoute()
	if err := s.Run(viper.GetString("server.port")); err != nil {
		log.Panic("server interrupted", zap.Error(err))
	}
}
