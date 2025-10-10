package indexer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/managedblockchainquery"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"

	"github.com/feral-file/ff-indexer/externals/opensea"
)

type TransactionDetails struct {
	From      string
	To        string
	IndexID   string
	Timestamp time.Time
}

// IndexETHTokenByOwner indexes all tokens owned by a specific ethereum address
func (e *IndexEngine) IndexETHTokenByOwner(ctx context.Context, owner string, next string) ([]AssetUpdates, string, error) {
	if _, excluded := EthereumIndexExcludedOwners[owner]; excluded {
		return nil, "", nil
	}

	assets, err := e.opensea.RetrieveAssets(ctx, owner, next)
	if err != nil {
		return nil, "", err
	}

	tokenUpdates := make([]AssetUpdates, 0, len(assets.NFTs))

	for _, a := range assets.NFTs {
		balance := int64(1) // set default balance to 1 to reduce extra call to opensea

		detailedAsset, err := e.opensea.RetrieveAsset(ctx, a.Contract, a.Identifier)
		if err != nil {
			log.WarnWithContext(ctx, "fail to fetch detailed index token data", zap.Error(err))
			continue
		}

		update, err := e.indexETHToken(ctx, detailedAsset, owner, balance)
		if err != nil {
			log.WarnWithContext(ctx, "fail to index token data", zap.Error(err))
		}

		if update != nil {
			tokenUpdates = append(tokenUpdates, *update)
		}
	}

	return tokenUpdates, assets.Next, nil
}

// getEthereumTokenBalanceOfOwner returns current balance of a token that the owner owns
func (e *IndexEngine) getEthereumTokenBalanceOfOwner(_ context.Context, contract, tokenID, owner string) (int64, error) {
	if _, excluded := EthereumIndexExcludedOwners[owner]; excluded {
		return 0, nil
	}

	id, ok := big.NewInt(0).SetString(tokenID, 10)
	if !ok {
		return 0, fmt.Errorf("fail to convert token id to hex")
	}

	network := managedblockchainquery.QueryNetworkEthereumMainnet

	result, err := e.blockchainQueryClient.GetTokenBalance(&managedblockchainquery.GetTokenBalanceInput{
		OwnerIdentifier: &managedblockchainquery.OwnerIdentifier{
			Address: aws.String(owner),
		},
		TokenIdentifier: &managedblockchainquery.TokenIdentifier{
			Network:         &network,
			ContractAddress: aws.String(contract),
			TokenId:         aws.String(fmt.Sprintf("0x%064x", id)),
		},
	})
	if err != nil {
		return 0, err
	}

	balance, err := strconv.Atoi(*result.Balance)
	if err != nil {
		return 0, err
	}
	return int64(balance), nil
}

// IndexETHToken indexes an Ethereum token with a specific contract and ID
func (e *IndexEngine) IndexETHToken(ctx context.Context, contract, tokenID string) (*AssetUpdates, error) {
	a, err := e.opensea.RetrieveAsset(ctx, contract, tokenID)
	if err != nil {
		return nil, err
	}

	return e.indexETHToken(ctx, a, "", 0)
}

