package worker

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"
	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/externals/etherscan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/spf13/viper"
)

type TokenSaleInfo struct {
	ContractAddress string `json:"contractAddress" mapstructure:"contractAddress"`
	TokenID         string `json:"tokenID" mapstructure:"tokenID"`
	SellerAddress   string `json:"sellerAddress" mapstructure:"sellerAddress"`
	BuyerAddress    string `json:"buyerAddress" mapstructure:"buyerAddress"`
}

type TokenSale struct {
	Timestamp       time.Time           `json:"timestamp"`
	BundleTokenInfo []TokenSaleInfo     `json:"bundleTokenInfo"`
	Price           *big.Int            `json:"price"`
	Marketplace     string              `json:"marketplace"`
	Blockchain      string              `json:"blockchain"`
	Currency        string              `json:"currency"`
	TxID            string              `json:"txID"`
	PlatformFee     *big.Int            `json:"platformFee"`
	NetRevenue      *big.Int            `json:"netRevenue"`
	PaymentAmount   *big.Int            `json:"paymentAmount"`
	Shares          map[string]*big.Int `json:"shares"`
}

// IndexEthereumTokenSale is a workflow to index the sale of an Ethereum token
func (w *NFTIndexerWorker) IndexEthereumTokenSale(
	ctx workflow.Context,
	txID string,
	skipIndexed bool) error {
	ctx = ContextRegularActivity(ctx, TaskListName)
	ctx = ContextRegularChildWorkflow(ctx, TaskListName)
	logger := log.CadenceWorkflowLogger(ctx)

	if skipIndexed {
		// Check if sale tx is indexed already
		var indexed bool
		if err := workflow.ExecuteActivity(
			ctx,
			w.IndexedSaleTx,
			txID,
			utils.EthereumBlockchain).
			Get(ctx, &indexed); nil != err {
			logger.Error(errors.New("fail to check if sale tx is indexed"), zap.Error(err))
			return err
		}

		if indexed {
			return nil
		}
	}

	// Parse tx
	var tokenSale *TokenSale
	if err := workflow.ExecuteChildWorkflow(
		ctx,
		w.ParseEthereumTokenSale,
		txID).
		Get(ctx, &tokenSale); nil != err {
		logger.Error(errors.New("fail to parse ethereum token sale"), zap.Error(err))
		return err
	}
	if nil == tokenSale {
		return nil
	}

	// Index token sale
	var saleTimeSeries *indexer.GenericSalesTimeSeries
	if err := workflow.ExecuteActivity(
		ctx,
		w.ParseTokenSaleToGenericSalesTimeSeries,
		*tokenSale).
		Get(ctx, &saleTimeSeries); err != nil {
		logger.Error(errors.New("fail to parse token sale to generic sales time series"), zap.Error(err))
		return err
	}

	data := []indexer.GenericSalesTimeSeries{*saleTimeSeries}
	if err := workflow.ExecuteActivity(
		ctx,
		w.WriteSaleTimeSeriesData,
		data).
		Get(ctx, nil); nil != err {
		logger.Error(errors.New("fail to write sale time series data"), zap.Error(err))
		return err
	}

	return nil
}

