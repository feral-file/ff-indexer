package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/bitmark-inc/nft-indexer/traceutils"
)

func abortWithError(c *gin.Context, code int, message string, traceErr error) {
	log.WithError(traceErr).Error(message)
	if code == http.StatusInternalServerError {
		traceutils.CaptureException(c, traceErr)
	}

	c.AbortWithStatusJSON(code, gin.H{
		"message": message,
	})
}
