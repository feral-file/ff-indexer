package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	goethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/contracts"
	"github.com/bitmark-inc/nft-indexer/traceutils"
	"github.com/bitmark-inc/tzkt-go"
)

var (
	ErrMapKeyNotFound   = errors.New("key is not found")
	ErrValueNotString   = errors.New("value is not of string type")
	ErrInvalidEditionID = errors.New("invalid edition id")
	QueryPageSize       = 50
)

// GetOwnedERC721TokenIDByContract returns a list of token id belongs to an owner for a specific contract
func (w *NFTIndexerWorker) GetOwnedERC721TokenIDByContract(_ context.Context, contractAddress, ownerAddress string) ([]*big.Int, error) {
	rpcClient, err := ethclient.Dial(viper.GetString("ethereum.rpc_url"))
	if err != nil {
		return nil, err
	}

	contractAddr := common.HexToAddress(contractAddress)
	instance, err := contracts.NewERC721Enumerable(contractAddr, rpcClient)
	if err != nil {
		return nil, err
	}

	var tokenIDs []*big.Int

	ownerAddr := common.HexToAddress(ownerAddress)
	n, err := instance.BalanceOf(nil, ownerAddr)
	if err != nil {
		return nil, err
	}
	for i := int64(0); i < n.Int64(); i++ {
		id, err := instance.TokenOfOwnerByIndex(nil, ownerAddr, big.NewInt(i))
		if err != nil {
			return nil, err
		}

		tokenIDs = append(tokenIDs, id)
	}

	return tokenIDs, nil
}

// IndexETHTokenByOwner indexes ETH token data for an owner into the format of AssetUpdates
func (w *NFTIndexerWorker) IndexETHTokenByOwner(ctx context.Context, owner string, next string) (string, error) {
	log.Debug("IndexETHTokenByOwner", zap.String("owner", owner))
	updates, next, err := w.indexerEngine.IndexETHTokenByOwner(ctx, owner, next)
	if err != nil {
		return "", err
	}

	if len(updates) == 0 {
		return "", nil
	}

	accountTokens := []indexer.AccountToken{}

	for _, update := range updates {
		if err := w.indexerStore.IndexAsset(ctx, update.ID, update); err != nil {
			return "", err
		}

		accountTokens = append(accountTokens, indexer.AccountToken{
			BaseTokenInfo:     update.Tokens[0].BaseTokenInfo,
			IndexID:           update.Tokens[0].IndexID,
			OwnerAccount:      update.Tokens[0].Owner,
			Balance:           update.Tokens[0].Balance,
			LastActivityTime:  update.Tokens[0].LastActivityTime,
			LastRefreshedTime: update.Tokens[0].LastRefreshedTime,
		})
	}

	if err := w.IndexAccountTokens(ctx, owner, accountTokens); err != nil {
		return "", err
	}

	return next, nil
}

