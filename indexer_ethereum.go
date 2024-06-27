package indexer

import (
	"context"
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
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"
	"github.com/bitmark-inc/nft-indexer/externals/opensea"
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

	assets, err := e.opensea.RetrieveAssets(owner, next)
	if err != nil {
		return nil, "", err
	}

	tokenUpdates := make([]AssetUpdates, 0, len(assets.NFTs))

	for _, a := range assets.NFTs {
		balance := int64(1) // set default balance to 1 to reduce extra call to opensea
		log.Debug("get token balance",
			zap.String("contract", a.Contract),
			zap.String("tokenID", a.Identifier),
			zap.String("owner", owner),
			zap.Int64("balance", balance))

		detailedAsset, err := e.opensea.RetrieveAsset(a.Contract, a.Identifier)
		if err != nil {
			log.Error("fail to fetch detailed index token data", zap.Error(err))
			continue
		}

		update, err := e.indexETHToken(ctx, detailedAsset, owner, balance)
		if err != nil {
			log.Error("fail to index token data", zap.Error(err))
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
	a, err := e.opensea.RetrieveAsset(contract, tokenID)
	if err != nil {
		return nil, err
	}

	return e.indexETHToken(ctx, a, "", 0)
}

// indexETHToken prepares indexing data for a specific asset read from opensea
// The reason to use owner as a parameter is that opensea sometimes returns zero address for it owners. Why?
func (e *IndexEngine) indexETHToken(ctx context.Context, a *opensea.DetailedAssetV2, owner string, balance int64) (*AssetUpdates, error) {
	dataSource := SourceOpensea

	contractAddress := EthereumChecksumAddress(a.Contract)
	switch contractAddress {
	case ENSContractAddress1, ENSContractAddress2:
		return nil, nil
	}

	source := getTokenSourceByMetadataURL(a.MetadataURL)

	if source == "" {
		source = getTokenSourceByPreviewURL(a.AnimationURL)
	}

	if source == "" {
		source = getTokenSourceByContract(contractAddress)
	}

	metadataDetail := NewAssetMetadataDetail(contractAddress)
	metadataDetail.FromOpenseaAsset(a, source)
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

	// FIXME: does not support testnet indexing for now
	network := managedblockchainquery.QueryNetworkEthereumMainnet

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
		if err == ethereum.NotFound {
			return time.Time{}, ErrTXNotFound
		}
		return time.Time{}, err
	}

	if receipt.Status == 0 {
		return time.Time{}, fmt.Errorf("the transaction is not success")
	} else if receipt.Status == 1 {
		t, err := GetETHBlockTime(ctx, e.cacheStore, e.ethereum, receipt.BlockHash)
		if err != nil {
			return time.Time{}, err
		}
		return t, nil
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

func (e *IndexEngine) GetOpenseaAccountByAddress(address string) (*opensea.Account, error) {
	account, err := e.opensea.RetrieveAccount(address)
	if err != nil {
		return nil, err
	}

	return account, nil
}

// IndexETHCollectionByCreator indexes all collections owned by a specific eth address
func (e *IndexEngine) IndexETHCollectionByCreator(_ context.Context, account opensea.Account, next string) ([]Collection, string, error) {
	collectionsResp, err := e.opensea.RetrieveColections(account.Username, next)
	if err != nil {
		return nil, "", err
	}

	log.Debug("retrieve eth collections for creator", zap.Any("collections", collectionsResp), zap.String("username", account.Username))

	collectionUpdates := make([]Collection, 0, len(collectionsResp.Collections))

	for _, c := range collectionsResp.Collections {
		collection, err := e.opensea.RetrieveColection(account.Username, c.ID)
		if err != nil {
			log.Error("failed to RetrieveColection", zap.Error(err), zap.String("collectionID", c.ID))
			return nil, "", err
		}

		contracts := []string{}
		for _, c := range c.Contracts {
			contracts = append(contracts, c.Address)
		}

		createdAt, err := time.Parse("2006-01-02", collection.CreatedDate)
		if err != nil {
			log.Warn("error parsing date collection created date", zap.Error(err))
			createdAt = time.Time{}
		}

		update := Collection{
			ID:               fmt.Sprint("opensea-", c.ID),
			ExternalID:       c.ID,
			Blockchain:       utils.EthereumBlockchain,
			Creator:          EthereumChecksumAddress(account.Address),
			Name:             c.Name,
			Description:      c.Description,
			ImageURL:         c.ImageURL,
			Contracts:        contracts,
			Source:           "opensea",
			Published:        !c.IsDisabled,
			SourceURL:        c.OpenseaURL,
			ProjectURL:       c.ProjectURL,
			Items:            collection.TotalSupply,
			LastActivityTime: createdAt,
			CreatedAt:        createdAt,
		}

		collectionUpdates = append(collectionUpdates, update)
	}

	return collectionUpdates, collectionsResp.Next, nil
}

// IndexETHTokenByCollection indexes all tokens from a given collection slug
func (e *IndexEngine) IndexETHTokenByCollection(ctx context.Context, slug string, next string) ([]AssetUpdates, string, error) {
	assets, err := e.opensea.RetrieveColectionAssets(slug, next)
	if err != nil {
		return nil, "", err
	}

	tokenUpdates := make([]AssetUpdates, 0, len(assets.NFTs))

	for _, a := range assets.NFTs {
		balance := int64(1) // set default balance to 1 to reduce extra call to opensea
		log.Debug("get token balance",
			zap.String("contract", a.Contract),
			zap.String("tokenID", a.Identifier),
			zap.String("collection", slug),
			zap.Int64("balance", balance))

		detailedAsset, err := e.opensea.RetrieveAsset(a.Contract, a.Identifier)
		if err != nil {
			log.Error("fail to fetch detailed index token data", zap.Error(err))
			continue
		}

		update, err := e.indexETHToken(ctx, detailedAsset, "", balance)
		if err != nil {
			log.Error("fail to index token data", zap.Error(err))
		}

		if update != nil {
			tokenUpdates = append(tokenUpdates, *update)
		}
	}

	return tokenUpdates, assets.Next, nil
}
