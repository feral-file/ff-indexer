package log

import "go.uber.org/zap"

var (
	SourceFXHASH       = zap.String("source", "fxhash")
	SourceTZKT         = zap.String("source", "tzkt")
	SourceObjkt        = zap.String("source", "objkt")
	SourceBitmark      = zap.String("source", "bitmark")
	SourceFeralFile    = zap.String("source", "feralfile")
	SourceOpensea      = zap.String("source", "opensea")
	SourceGRPC         = zap.String("source", "gRPC")
	SourceETHClient    = zap.String("source", "ETHClient")
	SourcePG           = zap.String("source", "pq")
	SourceImageCaching = zap.String("source", "imageCaching")
)