// ParseEthereumTokenSale is a workflow to parse the sale of an Ethereum token
func (w *NFTIndexerWorker) ParseEthereumTokenSale(ctx workflow.Context, txID string) (*TokenSale, error) {
	ctx = ContextRegularActivity(ctx, TaskListName)
	logger := log.CadenceWorkflowLogger(ctx)

	// Get tx receipt
	var txReceipt *types.Receipt
	if err := workflow.ExecuteActivity(
		ctx,
		w.GetEthereumTxReceipt,
		txID).
		Get(ctx, &txReceipt); nil != err {
		logger.Error(errors.New("fail to get ethereum tx receipt"), zap.Error(err), zap.String("txID", txID))
		return nil, err
	}
	if nil == txReceipt {
		logger.Warn("tx receipt is not found", zap.String("txID", txID))
		return nil, nil
	}
	if txReceipt.Status == uint64(0) {
		logger.Warn("tx status is failed", zap.String("txID", txID))
		return nil, nil
	}

	// Query tx
	var tx *types.Transaction
	if err := workflow.ExecuteActivity(
		ctx,
		w.GetEthereumTx,
		txID).
		Get(ctx, &tx); nil != err {
		logger.Error(errors.New("fail to get ethereum tx"), zap.Error(err), zap.String("txID", txID))
		return nil, err
	}

	// Tx to
	txTo := tx.To()
	if nil == txTo {
		logger.Warn("tx to is not found", zap.String("txID", txID))
		return nil, nil
	}
	txToHex := txTo.Hex()

	// Classify transfers from the event logs
	erc20Transfers, tokenTransfers := classifyTxLogs(txReceipt.Logs)

	// currency
	var currency string
	if tx.Value().BitLen() > 0 {
		currency = "ETH"
	}

	// The tx is a token sale if it's either paid by ETH or an ERC20 token
	if currency == "ETH" && len(erc20Transfers) > 0 {
		return nil, nil
	}

	// The tx should only includes one ERC20 contract interaction
	if len(erc20Transfers) > 1 {
		return nil, nil
	}

	// Struct for internal token transfer data
	type tokenTransferData struct {
		From     string
		To       string
		TokenID  string
		Contract string
	}

	tokenTxMap := make(map[string]tokenTransferData) // [tokenID] => []tokenTransferData

	// Iterate over internal token transfers and turn them into appropriate data structures
	for _, l := range tokenTransfers {
		isERC721 := indexer.ERC721Transfer(l) // if not, it will be ERC-1155
		topic1 := l.Topics[1]
		topic2 := l.Topics[2]
		topic3 := l.Topics[3]
		data := l.Data

		var senderAddrHex string
		var recipientAddrHex string
		var tokenIDHex string
		if isERC721 { // ERC-721
			senderAddrHex = common.HexToAddress(topic1.Hex()).Hex()
			recipientAddrHex = common.HexToAddress(topic2.Hex()).Hex()
			tokenIDHex = strings.TrimPrefix(topic3.Hex(), "0x")
		} else { // ERC-1155 TransferSingle
			senderAddrHex = common.HexToAddress(topic2.Hex()).Hex()
			recipientAddrHex = common.HexToAddress(topic3.Hex()).Hex()

			if len(data) != 64 { // 64 bytes for uint256 id and uint256 value
				logger.Warn("invalid ERC-1155 transfer single event data", zap.String("txID", txID))
				return nil, nil
			}

			// Parse event data
			tokenID := new(big.Int).SetBytes(data[:32])
			value := new(big.Int).SetBytes(data[32:])
			if value.Uint64() != 1 {
				logger.Warn("invalid ERC-1155 transfer single event data", zap.String("txID", txID))
				return nil, nil
			}

			tokenIDHex = hex.EncodeToString(tokenID.Bytes())
		}

		tokenID := indexer.HexToDec(tokenIDHex)
		tokenContract := l.Address.Hex()

		// Check if the token is published by Feral File
		// TODO remove after supporting all tokens not only Feral File one
		indexID := indexer.TokenIndexID(utils.EthereumBlockchain, tokenContract, tokenID)
		var token *indexer.Token
		if err := workflow.ExecuteActivity(
			ctx,
			w.GetTokenByIndexID,
			indexID).
			Get(ctx, &token); err != nil {
			return nil, err
		}
		if nil == token || token.Source != "feralfile" {
			logger.Warn("token is not found or not published by Feral File", zap.String("txID", txID))
			return nil, nil
		}

		// If there are multiple transfers for the same token,
		// the sender is the first transfer sender,
		// the recipient is the last transfer recipient
		tokenTxMapKey := fmt.Sprintf("%s-%s", tokenContract, tokenID)
		var ttd tokenTransferData
		if token, ok := tokenTxMap[tokenTxMapKey]; ok {
			ttd = tokenTransferData{
				From:     token.From,
				To:       recipientAddrHex,
				TokenID:  tokenID,
				Contract: tokenContract,
			}
		} else {
			ttd = tokenTransferData{
				From:     senderAddrHex,
				To:       recipientAddrHex,
				TokenID:  tokenID,
				Contract: tokenContract,
			}
		}
		tokenTxMap[tokenTxMapKey] = ttd
	}

	var tokenTxs []tokenTransferData
	for _, t := range tokenTxMap {
		tokenTxs = append(tokenTxs, t)
	}

	// Struct for payment transfer data
	type payTransferData struct {
		From   string
		To     string
		Amount *big.Int
	}

	var payTxs []payTransferData // payment transfers
	if len(erc20Transfers) > 0 { // pay by ERC20 tokens
		contractMap := viper.GetStringMapString("ethereum.erc20") // key is lower case
		if len(contractMap) == 0 {
			err := errors.New("couldn't load the ERC20 contracts")
			logger.Error(err, zap.String("txID", txID))
			return nil, err
		}

		// flatten ERC20 transfers
		var erc20TransferLogs []types.Log
		for _, logs := range erc20Transfers {
			erc20TransferLogs = append(erc20TransferLogs, logs...)
		}

		for _, l := range erc20TransferLogs {
			address := l.Address.Hex()
			if c, ok := contractMap[strings.ToLower(address)]; ok {
				currency = c
			}

			topic1 := l.Topics[1]
			topic2 := l.Topics[2]
			from := common.HexToAddress(topic1.Hex()).Hex()
			to := common.HexToAddress(topic2.Hex()).Hex()
			amount := new(big.Int).SetBytes(l.Data)
			payTxs = append(payTxs, payTransferData{
				From:   from,
				To:     to,
				Amount: amount,
			})
		}
	}

	// List internal txs
	var itxs []etherscan.Transaction
	if err := workflow.ExecuteActivity(
		ctx,
		w.GetEthereumInternalTxs,
		txID).
		Get(ctx, &itxs); nil != err {
		logger.Error(errors.New("fail to get ethereum internal txs"), zap.Error(err))
		return nil, err
	}

	if len(itxs) > 0 {
		// Assume the currency is ETH if not paid by ERC20 tokens and there are internal txs
		if currency == "" {
			currency = "ETH"
		}
		for _, itx := range itxs {
			srcAddrHex := common.HexToAddress(itx.From).Hex()
			dstAddrHex := common.HexToAddress(itx.To).Hex()
			val, ok := big.NewInt(0).SetString(itx.Value, 10)
			if !ok {
				err := errors.New("fail to parse internal tx value")
				logger.Error(err, zap.String("txID", txID))
				return nil, err
			}
			transfer := payTransferData{
				From:   srcAddrHex,
				To:     dstAddrHex,
				Amount: val,
			}

			payTxs = append(payTxs, transfer)
		}
	}

	// Check if sale is on supported marketplaces
	marketplaceContracts := viper.GetStringMapString("marketplace.contracts") // key is lower case
	marketplace, ok := marketplaceContracts[strings.ToLower(txToHex)]
	if !ok {
		for _, itx := range payTxs {
			marketplace, ok = marketplaceContracts[strings.ToLower(itx.To)]
			if !ok {
				marketplace, ok = marketplaceContracts[strings.ToLower(itx.From)]
			}

			if ok {
				break
			}
		}
	}

	if marketplace == "" {
		return nil, nil
	}

	if len(payTxs) == 0 {
		return nil, nil
	}

	// Fee wallets
	platformFeeWallets := viper.GetStringMapString("marketplace.fee_wallets") // key is lower case
	if len(platformFeeWallets) == 0 {
		return nil, nil
	}

	shares := make(map[string]*big.Int)
	platformFee := big.NewInt(0)
	price := big.NewInt(0)
	for _, tx := range payTxs {

		if _, ok := marketplaceContracts[strings.ToLower(tx.To)]; ok {
			// This is mainly internal tx to pay for the sale from another contract
			// like indirect sale from other marketplaces
			continue
		}

		// Accumulate share
		if s, ok := shares[tx.To]; ok {
			shares[tx.To] = big.NewInt(0).Add(s, tx.Amount)
		} else {
			shares[tx.To] = tx.Amount
		}
		price.Add(price, tx.Amount)

		// Deduct share
		if s, ok := shares[tx.From]; ok {
			shares[tx.From] = big.NewInt(0).Sub(s, tx.Amount)
			price.Sub(price, tx.Amount)
		}

		// Accumulate platform fee
		if _, ok := platformFeeWallets[strings.ToLower(tx.To)]; ok {
			platformFee.Add(platformFee, tx.Amount)
		}
	}

	openSeaOldFeeWallet := viper.GetString("opensea.oldfeewallet")
	if platformFee.BitLen() == 0 {
		if _, ok := shares[openSeaOldFeeWallet]; ok {
			// 2.5% fixed fee
			new(big.Float).
				Mul(new(big.Float).SetInt(price), big.NewFloat(0.025)).
				Int(platformFee)
		}
	}

	// Sale timestamp
	var blkHeader *types.Header
	if err := workflow.ExecuteActivity(
		ctx,
		w.GetEthereumBlockHeaderHash,
		txReceipt.BlockHash.Hex()).
		Get(ctx, &blkHeader); nil != err {
		logger.Error(errors.New("fail to get ethereum block header hash"), zap.Error(err), zap.String("txID", txID))
		return nil, err
	}
	if nil == blkHeader {
		err := errors.New("block not found")
		logger.Error(err, zap.String("txID", txID))
		return nil, err
	}

	bundleTokenInfo := []TokenSaleInfo{}
	for _, t := range tokenTxs {
		bundleTokenInfo = append(bundleTokenInfo, TokenSaleInfo{
			BuyerAddress:    t.To,
			SellerAddress:   t.From,
			ContractAddress: t.Contract,
			TokenID:         t.TokenID,
		})
	}

	return &TokenSale{
		Timestamp:       time.Unix(int64(blkHeader.Time), 0),
		Price:           price,
		Marketplace:     marketplace,
		Blockchain:      "ethereum",
		Currency:        currency,
		TxID:            txID,
		PlatformFee:     platformFee,
		NetRevenue:      big.NewInt(0).Sub(price, platformFee),
		BundleTokenInfo: bundleTokenInfo,
		PaymentAmount:   price,
		Shares:          shares,
	}, nil
}

