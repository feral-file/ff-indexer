package main

import (
	"context"
	"net/http"
	"time"

	"github.com/bitmark-inc/nft-indexer/traceutils"
	"github.com/gin-gonic/gin"
)

type Sales struct {
	Timestamp string            `json:"timestamp"`
	Metadata  map[string]string `json:"metadata"`
	Values    map[string]string `json:"values"`
}

// SalesTimeSeries - store a time series record
func (s *NFTIndexerServer) SalesTimeSeries(c *gin.Context) {
	traceutils.SetHandlerTag(c, "Sales")

	var request Sales
	if err := c.Bind(&request); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}
	timestamp, err := time.Parse(time.RFC3339Nano, request.Timestamp)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid timestamp", err)
		return
	}

	ctx := context.Background()
	err = s.indexerStore.WriteTimeSeriesData(ctx, timestamp, request.Metadata, request.Values)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "failed to store record", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}
