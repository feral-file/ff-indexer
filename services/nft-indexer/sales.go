package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

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

type Sales struct {
	Timestamp string            `json:"timestamp"`
	Metadata  map[string]string `json:"metadata"`
	Values    map[string]string `json:"values"`
	Shares    map[string]string `json:"shares"`
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
	err = s.indexerStore.WriteTimeSeriesData(ctx, timestamp, request.Metadata, request.Values, request.Shares)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "failed to store record", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}

func (s *NFTIndexerServer) GetSalesTimeSeries(c *gin.Context) {
	traceutils.SetHandlerTag(c, "GetSalesTimeSeries")

	var reqParams = SalesQueryParams{
		Offset: 0,
		Size:   50,
	}

	if err := c.BindQuery(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if len(reqParams.Addresses) == 0 && len(reqParams.RoyaltyAddresses) == 0 {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", fmt.Errorf("addresses or royaltyAddresses is required"))
		return
	}

	saleTimeSeries, err := s.indexerStore.GetSaleTimeSeriesData(c, reqParams.Addresses, reqParams.RoyaltyAddresses, reqParams.Marketplace, reqParams.Offset, reqParams.Size)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query sale time series from indexer store", err)
		return
	}

	c.JSON(http.StatusOK, saleTimeSeries)
}

func (s *NFTIndexerServer) AggregateSaleRevenues(c *gin.Context) {
	traceutils.SetHandlerTag(c, "GetSalesRevenues")

	var reqParams SalesQueryParams

	if err := c.BindQuery(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	saleRevenues, err := s.indexerStore.AggregateSaleRevenues(c, reqParams.Addresses, reqParams.Marketplace)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query sale time series from indexer store", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"earning": saleRevenues,
	})
}
