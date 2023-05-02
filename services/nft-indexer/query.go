package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	indexer "github.com/bitmark-inc/nft-indexer"
	indexerWorker "github.com/bitmark-inc/nft-indexer/background/worker"
	"github.com/bitmark-inc/nft-indexer/log"
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
}

// PreprocessTokens takes an array of token ids and return an array formatted token ids
// which includes formatting ethereum address and converting token id from hex to decimal if
// isConvertToDecimal is set to true. NOTE: There is no error return in this call.
func PreprocessTokens(indexIDs []string, isConvertToDecimal bool) []string {
	var processedAddresses = []string{}
	for _, indexID := range indexIDs {
		blockchain, contractAddress, tokenID, err := indexer.ParseTokenIndexID(indexID)
		if err != nil {
			continue
		}

		if blockchain == indexer.BlockchainAlias[indexer.EthereumBlockchain] {
			if isConvertToDecimal {
				decimalTokenID, ok := big.NewInt(0).SetString(tokenID, 16)
				if !ok {
					continue
				}
				tokenID = decimalTokenID.String()
			}
			indexID = fmt.Sprintf("%s-%s-%s", blockchain, contractAddress, tokenID)
		}

		processedAddresses = append(processedAddresses, indexID)
	}
	return processedAddresses
}

