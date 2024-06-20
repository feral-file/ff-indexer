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
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// PendingTxFollowUpWorkflow is a workflow to follow up and update pending tokens
func (w *NFTIndexerWorker) PendingTxFollowUpWorkflow(ctx workflow.Context, delay time.Duration) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.AccountTokenTaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
	}

	log := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, ao)
	log.Debug("start PendingTxFollowUpWorkflow")

	var pendingAccountTokens []indexer.AccountToken
	if err := workflow.ExecuteActivity(ctx, w.GetPendingAccountTokens).Get(ctx, &pendingAccountTokens); err != nil {
		log.Error("fail to get pending account tokens", zap.Error(err))
		return err
	}

	if len(pendingAccountTokens) == 0 {
		_ = workflow.Sleep(ctx, 1*time.Minute)
		return workflow.NewContinueAsNewError(ctx, w.PendingTxFollowUpWorkflow, delay)
	}

	for _, a := range pendingAccountTokens {
		log.Debug("start checking txs for pending token", zap.String("indexID", a.IndexID))
		pendindTxCount := len(a.PendingTxs)

		var hasNewTx bool
		remainingPendingTxs := make([]string, 0, pendindTxCount)
		remainingPendingTxTimes := make([]time.Time, 0, pendindTxCount)

		// The loop checks all new confirmed txs.
	PendingTxChecking:
		for i := 0; i < pendindTxCount; i++ {
			pendingTime := a.LastPendingTime[i]
			pendingTx := a.PendingTxs[i]

			var txComfirmedTime time.Time
			if err := workflow.ExecuteActivity(ctx, w.GetTxTimestamp, a.Blockchain, pendingTx).Get(ctx, &txComfirmedTime); err != nil {
				log.Error("fail to get tx status for the account token", zap.Error(err),
					zap.String("txID", pendingTx), zap.String("indexID", a.IndexID), zap.String("ownerAccount", a.OwnerAccount))
				switch err.Error() {
				case indexer.ErrTXNotFound.Error():
					// drop not found tx which exceed an hour
					if time.Since(pendingTime) > time.Hour {
						// TODO: should be check if the tx is remaining in the mempool of the blockchain network
						continue PendingTxChecking
					}
				case indexer.ErrUnsupportedBlockchain.Error():
					// drop unsupported pending tx
					continue PendingTxChecking
				default:
					// leave non-handled error in the remaining pending txs for next processing
					remainingPendingTxs = append(remainingPendingTxs, pendingTx)
					remainingPendingTxTimes = append(remainingPendingTxTimes, pendingTime)
				}
			} else {
				log.Debug("found a confirme pending tx", zap.String("pendingTx", pendingTx))
				if txComfirmedTime.Sub(a.LastActivityTime) > 0 {
					hasNewTx = true
				}
			}
		}
		log.Debug("finish checking txs for pending token", zap.String("indexID", a.IndexID), zap.Any("hasNewTx", hasNewTx))

		// update the balance of "this" token immediately
		var balance int64
		if err := workflow.ExecuteActivity(ContextFastActivity(ctx, AccountTokenTaskListName), w.GetTokenBalanceOfOwner, a.ContractAddress, a.ID, a.OwnerAccount).
			Get(ctx, &balance); err != nil {
			log.Error("fail to get the latest balance for the account token", zap.Error(err),
				zap.String("indexID", a.IndexID), zap.String("ownerAccount", a.OwnerAccount))
			continue
		}

		accountTokens := []indexer.AccountToken{{
			IndexID:           a.IndexID,
			OwnerAccount:      a.OwnerAccount,
			Balance:           balance,
			LastActivityTime:  a.LastActivityTime,
			LastRefreshedTime: time.Now(),
		}}

		if err := workflow.ExecuteActivity(ContextFastActivity(ctx, AccountTokenTaskListName), w.IndexAccountTokens, a.OwnerAccount, accountTokens).Get(ctx, nil); err != nil {
			log.Error("fail to update the latest balance for the account token", zap.Error(err),
				zap.String("indexID", a.IndexID), zap.String("ownerAccount", a.OwnerAccount))
			continue
		}

		// trigger async token ownership / provenance refreshing for the updated token
		if hasNewTx {
			var childFuture workflow.ChildWorkflowFuture
			if a.Fungible {
				childFuture = workflow.ExecuteChildWorkflow(
					ContextDetachedChildWorkflow(ctx, WorkflowIDIndexTokenOwnershipByIndexID(
						"pending-tx-follower", a.IndexID), ProvenanceTaskListName),
					w.RefreshTokenOwnershipWorkflow, []string{a.IndexID}, delay)
			} else {
				childFuture = workflow.ExecuteChildWorkflow(
					ContextDetachedChildWorkflow(ctx, WorkflowIDIndexTokenProvenanceByIndexID(
						"pending-tx-follower", a.IndexID), ProvenanceTaskListName),
					w.RefreshTokenProvenanceWorkflow, []string{a.IndexID}, delay)
			}

			if err := childFuture.GetChildWorkflowExecution().Get(ctx, nil); err != nil {
				log.Error("fail to spawn ownership / provenance updating workflow for indexID", zap.Error(err),
					zap.Bool("fungible", a.Fungible), zap.String("indexID", a.IndexID))
			}
		}

		log.Debug("remaining pending txs", zap.Any("remainingPendingTx", remainingPendingTxs), zap.Any("remainingPendingTxTime", remainingPendingTxs))
		if err := workflow.ExecuteActivity(ctx, w.UpdatePendingTxsToAccountToken,
			a.OwnerAccount, a.IndexID, remainingPendingTxs, remainingPendingTxTimes).Get(ctx, nil); err != nil {
			// log the error only so the loop will continuously check the next pending token
			log.Error("fail to update remaining pending txs into account token", zap.Error(err),
				zap.String("indexID", a.IndexID), zap.String("ownerAccount", a.OwnerAccount), zap.Time("astRefreshedTime", a.LastRefreshedTime))
		}
	}

	return workflow.NewContinueAsNewError(ctx, w.PendingTxFollowUpWorkflow, delay)
}

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

