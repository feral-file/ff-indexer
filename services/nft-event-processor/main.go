package main

import (
	"net"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/bitmark-inc/config-loader"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/grpc/processor"
	"github.com/bitmark-inc/nft-indexer/services/nft-event-processor/rpc-services"
)

func main() {
	config.LoadConfig("NFT_INDEXER")

	s := grpc.NewServer()
	server := rpc_services.NewEventProcessor()

	reflection.Register(s)
	processor.RegisterEventProcessorServer(s, server)

	tl, err := net.Listen(viper.GetString("event_processor_server.network"), viper.GetString("event_processor_server.address"))
	if err != nil {
		log.WithError(err).Panic("server interrupted")
	}

	s.Serve(tl)
}
