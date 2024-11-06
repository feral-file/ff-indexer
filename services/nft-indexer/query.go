package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"
	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
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

	checksumDecimalIDs := indexer.NormalizeIndexIDs(reqParams.IDs, true)
	tokenInfo, err := s.indexerStore.GetDetailedTokens(c, indexer.FilterParameter{
		IDs: checksumDecimalIDs,
	}, reqParams.Offset, reqParams.Size)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query tokens from indexer store", err)
		return
	}

	for i, t := range tokenInfo {
		if t.Blockchain != utils.EthereumBlockchain {
			continue
		}

		id, err := strconv.Atoi(t.ID)
		if err != nil {
			continue
		}

		oldIndexID := indexer.TokenIndexID(utils.EthereumBlockchain, t.ContractAddress, fmt.Sprintf("%x", id))
		tokenInfo[i].IndexID = oldIndexID
	}

	c.JSON(http.StatusOK, tokenInfo)
}

// IndexMissingTokens indexes tokens that have not been indexed yet.
func (s *NFTIndexerServer) IndexMissingTokens(c *gin.Context, idMap map[string]bool) {
	// index redundant reqParams.IDs
	for redundantID := range idMap {
		_, contract, tokenID, err := indexer.ParseTokenIndexID(redundantID)
		if err != nil {
			continue
		}

		go indexerWorker.StartIndexTokenWorkflow(c, s.cadenceWorker, "", contract, tokenID, true, false)
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

	tokens, err := s.indexerStore.GetOwnedTokenIDsByOwner(c, reqParams.Owner)
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
	blockchain := utils.GetBlockchainByAddress(accountNumber)

	id := indexer.AccountIdentity{
		AccountNumber: accountNumber,
		Blockchain:    blockchain,
	}

	switch blockchain {
	case utils.EthereumBlockchain:
		domain, err := s.ensClient.ResolveDomain(accountNumber)
		if err != nil {
			return nil, err
		}
		id.Name = domain
	case utils.TezosBlockchain:
		domain, err := s.tezosDomain.ResolveDomain(c, accountNumber)
		if err != nil {
			return nil, err
		}
		id.Name = domain
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
		log.Error("fail to query account identity from blockchain", zap.Any("identity", id), zap.Error(err))
		return
	}

	if err := s.indexerStore.IndexIdentity(c, *id); err != nil {
		log.Error("fail to index identity to indexer store", zap.Any("identity", id), zap.Error(err))
	}
}

// GetIdentity returns the identity of an given account by querying indexer store. If an identity is not existent,
// it will read it from blockchain and set to indexer store before return.
func (s *NFTIndexerServer) GetIdentity(c *gin.Context) {
	traceutils.SetHandlerTag(c, "GetIdentity")

	accountNumber := c.Param("account_number")

	account, err := s.indexerStore.GetIdentity(c, accountNumber)
	if err != nil {
		log.Error("fail to get identity from indexer store", zap.Error(err))
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
		log.Error("fail to index identity to indexer store", zap.Any("identity", id), zap.Error(err))
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

func (s *NFTIndexerServer) ForceReindexNFT(c *gin.Context) {
	traceutils.SetHandlerTag(c, "ForceReIndexToken")
	var req struct {
		Owner      indexer.BlockchainAddress `json:"owner"`
		LastUpdate int64                     `json:"lastUpdated"`
	}

	if err := c.Bind(&req); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	owner := string(req.Owner)
	blockchain := utils.GetBlockchainByAddress(owner)
	if blockchain == utils.UnknownBlockchain {
		abortWithError(c, http.StatusInternalServerError, "unknow blockchain", fmt.Errorf("unknow blockchain"))
		return
	}

	account := indexer.Account{
		Account:          owner,
		Blockchain:       blockchain,
		LastUpdatedTime:  time.Unix(req.LastUpdate, 0),
		LastActivityTime: time.Unix(req.LastUpdate, 0),
	}

	if err := s.indexerStore.IndexAccount(c, account); err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to update account status", err)
		return
	}

	var w indexerWorker.NFTIndexerWorker

	switch blockchain {
	case "eth":
		go s.startIndexWorkflow(c, owner, blockchain, w.IndexETHTokenWorkflow)
	case "tezos":
		go s.startIndexWorkflow(c, owner, blockchain, w.IndexTezosTokenWorkflow)
	default:
		if strings.HasPrefix(owner, "0x") {
			go s.startIndexWorkflow(c, owner, indexer.BlockchainAlias[utils.EthereumBlockchain], w.IndexETHTokenWorkflow)
		} else if strings.HasPrefix(owner, "tz") {
			go s.startIndexWorkflow(c, owner, indexer.BlockchainAlias[utils.TezosBlockchain], w.IndexTezosTokenWorkflow)
		} else {
			abortWithError(c, http.StatusInternalServerError, "owner address with unsupported blockchain", fmt.Errorf("owner address with unsupported blockchain"))
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}

func (s *NFTIndexerServer) CreateDemoTokens(c *gin.Context) {
	traceutils.SetHandlerTag(c, "CreateDemoTokens")

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

	owner := reqParams.Owner

	if owner == "" {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", fmt.Errorf("owner is required"))
		return
	}

	if len(reqParams.IDs) == 0 {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", fmt.Errorf("IDs are required"))
		return
	}

	for _, indexID := range reqParams.IDs {
		if len(strings.Split(indexID, "-")) != 3 {
			abortWithError(c, http.StatusBadRequest, "invalid parameter", fmt.Errorf("indexID structure is not correct"))
			return
		}
	}

	err := s.indexerStore.IndexDemoTokens(c, owner, reqParams.IDs)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to index all demo tokens", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":      1,
		"message": "tokens in the system are added",
	})
}

// GetAccountNFTsV2 queries NFTsV2 based on by owners & lastUpdatedAt
func (s *NFTIndexerServer) GetAccountNFTsV2(c *gin.Context) {
	traceutils.SetHandlerTag(c, "GetAccountNFTsV2")

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
	lastUpdatedAt := time.Unix(reqParams.LastUpdatedAt, 0)

	tokensInfo, err := s.indexerStore.GetDetailedAccountTokensByOwners(
		c,
		owners,
		indexer.FilterParameter{
			Source: reqParams.Source,
		},
		lastUpdatedAt,
		reqParams.SortBy,
		reqParams.Offset,
		reqParams.Size,
	)

	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query tokens from indexer store", err)
		return
	}

	c.JSON(http.StatusOK, tokensInfo)
}

// CountAccountNFTsV2 count NFTsV2 based on by owner
func (s *NFTIndexerServer) CountAccountNFTsV2(c *gin.Context) {
	traceutils.SetHandlerTag(c, "CountAccountNFTsV2")

	var reqParams struct {
		Owner string `form:"owner"`
	}

	if err := c.BindQuery(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if reqParams.Owner == "" {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", fmt.Errorf("owner is required"))
		return
	}

	count, err := s.indexerStore.CountDetailedAccountTokensByOwner(
		c,
		reqParams.Owner,
	)

	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to count tokens from indexer store", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"total": count})
}

// QueryNFTsV2 queries NFTsV2 based on given criteria (decimal input)
func (s *NFTIndexerServer) QueryNFTsV2(c *gin.Context) {
	traceutils.SetHandlerTag(c, "QueryNFTsV2")

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

	if len(reqParams.IDs) > 0 {
		checksumIDs := indexer.NormalizeIndexIDs(reqParams.IDs, false)
		tokenInfo, err := s.indexerStore.GetDetailedTokensV2(c, indexer.FilterParameter{
			IDs: checksumIDs,
		}, reqParams.Offset, reqParams.Size)
		if err != nil {
			abortWithError(c, http.StatusInternalServerError, "fail to query tokens from indexer store", err)
			return
		}

		// check and IndexMissingTokens
		if len(reqParams.IDs) > len(tokenInfo) {
			m := make(map[string]bool, len(reqParams.IDs))
			for _, id := range reqParams.IDs {
				m[id] = true
			}

			for _, info := range tokenInfo {
				if m[info.IndexID] {
					delete(m, info.IndexID)
				}
			}
			go s.IndexMissingTokens(c, m)
		}

		c.JSON(http.StatusOK, tokenInfo)
	} else {
		tokenInfo, err := s.indexerStore.GetDetailedTokensByCollectionID(c, reqParams.CollectionID, reqParams.SortBy, reqParams.Offset, reqParams.Size)
		if err != nil {
			abortWithError(c, http.StatusInternalServerError, "fail to query collection tokens from indexer store", err)
			return
		}

		c.JSON(http.StatusOK, tokenInfo)
	}
}

func (s *NFTIndexerServer) GetETHBlockTime(c *gin.Context) {
	traceutils.SetHandlerTag(c, "GetETHBlockTime")

	blockHash := c.Param("block_hash")

	blockTime, err := indexer.GetETHBlockTime(c, s.cacheStore, s.ethClient, common.HexToHash(blockHash))
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to get block", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"blockTime": blockTime,
	})

}

