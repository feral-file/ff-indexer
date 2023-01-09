package indexer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/bitmark-inc/nft-indexer/externals/opensea"
)

type TransactionDetails struct {
	From      string
	To        string
	IndexID   string
	Timestamp time.Time
}

// IndexETHTokenByOwner indexes all tokens owned by a specific ethereum address
func (e *IndexEngine) IndexETHTokenByOwner(ctx context.Context, owner string, offset int) ([]AssetUpdates, error) {
	assets, err := e.opensea.RetrieveAssets(owner, offset)
	if err != nil {
		return nil, err
	}

	tokenUpdates := make([]AssetUpdates, 0, len(assets))

	for _, a := range assets {
		// balance, err := e.opensea.GetTokenBalanceForOwner(a.AssetContract.Address, a.TokenID, owner)
		// if err != nil {
		// 	log.WithError(err).Error("fail to get token balance from owner")
		// 	continue
		// }
		balance := int64(1) // set default balance to 1 to reduce extra call to opensea

		log.WithFields(log.Fields{
			"contract": a.AssetContract.Address,
			"tokenID":  a.TokenID,
			"owner":    owner,
			"balance":  balance,
		}).Trace("get token balance")

		update, err := e.indexETHToken(&a, owner, balance)
		if err != nil {
			log.WithError(err).Error("fail to index token data")
		}

		if update != nil {
			tokenUpdates = append(tokenUpdates, *update)
		}
	}

	return tokenUpdates, nil
}

// IndexETHToken indexes an Ethereum token with a specific contract and ID
func (e *IndexEngine) IndexETHToken(ctx context.Context, owner, contract, tokenID string) (*AssetUpdates, error) {
	a, err := e.opensea.RetrieveAsset(contract, tokenID)
	if err != nil {
		return nil, err
	}

	balance, err := e.opensea.GetTokenBalanceForOwner(contract, tokenID, owner)
	if err != nil {
		return nil, err
	}

	return e.indexETHToken(a, owner, balance)
}