// IndexMissingTokens indexes tokens that have not been indexed yet.
func (s *NFTIndexerServer) IndexMissingTokens(c *gin.Context, idMap map[string]bool) {
	// index redundant reqParams.IDs
	for redundantID := range idMap {
		_, contract, tokenID, err := indexer.ParseTokenIndexID(redundantID)
		if err != nil {
			continue
		}

		owner, err := s.indexerEngine.GetTokenOwnerAddress(contract, tokenID)
		if err != nil {
			log.Warn("unexpected error while getting token owner address of the contract",
				zap.String("contract", contract),
				zap.String("tokenId", tokenID),
				zap.Error(err))
			continue
		}

		go indexerWorker.StartIndexTokenWorkflow(c, s.cadenceWorker, owner, contract, tokenID, false, false)
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
	blockchain := indexer.GetBlockchainByAddress(accountNumber)

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

func (s *NFTIndexerServer) SetTokenPending(c *gin.Context) {
	traceutils.SetHandlerTag(c, "TokenPending")

	var reqParams indexer.PendingTxUpdate

	if err := c.BindQuery(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if err := c.Bind(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if reqParams.PendingTx == "" {
		abortWithError(c, http.StatusBadRequest, "invalid parameter", fmt.Errorf("pendingTx is required"))
		return
	}

	if len(strings.Split(reqParams.IndexID, "-")) != 3 {
		abortWithError(c, http.StatusBadRequest, "invalid parameter", fmt.Errorf("indexID structure is not correct"))
		return
	}

	if reqParams.Blockchain == indexer.EthereumBlockchain {
		reqParams.IndexID = fmt.Sprintf("%s-%s-%s", indexer.BlockchainAlias[reqParams.Blockchain], reqParams.ContractAddress, reqParams.ID)
	}

	if err := s.indexerStore.AddPendingTxToAccountToken(c, string(reqParams.OwnerAccount), reqParams.IndexID, reqParams.PendingTx, reqParams.Blockchain, reqParams.ID); err != nil {
		log.Warn("fail to index identity to indexer store", zap.Error(err))
		return
	}
	log.Debug("a pending account token is added", zap.String("pendingTx", reqParams.PendingTx))

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
}

func (s *NFTIndexerServer) verifyAddressOwner(blockchain, message, signature, address, publicKey string) (bool, error) {
	switch blockchain {
	case indexer.EthereumBlockchain:
		return indexer.VerifyETHPersonalSignature(message, signature, address)
	case indexer.TezosBlockchain:
		return indexer.VerifyTezosSignature(message, signature, address, publicKey)
	default:
		return false, fmt.Errorf("unsupported blockchain")
	}
}

func (s *NFTIndexerServer) SetTokenPendingV1(c *gin.Context) {
	traceutils.SetHandlerTag(c, "TokenPending")

	var reqParams PendingTxParamsV1

	if err := c.BindQuery(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if err := c.Bind(&reqParams); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if reqParams.PendingTx == "" {
		abortWithError(c, http.StatusBadRequest, "invalid parameter", fmt.Errorf("pendingTx is required"))
		return
	}

	createdAt, err := indexer.EpochStringToTime(reqParams.Timestamp)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameter", err)
		return
	}

	now := time.Now()
	if !indexer.IsTimeInRange(createdAt, now, 5) {
		abortWithError(c, http.StatusBadRequest, "invalid parameter", fmt.Errorf("request time too skewed"))
		return
	}

	isValidAddress, err := s.verifyAddressOwner(reqParams.Blockchain, reqParams.Timestamp, reqParams.Signature, reqParams.OwnerAccount, reqParams.PublicKey)

	if err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	if !isValidAddress {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", fmt.Errorf("invalid signature for ownerAddress"))
		return
	}

	indexID := indexer.TokenIndexID(reqParams.Blockchain, reqParams.ContractAddress, reqParams.ID)

	if err := s.indexerStore.AddPendingTxToAccountToken(c, reqParams.OwnerAccount, indexID, reqParams.PendingTx, reqParams.Blockchain, reqParams.ID); err != nil {
		log.Warn("error while adding pending accountToken", zap.Error(err))
		return
	}
	log.Debug("a pending account token is added", zap.String("pendingTx", reqParams.PendingTx))

	c.JSON(http.StatusOK, gin.H{
		"ok": 1,
	})
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

	switch indexer.GetBlockchainByAddress(owner) {
	case indexer.EthereumBlockchain:
		owner = indexer.EthereumChecksumAddress(owner)
		fallthrough
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
	blockchain := indexer.GetBlockchainByAddress(owner)
	if blockchain == indexer.UnknownBlockchain {
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
		go s.startIndexWorkflow(c, owner, blockchain, w.IndexOpenseaTokenWorkflow)
	case "tezos":
		go s.startIndexWorkflow(c, owner, blockchain, w.IndexTezosTokenWorkflow)
	default:
		if strings.HasPrefix(owner, "0x") {
			go s.startIndexWorkflow(c, owner, indexer.BlockchainAlias[indexer.EthereumBlockchain], w.IndexOpenseaTokenWorkflow)
		} else if strings.HasPrefix(owner, "tz") {
			go s.startIndexWorkflow(c, owner, indexer.BlockchainAlias[indexer.TezosBlockchain], w.IndexTezosTokenWorkflow)
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

func (s *NFTIndexerServer) GetAbsentMimeTypeTokens(c *gin.Context) {
	traceutils.SetHandlerTag(c, "GetAbsentMimeTypeTokens")

	userDID := c.GetString("requester")

	absentMIMETypeToken, err := s.indexerStore.GetAbsentMimeTypeTokens(c, 5)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query tokens from indexer store", err)
		return
	}

	tokenIDs := map[string]bool{}
	for _, t := range absentMIMETypeToken {
		tokenIDs[t.IndexID] = true
	}

	rq := RequestedTokenFeedback{
		DID:       userDID,
		Timestamp: time.Now().Unix(),
		Tokens:    tokenIDs,
	}

	data, err := json.Marshal(rq)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to encode tokens", err)
		return
	}

	sealedRequest, err := indexer.AESSeal(data, s.secretSymmetricKey)

	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to generate signature", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tokens":    absentMIMETypeToken,
		"requestID": sealedRequest,
	})
}

func (s *NFTIndexerServer) FeedbackMimeTypeTokens(c *gin.Context) {
	traceutils.SetHandlerTag(c, "FeedbackMimeTypeTokens")

	userDID := c.GetString("requester")

	var tokenFeedbacks TokenFeedbackParams

	if err := c.Bind(&tokenFeedbacks); err != nil {
		abortWithError(c, http.StatusBadRequest, "invalid parameters", err)
		return
	}

	data, err := indexer.AESOpen(tokenFeedbacks.RequestID, s.secretSymmetricKey)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, "fail to decrypt signature", err)
		return
	}

	var rq RequestedTokenFeedback
	if err := json.Unmarshal(data, &rq); err != nil {
		abortWithError(c, http.StatusBadRequest, "fail to decode signature", err)
		return
	}

	if rq.DID != userDID {
		abortWithError(c, http.StatusBadRequest, "user DID mismatch", err)
		return
	}

	if rq.Timestamp < time.Now().Add(-30*time.Minute).Unix() {
		abortWithError(c, http.StatusBadRequest, "error request time too skewed", err)
		return
	}

	for _, tokenFeedback := range tokenFeedbacks.Tokens {
		if _, contains := rq.Tokens[tokenFeedback.IndexID]; !contains {
			abortWithError(c, http.StatusBadRequest, "indexIDs mismatch", err)
			return
		}
	}

	if s.indexerStore.UpdateTokenFeedback(c, tokenFeedbacks.Tokens, userDID) != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to update token feedback to indexer store", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": 1})
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

	checksumIDs := PreprocessTokens(reqParams.IDs, false)
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
}