// indexETHToken prepares indexing data for a specific asset read from opensea
// The reason to use owner as a parameter is that opensea sometimes returns zero address for it owners. Why?
func (e *IndexEngine) indexETHToken(ctx context.Context, a *opensea.DetailedAssetV2, owner string, balance int64) (*AssetUpdates, error) {
	dataSource := SourceOpensea

	// Skip if the contract is ENS
	contractAddress := EthereumChecksumAddress(a.Contract)
	switch contractAddress {
	case ENSContractAddress1, ENSContractAddress2:
		return nil, nil
	}

	// Get token source
	source := getTokenSourceByMetadataURL(a.MetadataURL)
	if source == "" {
		source = getTokenSourceByPreviewURL(a.AnimationURL)
	}
	if source == "" {
		source = getTokenSourceByContract(contractAddress)
	}

	// Get metadata detail
	metadataDetail := NewAssetMetadataDetail(contractAddress)
	metadataDetail.FromOpenseaAsset(a, source)

	// Lookup artist name from metadata
	if metadataDetail.ArtistName == "" {
		metadata, err := e.fetchTokenMetadata(a.MetadataURL)
		if err != nil {
			return nil, err
		}
		metadataDetail.ArtistName = lookupArtistName(metadata)

		if len(metadataDetail.Artists) != 1 {
			return nil, fmt.Errorf("unexpected number of artists: %d", len(metadataDetail.Artists))
		}
		metadataDetail.Artists[0].Name = metadataDetail.ArtistName
	}

	tokenDetail := TokenDetail{
		MintedAt: a.CreatedAt.Time, // set minted_at to the contract creation time,
		Edition:  e.GetEditionNumberByName(a.Name),
	}

	contractType := strings.ToLower(a.TokenStandard)
	tokenDetail.Fungible = contractType != "erc721"

	// Check source and get data from source's API
	if e.environment != DevelopmentEnvironment {
		if source == sourceFxHash {
			fxObjktID := fmt.Sprintf("%s-%s", a.Contract, a.Identifier)
			e.indexTokenFromFXHASH(ctx, fxObjktID, metadataDetail, &tokenDetail)
		}
	}

	pm := ProjectMetadata{
		AssetID:   metadataDetail.AssetID,
		Source:    metadataDetail.Source,
		SourceURL: metadataDetail.SourceURL,
		AssetURL:  metadataDetail.AssetURL,

		Title:       metadataDetail.Name,
		Description: metadataDetail.Description,
		MIMEType:    metadataDetail.MIMEType,
		Medium:      metadataDetail.Medium,

		ArtistID:   metadataDetail.ArtistID,
		ArtistName: metadataDetail.ArtistName,
		ArtistURL:  metadataDetail.ArtistURL,
		Artists:    metadataDetail.Artists,
		MaxEdition: metadataDetail.MaxEdition,

		PreviewURL: metadataDetail.PreviewURI,
		// use the thumbnail in metadata for ThumbnailURL
		ThumbnailURL: metadataDetail.ThumbnailURI,
		// use the high quality image for GalleryThumbnailURL
		GalleryThumbnailURL: metadataDetail.DisplayURI,

		ArtworkMetadata: metadataDetail.ArtworkMetadata,

		LastUpdatedAt: time.Now(),
	}

	token := Token{
		BaseTokenInfo: BaseTokenInfo{
			ID:              a.Identifier,
			Blockchain:      utils.EthereumBlockchain,
			Fungible:        tokenDetail.Fungible,
			ContractType:    contractType,
			ContractAddress: contractAddress,
		},
		IndexID:           TokenIndexID(utils.EthereumBlockchain, contractAddress, a.Identifier),
		Edition:           tokenDetail.Edition,
		Balance:           balance,
		Owner:             owner,
		MintedAt:          tokenDetail.MintedAt,
		LastRefreshedTime: time.Now(),
		LastActivityTime:  a.UpdatedAt.Time,
	}

	if owner != "" {
		token.Owners = map[string]int64{owner: balance}
	} else if a.Owners != nil {
		owners := make(map[string]int64)
		for _, o := range a.Owners {
			owners[o.Address] = o.Quantity
		}
		token.Owners = owners

		if len(a.Owners) == 1 {
			token.Owner = a.Owners[0].Address
		}
	}

	tokenUpdate := &AssetUpdates{
		ID:              fmt.Sprintf("%s-%s", a.Contract, a.Identifier),
		Source:          dataSource,
		ProjectMetadata: pm,
		Tokens:          []Token{token},
	}

	log.Debug("asset updating data prepared",
		zap.String("blockchain", utils.EthereumBlockchain),
		zap.String("id", TokenIndexID(utils.EthereumBlockchain, contractAddress, a.Identifier)),
		zap.Any("tokenUpdate", tokenUpdate))

	return tokenUpdate, nil
}

