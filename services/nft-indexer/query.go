package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	indexer "github.com/bitmark-inc/nft-indexer"
)

// QueryNFTs queries NFTs based on given criteria
func (s *NFTIndexerServer) QueryNFTs(c *gin.Context) {
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

// GetIdentity querys and returns the blockchain identity of a client
func (s *NFTIndexerServer) GetIdentity(c *gin.Context) {
	accountNumber := c.Param("account_number")
	var identityName string

	blockchain := indexer.DetectAccountBlockchain(accountNumber)

	switch blockchain {
	case indexer.EthereumBlockchain:
		domain, err := s.ensClient.ResolveDomain(accountNumber)
		if err != nil {
			abortWithError(c, http.StatusInternalServerError, "fail to resolve name by account number", err)
			return
		}
		identityName = domain
	case indexer.TezosBlockchain:
		domain, err := s.tezosDomain.ResolveDomain(c, accountNumber)
		if err != nil {
			abortWithError(c, http.StatusInternalServerError, "fail to resolve name by account number", err)
			return
		}
		identityName = domain
	case indexer.BitmarkBlockchain:
		account, err := s.feralfile.GetAccountInfo(accountNumber)
		if err != nil {
			abortWithError(c, http.StatusInternalServerError, "fail to resolve name by account number", err)
			return
		}
		identityName = account.Alias
	default:
		abortWithError(c, http.StatusInternalServerError, "unsupported blockchain address format", nil)
		return
	}

	c.JSON(200, gin.H{
		"blockchain": blockchain,
		"name":       identityName,
	})
}
