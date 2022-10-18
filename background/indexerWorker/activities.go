package indexerWorker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	goethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	indexer "github.com/bitmark-inc/nft-indexer"
	"github.com/bitmark-inc/nft-indexer/contracts"
	"github.com/bitmark-inc/nft-indexer/externals/tzkt"
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
	updates, err := w.indexerEngine.IndexETHTokenByOwner(ctx, owner, offset)
	if err != nil {
		return 0, err
	}

	if len(updates) == 0 {
		return 0, nil
	}

	for _, update := range updates {
		if err := w.indexerStore.IndexAsset(ctx, update.ID, update); err != nil {
			return 0, err
		}
	}

	return len(updates), nil
}

// IndexTezosTokenByOwner indexes Tezos token data for an owner into the format of AssetUpdates
func (w *NFTIndexerWorker) IndexTezosTokenByOwner(ctx context.Context, runID string, owner string, offset int) (int, error) {
	if offset == 0 {
		delay := time.Minute
		account, err := w.indexerStore.GetAccount(ctx, owner)

		if err != nil {
			return 0, err
		}

		if account.LastUpdatedTime.Unix() > time.Now().Add(-delay).Unix() {
			log.WithField("lastUpdatedTime", account.LastUpdatedTime.Unix()).
				WithField("now", time.Now().Add(-delay).Unix()).
				WithField("owner", account.Account).Trace("owner refresh too frequently")
			return 0, nil
		}
	}

	updates, err := w.indexerEngine.IndexTezosTokenByOwner(ctx, owner, offset)
	if err != nil {
		return 0, err
	}

	if len(updates) == 0 {
		err := w.indexerStore.CleanupAccountTokens(ctx, runID, owner)
		return 0, err
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
			RunID:             runID,
		})
	}

	if err := w.indexTezosAccount(ctx, owner); err != nil {
		return 0, err
	}

	if err := w.indexTezosAccountTokens(ctx, owner, accountTokens); err != nil {
		return 0, err
	}

	return len(updates), nil
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

// IndexAsset saves asset data into indexer's storage
func (w *NFTIndexerWorker) GetTokenIDsByOwner(ctx context.Context, owner string) ([]string, error) {
	return w.indexerStore.GetTokenIDsByOwner(ctx, owner)
}

// IndexAsset saves asset data into indexer's storage
func (w *NFTIndexerWorker) GetOutdatedTokensByOwner(ctx context.Context, owner string) ([]indexer.Token, error) {
	return w.indexerStore.GetOutdatedTokensByOwner(ctx, owner)
}

// IndexAsset saves asset data into indexer's storage
func (w *NFTIndexerWorker) IndexAsset(ctx context.Context, updates indexer.AssetUpdates) error {
	return w.indexerStore.IndexAsset(ctx, updates.ID, updates)
}

// indexTezosAccount saves tezos account data into indexer's storage
func (w *NFTIndexerWorker) indexTezosAccount(ctx context.Context, owner string) error {
	account := indexer.Account{
		Account:         owner,
		Blockchain:      "tezos",
		LastUpdatedTime: time.Now(),
	}
	return w.indexerStore.IndexAccount(ctx, account)
}

// indexTezosAccountTokens saves tezos account tokens data into indexer's storage
func (w *NFTIndexerWorker) indexTezosAccountTokens(ctx context.Context, owner string, accountTokens []indexer.AccountToken) error {
	return w.indexerStore.IndexAccountTokens(ctx, owner, accountTokens)
}

