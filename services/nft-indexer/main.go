package main

import (
	"context"
	"io/ioutil"

	"github.com/dgrijalva/jwt-go"
	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/bitmark-inc/config-loader"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/externals/ens"
	"github.com/bitmark-inc/nft-indexer/externals/feralfile"
	"github.com/bitmark-inc/nft-indexer/externals/fxhash"
	"github.com/bitmark-inc/nft-indexer/externals/objkt"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
	tezosDomain "github.com/bitmark-inc/nft-indexer/externals/tezos-domain"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
)

func main() {
	ctx := context.Background()

	config.LoadConfig("NFT_INDEXER")

	environment := viper.GetString("environment")

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         viper.GetString("sentry.dsn"),
		Environment: environment,
	}); err != nil {
		log.WithError(err).Panic("Sentry initialization failed")
	}

	indexerStore, err := indexer.NewMongodbIndexerStore(ctx, viper.GetString("store.db_uri"), viper.GetString("store.db_name"))
	if err != nil {
		log.WithError(err).Panic("fail to initiate indexer store")
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

	jwtPublicByte, err := ioutil.ReadFile(viper.GetString("jwt.pubkeyfile"))
	if err != nil {
		log.WithError(err).Fatal("fail to read jwt key file")
	}

	jwtPubkey, err := jwt.ParseRSAPublicKeyFromPEM(jwtPublicByte)
	if err != nil {
		log.WithError(err).Panic("jwt public key parsing failed")
	}

	s := NewNFTIndexerServer(cadenceClient, ensClient, tezosDomain, feralfileClient, indexerStore, engine, jwtPubkey, viper.GetString("server.api_token"), viper.GetString("server.admin_api_token"), viper.GetString("server.secret_symmetric_key"))
	s.SetupRoute()
	if err := s.Run(viper.GetString("server.port")); err != nil {
		log.WithError(err).Panic("server interrupted")
	}
}
