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
	"github.com/spf13/viper"
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
func (e *IndexEngine) IndexETHTokenByOwner(owner string, next string) ([]AssetUpdates, string, error) {
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

		update, err := e.indexETHToken(detailedAsset, owner, balance)
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
func (e *IndexEngine) IndexETHToken(_ context.Context, contract, tokenID string) (*AssetUpdates, error) {
	a, err := e.opensea.RetrieveAsset(contract, tokenID)
	if err != nil {
		return nil, err
	}

	return e.indexETHToken(a, "", 0)
}

// indexETHToken prepares indexing data for a specific asset read from opensea
// The reason to use owner as a parameter is that opensea sometimes returns zero address for it owners. Why?
func (e *IndexEngine) indexETHToken(a *opensea.DetailedAssetV2, owner string, balance int64) (*AssetUpdates, error) {
	dataSource := SourceOpensea

	var sourceURL string
	var artistURL string
	artistID := EthereumChecksumAddress(a.Creator)
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

	switch source {
	case sourceArtBlocks:
		sourceURL = "https://www.artblocks.io/"
		artistURL = fmt.Sprintf("%s/%s", sourceURL, a.Creator)
	case sourceCrayonCodes:
		sourceURL = "https://openprocessing.org/crayon/"
		artistURL = fmt.Sprintf("https://opensea.io/%s", a.Creator)
	case sourceFxHash:
		sourceURL = "https://www.fxhash.xyz/"
		artistURL = fmt.Sprintf("https://www.fxhash.xyz/u/%s", a.Creator)
	default:
		if viper.GetString("network") == "testnet" {
			sourceURL = "https://testnets.opensea.io"
		} else {
			sourceURL = "https://opensea.io"
		}
		artistURL = fmt.Sprintf("https://opensea.io/%s", a.Creator)
	}

	log.Debug("source debug", zap.String("source", source), zap.String("contract", a.Contract), zap.String("id", a.Identifier))

	// Opensea GET assets API just provide a creator, not multiple creator
	artists := []Artist{
		{
			ID:   artistID,
			Name: artistID,
			URL:  artistURL,
		},
	}

	assetURL := a.OpenseaURL
	animationURL := a.AnimationURL

	imageURL, err := OptimizedOpenseaImageURL(a.ImageURL)
	if err != nil {
		log.Warn("invalid opensea image url", zap.String("imageURL", a.ImageURL))
	}

	// fallback to project origin image url
	if imageURL == "" {
		imageURL = a.ImageURL
	}

	if source == sourceFxHash {
		assetURL = fmt.Sprintf("https://www.fxhash.xyz/gentk/%s-%s", a.Contract, a.Identifier)
		imageURL = OptimizeFxHashIPFSURL(imageURL)
		animationURL = OptimizeFxHashIPFSURL(animationURL)
	}

	metadata := ProjectMetadata{
		ArtistID:            artistID,
		ArtistName:          artistID,
		ArtistURL:           artistURL,
		AssetID:             contractAddress,
		Title:               a.Name,
		Description:         a.Description,
		MIMEType:            GetMIMETypeByURL(imageURL),
		Medium:              MediumUnknown,
		Source:              source,
		SourceURL:           sourceURL,
		PreviewURL:          imageURL,
		ThumbnailURL:        imageURL,
		GalleryThumbnailURL: imageURL,
		AssetURL:            assetURL,
		LastUpdatedAt:       time.Now(),
		Artists:             artists,
	}

	if animationURL != "" {
		metadata.PreviewURL = animationURL
		metadata.MIMEType = GetMIMETypeByURL(animationURL)

		if source == sourceArtBlocks {
			metadata.Medium = MediumSoftware
		} else {
			medium := mediumByPreviewFileExtension(metadata.PreviewURL)
			log.Debug("fallback medium check", zap.String("previewURL", metadata.PreviewURL), zap.Any("medium", medium))
			metadata.Medium = medium
		}
	} else if imageURL != "" {
		metadata.Medium = MediumImage
	}

	contractType := strings.ToLower(a.TokenStandard)
	fungible := contractType != "erc721"

	token := Token{
		BaseTokenInfo: BaseTokenInfo{
			ID:              a.Identifier,
			Blockchain:      utils.EthereumBlockchain,
			Fungible:        fungible,
			ContractType:    contractType,
			ContractAddress: contractAddress,
		},
		IndexID:           TokenIndexID(utils.EthereumBlockchain, contractAddress, a.Identifier),
		Edition:           e.GetEditionNumberByName(a.Name),
		Balance:           balance,
		Owner:             owner,
		MintedAt:          a.CreatedAt.Time, // set minted_at to the contract creation time
		LastRefreshedTime: time.Now(),
		LastActivityTime:  a.UpdatedAt.Time,
	}

	if owner != "" {
		token.Owners = map[string]int64{owner: balance}
	}

	tokenUpdate := &AssetUpdates{
		ID:              fmt.Sprintf("%s-%s", a.Contract, a.Identifier),
		Source:          dataSource,
		ProjectMetadata: metadata,
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
