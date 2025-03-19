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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	utils "github.com/bitmark-inc/autonomy-utils"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/contracts"
	"github.com/bitmark-inc/nft-indexer/externals/coinbase"
	"github.com/bitmark-inc/nft-indexer/externals/etherscan"
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

// GetTokenByIndexID gets a token by indexID
func (w *NFTIndexerWorker) GetTokenByIndexID(ctx context.Context, indexID string) (*indexer.Token, error) {
	return w.indexerStore.GetTokenByIndexID(ctx, indexID)
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

// WriteSaleTimeSeriesData saves sales time series data into indexer's storage
func (w *NFTIndexerWorker) WriteSaleTimeSeriesData(ctx context.Context, data []indexer.GenericSalesTimeSeries) error {
	return w.indexerStore.WriteTimeSeriesData(ctx, data)
}

// IndexedSaleTx checks if a sale tx is indexed
func (w *NFTIndexerWorker) IndexedSaleTx(ctx context.Context, txID, blockchain string) (bool, error) {
	return w.indexerStore.SaleTimeSeriesDataExists(ctx, txID, blockchain)
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

func (w *NFTIndexerWorker) GetExchangeRateLastTime(ctx context.Context) (time.Time, error) {
	return w.indexerStore.GetExchangeRateLastTime(ctx)
}

func (w *NFTIndexerWorker) WriteHistoricalExchangeRate(ctx context.Context, exchangeRate []coinbase.HistoricalExchangeRate) error {
	return w.indexerStore.WriteHistoricalExchangeRate(ctx, exchangeRate)
}

func (w *NFTIndexerWorker) CrawlExchangeRateFromCoinbase(
	ctx context.Context,
	currencyPair string,
	granularity string,
	start int64,
	end int64,
) ([]coinbase.HistoricalExchangeRate, error) {
	client := coinbase.NewClient()
	return client.GetCandles(ctx, currencyPair, granularity, start, end)
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
func (w *NFTIndexerWorker) fetchBitmarkProvenance(_ context.Context, bitmarkID string) ([]indexer.Provenance, error) {
	provenanceResp, err := w.bitmarkdClient.GetBitmarkFullProvenance(bitmarkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bitmark provenance: %w", err)
	}

	var data struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(provenanceResp, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal provenance data: %w", err)
	}

	s := len(data.Data)
	if s == 0 {
		return nil, fmt.Errorf("no provenance data found for bitmark ID: %s", bitmarkID)
	}

	provenanceData := data.Data[0 : s-1] // the last item is the asset data
	provenances := make([]indexer.Provenance, 0, len(provenanceData))

	for _, d := range provenanceData {
		// unmarshal the provenance data
		var p struct {
			Record string                 `json:"record"`
			TxID   string                 `json:"txId"`
			Block  string                 `json:"inBlock"`
			Data   map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(d, &p); err != nil {
			return nil, fmt.Errorf("failed to unmarshal provenance item: %w", err)
		}

		// get the owner
		owner, ok := p.Data["owner"].(string)
		if !ok {
			return nil, fmt.Errorf("owner is not a string in provenance data")
		}

		// get the tx type
		txType := "transfer"
		if p.Record == "BitmarkIssue" {
			txType = "issue"
		} else if owner == w.bitmarkZeroAddress {
			txType = "burn"
		}

		// get the block height and timestamp
		blockHeight, err := strconv.ParseUint(p.Block, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse block height: %w", err)
		}

		blockResp, err := w.bitmarkdClient.BlockDump(blockHeight, false)
		if err != nil {
			return nil, fmt.Errorf("failed to get block dump: %w", err)
		}

		var blockData struct {
			Block struct {
				Header struct {
					Timestamp string `json:"timestamp"`
				} `json:"header"`
			} `json:"block"`
		}
		if err := json.Unmarshal(blockResp, &blockData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal block data: %w", err)
		}

		timestamp, err := strconv.ParseInt(blockData.Block.Header.Timestamp, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp: %w", err)
		}

		provenances = append(provenances, indexer.Provenance{
			Type:        txType,
			Owner:       owner,
			Blockchain:  utils.BitmarkBlockchain,
			BlockNumber: &blockHeight,
			Timestamp:   time.Unix(timestamp, 0),
			TxID:        p.TxID,
			TxURL:       indexer.TxURL(utils.BitmarkBlockchain, w.Environment, p.TxID),
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
func (w *NFTIndexerWorker) fetchTezosProvenance(ctx context.Context, tokenID, contractAddress string) ([]indexer.Provenance, error) {
	return w.indexerEngine.IndexTezosTokenProvenance(contractAddress, tokenID)
}

// RefreshTokenProvenance refresh provenance. This is a heavy task
func (w *NFTIndexerWorker) RefreshTokenProvenance(ctx context.Context, indexIDs []string, delay time.Duration) error {
	tokens, err := w.indexerStore.GetTokensByIndexIDs(ctx, indexIDs)
	if err != nil {
		return err
	}

	for _, token := range tokens {
		if delay > 0 {
			if token.LastRefreshedTime.Unix() > time.Now().Add(-delay).Unix() {
				continue
			}
		}

		if token.Fungible {
			continue
		}

		totalProvenances := []indexer.Provenance{}
		switch token.Blockchain {
		case utils.BitmarkBlockchain:
			provenance, err := w.fetchBitmarkProvenance(ctx, token.ID)
			if err != nil {
				return err
			}

			totalProvenances = append(totalProvenances, provenance...)
		case utils.EthereumBlockchain:
			provenance, err := w.fetchEthereumProvenance(ctx, token.ID, token.ContractAddress)
			if err != nil {
				return err
			}
			totalProvenances = append(totalProvenances, provenance...)
		case utils.TezosBlockchain:
			lastActivityTime, err := w.indexerEngine.IndexTezosTokenLastActivityTime(token.ContractAddress, token.ID)
			if err != nil {
				return err
			}

			// Ignore the refreshing process when
			// 1. provenances is not empty
			// 2. the latest provenance timestamp match the token's lastActivityTime
			// 3. the token's lastActivityTime is greater than the record from tzkt
			if len(token.Provenances) != 0 &&
				token.Provenances[0].Timestamp.Sub(token.LastActivityTime) == 0 &&
				lastActivityTime.Sub(token.LastActivityTime) <= 0 {
				continue
			}

			provenance, err := w.fetchTezosProvenance(ctx, token.ID, token.ContractAddress)
			if err != nil {
				return err
			}
			totalProvenances = append(totalProvenances, provenance...)
		}

		// recursively fetch provenance records from migrated blockchains
		for _, tokenInfo := range token.OriginTokenInfo {
			switch tokenInfo.Blockchain {
			case utils.BitmarkBlockchain:
				provenance, err := w.fetchBitmarkProvenance(ctx, tokenInfo.ID)
				if err != nil {
					return err
				}

				totalProvenances = append(totalProvenances, provenance...)
			case utils.EthereumBlockchain:
				provenance, err := w.fetchEthereumProvenance(ctx, tokenInfo.ID, tokenInfo.ContractAddress)
				if err != nil {
					return err
				}
				totalProvenances = append(totalProvenances, provenance...)
			case utils.TezosBlockchain:
				provenance, err := w.fetchTezosProvenance(ctx, tokenInfo.ID, tokenInfo.ContractAddress)
				if err != nil {
					return err
				}
				totalProvenances = append(totalProvenances, provenance...)
			}
		}

		if err := w.indexerStore.UpdateTokenProvenance(ctx, token.IndexID, totalProvenances); err != nil {
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
				return err
			}
		}
	}

	return nil
}

// RefreshTokenOwnership refreshes ownership for each tokens
func (w *NFTIndexerWorker) RefreshTokenOwnership(ctx context.Context, indexIDs []string, delay time.Duration) error {
	accountTokenLatestActivityTimes, err := w.indexerStore.GetLatestActivityTimeByIndexIDs(ctx, indexIDs)
	if err != nil {
		return err
	}

	tokens, err := w.indexerStore.GetTokensByIndexIDs(ctx, indexIDs)
	if err != nil {
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
			continue
		}

		if !token.Fungible {
			continue
		}

		var (
			ownerBalances           []indexer.OwnerBalance
			onChainLastActivityTime time.Time
			err                     error
		)
		switch token.Blockchain {
		case utils.EthereumBlockchain:
			// update ethereum last activity time by daily manner for now since this is a costy action
			if time.Since(token.LastActivityTime) <= 86400*time.Second && len(token.OwnersArray) != 0 {
				continue
			}

			ownerBalances, err = w.indexerEngine.IndexETHTokenOwners(token.ContractAddress, token.ID)
			if err != nil {
				return err
			}
		case utils.TezosBlockchain:
			onChainLastActivityTime, err = w.indexerEngine.IndexTezosTokenLastActivityTime(token.ContractAddress, token.ID)
			if err != nil {
				return err
			}

			if onChainLastActivityTime.Sub(tokenLastActivityTime) <= 0 {
				continue
			}

			ownerBalances, err = w.indexerEngine.IndexTezosTokenOwners(token.ContractAddress, token.ID)
			if err != nil {
				return err
			}
		}

		if err := w.indexerStore.UpdateTokenOwners(ctx, token.IndexID, onChainLastActivityTime, ownerBalances); err != nil {
			return err
		}

		if err := w.indexerStore.UpdateAccountTokenOwners(ctx, token.IndexID, ownerBalances); err != nil {
			return err
		}
	}
	return nil
}

// GetBalanceDiffFromTezosTransaction gets the balance difference of TEZOS account tokens in a transaction.
func (w *NFTIndexerWorker) GetBalanceDiffFromTezosTransaction(transactionDetails tzkt.DetailedTransaction, accountToken indexer.AccountToken) ([]indexer.AccountToken, error) {
	var updatedAccountTokens = []indexer.AccountToken{}
	var totalTransferredAmount = int64(0)

	paramValues, err := decodeParametersValue(transactionDetails.Parameter.Value)
	if err != nil {
		return nil, err
	}

	for _, paramValue := range paramValues {
		for _, txs := range paramValue.Txs {
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

// GetEthereumTxReceipt returns the ethereum transaction receipt object of a tx hash
func (w *NFTIndexerWorker) GetEthereumTxReceipt(ctx context.Context, txID string) (*types.Receipt, error) {
	return w.ethClient.TransactionReceipt(ctx, common.HexToHash(txID))
}

// GetEthereumTx returns the ethereum transaction object of a tx hash
func (w *NFTIndexerWorker) GetEthereumTx(ctx context.Context, txID string) (*types.Transaction, error) {
	tx, _, err := w.ethClient.TransactionByHash(ctx, common.HexToHash(txID))
	return tx, err
}

// GetEthereumBlockHeaderHash returns the ethereum block object of a block hash
func (w *NFTIndexerWorker) GetEthereumBlockHeaderHash(ctx context.Context, blkHash string) (*types.Header, error) {
	return w.ethClient.HeaderByHash(ctx, common.HexToHash(blkHash))
}

// GetEthereumBlockHeaderByNumber returns the ethereum block object of a block number
func (w *NFTIndexerWorker) GetEthereumBlockHeaderByNumber(ctx context.Context, blkNumber *big.Int) (*types.Header, error) {
	return w.ethClient.HeaderByNumber(ctx, blkNumber)
}

// GetEthereumInternalTxs returns the ethereum internal transactions of a tx hash
func (w *NFTIndexerWorker) GetEthereumInternalTxs(ctx context.Context, txID string) ([]etherscan.Transaction, error) {
	ec := etherscan.NewClient(
		viper.GetString("etherscan.url"),
		viper.GetString("etherscan.apikey"))
	return ec.Account.ListInternalTxs(
		ctx,
		etherscan.TransactionQueryParams{TxHash: &txID})
}

// FilterEthereumNFTTxByEventLogs filters ethereum NFT txs by event logs
func (w *NFTIndexerWorker) FilterEthereumNFTTxByEventLogs(
	ctx context.Context,
	addresses []string,
	fromBlk uint64,
	toBlk uint64) ([]string, error) {
	topics := [][]common.Hash{
		{
			common.HexToHash(indexer.TransferEventSignature),
			common.HexToHash(indexer.TransferSingleEventSignature)},
	}

	var filterAddress []common.Address
	for _, addr := range addresses {
		filterAddress = append(filterAddress, common.HexToAddress(addr))
	}

	// Filter logs
	evts, err := w.ethClient.FilterLogs(ctx, goethereum.FilterQuery{
		Addresses: filterAddress,
		Topics:    topics,
		FromBlock: new(big.Int).SetUint64(fromBlk),
		ToBlock:   new(big.Int).SetUint64(toBlk),
	})
	if nil != err {
		return nil, err
	}

	// Dedup txs, collect only ERC721 and ERC1155 txs
	txMap := make(map[string]struct{})
	for _, evt := range evts {
		if indexer.ERC721Transfer(evt) || indexer.ERC1155SingleTransfer(evt) {
			txMap[evt.TxHash.Hex()] = struct{}{}
		}
	}

	// Convert to array
	txs := make([]string, 0, len(txMap))
	for tx := range txMap {
		txs = append(txs, tx)
	}

	return txs, nil
}

func (w *NFTIndexerWorker) ParseTokenSaleToGenericSalesTimeSeries(
	ctx context.Context,
	tokenSale TokenSale) (*indexer.GenericSalesTimeSeries, error) {
	shares := make(map[string]string)
	for address, share := range tokenSale.Shares {
		shares[address] = share.String()
	}

	values := make(map[string]string)
	values["price"] = tokenSale.Price.String()
	values["platformFee"] = tokenSale.PlatformFee.String()
	values["netRevenue"] = tokenSale.NetRevenue.String()
	values["paymentAmount"] = tokenSale.PaymentAmount.String()
	values["exchangeRate"] = "1"

	bundleTokenInfo := []map[string]interface{}{}
	for _, info := range tokenSale.BundleTokenInfo {
		var tokenInfo map[string]interface{}
		err := mapstructure.Decode(info, &tokenInfo)
		if err != nil {
			return nil, err
		}
		bundleTokenInfo = append(bundleTokenInfo, tokenInfo)
	}

	metadata := map[string]interface{}{
		"blockchain":      tokenSale.Blockchain,
		"marketplace":     tokenSale.Marketplace,
		"paymentCurrency": tokenSale.Currency,
		"paymentMethod":   "crypto",
		"pricingCurrency": tokenSale.Currency,
		"revenueCurrency": tokenSale.Currency,
		"saleType":        "secondary",
		"transactionIDs":  []string{tokenSale.TxID},
		"bundleTokenInfo": bundleTokenInfo,
	}

	return &indexer.GenericSalesTimeSeries{
		Timestamp: tokenSale.Timestamp.Format(time.RFC3339Nano),
		Metadata:  metadata,
		Shares:    shares,
		Values:    values,
	}, nil
}

// GetTezosTxHashFromTzktTransactionID get tezos hash from transaction ID
func (w *NFTIndexerWorker) GetTezosTxHashFromTzktTransactionID(_ context.Context, id uint64) (*string, error) {
	tx, err := w.indexerEngine.GetTzktTransactionByID(id)
	if err != nil {
		return nil, err
	}

	return &tx.Hash, nil
}

// GetObjktSaleTransactionHashes get objkt sale transaction hashes by time with paging
func (w *NFTIndexerWorker) GetObjktSaleTransactionHashes(_ context.Context, lastTime *time.Time, offset, limit int) ([]string, error) {
	var contracts []string
	if viper.GetString("network") == "testnet" {
		contracts = []string{indexer.TezosOBJKTMarketplaceAddressTestnet}
	} else {
		contracts = []string{indexer.TezosOBJKTMarketplaceAddress, indexer.TezosOBJKTMarketplaceAddressV2}
	}

	txs, err := w.indexerEngine.GetTzktTransactionByContractsAndEntrypoint(
		contracts,
		indexer.OBJKTSaleEntrypoints,
		lastTime,
		offset,
		limit)
	if err != nil {
		return nil, err
	}

	hashes := []string{}
	for _, tx := range txs {
		hashes = append(hashes, tx.Hash)
	}

	return hashes, nil
}

// GetTzktTransactionByID get tezos transaction hash by tzkt transaction id
func (w *NFTIndexerWorker) ParseTezosObjktTokenSale(ctx context.Context, hash string) (*TokenSale, error) {

	txs, err := w.indexerEngine.GetTzktTransactionsByHash(hash)
	if err != nil {
		return nil, err
	}

	if len(txs) < 2 {
		return nil, errors.New("invalid objkt tx")
	}

	saleEntrypointMap := make(map[string]bool)
	for _, entrypoint := range indexer.OBJKTSaleEntrypoints {
		saleEntrypointMap[entrypoint] = true
	}

	isValidObjktSaleOperation := false
	bundleTokenInfo := []TokenSaleInfo{}
	price := big.NewInt(0)
	platformFeeWallets := viper.GetStringMapString("marketplace.fee_wallets") // key is lower case
	platformFee := big.NewInt(0)
	shares := make(map[string]*big.Int)
	for _, tx := range txs {
		if tx.Status != "applied" {
			return nil, nil
		}

		if tx.Parameter != nil {
			// check for fulfill entrypoints
			if saleEntrypointMap[tx.Parameter.EntryPoint] {
				isValidObjktSaleOperation = true
				continue
			}

			// process token transfers
			if tx.Parameter.EntryPoint == "transfer" {
				paramValues, err := decodeParametersValue(tx.Parameter.Value)
				if err != nil {
					// We don't support sale operations contain coin transfer operations
					// Any sale operations contain invalid "transfer" will be ignored
					return nil, errors.New("invalid transfer transaction - not supported buying using token")
				}

				for _, paramValue := range paramValues {
					for _, ptx := range paramValue.Txs {
						bundleTokenInfo = append(bundleTokenInfo, TokenSaleInfo{
							SellerAddress:   paramValue.From,
							BuyerAddress:    ptx.To,
							TokenID:         ptx.TokenID,
							ContractAddress: tx.Target.Address,
						})
					}
				}
			}
		} else {
			// process revenue shares transfers
			amount := big.NewInt(int64(tx.Amount))

			// ignore proxy transfer to ProxyAddress for objktV1 contract
			if tx.Target.Address == indexer.TezosOBJKTTreasuryProxyAddress {
				continue
			}

			// Accumulate shares
			if s, ok := shares[tx.Target.Address]; ok {
				shares[tx.Target.Address] = big.NewInt(0).Add(s, amount)
			} else {
				shares[tx.Target.Address] = amount
			}
			price = big.NewInt(0).Add(price, amount)

			// Deduct share
			if s, ok := shares[tx.Sender.Address]; ok {
				shares[tx.Sender.Address] = big.NewInt(0).Sub(s, amount)
				price.Sub(price, amount)
			}

			// Accumulate platform fee
			if platformFeeWallets[strings.ToLower(tx.Target.Address)] == "Objkt" {
				platformFee = big.NewInt(0).Add(platformFee, amount)
			}
		}
	}

	if !isValidObjktSaleOperation {
		return nil, errors.New("invalid objkt tx")
	}

	if len(bundleTokenInfo) == 0 {
		return nil, errors.New("invalid sale transaction - no tokens transfer")
	}

	return &TokenSale{
		Timestamp:       txs[0].Timestamp,
		Price:           price,
		Marketplace:     "Objkt",
		Blockchain:      "tezos",
		Currency:        "XTZ",
		TxID:            hash,
		PlatformFee:     platformFee,
		NetRevenue:      big.NewInt(0).Sub(price, platformFee),
		BundleTokenInfo: bundleTokenInfo,
		PaymentAmount:   price,
		Shares:          shares,
	}, nil
}

func parseArraryMapInterface(input interface{}) ([]map[string]interface{}, error) {
	slice, ok := input.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid slice of interface")
	}

	data := make([]map[string]interface{}, len(slice))
	for i, item := range slice {
		m, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid slice of map string interface")
		}
		data[i] = m
	}

	return data, nil
}

// Decode Tzkt transfer ParametersValue from an interface
func decodeParametersValue(input interface{}) (paramValues []tzkt.ParametersValue, err error) {
	data, err := parseArraryMapInterface(input)
	if err != nil {
		return nil, err
	}

	var ok bool
	for _, paramValue := range data {
		var valueSlice []interface{}
		for _, value := range paramValue {
			valueSlice = append(valueSlice, value)
		}

		if len(valueSlice) != 2 {
			return nil, fmt.Errorf("invalid param value legth")
		}

		var txs []map[string]interface{}
		var from string
		txs, err := parseArraryMapInterface(valueSlice[0])
		if err != nil {
			txs, err = parseArraryMapInterface(valueSlice[1])
			if err != nil {
				return nil, fmt.Errorf("invalid param value parse txs")
			}

			from, ok = valueSlice[0].(string)
			if !ok {
				return nil, fmt.Errorf("invalid parame value parse from")
			}
		} else {
			from, ok = valueSlice[1].(string)
			if !ok {
				return nil, fmt.Errorf("invalid parame value parse from")
			}
		}

		txsResult := []tzkt.TxsFormat{}
		for _, tx := range txs {
			to, ok := tx["to_"].(string)
			if !ok {
				to, ok = tx["to"].(string)
				if !ok {
					return nil, fmt.Errorf("invalid tx value parse to")
				}
			}

			amount, ok := tx["amount"].(string)
			if !ok {
				return nil, fmt.Errorf("invalid tx value parse amount")
			}

			tokenID, ok := tx["token_id"].(string)
			if !ok {
				return nil, fmt.Errorf("invalid tx value parse token_id")
			}

			txsResult = append(txsResult, tzkt.TxsFormat{
				To:      to,
				Amount:  amount,
				TokenID: tokenID,
			})
		}

		paramValues = append(paramValues, tzkt.ParametersValue{
			Txs:  txsResult,
			From: from,
		})
	}

	return paramValues, nil
}
