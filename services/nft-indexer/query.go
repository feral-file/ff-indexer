package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/traceutils"
)

// QueryNFTs queries NFTs based on given criteria
func (s *NFTIndexerServer) QueryNFTs(c *gin.Context) {
	traceutils.SetHandlerTag(c, "QueryNFTs")

	var reqParams = NFTQueryParams{
		Offset: 0,
		Size:   50,
	}

	if err := c.BindQuery(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if err := c.Bind(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	tokenInfo, err := s.indexerStore.GetDetailedTokens(c, indexer.FilterParameter{
		IDs: reqParams.IDs,
	}, reqParams.Offset, reqParams.Size)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query tokens from indexer store", err)
		return
	}

	c.JSON(http.StatusOK, tokenInfo)
}

// ListNFTs returns information for a list of NFTs with some criterias.
// It currently only supports listing by owners.
func (s *NFTIndexerServer) ListNFTs(c *gin.Context) {
	traceutils.SetHandlerTag(c, "ListNFTs")

	var reqParams = NFTQueryParams{
		Offset: 0,
		Size:   50,
	}

	if err := c.BindQuery(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if reqParams.Owner == "" {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", fmt.Errorf("owner is required"))
		return
	}

	tokenInfo, err := s.indexerStore.GetDetailedTokensByOwner(c, reqParams.Owner, reqParams.Offset, reqParams.Size)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query tokens from indexer store", err)
		return
	}

	c.JSON(http.StatusOK, tokenInfo)
}

// SearchNFTs returns a list of NFTs by searching criteria
func (s *NFTIndexerServer) SearchNFTs(c *gin.Context) {
	traceutils.SetHandlerTag(c, "SearchNFTs")

	var reqParams = NFTQueryParams{
		Offset: 0,
		Size:   50,
	}

	if err := c.BindQuery(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if reqParams.Text == "" {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", fmt.Errorf("text is required"))
		return
	}

	tokens, err := s.indexerStore.GetTokensByTextSearch(c, reqParams.Text, reqParams.Offset, reqParams.Size)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query tokens from indexer store", err)
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// fetchIdentity collects information from the blockchains and returns an identity object
func (s *NFTIndexerServer) fetchIdentity(c context.Context, accountNumber string) (*indexer.AccountIdentity, error) {
	blockchain := indexer.DetectAccountBlockchain(accountNumber)

	id := indexer.AccountIdentity{
		AccountNumber: accountNumber,
		Blockchain:    blockchain,
	}

	switch blockchain {
	case indexer.EthereumBlockchain:
		domain, err := s.ensClient.ResolveDomain(accountNumber)
		if err != nil {
			return nil, err
		}
		id.Name = domain
	case indexer.TezosBlockchain:
		domain, err := s.tezosDomain.ResolveDomain(c, accountNumber)
		if err != nil {
			return nil, err
		}
		id.Name = domain
	case indexer.BitmarkBlockchain:
		account, err := s.feralfile.GetAccountInfo(accountNumber)
		if err != nil {
			return nil, err
		}
		id.Name = account.Alias
	default:
		return nil, fmt.Errorf("unable to detech blockchain for the given account number")
	}

	return &id, nil
}

// FIXME: move the refresh call out of the API server
// refreshIdentity update the latest identity information to indexer storage
func (s *NFTIndexerServer) refreshIdentity(accountNumber string) {
	c := context.Background()
	id, err := s.fetchIdentity(c, accountNumber)
	if err != nil {
		log.WithError(err).WithField("identity", id).Error("fail to query account identity from blockchain")
		return
	}

	if err := s.indexerStore.IndexIdentity(c, *id); err != nil {
		log.WithError(err).WithField("identity", id).Error("fail to index identity to indexer store")
	}
}

// GetIdentity returns the identity of an given account by querying indexer store. If an identity is not existent,
// it will read it from blockchain and set to indexer store before return.
func (s *NFTIndexerServer) GetIdentity(c *gin.Context) {
	traceutils.SetHandlerTag(c, "GetIdentity")

	accountNumber := c.Param("account_number")

	account, err := s.indexerStore.GetIdentity(c, accountNumber)
	if err != nil {
		log.WithError(err).Error("fail to get identity from indexer store")
	}

	if account.AccountNumber != "" {
		// FIXME: define the cache expiry for identities
		if time.Since(account.LastUpdatedTime) > time.Hour {
			go s.refreshIdentity(accountNumber)
		}
		c.JSON(200, account)
		return
	}

	id, err := s.fetchIdentity(c, accountNumber)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query account identity from blockchain", err)
		return
	}

	if err := s.indexerStore.IndexIdentity(c, *id); err != nil {
		log.WithError(err).WithField("identity", id).Error("fail to index identity to indexer store")
	}

	c.JSON(200, id)
}

// GetIdentities a map of identities which has already updated from the store.
func (s *NFTIndexerServer) GetIdentities(c *gin.Context) {
	traceutils.SetHandlerTag(c, "GetIdentities")

	var reqParams struct {
		AccountNumbers []string `json:"account_numbers" binding:"required"`
	}

	if err := c.Bind(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", fmt.Errorf("text is required"))
		return
	}

	ids, err := s.indexerStore.GetIdentities(c, reqParams.AccountNumbers)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to resolve name by account numbers", err)
		return
	}

	c.JSON(200, ids)
}
