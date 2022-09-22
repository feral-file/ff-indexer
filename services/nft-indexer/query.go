package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

	tokenInfo, err := s.indexerStore.GetDetailedTokens(c, indexer.FilterParameter{
		IDs: reqParams.IDs,
	}, reqParams.Offset, reqParams.Size)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "fail to query tokens from indexer store", err)
		return
	}

	if len(reqParams.IDs) > len(tokenInfo) {
		// find redundant reqParams.IDs to index
		m := make(map[string]bool, len(reqParams.IDs))
		for _, id := range reqParams.IDs {
			m[id] = true
		}

		for _, info := range tokenInfo {
			if m[info.IndexID] {
				delete(m, info.IndexID)
			}
		}

		// index redundant reqParams.IDs
		for redundantID := range m {
			owner := "" //"tz1fRXMLR27hWoD49tdtKunHyfy3CQb5XZst"
			blockchain := strings.Split(redundantID, "-")[0]
			contract := strings.Split(redundantID, "-")[1]
			tokenId := strings.Split(redundantID, "-")[2]

			fmt.Printf("\n\n blockchain: %s. contract: %s, tokenId: %s\n", blockchain, contract, tokenId)

			if contract != "" {
				// var e indexer.IndexEngine
				// tokenOwner, err := e.GetTokenOwners(contract, tokenId)

				// fmt.Printf("\n\t IndexTezosToken: tokenID: %s tokenOwner: %v", tokenId, tokenOwner)

				// if err != nil {
				// 	fmt.Println("\n\n\t Some error with GetTokenOwners: ", err)
				// 	return
				// }
				// owner = tokenOwner[0].Address

				var w indexerWorker.NFTIndexerWorker
				switch blockchain {
				case "tez":
					fmt.Println("\n\t\t Go to Tezos token")
					go s.indexerEngine.IndexTezosToken(c, owner, contract, tokenId)
					// go s.startIndexWorkflow(c, owner, indexer.BlockchainAlias[indexer.TezosBlockchain], w.IndexTezosTokenWorkflow)
				case "eth":
					fmt.Println("\n\t Go to ETH token")
					go s.startIndexWorkflow(c, owner, indexer.BlockchainAlias[indexer.EthereumBlockchain], w.IndexTezosTokenWorkflow)
				}
			} else {
				fmt.Println("\n\t === Contract = nil")
			}
		}
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

	owners := strings.Split(reqParams.Owner, ",")

	tokens, err := s.indexerStore.GetTokenIDsByOwners(c, owners)
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
