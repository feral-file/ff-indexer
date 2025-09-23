package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
)

var ErrUnsupportedBlockchain = fmt.Errorf("unsupported blockchain")

func abortWithError(c *gin.Context, code int, message string, traceErr error) {
	if code == http.StatusInternalServerError {
		log.ErrorWithContext(c, errors.New(message), zap.Error(traceErr))
	} else {
		log.WarnWithContext(c, message, zap.Error(traceErr))
	}

	c.AbortWithStatusJSON(code, gin.H{
		"message": message,
	})
}