// GetCollectionsByCreators queries list of collections base on the given addresses
func (s *NFTIndexerServer) GetCollectionsByCreators(c *gin.Context) {
	traceutils.SetHandlerTag(c, "GetCollectionsByCreators")

	var reqParams = CollectionQueryParams{
		Offset: 0,
		Size:   50,
	}

	if err := c.BindQuery(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if reqParams.Creators == "" {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", fmt.Errorf("creators is required"))
		return
	}

	creators := strings.Split(reqParams.Creators, ",")

	collections, err := s.indexerStore.GetCollectionsByCreators(
		c,
		creators,
		reqParams.Offset,
		reqParams.Size,
	)

	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query collections from indexer store", err)
		return
	}

	c.JSON(http.StatusOK, collections)
}

// GetCollectionByID queries collection by the collectionID
func (s *NFTIndexerServer) GetCollectionByID(c *gin.Context) {
	traceutils.SetHandlerTag(c, "GetCollectionByID")

	collectionID := c.Param("collection_id")

	collection, err := s.indexerStore.GetCollectionByID(
		c,
		collectionID,
	)

	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query collections from indexer store", err)
		return
	}

	if collection == nil {
		abortWithError(c, http.StatusInternalServerError, "collection is not found", err)
		return
	}

	c.JSON(http.StatusOK, collection)
}

// GetCollectionByID queries the exchange rate by currency pair and timestamp
func (s *NFTIndexerServer) GetExchangeRate(c *gin.Context) {
	traceutils.SetHandlerTag(c, "GetExchangeRate")

	var reqParams = ExchangeRateQueryParams{
		Timestamp: time.Now(), // default is latest exchange rate
	}

	if err := c.BindQuery(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if !indexer.SupportedCurrencyPairs[reqParams.CurrencyPair] {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", fmt.Errorf("unsupported currency pair"))
		return
	}

	result, err := s.indexerStore.GetHistoricalExchangeRate(c, indexer.HistoricalExchangeRateFilter{
		CurrencyPair: reqParams.CurrencyPair,
		Timestamp:    reqParams.Timestamp,
	})
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query exchange rate from indexer store", err)
		return
	}

	c.JSON(http.StatusOK, result)
}
