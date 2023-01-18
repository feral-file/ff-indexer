package log

import (
	"encoding/json"

	"go.uber.org/zap"
)

var Logger *zap.Logger = InitializeLogger()

const (
	Fxhash       = "fxhas"
	TZKT         = "tzkt"
	Objkt        = "objkt"
	Bitmark      = "bitmark"
	FeralFile    = "feralfile"
	Opensea      = "opensea"
	GRPC         = "gRPC"
	ETHClient    = "ETHClient"
	Pq           = "pq"
	ImageCaching = "imageCaching"
)

func InitializeLogger() *zap.Logger {
	rawJSON := []byte(`{
		"level": "debug",
		"encoding": "json",
		"outputPaths": ["stdout"],
		"errorOutputPaths": ["stderr"],
		"encoderConfig": {
		  "messageKey": "message",
		  "levelKey": "level",
		  "levelEncoder": "lowercase"
		}
	  }`)
	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}

	Logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	return Logger
}
