package main

import (
	"net/http"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/traceutils"
	"github.com/gin-gonic/gin"
)

type SalesQueryParams struct {
	// global
	Offset int64 `form:"offset"`
	Size   int64 `form:"size"`

	// list by owners
	Addresses        []string `form:"addresses"`
	RoyaltyAddresses []string `form:"royaltyAddresses"`
	Marketplace      string   `form:"marketplace"`
}

// SalesTimeSeries - store a time series record
func (s *NFTIndexerServer) SalesTimeSeries(c *gin.Context) {
	traceutils.SetHandlerTag(c, "Sales")

	var request []indexer.GenericSalesTimeSeries
	if err := c.Bind(&request); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	err := s.indexerStore.WriteTimeSeriesData(c, request)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "failed to store record", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}