// IndexETHTokenOwners indexes owners of a given token
func (e *IndexEngine) IndexETHTokenOwners(contract, tokenID string) ([]OwnerBalance, error) {
	log.Debug("index eth token owners",
		zap.String("blockchain", utils.EthereumBlockchain),
		zap.String("contract", contract), zap.String("tokenID", tokenID))

	network := managedblockchainquery.QueryNetworkEthereumMainnet

	if viper.GetString("network.ethereum") == "sepolia" {
		network = managedblockchainquery.QueryNetworkEthereumSepoliaTestnet
	}

	var nextToken *string
	ownerBalances := []OwnerBalance{}
	for {
		id, ok := big.NewInt(0).SetString(tokenID, 10)
		if !ok {
			return nil, fmt.Errorf("fail to convert to hex")
		}

		result, err := e.blockchainQueryClient.ListTokenBalances(&managedblockchainquery.ListTokenBalancesInput{
			MaxResults: aws.Int64(250),
			NextToken:  nextToken,

			TokenFilter: &managedblockchainquery.TokenFilter{
				Network:         &network,
				ContractAddress: aws.String(contract),
				TokenId:         aws.String(fmt.Sprintf("0x%064x", id)),
			},
		})
		if err != nil {
			return nil, err
		}

		nextToken = result.NextToken
		for _, o := range result.TokenBalances {

			balance, err := strconv.Atoi(*o.Balance)
			if err != nil {
				return nil, err
			}

			ownerBalances = append(ownerBalances, OwnerBalance{
				Address:  EthereumChecksumAddress(*o.OwnerIdentifier.Address),
				Balance:  int64(balance),
				LastTime: *o.LastUpdatedTime.Time,
			})
		}

		if nextToken == nil {
			break
		}
	}

	log.Debug("indexed eth token owners",
		zap.String("blockchain", utils.EthereumBlockchain),
		zap.String("contract", contract), zap.String("tokenID", tokenID))
	return ownerBalances, nil
}

// GetEthereumTxTimestamp returns the timestamp of an transaction if it exists
func (e *IndexEngine) GetEthereumTxTimestamp(ctx context.Context, txHashString string) (time.Time, error) {
	txHash := common.HexToHash(txHashString)
	receipt, err := e.ethereum.TransactionReceipt(ctx, txHash)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return time.Time{}, ErrTXNotFound
		}
		return time.Time{}, err
	}

	switch receipt.Status {
	case 0:
		return time.Time{}, fmt.Errorf("the transaction is not success")
	case 1:
		return GetETHBlockTime(ctx, e.cacheStore, e.ethereum, receipt.BlockHash)
	}

	return time.Time{}, fmt.Errorf("unexpected tx status for ethereum")
}

// GetETHTransactionDetailsByPendingTx gets transaction details by a specific pendingTx
func (e *IndexEngine) GetETHTransactionDetailsByPendingTx(ctx context.Context, client *ethclient.Client, txHash common.Hash, tokenID string) ([]TransactionDetails, error) {
	receipt, err := client.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, err
	}

	if len(receipt.Logs) == 0 || receipt.Status == 0 {
		return nil, fmt.Errorf("the transaction is not success")
	}

	timestamp, err := GetETHBlockTime(ctx, e.cacheStore, e.ethereum, receipt.BlockHash)
	if err != nil {
		return nil, fmt.Errorf("cannot get transaction timestamp")
	}

	transactionDetails := []TransactionDetails{}
	for _, log := range receipt.Logs {
		if len(log.Topics) != 4 || log.Topics[3].Big().String() != tokenID ||
			(log.Topics[0].String() != TransferEventSignature && log.Topics[0].String() != TransferSingleEventSignature) {
			continue
		}

		transactionDetail := TransactionDetails{
			From:      common.HexToAddress(log.Topics[1].String()).String(),
			To:        common.HexToAddress(log.Topics[2].String()).String(),
			IndexID:   TokenIndexID(utils.EthereumBlockchain, log.Address.String(), tokenID),
			Timestamp: timestamp,
		}

		transactionDetails = append(transactionDetails, transactionDetail)
	}

	return transactionDetails, nil
}