// IndexTezosTokenByOwner indexes Tezos token data for an owner into the format of AssetUpdates
func (w *NFTIndexerWorker) IndexTezosTokenByOwner(ctx context.Context, owner string, isFirstPage bool) (bool, error) {
	account, err := w.indexerStore.GetAccount(ctx, owner)

	if err != nil {
		return false, err
	}

	if isFirstPage {
		delay := time.Minute
		if account.LastUpdatedTime.Unix() > time.Now().Add(-delay).Unix() {
			log.Debug("owner refresh too frequently",
				zap.Int64("lastUpdatedTime", account.LastUpdatedTime.Unix()),
				zap.Int64("now", time.Now().Add(-delay).Unix()),
				zap.String("owner", account.Account))
			return false, nil
		}
	}

	// FIXME: currently both account_tokens and tokens rely on this activity to be done correctly
	// It would be better the separate the token indexing and account_tokens indexing since
	// account_tokens is for what you own and token is for what it is
	updates, newLastTime, err := w.indexerEngine.IndexTezosTokenByOwner(ctx, owner, account.LastActivityTime, 0)
	if err != nil {
		return false, err
	}

	if len(updates) == 0 {
		return false, err
	}

	accountTokens := []indexer.AccountToken{}

	for _, update := range updates {
		if err := w.indexerStore.IndexAsset(ctx, update.ID, update); err != nil {
			return false, err
		}

		accountTokens = append(accountTokens, indexer.AccountToken{
			BaseTokenInfo:     update.Tokens[0].BaseTokenInfo,
			IndexID:           update.Tokens[0].IndexID,
			OwnerAccount:      update.Tokens[0].Owner,
			Balance:           update.Tokens[0].Balance,
			LastActivityTime:  update.Tokens[0].LastActivityTime,
			LastRefreshedTime: update.Tokens[0].LastRefreshedTime,
		})
	}

	if err := w.indexTezosAccount(ctx, owner, newLastTime); err != nil {
		return false, err
	}

	if err := w.IndexAccountTokens(ctx, owner, accountTokens); err != nil {
		return false, err
	}

	return true, nil
}

type TezosTokenRawData struct {
	Token   tzkt.Token
	Owner   string
	Balance int64
}

// GetTokenBalanceOfOwner returns the balance of a token for an owner
func (w *NFTIndexerWorker) GetTokenBalanceOfOwner(ctx context.Context, contract, tokenID, owner string) (int64, error) {
	return w.indexerEngine.GetTokenBalanceOfOwner(ctx, contract, tokenID, owner)
}

// IndexToken indexes a token by the given contract and token id
func (w *NFTIndexerWorker) IndexToken(ctx context.Context, contract, tokenID string) (*indexer.AssetUpdates, error) {
	return w.indexerEngine.IndexToken(ctx, contract, tokenID)
}

// GetOwnedTokenIDsByOwner gets tokenIDs by the given owner
func (w *NFTIndexerWorker) GetOwnedTokenIDsByOwner(ctx context.Context, owner string) ([]string, error) {
	return w.indexerStore.GetOwnedTokenIDsByOwner(ctx, owner)
}

// FilterTokenIDsWithInconsistentProvenanceForOwner gets tokenIDs by the given owner
func (w *NFTIndexerWorker) FilterTokenIDsWithInconsistentProvenanceForOwner(ctx context.Context, tokenIDs []string, owner string) ([]string, error) {
	return w.indexerStore.FilterTokenIDsWithInconsistentProvenanceForOwner(ctx, tokenIDs, owner)
}

// IndexAsset saves asset data into indexer's storage
func (w *NFTIndexerWorker) IndexAsset(ctx context.Context, updates indexer.AssetUpdates) error {
	return w.indexerStore.IndexAsset(ctx, updates.ID, updates)
}

// indexTezosAccount saves tezos account data into indexer's storage
func (w *NFTIndexerWorker) indexTezosAccount(ctx context.Context, owner string, lastActivityTime time.Time) error {
	account := indexer.Account{
		Account:          owner,
		Blockchain:       "tezos",
		LastUpdatedTime:  time.Now(),
		LastActivityTime: lastActivityTime,
	}
	return w.indexerStore.IndexAccount(ctx, account)
}

// IndexAccountTokens saves account tokens data into indexer's storage
func (w *NFTIndexerWorker) IndexAccountTokens(ctx context.Context, owner string, accountTokens []indexer.AccountToken) error {
	return w.indexerStore.IndexAccountTokens(ctx, owner, accountTokens)
}

func (w *NFTIndexerWorker) MarkAccountTokenChanged(ctx context.Context, indexIDs []string) error {
	return w.indexerStore.MarkAccountTokenChanged(ctx, indexIDs)
}

