package indexerWorker

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

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/contracts"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/bitmark-inc/nft-indexer/traceutils"
)

var (
	ErrMapKeyNotFound   = errors.New("key is not found")
	ErrValueNotString   = errors.New("value is not of string type")
	ErrInvalidEditionID = errors.New("invalid edition id")
)

// GetOwnedERC721TokenIDByContract returns a list of token id belongs to an owner for a specific contract
func (w *NFTIndexerWorker) GetOwnedERC721TokenIDByContract(ctx context.Context, contractAddress, ownerAddress string) ([]*big.Int, error) {
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
func (w *NFTIndexerWorker) IndexETHTokenByOwner(ctx context.Context, owner string, offset int) (int, error) {
	log.Debug("IndexETHTokenByOwner", zap.String("owner", owner))
	updates, err := w.indexerEngine.IndexETHTokenByOwner(ctx, owner, offset)
	if err != nil {
		return 0, err
	}

	if len(updates) == 0 {
		return 0, nil
	}

	accountTokens := []indexer.AccountToken{}

	for _, update := range updates {
		if err := w.indexerStore.IndexAsset(ctx, update.ID, update); err != nil {
			return 0, err
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
		return 0, err
	}

	return len(updates), nil
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

// IndexToken indexes a token by the given contract and token id
func (w *NFTIndexerWorker) IndexToken(ctx context.Context, owner, contract, tokenID string) (*indexer.AssetUpdates, error) {
	return w.indexerEngine.IndexToken(ctx, owner, contract, tokenID)
}

// GetTokenIDsByOwner gets tokenIDs by the given owner
func (w *NFTIndexerWorker) GetTokenIDsByOwner(ctx context.Context, owner string) ([]string, error) {
	return w.indexerStore.GetTokenIDsByOwner(ctx, owner)
}

// GetOutdatedTokensByOwner sets outdated tokens by the given owner
func (w *NFTIndexerWorker) GetOutdatedTokensByOwner(ctx context.Context, owner string) ([]indexer.Token, error) {
	return w.indexerStore.GetOutdatedTokensByOwner(ctx, owner)
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
			Blockchain: indexer.BitmarkBlockchain,
			Timestamp:  p.CreatedAt,
			TxID:       p.TxID,
			TxURL:      indexer.TxURL(indexer.BitmarkBlockchain, w.Environment, p.TxID),
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
	transferLogs, err := w.wallet.RPCClient().FilterLogs(ctx, goethereum.FilterQuery{
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

		txTime, err := indexer.GetETHBlockTime(ctx, w.wallet.RPCClient(), l.BlockHash)
		if err != nil {
			return nil, err
		}

		provenances = append(provenances, indexer.Provenance{
			Timestamp:   txTime,
			Type:        txType,
			Owner:       indexer.EthereumChecksumAddress(toAccountHash.Hex()),
			Blockchain:  indexer.EthereumBlockchain,
			BlockNumber: &l.BlockNumber,
			TxID:        l.TxHash.Hex(),
			TxURL:       indexer.TxURL(indexer.EthereumBlockchain, w.Environment, l.TxHash.Hex()),
		})
	}

	return provenances, nil
}

// fetchTezosProvenance reads ethereum provenance through filterLogs
func (w *NFTIndexerWorker) fetchTezosProvenance(ctx context.Context, tokenID, contractAddress string) ([]indexer.Provenance, error) {
	return w.indexerEngine.IndexTezosTokenProvenance(ctx, contractAddress, tokenID)
}

// RefreshTokenProvenance refresh provenance. This is a heavy task
func (w *NFTIndexerWorker) RefreshTokenProvenance(ctx context.Context, indexIDs []string, delay time.Duration) error {
	tokens, err := w.indexerStore.GetTokensByIndexIDs(ctx, indexIDs)
	if err != nil {
		log.Error("cannot find tokens in DB", zap.Any("indexIDs", indexIDs), zap.Error(err))
		return err
	}

	for _, token := range tokens {
		if token.LastRefreshedTime.Unix() > time.Now().Add(-delay).Unix() {
			log.Debug("provenance refresh too frequently", zap.String("indexID", token.IndexID))
			continue
		}

		if token.Fungible {
			log.Debug("ignore fungible token", zap.String("indexID", token.IndexID))
			continue
		}

		log.Debug("start refresh token provenance updating flow", zap.String("indexID", token.IndexID))

		totalProvenances := []indexer.Provenance{}
		switch token.Blockchain {
		case indexer.BitmarkBlockchain:
			provenance, err := w.fetchBitmarkProvenance(token.ID)
			if err != nil {
				log.Error("cannot fetch bitmark provenance", zap.String("tokenID: ", token.ID), zap.Error(err))
				return err
			}

			totalProvenances = append(totalProvenances, provenance...)
		case indexer.EthereumBlockchain:
			provenance, err := w.fetchEthereumProvenance(ctx, token.ID, token.ContractAddress)
			if err != nil {
				log.Error("cannot fetch ethereum provenance", zap.String("contractAddress: ", token.ContractAddress), zap.String("tokenID: ", token.ID), zap.Error(err))
				return err
			}
			totalProvenances = append(totalProvenances, provenance...)
		case indexer.TezosBlockchain:
			lastActivityTime, err := w.indexerEngine.IndexTezosTokenLastActivityTime(ctx, token.ContractAddress, token.ID)
			if err != nil {
				log.Error("cannot fetch lastActivityTime", zap.String("contractAddress: ", token.ContractAddress), zap.String("tokenID: ", token.ID), zap.Error(err))
				return err
			}

			if delay > 0 && lastActivityTime.Sub(token.LastActivityTime) <= 0 {
				log.Debug("no new updates since last check", zap.String("indexID", token.IndexID))
				continue
			}

			provenance, err := w.fetchTezosProvenance(ctx, token.ID, token.ContractAddress)
			if err != nil {
				log.Error("cannot fetch tezos provenance", zap.String("contractAddress: ", token.ContractAddress), zap.String("tokenID: ", token.ID), zap.Error(err))
				return err
			}
			totalProvenances = append(totalProvenances, provenance...)
		}

		// recursively fetch provenance records from migrated blockchains
		for _, tokenInfo := range token.OriginTokenInfo {
			switch tokenInfo.Blockchain {
			case indexer.BitmarkBlockchain:
				provenance, err := w.fetchBitmarkProvenance(tokenInfo.ID)
				if err != nil {
					log.Error("cannot fetch bitmark provenance", zap.String("tokenID: ", tokenInfo.ID), zap.Error(err))
					return err
				}

				totalProvenances = append(totalProvenances, provenance...)
			case indexer.EthereumBlockchain:
				provenance, err := w.fetchEthereumProvenance(ctx, tokenInfo.ID, tokenInfo.ContractAddress)
				if err != nil {
					log.Error("cannot fetch ethereum provenance", zap.String("contractAddress: ", token.ContractAddress), zap.String("tokenID: ", token.ID), zap.Error(err))
					return err
				}
				totalProvenances = append(totalProvenances, provenance...)
			case indexer.TezosBlockchain:
				provenance, err := w.fetchTezosProvenance(ctx, tokenInfo.ID, tokenInfo.ContractAddress)
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
			owner := map[string]int64{totalProvenances[0].Owner: 1}
			if err := w.indexerStore.UpdateAccountTokenOwners(ctx, token.IndexID, totalProvenances[0].Timestamp, owner); err != nil {
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
	indexTokens := map[string]indexer.AccountToken{}

	accountTokens, err := w.indexerStore.GetAccountTokensByIndexIDs(ctx, indexIDs)
	if err != nil {
		log.Error("fail to get account tokens", zap.Any("indexIDs", indexIDs), zap.Error(err))
		return err
	}

	for _, token := range accountTokens {
		indexTokens[token.IndexID] = token
	}

	tokens, err := w.indexerStore.GetTokensByIndexIDs(ctx, indexIDs)
	if err != nil {
		log.Error("fail to get tokens", zap.Any("indexIDs", indexIDs), zap.Error(err))
		return err
	}

	for _, token := range tokens {
		_, tokenExist := indexTokens[token.IndexID]
		if !tokenExist {
			indexTokens[token.AssetID] = indexer.AccountToken{
				BaseTokenInfo:     token.BaseTokenInfo,
				IndexID:           token.IndexID,
				OwnerAccount:      token.Owner,
				Balance:           token.Balance,
				LastActivityTime:  token.LastActivityTime,
				LastRefreshedTime: token.LastRefreshedTime,
			}
		}
	}

	for _, token := range indexTokens {
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
			owners           map[string]int64
			lastActivityTime time.Time
			err              error
		)
		switch token.Blockchain {
		case indexer.EthereumBlockchain:
			lastActivityTime, err = w.indexerEngine.IndexETHTokenLastActivityTime(ctx, token.ContractAddress, token.ID)
			if err != nil {
				log.Error("fail to get lastActivityTime", zap.String("indexID", token.IndexID), zap.Error(err))
				return err
			}

			if lastActivityTime.Sub(token.LastActivityTime) <= 0 {
				log.Debug("no new updates since last check", zap.String("indexID", token.IndexID))
				continue
			}

			log.Debug("fetch eth ownership for the token", zap.String("indexID", token.IndexID))
			owners, err = w.indexerEngine.IndexETHTokenOwners(ctx, token.ContractAddress, token.ID)
			if err != nil {
				log.Error("fail to fetch ownership", zap.String("indexID", token.IndexID), zap.Error(err))
				return err
			}
		case indexer.TezosBlockchain:
			lastActivityTime, err = w.indexerEngine.IndexTezosTokenLastActivityTime(ctx, token.ContractAddress, token.ID)
			if err != nil {
				log.Error("fail to get lastActivityTime", zap.String("indexID", token.IndexID), zap.Error(err))
				return err
			}

			if lastActivityTime.Sub(token.LastActivityTime) <= 0 {
				log.Debug("no new updates since last check", zap.String("indexID", token.IndexID))
				continue
			}

			log.Debug("fetch tezos ownership for the token", zap.String("indexID", token.IndexID))
			owners, err = w.indexerEngine.IndexTezosTokenOwners(ctx, token.ContractAddress, token.ID)
			if err != nil {
				log.Error("fail to fetch ownership", zap.String("indexID", token.IndexID), zap.Error(err))
				return err
			}
		}

		if err := w.indexerStore.UpdateTokenOwners(ctx, token.IndexID, lastActivityTime, owners); err != nil {
			log.Error("fail to update token owners", zap.String("indexID", token.IndexID), zap.Any("owners", owners), zap.Error(err))
			return err
		}

		if err := w.indexerStore.UpdateAccountTokenOwners(ctx, token.IndexID, lastActivityTime, owners); err != nil {
			log.Error("fail to update account token owners", zap.String("indexID", token.IndexID), zap.Any("owners", owners), zap.Error(err))
			return err
		}
	}
	return nil
}

// UpdateAccountTokens updates all pending account tokens
func (w *NFTIndexerWorker) UpdateAccountTokens(ctx context.Context) error {
	pendingAccountTokens, err := w.indexerStore.GetPendingAccountTokens(ctx)
	if err != nil {
		log.Warn("errors in the pending account tokens")
		return err
	}

	delay := time.Hour
	for _, pendingAccountToken := range pendingAccountTokens {
		for idx, pendingTx := range pendingAccountToken.PendingTxs {
			if pendingAccountToken.LastPendingTime[idx].Unix() < time.Now().Add(-delay).Unix() {
				log.Warn("pending too long", zap.Any("pendingTxs", pendingAccountToken.PendingTxs))
				err := w.indexerStore.DeletePendingFieldsAccountToken(ctx, pendingAccountToken.OwnerAccount, pendingAccountToken.IndexID, pendingTx, pendingAccountToken.LastPendingTime[idx])
				if err != nil {
					log.Error("fail to clean up pending field", zap.Error(err),
						zap.String("indexID", pendingAccountToken.IndexID))
				}
				continue
			}

			accountTokens := []indexer.AccountToken{}
			switch pendingAccountToken.Blockchain {
			case indexer.TezosBlockchain:
				transactionDetails, err := w.indexerEngine.GetTransactionDetailsByPendingTx(pendingTx)
				if err != nil {
					log.Error("fail to get pending txs for tezos", zap.Error(err),
						zap.String("indexID", pendingAccountToken.IndexID))
					continue
				}

				if len(transactionDetails) == 0 {
					log.Error("pending txs not found", zap.String("indexID", pendingAccountToken.IndexID))
					continue
				}

				accountTokens, err = w.GetBalanceDiffFromTezosTransaction(transactionDetails[0], pendingAccountToken)
				if err != nil {
					log.Error("fail to calculate balance difference from tezos tx", zap.Error(err),
						zap.String("indexID", pendingAccountToken.IndexID))
					continue
				}
			case indexer.EthereumBlockchain:
				txHash := common.HexToHash(pendingTx)
				transactionDetails, err := w.indexerEngine.GetETHTransactionDetailsByPendingTx(ctx, w.wallet.RPCClient(), txHash, pendingAccountToken.ID)
				if err != nil {
					log.Error("fail to get pending txs for ethereum", zap.Error(err),
						zap.String("indexID", pendingAccountToken.IndexID))
					continue
				}

				if len(transactionDetails) == 0 {
					log.Error("pending txs not found", zap.String("indexID", pendingAccountToken.IndexID))
					continue
				}

				accountTokens, err = w.GetBalanceDiffFromETHTransaction(transactionDetails)
				if err != nil {
					log.Error("fail to calculate balance difference from ethereum tx", zap.Error(err),
						zap.String("indexID", pendingAccountToken.IndexID))
					continue
				}
			}

			for _, accountToken := range accountTokens {
				err = w.indexerStore.UpdateAccountTokenBalance(ctx, accountToken.OwnerAccount, accountToken.IndexID, accountToken.Balance, accountToken.LastActivityTime, pendingTx, pendingAccountToken.LastPendingTime[idx])
				if err != nil {
					log.Error("fail to update account token balance", zap.Error(err),
						zap.String("indexID", pendingAccountToken.IndexID))
					continue
				}
			}
		}
	}
	return nil
}

// GetBalanceDiffFromTezosTransaction gets the balance difference of TEZOS account tokens in a transaction.
func (w *NFTIndexerWorker) GetBalanceDiffFromTezosTransaction(transactionDetails tzkt.TransactionDetails, accountToken indexer.AccountToken) ([]indexer.AccountToken, error) {
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

// CalculateMimeTypeFromTokenFeedback calculate mimeType from token_feeback and update into token suggestedMimeType
func (w *NFTIndexerWorker) CalculateMIMETypeFromTokenFeedback(ctx context.Context) error {
	grouppedTokenFeedback, err := w.indexerStore.GetGrouppedTokenFeedbacks(ctx)

	if err != nil {
		log.Warn("errors in the GetGrouppedTokenFeedbacks")
		return err
	}

	for _, token := range grouppedTokenFeedback {
		max := 0
		total := 0
		suggestedMimeType := ""
		for _, m := range token.MimeTypes {
			total += m.Count
			if m.Count > max {
				max = m.Count
				suggestedMimeType = m.MimeType
			}
		}

		if total == 0 {
			continue
		}

		if max*100.0/total >= 75 {
			err = w.indexerStore.UpdateTokenSugesstedMIMEType(ctx, token.IndexID, suggestedMimeType)
			if err != nil {
				log.Error("failed to update token suggested MIME Type",
					zap.Error(err),
					zap.String("indexID", token.IndexID),
					zap.String("suggestedMimeType", suggestedMimeType),
				)
				return err
			}
		}
	}

	return nil
}

// UpdatePresignedThumbnailAssets detects pre-sign fxhash thumbnail and trigger IndexAsset
func (w *NFTIndexerWorker) UpdatePresignedThumbnailAssets(ctx context.Context) error {
	presignedThumbnailTokens, err := w.indexerStore.GetPresignedThumbnailTokens(ctx)
	if err != nil {
		log.Debug("errors in the pending account tokens", zap.Error(err))
		return err
	}

	updatedIndexIDs := []string{}
	for _, token := range presignedThumbnailTokens {
		assetUpdates, err := w.indexerEngine.IndexTezosToken(ctx, token.Owner, token.ContractAddress, token.ID)
		if err != nil {
			log.Error("fail to get updates of a tezos token", zap.String("indexID", token.IndexID), zap.Error(err))
			continue
		}

		err = w.indexerStore.IndexAsset(ctx, token.AssetID, *assetUpdates)
		if err != nil {
			log.Error("fail to update a tezos asset", zap.String("assetID", token.AssetID), zap.Error(err))
			continue
		}

		for _, token := range assetUpdates.Tokens {
			updatedIndexIDs = append(updatedIndexIDs, token.IndexID)
		}
	}

	if err := w.indexerStore.MarkAccountTokenChanged(ctx, updatedIndexIDs); err != nil {
		log.Error("fail to update account tokens", zap.Any("indexIDs", updatedIndexIDs), zap.Error(err))
		return err
	}

	return nil
}
