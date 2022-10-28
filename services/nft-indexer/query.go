package main

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
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

	checksumDecimalIDs := PreprocessTokens(reqParams.IDs, true)
	tokenInfo, err := s.indexerStore.GetDetailedTokens(c, indexer.FilterParameter{
		IDs: checksumDecimalIDs,
	}, reqParams.Offset, reqParams.Size)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query tokens from indexer store", err)
		return
	}

	// go s.IndexMissingTokens(c, checksumDecimalIDs, tokenInfo)
	for i, t := range tokenInfo {
		if t.Blockchain != indexer.EthereumBlockchain {
			continue
		}

		id, err := strconv.Atoi(t.ID)
		if err != nil {
			continue
		}

		oldIndexID := indexer.TokenIndexID(indexer.EthereumBlockchain, t.ContractAddress, fmt.Sprintf("%x", id))
		tokenInfo[i].IndexID = oldIndexID
	}

	c.JSON(http.StatusOK, tokenInfo)
}

// QueryNFTsV1 queries NFTsV1 based on given criteria (decimal input)
func (s *NFTIndexerServer) QueryNFTsV1(c *gin.Context) {
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

	checksumIDs := PreprocessTokens(reqParams.IDs, false)
	tokenInfo, err := s.indexerStore.GetDetailedTokens(c, indexer.FilterParameter{
		IDs: checksumIDs,
	}, reqParams.Offset, reqParams.Size)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query tokens from indexer store", err)
		return
	}

	go s.IndexMissingTokens(c, reqParams.IDs, tokenInfo)

	c.JSON(http.StatusOK, tokenInfo)
}

func PreprocessTokens(addresses []string, isConvertToDecimal bool) []string {
	var processedAddresses = []string{}
	for _, address := range addresses {
		blockchain, contractAddress, tokenID, err := indexer.ParseIndexID(address)
		if err != nil {
			continue
		}

		if blockchain == "eth" {
			if isConvertToDecimal {
				decimalTokenID, ok := big.NewInt(0).SetString(tokenID, 16)
				if !ok {
					continue
				}
				processedAddresses = append(processedAddresses, fmt.Sprintf("%s-%s-%s", blockchain, indexer.EthereumChecksumAddress(contractAddress), decimalTokenID.String()))
			} else {
				processedAddresses = append(processedAddresses, fmt.Sprintf("%s-%s-%s", blockchain, indexer.EthereumChecksumAddress(contractAddress), tokenID))
			}

		} else {
			processedAddresses = append(processedAddresses, address)
		}
	}
	return processedAddresses
}

// IndexMissingTokens indexes tokens that have not been indexed yet.
func (s *NFTIndexerServer) IndexMissingTokens(c *gin.Context, reqParamsIDs []string, tokenInfo []indexer.DetailedToken) {
	if len(reqParamsIDs) > len(tokenInfo) {
		// find redundant reqParams.IDs to index
		m := make(map[string]bool, len(reqParamsIDs))
		for _, id := range reqParamsIDs {
			m[id] = true
		}

		for _, info := range tokenInfo {
			if m[info.IndexID] {
				delete(m, info.IndexID)
			}
		}

		// index redundant reqParams.IDs
		for redundantID := range m {
			_, contract, tokenId, err := indexer.ParseIndexID(redundantID)
			if err != nil {
				panic(err)
			}

			owner, err := s.indexerEngine.GetTokenOwnerAddress(contract, tokenId)
			if err != nil {
				logrus.
					WithField("contract", contract).
					WithField("tokenId", tokenId).
					WithError(err).
					Warn("unexpected error while getting token owner address of the contract")
				continue
			}

			go indexerWorker.StartIndexTokenWorkflow(c, s.cadenceWorker, owner, contract, tokenId, false)
		}
	}
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

	owners := strings.Split(reqParams.Owner, ",")

	tokenInfo, err := s.indexerStore.GetDetailedTokensByOwners(c, owners,
		indexer.FilterParameter{
			Source: reqParams.Source,
		},
		reqParams.Offset, reqParams.Size)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query tokens from indexer store", err)
		return
	}

	c.JSON(http.StatusOK, tokenInfo)
}

// OwnedNFTIDs returns a list of token ids for a given list of owners
func (s *NFTIndexerServer) OwnedNFTIDs(c *gin.Context) {
	traceutils.SetHandlerTag(c, "OwnedNFTIDs")

	var reqParams = NFTQueryParams{}

	if err := c.BindQuery(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if reqParams.Owner == "" {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", fmt.Errorf("owner is required"))
		return
	}

	tokens, err := s.indexerStore.GetTokenIDsByOwner(c, reqParams.Owner)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query tokens from indexer store", err)
		return
	}

	c.JSON(http.StatusOK, tokens)
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
		return nil, ErrUnsupportedBlockchain
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
		if err == ErrUnsupportedBlockchain {
			abortWithError(c, http.StatusBadRequest, "fail to query account identity from blockchain", err)
		} else {
			abortWithError(c, http.StatusInternalServerError, "fail to query account identity from blockchain", err)
		}
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

func (s *NFTIndexerServer) GetAccountNFTs(c *gin.Context) {
	traceutils.SetHandlerTag(c, "GetNewAccountTokens")

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

	owner := reqParams.Owner

	var tokensInfo []indexer.DetailedToken
	var err error

	switch indexer.DetectAccountBlockchain(owner) {
	case indexer.TezosBlockchain:
		tokensInfo, err = s.indexerStore.GetDetailedAccountTokensByOwner(c, owner,
			indexer.FilterParameter{
				Source: reqParams.Source,
			},
			reqParams.Offset, reqParams.Size)
	default:
		tokensInfo, err = s.indexerStore.GetDetailedTokensByOwners(c, []string{owner},
			indexer.FilterParameter{
				Source: reqParams.Source,
			},
			reqParams.Offset, reqParams.Size)
	}

	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query tokens from indexer store", err)
		return
	}

	c.JSON(http.StatusOK, tokensInfo)
}