type Provenance struct {
	TxID      string    `json:"tx_id"`
	Owner     string    `json:"owner"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Bitmark is the response structure of bitmark registry
type Bitmark struct {
	ID               string       `json:"id"`
	HeadID           string       `json:"head_id"`
	Owner            string       `json:"owner"`
	AssetID          string       `json:"asset_id"`
	Issuer           string       `json:"issuer"`
	Head             string       `json:"head"`
	Status           string       `json:"status"`
	IssueBlockNumber int64        `json:"issue_block_number"`
	Provenance       []Provenance `json:"provenance"`
}

// fetchBitmarkProvenance reads bitmark provenances through bitmark api
func (w *NFTIndexerWorker) fetchBitmarkProvenance(bitmarkID string) ([]indexer.Provenance, error) {
	provenances := []indexer.Provenance{}

	var data struct {
		Bitmark Bitmark `json:"bitmark"`
	}

	resp, err := w.http.Get(fmt.Sprintf("%s/v1/bitmarks/%s?provenance=true", w.bitmarkAPIEndpoint, bitmarkID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Error("fail to decode bitmark payload", zap.Error(err),
			log.SourceBitmark,
			zap.String("respData", traceutils.DumpResponse(resp)))
		return nil, err
	}

	for i, p := range data.Bitmark.Provenance {
		txType := "transfer"

		if i == len(data.Bitmark.Provenance)-1 {
			txType = "issue"
		} else if p.Owner == w.bitmarkZeroAddress {
			txType = "burn"
		}

		provenances = append(provenances, indexer.Provenance{
			Type:       txType,
			Owner:      p.Owner,
			Blockchain: utils.BitmarkBlockchain,
			Timestamp:  p.CreatedAt,
			TxID:       p.TxID,
			TxURL:      indexer.TxURL(utils.BitmarkBlockchain, w.Environment, p.TxID),
		})
	}

	return provenances, nil
}

// fetchEthereumProvenance reads ethereum provenance through filterLogs
func (w *NFTIndexerWorker) fetchEthereumProvenance(ctx context.Context, tokenID, contractAddress string) ([]indexer.Provenance, error) {
	hexID, err := indexer.OpenseaTokenIDToHex(tokenID)
	if err != nil {
		return nil, err
	}
	transferLogs, err := w.ethClient.FilterLogs(ctx, goethereum.FilterQuery{
		Addresses: []common.Address{common.HexToAddress(contractAddress)},
		Topics: [][]common.Hash{
			{common.HexToHash(indexer.TransferEventSignature)},
			nil, nil,
			{common.HexToHash(hexID)},
		},
	})
	if err != nil {
		return nil, err
	}

	log.Debug("token provenance", zap.String("tokenID", hexID), zap.Any("logs", transferLogs))

	totalTransferLogs := len(transferLogs)

	provenances := make([]indexer.Provenance, 0, totalTransferLogs)

	for i := range transferLogs {
		l := transferLogs[totalTransferLogs-i-1]

		fromAccountHash := l.Topics[1]
		toAccountHash := l.Topics[2]
		txType := "transfer"
		if fromAccountHash.Big().Cmp(big.NewInt(0)) == 0 {
			txType = "mint"
		}

		txTime, err := indexer.GetETHBlockTime(ctx, w.cacheStore, w.ethClient, l.BlockHash)
		if err != nil {
			return nil, err
		}

		provenances = append(provenances, indexer.Provenance{
			Timestamp:   txTime,
			Type:        txType,
			Owner:       indexer.EthereumChecksumAddress(toAccountHash.Hex()),
			Blockchain:  utils.EthereumBlockchain,
			BlockNumber: &l.BlockNumber,
			TxID:        l.TxHash.Hex(),
			TxURL:       indexer.TxURL(utils.EthereumBlockchain, w.Environment, l.TxHash.Hex()),
		})
	}

	return provenances, nil
}

// fetchTezosProvenance reads ethereum provenance through filterLogs
func (w *NFTIndexerWorker) fetchTezosProvenance(tokenID, contractAddress string) ([]indexer.Provenance, error) {
	return w.indexerEngine.IndexTezosTokenProvenance(contractAddress, tokenID)
}

// RefreshTokenProvenance refresh provenance. This is a heavy task
func (w *NFTIndexerWorker) RefreshTokenProvenance(ctx context.Context, indexIDs []string, delay time.Duration) error {
	tokens, err := w.indexerStore.GetTokensByIndexIDs(ctx, indexIDs)
	if err != nil {
		log.Error("cannot find tokens in DB", zap.Any("indexIDs", indexIDs), zap.Error(err))
		return err
	}

	for _, token := range tokens {
		if delay > 0 {
			if token.LastRefreshedTime.Unix() > time.Now().Add(-delay).Unix() {
				log.Debug("provenance refresh too frequently", zap.String("indexID", token.IndexID))
				continue
			}
		}

		if token.Fungible {
			log.Debug("ignore fungible token", zap.String("indexID", token.IndexID))
			continue
		}

		log.Debug("start refresh token provenance updating flow", zap.String("indexID", token.IndexID))

		totalProvenances := []indexer.Provenance{}
		switch token.Blockchain {
		case utils.BitmarkBlockchain:
			provenance, err := w.fetchBitmarkProvenance(token.ID)
			if err != nil {
				log.Error("cannot fetch bitmark provenance", zap.String("tokenID: ", token.ID), zap.Error(err))
				return err
			}

			totalProvenances = append(totalProvenances, provenance...)
		case utils.EthereumBlockchain:
			provenance, err := w.fetchEthereumProvenance(ctx, token.ID, token.ContractAddress)
			if err != nil {
				log.Error("cannot fetch ethereum provenance", zap.String("contractAddress: ", token.ContractAddress), zap.String("tokenID: ", token.ID), zap.Error(err))
				return err
			}
			totalProvenances = append(totalProvenances, provenance...)
		case utils.TezosBlockchain:
			lastActivityTime, err := w.indexerEngine.IndexTezosTokenLastActivityTime(token.ContractAddress, token.ID)
			if err != nil {
				log.Error("cannot fetch lastActivityTime", zap.String("contractAddress: ", token.ContractAddress), zap.String("tokenID: ", token.ID), zap.Error(err))
				return err
			}

			// Ignore the refreshing process when
			// 1. provenances is not empty
			// 2. the latest provenance timestamp match the token's lastActivityTime
			// 3. the token's lastActivityTime is greater than the record from tzkt
			if len(token.Provenances) != 0 &&
				token.Provenances[0].Timestamp.Sub(token.LastActivityTime) == 0 &&
				lastActivityTime.Sub(token.LastActivityTime) <= 0 {
				log.Debug("no new updates since last check", zap.String("indexID", token.IndexID))
				continue
			}

			provenance, err := w.fetchTezosProvenance(token.ID, token.ContractAddress)
			if err != nil {
				log.Error("cannot fetch tezos provenance", zap.String("contractAddress: ", token.ContractAddress), zap.String("tokenID: ", token.ID), zap.Error(err))
				return err
			}
			totalProvenances = append(totalProvenances, provenance...)
		}

		// recursively fetch provenance records from migrated blockchains
		for _, tokenInfo := range token.OriginTokenInfo {
			switch tokenInfo.Blockchain {
			case utils.BitmarkBlockchain:
				provenance, err := w.fetchBitmarkProvenance(tokenInfo.ID)
				if err != nil {
					log.Error("cannot fetch bitmark provenance", zap.String("tokenID: ", tokenInfo.ID), zap.Error(err))
					return err
				}

				totalProvenances = append(totalProvenances, provenance...)
			case utils.EthereumBlockchain:
				provenance, err := w.fetchEthereumProvenance(ctx, tokenInfo.ID, tokenInfo.ContractAddress)
				if err != nil {
					log.Error("cannot fetch ethereum provenance", zap.String("contractAddress: ", token.ContractAddress), zap.String("tokenID: ", token.ID), zap.Error(err))
					return err
				}
				totalProvenances = append(totalProvenances, provenance...)
			case utils.TezosBlockchain:
				provenance, err := w.fetchTezosProvenance(tokenInfo.ID, tokenInfo.ContractAddress)
				if err != nil {
					log.Error("cannot fetch tezos provenance", zap.String("contractAddress: ", token.ContractAddress), zap.String("tokenID: ", token.ID), zap.Error(err))
					return err
				}
				totalProvenances = append(totalProvenances, provenance...)
			}
		}

		if err := w.indexerStore.UpdateTokenProvenance(ctx, token.IndexID, totalProvenances); err != nil {
			log.Error("cannot update token provenance", zap.String("contractAddress: ", token.ContractAddress), zap.String("tokenID: ", token.ID), zap.Error(err))
			return err
		}

		if len(totalProvenances) != 0 {
			ownerBalance := []indexer.OwnerBalance{
				{
					Address:  totalProvenances[0].Owner,
					Balance:  1,
					LastTime: totalProvenances[0].Timestamp,
				},
			}

			if err := w.indexerStore.UpdateAccountTokenOwners(ctx, token.IndexID, ownerBalance); err != nil {
				log.Error("cannot update account token owners", zap.String("tokenID: ", token.IndexID), zap.Error(err))
				return err
			}

			log.Debug("finish updating token owners")
		}
	}

	return nil
}

// RefreshTokenOwnership refreshes ownership for each tokens
func (w *NFTIndexerWorker) RefreshTokenOwnership(ctx context.Context, indexIDs []string, delay time.Duration) error {
	accountTokenLatestActivityTimes, err := w.indexerStore.GetLatestActivityTimeByIndexIDs(ctx, indexIDs)
	if err != nil {
		log.Error("fail to get tokens last activities", zap.Any("indexIDs", indexIDs), zap.Error(err))
		return err
	}

	tokens, err := w.indexerStore.GetTokensByIndexIDs(ctx, indexIDs)
	if err != nil {
		log.Error("fail to get tokens", zap.Any("indexIDs", indexIDs), zap.Error(err))
		return err
	}

	for _, token := range tokens {
		tokenLastActivityTime := token.LastActivityTime
		// replace the tokenLastActivityTime by the latest value in `account_tokens` collection.
		// this prevents the out sync of `account_tokens`
		if t, ok := accountTokenLatestActivityTimes[token.IndexID]; ok {
			tokenLastActivityTime = t
		}

		if token.LastRefreshedTime.Unix() > time.Now().Add(-delay).Unix() {
			log.Debug("ownership refresh too frequently",
				zap.Int64("lastRefresh", token.LastRefreshedTime.Unix()),
				zap.Int64("now", time.Now().Add(-delay).Unix()),
				zap.String("indexID", token.IndexID))
			continue
		}

		if !token.Fungible {
			log.Debug("ignore non-fungible token", zap.String("indexID", token.IndexID))
			continue
		}

		log.Debug("start refresh token ownership updating flow", zap.String("indexID", token.IndexID))
		var (
			ownerBalances           []indexer.OwnerBalance
			onChainLastActivityTime time.Time
			err                     error
		)
		switch token.Blockchain {
		case utils.EthereumBlockchain:
			// update ethereum last activity time by daily manner for now since this is a costy action
			if time.Since(token.LastActivityTime) <= 86400*time.Second && len(token.OwnersArray) != 0 {
				log.Debug("no new updates since last check", zap.String("indexID", token.IndexID))
				continue
			}

			log.Debug("fetch eth ownership for the token", zap.String("indexID", token.IndexID))
			ownerBalances, err = w.indexerEngine.IndexETHTokenOwners(token.ContractAddress, token.ID)
			if err != nil {
				log.Error("fail to fetch ownership", zap.String("indexID", token.IndexID), zap.Error(err))
				return err
			}
		case utils.TezosBlockchain:
			onChainLastActivityTime, err = w.indexerEngine.IndexTezosTokenLastActivityTime(token.ContractAddress, token.ID)
			if err != nil {
				log.Error("fail to get lastActivityTime", zap.String("indexID", token.IndexID), zap.Error(err))
				return err
			}

			if onChainLastActivityTime.Sub(tokenLastActivityTime) <= 0 {
				log.Debug("no new updates since last check", zap.String("indexID", token.IndexID))
				continue
			}

			log.Debug("fetch tezos ownership for the token", zap.String("indexID", token.IndexID))
			ownerBalances, err = w.indexerEngine.IndexTezosTokenOwners(token.ContractAddress, token.ID)
			if err != nil {
				log.Error("fail to fetch ownership", zap.String("indexID", token.IndexID), zap.Error(err))
				return err
			}
		}

		if err := w.indexerStore.UpdateTokenOwners(ctx, token.IndexID, onChainLastActivityTime, ownerBalances); err != nil {
			log.Error("fail to update token owners", zap.String("indexID", token.IndexID), zap.Any("owners", ownerBalances), zap.Error(err))
			return err
		}

		if err := w.indexerStore.UpdateAccountTokenOwners(ctx, token.IndexID, ownerBalances); err != nil {
			log.Error("fail to update account token owners", zap.String("indexID", token.IndexID), zap.Any("owners", ownerBalances), zap.Error(err))
			return err
		}
	}
	return nil
}

// GetPendingAccountTokens returns all account tokens which have pending txs
func (w *NFTIndexerWorker) GetPendingAccountTokens(ctx context.Context) ([]indexer.AccountToken, error) {
	return w.indexerStore.GetPendingAccountTokens(ctx)
}

// GetTxTimestamp returns transaction timestamp of a blockchain
func (w *NFTIndexerWorker) GetTxTimestamp(ctx context.Context, blockchain, txHash string) (time.Time, error) {
	return w.indexerEngine.GetTxTimestamp(ctx, blockchain, txHash)
}

// UpdatePendingTxsToAccountToken updates the the pending txs of an account token
func (w *NFTIndexerWorker) UpdatePendingTxsToAccountToken(ctx context.Context, ownerAccount, indexID string, pendingTxs []string, lastPendingTimes []time.Time) error {
	return w.indexerStore.UpdatePendingTxsToAccountToken(ctx, ownerAccount, indexID, pendingTxs, lastPendingTimes)
}

// GetBalanceDiffFromTezosTransaction gets the balance difference of TEZOS account tokens in a transaction.
func (w *NFTIndexerWorker) GetBalanceDiffFromTezosTransaction(transactionDetails tzkt.DetailedTransaction, accountToken indexer.AccountToken) ([]indexer.AccountToken, error) {
	var updatedAccountTokens = []indexer.AccountToken{}
	var totalTransferredAmount = int64(0)

	for _, txs := range transactionDetails.Parameter.Value[0].Txs {
		if txs.TokenID == strings.Split(accountToken.IndexID, "-")[2] {
			amount, err := strconv.ParseInt(txs.Amount, 10, 64)
			if err != nil {
				continue
			}

			receiverAccountToken := indexer.AccountToken{
				IndexID:          accountToken.IndexID,
				OwnerAccount:     txs.To,
				Balance:          amount,
				LastActivityTime: transactionDetails.Timestamp,
			}

			updatedAccountTokens = append(updatedAccountTokens, receiverAccountToken)
			totalTransferredAmount += amount
		}
	}
	senderAccountToken := indexer.AccountToken{
		IndexID:          accountToken.IndexID,
		OwnerAccount:     accountToken.OwnerAccount,
		Balance:          -totalTransferredAmount,
		LastActivityTime: transactionDetails.Timestamp,
	}

	updatedAccountTokens = append(updatedAccountTokens, senderAccountToken)
	return updatedAccountTokens, nil
}

// GetBalanceDiffFromETHTransaction gets the balance difference of account tokens in a transaction for a specific indexID.
func (w *NFTIndexerWorker) GetBalanceDiffFromETHTransaction(transactionDetails []indexer.TransactionDetails) ([]indexer.AccountToken, error) {
	var updatedAccountTokens = []indexer.AccountToken{}

	for _, transactionDetail := range transactionDetails {
		if transactionDetail.To != indexer.EthereumZeroAddress {
			receiverAccountToken := indexer.AccountToken{
				IndexID:          transactionDetail.IndexID,
				OwnerAccount:     transactionDetail.To,
				Balance:          1,
				LastActivityTime: transactionDetail.Timestamp,
			}
			updatedAccountTokens = append(updatedAccountTokens, receiverAccountToken)

		}

		if transactionDetail.From != indexer.EthereumZeroAddress {
			senderAccountToken := indexer.AccountToken{
				IndexID:          transactionDetail.IndexID,
				OwnerAccount:     transactionDetail.From,
				Balance:          -1,
				LastActivityTime: transactionDetail.Timestamp,
			}
			updatedAccountTokens = append(updatedAccountTokens, senderAccountToken)
		}
	}

	return updatedAccountTokens, nil
}

// IndexTezosTokenByOwner indexes Tezos token data for an owner into the format of AssetUpdates
func (w *NFTIndexerWorker) IndexTezosCollectionsByOwner(ctx context.Context, owner string, offset int) (int, error) {

	// account, err := w.indexerStore.GetAccount(ctx, owner)

	// if err != nil {
	// 	return 0, err
	// }

	// if offset == 0 {
	// 	delay := time.Minute
	// 	if account.LastUpdatedTime.Unix() > time.Now().Add(-delay).Unix() {
	// 		log.Debug("owner refresh too frequently",
	// 			zap.Int64("lastUpdatedTime", account.LastUpdatedTime.Unix()),
	// 			zap.Int64("now", time.Now().Add(-delay).Unix()),
	// 			zap.String("owner", account.Account))
	// 		return 0, nil
	// 	}
	// }

	collectionUpdates, err := w.indexerEngine.IndexTezosCollectionByOwner(ctx, owner, offset, QueryPageSize)
	if err != nil {
		return 0, err
	}

	if len(collectionUpdates) == 0 {
		return 0, err
	}

	for _, collection := range collectionUpdates {
		if err := w.indexerStore.IndexCollection(ctx, collection); err != nil {
			return 0, err
		}

		// Index gallery tokens
		nextOffset := 0
		for {
			galleryPK, err := strconv.ParseInt(collection.ExternalID, 10, 64)
			if err != nil {
				return 0, err
			}

			assetUpdates, err := w.indexerEngine.GetObjktTokensByGalleryPK(ctx, galleryPK, nextOffset, QueryPageSize)
			if err != nil {
				return 0, err
			}

			collectionAssets := []indexer.CollectionAsset{}

			// Index assets of the colections
			for _, asset := range assetUpdates {
				if err := w.indexerStore.IndexAsset(ctx, asset.ID, asset); err != nil {
					return 0, err
				}

				token := asset.Tokens[0]
				indexID := indexer.TokenIndexID(token.Blockchain, token.ContractAddress, token.ID)

				collectionAssets = append(collectionAssets, indexer.CollectionAsset{
					CollectionID: indexID,
					TokenIndexID: asset.ID,
				})
			}

			if err := w.indexerStore.IndexCollectionAsset(ctx, collection.ID, collectionAssets); err != nil {
				return 0, err
			}

			if len(assetUpdates) == QueryPageSize {
				nextOffset += QueryPageSize
			} else {
				break
			}
		}
	}

	if len(collectionUpdates) == QueryPageSize {
		return offset + QueryPageSize, nil
	} else {
		return 0, nil
	}
}