var (
	errTxNotFound                       = errors.New("Tx not found")
	errTxFailed                         = errors.New("Tx failed")
	errUnknownTx                        = errors.New("Unknown tx")
	errMultipleERC20Contracts           = errors.New("Multiple ERC20 contracts in the tx")
	errMultipleTokenContracts           = errors.New("Multiple token contracts in the tx")
	errERC1155TransferAmountNotOne      = errors.New("ERC1155 transfer amount isn't 1")
	errERC1155TransferInvalidDataLength = errors.New("ERC1155 transfer invalid data length")
	errERC1155TransferInvalidTokenID    = errors.New("ERC1155 transfer invalid token ID")
	errUnsupportedTokenSale             = errors.New("Unsupported token sale")
	errParseInternalTxVal               = errors.New("Fail to parse internal tx value")
	errNoMarketContractInteractionTx    = errors.New("No market contract interaction transaction found")
	errNoPaymentTx                      = errors.New("No payment transaction found")
)

// IndexEthereumTokenSale is a workflow to index the sale of an Ethereum token
func (w *NFTIndexerWorker) IndexEthereumTokenSale(
	ctx workflow.Context,
	txID string,
	skipIndexed bool) error {
	ctx = ContextRegularActivity(ctx, TaskListName)
	ctx = ContextRegularChildWorkflow(ctx, TaskListName)

	// TODO remove in the future
	if !skipIndexed {
		return errors.New("skipIndexed must be true until we have a unique index handled properly for sale time series data")
	}

	if skipIndexed {
		// Check if sale tx is indexed already
		var indexed bool
		if err := workflow.ExecuteActivity(
			ctx,
			w.IndexedSaleTx,
			txID).
			Get(ctx, &indexed); nil != err {
			return err
		}

		if indexed {
			log.Info("skip tx already indexed", zap.String("txID", txID))
			return nil
		}
	}

	log.Info("start indexing token sale", zap.String("txID", txID))

	// Parse tx
	var tokenSale *TokenSale
	if err := workflow.ExecuteChildWorkflow(
		ctx,
		w.ParseEthereumSingleTokenSale,
		txID).
		Get(ctx, &tokenSale); nil != err {
		switch err.(type) {
		case *workflow.GenericError:
			switch err.Error() {
			case errNoMarketContractInteractionTx.Error():
				// This mainly the normal token transfer
			case
				errTxNotFound.Error(),
				errNoPaymentTx.Error(),
				errParseInternalTxVal.Error(),
				errUnknownTx.Error(),
				errTxFailed.Error(),
				errMultipleERC20Contracts.Error(),
				errMultipleTokenContracts.Error(),
				errERC1155TransferAmountNotOne.Error(),
				errERC1155TransferInvalidDataLength.Error(),
				errERC1155TransferInvalidTokenID.Error(),
				errUnsupportedTokenSale.Error(),
				errParseInternalTxVal.Error():
				log.Warn("fail to parse token sale", zap.String("error", err.Error()))
				return nil
			default:
				return err
			}
		default:
			return err
		}
	}
	if nil == tokenSale {
		log.Info("no token sale found in the tx", zap.String("txID", txID))
		return nil
	}

	// Index token sale
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
			return err
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
		"transactionID":   tokenSale.TxID,
		"bundleTokenInfo": bundleTokenInfo,
	}

	data := []indexer.GenericSalesTimeSeries{
		{
			Timestamp: tokenSale.Timestamp.Format(time.RFC3339Nano),
			Metadata:  metadata,
			Shares:    shares,
			Values:    values,
		},
	}
	if err := workflow.ExecuteActivity(
		ctx,
		w.WriteSaleTimeSeriesData,
		data).
		Get(ctx, nil); nil != err {
		log.Error("fail to write time series data", zap.Error(err))
		return err
	}

	log.Info("token sale indexed", zap.String("txID", txID))

	return nil
}