func classifyTxLogs(logs []*types.Log) (map[string][]types.Log, []types.Log) {
	erc20Transfers := make(map[string][]types.Log) // address => []types.Log
	tokenTransfers := []types.Log{}                // address => []types.Log
	for _, l := range logs {
		if nil == l {
			continue
		}
		address := l.Address.Hex()

		switch {
		case indexer.ERC20Transfer(*l):
			erc20Transfers[address] = append(erc20Transfers[address], *l)
		case indexer.ERC1155SingleTransfer(*l), indexer.ERC721Transfer(*l):
			tokenTransfers = append(tokenTransfers, *l)
		default:
			// Ignore
		}
	}
	return erc20Transfers, tokenTransfers
}

// IndexTezosTokenSaleFromTzktTxID is a workflow to get the tezos transaction hash by tzkt txid
func (w *NFTIndexerWorker) IndexTezosTokenSaleFromTzktTxID(
	ctx workflow.Context,
	id uint64) error {
	ctx = ContextRegularActivity(ctx, TaskListName)
	logger := log.CadenceWorkflowLogger(ctx)

	var txHash *string
	if err := workflow.ExecuteActivity(
		ctx,
		w.GetTezosTxHashFromTzktTransactionID,
		id).
		Get(ctx, &txHash); err != nil {
		logger.Error(errors.New("fail to get tezos tx hash from tzkt txid"), zap.Error(err), zap.Uint64("id", id))
		return err
	}
	if nil == txHash {
		err := errors.New("no tx hash found")
		logger.Error(err, zap.Uint64("id", id))
		return err
	}

	workflowID := fmt.Sprintf("IndexTezosObjktTokenSale-%s", *txHash)
	cwctx := ContextNamedRegularChildWorkflow(ctx, workflowID, TaskListName)
	if err := workflow.ExecuteChildWorkflow(
		cwctx,
		w.IndexTezosObjktTokenSale,
		txHash,
		true).Get(ctx, nil); err != nil {
		logger.Error(errors.New("fail to execute tezos objkt token sale"), zap.Error(err), zap.String("txHash", *txHash))
		return err
	}

	return nil
}

