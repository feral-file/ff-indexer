package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/bitmark-inc/nft-indexer/traceutils"
)

var ErrUnsupportedBlockchain = fmt.Errorf("unsupported blockchain")

func abortWithError(c *gin.Context, code int, message string, traceErr error) {
	log.Error(message, zap.Error(traceErr))
	if code == http.StatusInternalServerError {
		traceutils.CaptureException(c, traceErr)
	}

	c.AbortWithStatusJSON(code, gin.H{
		"message": message,
	})
}