// ParseEthereumSingleTokenSale is a workflow to parse the sale of an Ethereum token
func (w *NFTIndexerWorker) ParseEthereumSingleTokenSale(ctx workflow.Context, txID string) (*TokenSale, error) {
	ctx = ContextRegularActivity(ctx, TaskListName)

	log.Info("start parsing token sale", zap.String("txID", txID))

	// Get tx receipt
	var txReceipt *types.Receipt
	if err := workflow.ExecuteActivity(
		ctx,
		w.GetEthereumTxReceipt,
		txID).
		Get(ctx, &txReceipt); nil != err {
		return nil, err
	}
	if nil == txReceipt {
		return nil, errTxNotFound
	}
	if txReceipt.Status == uint64(0) {
		return nil, errTxFailed
	}

	// Query tx
	var tx *types.Transaction
	if err := workflow.ExecuteActivity(
		ctx,
		w.GetEthereumTx,
		txID).
		Get(ctx, &tx); nil != err {
		return nil, err
	}

	// Tx to
	txTo := tx.To()
	if nil == txTo {
		return nil, errUnknownTx
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
		return nil, errUnknownTx
	}

	// The tx should only includes one ERC20 contract interaction
	if len(erc20Transfers) > 1 {
		return nil, errMultipleERC20Contracts
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
				return nil, errERC1155TransferInvalidDataLength
			}

			// Parse event data
			tokenID := new(big.Int).SetBytes(data[:32])
			value := new(big.Int).SetBytes(data[32:])
			if value.Uint64() != 1 {
				return nil, errERC1155TransferAmountNotOne
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
			continue
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
			return nil, errors.New("Couldn't load the ERC20 contracts")
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
	} else {
		currency = "ETH"

		// List internal txs
		var itxs []etherscan.Transaction
		if err := workflow.ExecuteActivity(
			ctx,
			w.GetEthereumInternalTxs,
			txID).
			Get(ctx, &itxs); nil != err {
			return nil, err
		}

		if len(itxs) > 0 {
			for _, itx := range itxs {
				srcAddrHex := common.HexToAddress(itx.From).Hex()
				dstAddrHex := common.HexToAddress(itx.To).Hex()
				val, ok := big.NewInt(0).SetString(itx.Value, 10)
				if !ok {
					return nil, errParseInternalTxVal
				}
				transfer := payTransferData{
					From:   srcAddrHex,
					To:     dstAddrHex,
					Amount: val,
				}

				payTxs = append(payTxs, transfer)
			}
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
		return nil, errNoMarketContractInteractionTx
	}

	if len(payTxs) == 0 {
		return nil, errNoPaymentTx
	}

	// Fee wallets
	platformFeeWallets := viper.GetStringMapString("marketplace.fee_wallets") // key is lower case
	if len(platformFeeWallets) == 0 {
		return nil, errors.New("Couldn't load the platform fee wallets")
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
		return nil, err
	}
	if nil == blkHeader {
		return nil, errors.New("Block not found")
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

// LookupTezosTokenSale is a workflow to index the sale of a Tezos token
func (w *NFTIndexerWorker) IndexTezosTokenSale(ctx workflow.Context, txID string) error {
	// TODO: implement
	return nil
}
