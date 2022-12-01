package main

import (
	"flag"

	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/bitmark-inc/config-loader"
)

func main() {
	// FIXME: add context for graceful shutdown
	// ctx := context.Background()

	var configPath string
	flag.StringVar(&configPath, "c", ".", "config path")
	flag.Parse()

	config.LoadConfig("NFT_INDEXER", configPath)

	environment := viper.GetString("environment")

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         viper.GetString("sentry.dsn"),
		Environment: environment,
	}); err != nil {
		log.WithError(err).Panic("Sentry initialization failed")
	}

	db, err := gorm.Open(postgres.Open(viper.GetString("store.dsn")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.LogLevel(viper.GetInt("store.log_level"))),
	})
	if err != nil {
		panic(err)
	}

	store := NewPostgresEventStore(db)
	if err := store.AutoMigrate(); err != nil {
		panic(err)
	}

	p := NewEventProcessor(
		viper.GetString("server.network"),
		viper.GetString("server.address"),
		NewPostgresEventStore(db))
	p.Run()
}