type Provenance struct {
	TxId      string    `json:"tx_id"`
	Owner     string    `json:"owner"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Bitmark is the response structure of bitmark registry
type Bitmark struct {
	Id         string       `json:"id"`
	HeadId     string       `json:"head_id"`
	Owner      string       `json:"owner"`
	AssetId    string       `json:"asset_id"`
	Issuer     string       `json:"issuer"`
	Head       string       `json:"head"`
	Status     string       `json:"status"`
	Provenance []Provenance `json:"provenance"`
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
		log.WithError(err).WithField("respData", traceutils.DumpResponse(resp)).Error("fail to decode bitmark payload")
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
			TxID:       p.TxId,
			TxURL:      indexer.TxURL(indexer.BitmarkBlockchain, w.Environment, p.TxId),
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
			{common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")},
			nil, nil,
			{common.HexToHash(hexID)},
		},
	})
	if err != nil {
		return nil, err
	}

	log.WithField("tokenID", hexID).WithField("logs", transferLogs).Debug("token provenance")

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

		block, err := w.wallet.RPCClient().BlockByHash(ctx, l.BlockHash)
		if err != nil {
			return nil, err
		}
		txTime := time.Unix(int64(block.Time()), 0)

		provenances = append(provenances, indexer.Provenance{
			Timestamp:  txTime,
			Type:       txType,
			Owner:      indexer.EthereumChecksumAddress(toAccountHash.Hex()),
			Blockchain: indexer.EthereumBlockchain,
			TxID:       l.TxHash.Hex(),
			TxURL:      indexer.TxURL(indexer.EthereumBlockchain, w.Environment, l.TxHash.Hex()),
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
		return err
	}

	for _, token := range tokens {
		if token.LastRefreshedTime.Unix() > time.Now().Add(-delay).Unix() {
			log.WithField("indexID", token.IndexID).Trace("provenance refresh too frequently")
			continue
		}

		if token.Fungible {
			log.WithField("indexID", token.IndexID).Trace("ignore fungible token")
			continue
		}

		log.WithField("indexID", token.IndexID).Debug("start refresh token provenance updating flow")

		totalProvenances := []indexer.Provenance{}
		switch token.Blockchain {
		case indexer.BitmarkBlockchain:
			provenance, err := w.fetchBitmarkProvenance(token.ID)
			if err != nil {
				return err
			}

			totalProvenances = append(totalProvenances, provenance...)
		case indexer.EthereumBlockchain:
			provenance, err := w.fetchEthereumProvenance(ctx, token.ID, token.ContractAddress)
			if err != nil {
				return err
			}
			totalProvenances = append(totalProvenances, provenance...)
		case indexer.TezosBlockchain:
			lastActivityTime, err := w.indexerEngine.IndexTezosTokenLastActivityTime(ctx, token.ContractAddress, token.ID)
			if err != nil {
				return err
			}

			if delay > 0 && lastActivityTime.Sub(token.LastActivityTime) <= 0 {
				log.WithField("indexID", token.IndexID).Trace("no new updates since last check")
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
			case indexer.BitmarkBlockchain:
				provenance, err := w.fetchBitmarkProvenance(tokenInfo.ID)
				if err != nil {
					return err
				}

				totalProvenances = append(totalProvenances, provenance...)
			case indexer.EthereumBlockchain:
				provenance, err := w.fetchEthereumProvenance(ctx, token.ID, token.ContractAddress)
				if err != nil {
					return err
				}
				totalProvenances = append(totalProvenances, provenance...)
			case indexer.TezosBlockchain:
				provenance, err := w.fetchTezosProvenance(ctx, token.ID, token.ContractAddress)
				if err != nil {
					return err
				}
				totalProvenances = append(totalProvenances, provenance...)
			}
		}

		if err := w.indexerStore.UpdateTokenProvenance(ctx, token.IndexID, totalProvenances); err != nil {
			return err
		}
	}

	return nil
}

// RefreshTezosTokenOwnership refreshes ownership for each tokens
func (w *NFTIndexerWorker) RefreshTezosTokenOwnership(ctx context.Context, indexIDs []string, delay time.Duration) error {
	indexTokens := map[string]indexer.AccountToken{}

	accountTokens, err := w.indexerStore.GetAccountTokensByIndexIDs(ctx, indexIDs)
	if err != nil {
		return err
	}

	for _, token := range accountTokens {
		indexTokens[token.IndexID] = token
	}

	tokens, err := w.indexerStore.GetTokensByIndexIDs(ctx, indexIDs)
	if err != nil {
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
			log.WithField("lastRefresh", token.LastRefreshedTime.Unix()).
				WithField("now", time.Now().Add(-delay).Unix()).
				WithField("indexID", token.IndexID).Trace("ownership refresh too frequently")
			continue
		}

		if !token.Fungible {
			log.WithField("indexID", token.IndexID).Trace("ignore non-fungible token")
			continue
		}

		log.WithField("indexID", token.IndexID).Debug("start refresh token ownership updating flow")
		var (
			owners           map[string]int64
			lastActivityTime time.Time
			err              error
		)
		switch token.Blockchain {
		case indexer.EthereumBlockchain:
			lastActivityTime, err = w.indexerEngine.IndexETHTokenLastActivityTime(ctx, token.ContractAddress, token.ID)
			if err != nil {
				return err
			}

			if lastActivityTime.Sub(token.LastActivityTime) <= 0 {
				log.WithField("indexID", token.IndexID).Trace("no new updates since last check")
				continue
			}

			log.WithField("indexID", token.IndexID).Debug("fetch eth ownership for the token")
			owners, err = w.indexerEngine.IndexETHTokenOwners(ctx, token.ContractAddress, token.ID)
			if err != nil {
				return err
			}
		case indexer.TezosBlockchain:
			lastActivityTime, err = w.indexerEngine.IndexTezosTokenLastActivityTime(ctx, token.ContractAddress, token.ID)
			if err != nil {
				return err
			}

			if lastActivityTime.Sub(token.LastActivityTime) <= 0 {
				log.WithField("indexID", token.IndexID).Trace("no new updates since last check")
				continue
			}

			log.WithField("indexID", token.IndexID).Debug("fetch tezos ownership for the token")
			owners, err = w.indexerEngine.IndexTezosTokenOwners(ctx, token.ContractAddress, token.ID)
			if err != nil {
				return err
			}
		}

		if err := w.indexerStore.UpdateTokenOwners(ctx, token.IndexID, lastActivityTime, owners); err != nil {
			return err
		}
	}
	return nil
}