// IndexTezosObjktTokenSale is a workflow to index the sale of a Tezos objkt token
func (w *NFTIndexerWorker) IndexTezosObjktTokenSale(ctx workflow.Context, txHash string, skipIndexed bool) error {
	ctx = ContextRegularActivity(ctx, TaskListName)
	logger := log.CadenceWorkflowLogger(ctx)

	if skipIndexed {
		// Check if sale tx is indexed already
		var indexed bool
		if err := workflow.ExecuteActivity(
			ctx,
			w.IndexedSaleTx,
			txHash,
			utils.TezosBlockchain).
			Get(ctx, &indexed); nil != err {
			logger.Error(errors.New("fail to check if sale tx is indexed"), zap.Error(err), zap.String("txHash", txHash))
			return err
		}

		if indexed {
			return nil
		}
	}

	// Fetch & parse token sale by tx hashe
	var tokenSale *TokenSale
	if err := workflow.ExecuteActivity(
		ctx,
		w.ParseTezosObjktTokenSale,
		txHash).
		Get(ctx, &tokenSale); err != nil {
		logger.Error(errors.New("fail to parse tezos objkt token sale"), zap.Error(err), zap.String("txHash", txHash))
		return err
	}

	if nil == tokenSale {
		return nil
	}

	// Check if the token is published by Feral File
	// TODO remove after supporting all tokens not only Feral File one
	for _, info := range tokenSale.BundleTokenInfo {
		indexID := indexer.TokenIndexID(
			utils.TezosBlockchain,
			info.ContractAddress,
			info.TokenID,
		)
		var token *indexer.Token
		if err := workflow.ExecuteActivity(
			ctx,
			w.GetTokenByIndexID,
			indexID).
			Get(ctx, &token); err != nil {
			return err
		}
		if nil == token || token.Source != "feralfile" {
			logger.Warn("token is not found or not published by Feral File", zap.String("txHash", txHash))
			return nil
		}
	}

	// Index token sale
	var saleTimeSeries *indexer.GenericSalesTimeSeries
	if err := workflow.ExecuteActivity(
		ctx,
		w.ParseTokenSaleToGenericSalesTimeSeries,
		*tokenSale).
		Get(ctx, &saleTimeSeries); err != nil {
		logger.Error(errors.New("fail to parse token sale to generic sales time series"), zap.Error(err), zap.String("txHash", txHash))
		return err
	}

	data := []indexer.GenericSalesTimeSeries{*saleTimeSeries}
	if err := workflow.ExecuteActivity(
		ctx,
		w.WriteSaleTimeSeriesData,
		data).
		Get(ctx, nil); nil != err {
		logger.Error(errors.New("fail to write sale time series data"), zap.Error(err), zap.String("txHash", txHash))
		return err
	}

	return nil
}