// indexETHToken prepares indexing data for a specific asset read from opensea
// The reason to use owner as a parameter is that opensea sometimes returns zero address for it owners. Why?
func (e *IndexEngine) indexETHToken(a *opensea.Asset, owner string, balance int64) (*AssetUpdates, error) {
	dataSource := SourceOpensea

	var sourceURL string
	var artistURL string
	artistID := EthereumChecksumAddress(a.Creator.Address)
	artistName := a.Creator.User.Username
	contractAddress := EthereumChecksumAddress(a.AssetContract.Address)
	switch contractAddress {
	case ENSContractAddress:
		return nil, nil
	}

	source := getTokenSourceByContract(contractAddress)

	switch source {
	case "Art Blocks":
		sourceURL = "https://www.artblocks.io/"
		artistURL = fmt.Sprintf("%s/%s", sourceURL, a.Creator.Address)
	case "Crayon Codes":
		sourceURL = "https://openprocessing.org/crayon/"
		artistURL = fmt.Sprintf("https://opensea.io/%s", a.Creator.Address)
	default:
		if viper.GetString("network") == "testnet" {
			sourceURL = "https://testnets.opensea.io"
		} else {
			sourceURL = "https://opensea.io"
		}
		artistURL = fmt.Sprintf("https://opensea.io/%s", a.Creator.Address)
	}

	log.WithField("source", source).WithField("assetID", a.ID).Debug("source debug")

	if a.Creator.Address != "" {
		if artistName == "" {
			artistName = artistID
		}
	}

	metadata := ProjectMetadata{
		ArtistID:            artistID,
		ArtistName:          artistName,
		ArtistURL:           artistURL,
		AssetID:             contractAddress,
		Title:               a.Name,
		Description:         a.Description,
		MIMEType:            GetMIMEType(a.ImageURL),
		Medium:              MediumUnknown,
		Source:              source,
		SourceURL:           sourceURL,
		PreviewURL:          a.ImageURL,
		ThumbnailURL:        a.ImageURL,
		GalleryThumbnailURL: a.ImagePreviewURL,
		AssetURL:            a.Permalink,
		LastUpdatedAt:       time.Now(),
	}

	if a.AnimationURL != "" {
		metadata.PreviewURL = a.AnimationURL
		metadata.MIMEType = GetMIMEType(a.AnimationURL)

		if source == "Art Blocks" {
			metadata.Medium = MediumSoftware
		} else {
			medium := mediumByPreviewFileExtension(metadata.PreviewURL)
			log.WithField("previewURL", metadata.PreviewURL).WithField("medium", medium).Debug("fallback medium check")
			metadata.Medium = medium
		}
	} else if a.ImageURL != "" {
		metadata.Medium = MediumImage
	}

	contractType := strings.ToLower(a.AssetContract.SchemaName)
	fungible := contractType != "erc721"

	tokenUpdate := &AssetUpdates{
		ID:              fmt.Sprintf("%d", a.ID),
		Source:          dataSource,
		ProjectMetadata: metadata,
		Tokens: []Token{
			{
				BaseTokenInfo: BaseTokenInfo{
					ID:              a.TokenID,
					Blockchain:      EthereumBlockchain,
					Fungible:        fungible,
					ContractType:    contractType,
					ContractAddress: contractAddress,
				},
				IndexID:           TokenIndexID(EthereumBlockchain, contractAddress, a.TokenID),
				Edition:           0,
				Balance:           balance,
				Owner:             owner,
				Owners:            map[string]int64{owner: balance},
				MintAt:            a.AssetContract.CreatedDate.Time, // set minted_at to the contract creation time
				LastRefreshedTime: time.Now(),
			},
		},
	}

	log.WithField("blockchain", EthereumBlockchain).
		WithField("id", TokenIndexID(EthereumBlockchain, contractAddress, a.TokenID)).
		WithField("tokenUpdate", tokenUpdate).
		Trace("asset updating data prepared")

	return tokenUpdate, nil
}

// IndexETHTokenLastActivityTime indexes the last activity timestamp of a given token
func (e *IndexEngine) IndexETHTokenLastActivityTime(ctx context.Context, contract, tokenID string) (time.Time, error) {
	return e.opensea.GetTokenLastActivityTime(contract, tokenID)
}

// IndexETHTokenOwners indexes owners of a given token
func (e *IndexEngine) IndexETHTokenOwners(ctx context.Context, contract, tokenID string) (map[string]int64, error) {
	log.WithField("blockchain", EthereumBlockchain).
		WithField("contract", contract).WithField("tokenID", tokenID).
		Trace("index eth token owners")

	var next *string
	ownersMap := map[string]int64{}
	for {
		owners, n, err := e.opensea.RetrieveTokenOwners(contract, tokenID, next)
		if err != nil {
			return nil, err
		}

		for _, o := range owners {
			ownersMap[o.Owner.Address] = o.Quantity
		}

		if n == nil {
			break
		}

		next = n
	}

	return ownersMap, nil
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

	timestamp, err := GetETHBlockTime(ctx, client, receipt.BlockHash)
	if err != nil {
		return nil, fmt.Errorf("cannot get transaction timestamp")
	}

	transactionDetails := []TransactionDetails{}
	for _, log := range receipt.Logs {
		if len(log.Topics) != 4 || log.Topics[3].Big().String() != tokenID || log.Topics[0].String() != TransferEventSignature {
			continue
		}

		transactionDetail := TransactionDetails{
			From:      common.HexToAddress(log.Topics[1].String()).String(),
			To:        common.HexToAddress(log.Topics[2].String()).String(),
			IndexID:   TokenIndexID(EthereumBlockchain, log.Address.String(), tokenID),
			Timestamp: timestamp,
		}

		transactionDetails = append(transactionDetails, transactionDetail)
	}

	return transactionDetails, nil
}
