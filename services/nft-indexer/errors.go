package main

import (
	sentryHelper "github.com/bitmark-inc/nft-indexer/sentry"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func abortWithError(c *gin.Context, code int, message string, traceErr error) {
	log.WithError(traceErr).Error(message)
	sentryHelper.CaptureException(c, traceErr)

	c.AbortWithStatusJSON(code, gin.H{
		"message": message,
	})
}
