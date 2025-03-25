package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"
	coinbase "github.com/bitmark-inc/nft-indexer/externals/coinbase"
	"github.com/fatih/structs"
	"github.com/meirf/gopart"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

const (
	QueryPageSize       = 25
	UnsignedFxhashCID   = "QmYwSwa5hP4346GqD7hAjutwJSmeYTdiLQ7Wec2C7Cez1D"
	UnresolvedFxhashURL = "https://gateway.fxhash.xyz/ipfs//"
)

const (
	assetCollectionName                   = "assets"
	tokenCollectionName                   = "tokens"
	identityCollectionName                = "identities"
	ffIdentityCollectionName              = "ff_identities"
	accountCollectionName                 = "accounts"
	accountTokenCollectionName            = "account_tokens"
	tokenAssetViewCollectionName          = "token_assets"
	collectionsCollectionName             = "collections"
	collectionAssetsCollectionName        = "collection_assets"
	salesTimeSeriesCollectionName         = "sales_time_series"
	historicalExchangeRatesCollectionName = "historical_exchange_rates"
)

var ErrNoRecordUpdated = fmt.Errorf("no record updated")

type Store interface {
	Healthz(ctx context.Context) error
	IndexAsset(ctx context.Context, id string, assetUpdates AssetUpdates) error
	SwapToken(ctx context.Context, swapUpdate SwapUpdate) (string, error)
	UpdateOwner(ctx context.Context, indexID, owner string, updatedAt time.Time) error
	UpdateTokenProvenance(ctx context.Context, indexID string, provenances []Provenance) error
	UpdateTokenOwners(ctx context.Context, indexID string, lastActivityTime time.Time, ownerBalances []OwnerBalance) error
	PushProvenance(ctx context.Context, indexID string, lockedTime time.Time, provenance Provenance) error
	FilterTokenIDsWithInconsistentProvenanceForOwner(ctx context.Context, indexIDs []string, owner string) ([]string, error)
	GetTokensByIndexIDs(ctx context.Context, indexIDs []string) ([]Token, error)
	GetTokenByIndexID(ctx context.Context, indexID string) (*Token, error)
	GetOwnedTokenIDsByOwner(ctx context.Context, owner string) ([]string, error)
	GetDetailedTokens(ctx context.Context, filterParameter FilterParameter, offset, size int64) ([]DetailedToken, error)
	GetDetailedTokensByOwners(ctx context.Context, owner []string, filterParameter FilterParameter, offset, size int64) ([]DetailedToken, error)
	GetTokensByTextSearch(ctx context.Context, searchText string, offset, size int64) ([]DetailedToken, error)
	GetIdentity(ctx context.Context, accountNumber string) (AccountIdentity, error)
	GetIdentities(ctx context.Context, accountNumbers []string) (map[string]AccountIdentity, error)
	IndexIdentity(ctx context.Context, identity AccountIdentity) error
	IndexAccount(ctx context.Context, account Account) error
	IndexAccountTokens(ctx context.Context, owner string, accountTokens []AccountToken) error
	GetAccount(ctx context.Context, owner string) (Account, error)
	UpdateAccountTokenOwners(ctx context.Context, indexID string, tokenBalances []OwnerBalance) error
	IndexDemoTokens(ctx context.Context, owner string, indexIDs []string) error
	DeleteDemoTokens(ctx context.Context, owner string) error
	UpdateOwnerForFungibleToken(ctx context.Context, indexID string, lockedTime time.Time, to string, total int64) error
	GetLatestActivityTimeByIndexIDs(ctx context.Context, indexIDs []string) (map[string]time.Time, error)
	MarkAccountTokenChanged(ctx context.Context, indexIDs []string) error
	GetDetailedTokensV2(ctx context.Context, filterParameter FilterParameter, offset, size int64) ([]DetailedTokenV2, error)
	GetDetailedAccountTokensByOwners(ctx context.Context, owner []string, filterParameter FilterParameter, lastUpdatedAt time.Time, sortBy string, offset, size int64) ([]DetailedTokenV2, error)
	CountDetailedAccountTokensByOwner(ctx context.Context, owner string) (int64, error)
	GetDetailedToken(ctx context.Context, indexID string, burnedIncluded bool) (DetailedToken, error)
	GetTotalBalanceOfOwnerAccounts(ctx context.Context, addresses []string) (int, error)
	GetNullProvenanceTokensByIndexIDs(ctx context.Context, indexIDs []string) ([]string, error)
	GetOwnerAccountsByIndexIDs(ctx context.Context, indexIDs []string) ([]string, error)
	CheckAddressOwnTokenByCriteria(ctx context.Context, address string, criteria Criteria) (bool, error)
	GetOwnersByBlockchainContracts(context.Context, map[string][]string) ([]string, error)
	IndexCollection(ctx context.Context, collection Collection) error
	IndexCollectionAsset(ctx context.Context, collectionID string, collectionAssets []CollectionAsset) error
	DeleteCollection(ctx context.Context, collectionID string) error
	ReplaceCollectionCreator(ctx context.Context, oldCreator, newCreator string) error
	UpdateCollectionCreators(ctx context.Context, collectionID string, creators []string) error
	DeleteDeprecatedCollectionAsset(ctx context.Context, collectionID, runID string) error
	GetCollectionLastUpdatedTimeForCreator(ctx context.Context, creator string) (time.Time, error)
	GetCollectionLastUpdatedTime(ctx context.Context, collectionID string) (time.Time, error)
	GetCollectionByID(ctx context.Context, id string) (*Collection, error)
	GetCollectionsByCreators(ctx context.Context, creators []string, offset, size int64) ([]Collection, error)
	GetDetailedTokensByCollectionID(ctx context.Context, collectionID string, sortBy string, offset, size int64) ([]DetailedTokenV2, error)
	FilterBurnedIndexIDs(ctx context.Context, indexIDs []string) ([]string, error)
	WriteTimeSeriesData(
		ctx context.Context,
		records []GenericSalesTimeSeries,
	) error
	SaleTimeSeriesDataExists(ctx context.Context, txID, blockchain string) (bool, error)
	GetSaleTimeSeriesData(ctx context.Context, filter SalesFilterParameter) ([]SaleTimeSeries, error)
	AggregateSaleRevenues(ctx context.Context, filter SalesFilterParameter) (map[string]primitive.Decimal128, error)
	WriteHistoricalExchangeRate(ctx context.Context, exchangeRate []coinbase.HistoricalExchangeRate) error
	GetHistoricalExchangeRate(ctx context.Context, filter HistoricalExchangeRateFilter) (ExchangeRate, error)
	GetExchangeRateLastTime(ctx context.Context) (time.Time, error)
	UpdateAssetConfiguration(ctx context.Context, indexID string, configuration *AssetConfiguration) (int64, error)
}

type FilterParameter struct {
	Source         string
	IDs            []string
	BurnedIncluded bool
}

type Criteria struct {
	IndexID string `bson:"indexID"`
	Source  string `bson:"source"`
}

type OwnerBalance struct {
	Address  string    `json:"address"`
	Balance  int64     `json:"balance,string"`
	LastTime time.Time `json:"lastTime"`
}

type SalesFilterParameter struct {
	Addresses   []string
	Marketplace string
	From        *time.Time
	To          *time.Time
	Limit       int64
	Offset      int64
	SortASC     bool
}

type HistoricalExchangeRateFilter struct {
	CurrencyPair string
	Timestamp    time.Time
}

func NewMongodbIndexerStore(ctx context.Context, mongodbURI, dbName, environment string) (*MongodbIndexerStore, error) {
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongodbURI))
	if err != nil {
		return nil, err
	}

	db := mongoClient.Database(dbName)
	tokenCollection := db.Collection(tokenCollectionName)
	assetCollection := db.Collection(assetCollectionName)
	identityCollection := db.Collection(identityCollectionName)
	ffIdentityCollection := db.Collection(ffIdentityCollectionName)
	accountCollection := db.Collection(accountCollectionName)
	accountTokenCollection := db.Collection(accountTokenCollectionName)
	tokenAssetCollection := db.Collection(tokenAssetViewCollectionName)
	collectionsCollection := db.Collection(collectionsCollectionName)
	collectionAssetsCollection := db.Collection(collectionAssetsCollectionName)
	salesTimeSeriesCollection := db.Collection(salesTimeSeriesCollectionName)
	historicalExchangeRatesCollection := db.Collection(historicalExchangeRatesCollectionName)

	return &MongodbIndexerStore{
		environment:                       environment,
		dbName:                            dbName,
		mongoClient:                       mongoClient,
		tokenCollection:                   tokenCollection,
		assetCollection:                   assetCollection,
		identityCollection:                identityCollection,
		ffIdentityCollection:              ffIdentityCollection,
		accountCollection:                 accountCollection,
		accountTokenCollection:            accountTokenCollection,
		tokenAssetCollection:              tokenAssetCollection,
		collectionsCollection:             collectionsCollection,
		collectionAssetsCollection:        collectionAssetsCollection,
		salesTimeSeriesCollection:         salesTimeSeriesCollection,
		historicalExchangeRatesCollection: historicalExchangeRatesCollection,
	}, nil
}

type MongodbIndexerStore struct {
	environment                       string
	dbName                            string
	mongoClient                       *mongo.Client
	tokenCollection                   *mongo.Collection
	assetCollection                   *mongo.Collection
	identityCollection                *mongo.Collection
	ffIdentityCollection              *mongo.Collection
	accountCollection                 *mongo.Collection
	accountTokenCollection            *mongo.Collection
	tokenAssetCollection              *mongo.Collection
	collectionsCollection             *mongo.Collection
	collectionAssetsCollection        *mongo.Collection
	salesTimeSeriesCollection         *mongo.Collection
	historicalExchangeRatesCollection *mongo.Collection
}

type AssetUpdateSet struct {
	ID                 string                   `structs:"id,omitempty"`
	IndexID            string                   `structs:"indexID,omitempty"`
	Source             string                   `structs:"source,omitempty"`
	BlockchainMetadata interface{}              `structs:"blockchainMetadata,omitempty"`
	ProjectMetadata    VersionedProjectMetadata `structs:"projectMetadata,omitempty"`
	LastRefreshedTime  time.Time                `structs:"lastRefreshedTime"`
}

type TokenUpdateSet struct {
	Source            string    `structs:"source,omitempty"`
	AssetID           string    `structs:"assetID,omitempty"`
	Fungible          bool      `structs:"fungible,omitempty"`
	Edition           int64     `structs:"edition,omitempty"`
	EditionName       string    `structs:"editionName,omitempty"`
	ContractAddress   string    `structs:"contractAddress,omitempty"`
	MintedAt          time.Time `structs:"mintedAt,omitempty"`
	LastRefreshedTime time.Time `structs:"lastRefreshedTime"`
	LastActivityTime  time.Time `structs:"lastActivityTime,omitempty"`
}

// checkIfTokenNeedToUpdate returns true if the new token data is suppose to be
// better than existent one.
func checkIfTokenNeedToUpdate(assetSource string, currentToken Token) bool {
	// check if we need to update an existent token
	if assetSource == SourceFeralFile {
		return true
	}

	// ignore updates for swapped and burned token
	if currentToken.Swapped || currentToken.Burned {
		return false
	}

	// update only if the token source is not feral file
	if currentToken.Source != SourceFeralFile {
		return true
	}

	return false
}

// Healthz checks the db health status and returns errors if any
func (s *MongodbIndexerStore) Healthz(ctx context.Context) error {
	if err := s.mongoClient.Ping(ctx, readpref.Primary()); err != nil {
		return err
	}

	return s.mongoClient.Ping(ctx, readpref.Secondary())
}

// IndexAsset creates an asset and its corresponded tokens by inputs
func (s *MongodbIndexerStore) IndexAsset(ctx context.Context, id string, assetUpdates AssetUpdates) error {
	assetCreated := false

	assetIndexID := fmt.Sprintf("%s-%s", strings.ToLower(assetUpdates.Source), id)
	indexTime := time.Now()

	if assetUpdates.ProjectMetadata.LastUpdatedAt.IsZero() {
		assetUpdates.ProjectMetadata.LastUpdatedAt = indexTime
	}

	// insert or update an incoming asset
	assetResult := s.assetCollection.FindOne(ctx, bson.M{"indexID": assetIndexID},
		options.FindOne().SetProjection(bson.M{"source": 1, "projectMetadata": 1}))
	if err := assetResult.Err(); err != nil {
		if assetResult.Err() == mongo.ErrNoDocuments {
			// Create a new asset if it is not added
			assetUpdateSet := AssetUpdateSet{
				ID:                 id,
				IndexID:            assetIndexID,
				Source:             assetUpdates.Source,
				BlockchainMetadata: assetUpdates.BlockchainMetadata,
				ProjectMetadata: VersionedProjectMetadata{
					Origin: assetUpdates.ProjectMetadata,
					Latest: assetUpdates.ProjectMetadata,
				},
				LastRefreshedTime: indexTime,
			}

			if _, err := s.assetCollection.InsertOne(ctx, structs.Map(assetUpdateSet)); err != nil {
				return err
			}
			assetCreated = true
		} else {
			return err
		}
	}

	// update an existent asset
	if !assetCreated {
		var currentAsset struct {
			Source          string                   `json:"source" bson:"source"`
			ProjectMetadata VersionedProjectMetadata `json:"projectMetadata" bson:"projectMetadata"`
		}

		if err := assetResult.Decode(&currentAsset); err != nil {
			return err
		}
		var requireUpdates bool
		// check if we need to update an existent asset
		// 1. the incoming update's source IS Feralfile => update
		// 2. the incoming update's source IS NOT Feralfile AND current asset's source is not FeralFile
		if assetUpdates.Source == SourceFeralFile {
			requireUpdates = true
		} else {
			// incoming update's source IS NOT Feralfile
			if currentAsset.Source != SourceFeralFile {
				requireUpdates = true
			}
		}

		if requireUpdates {
			updates := bson.D{{Key: "$set", Value: bson.D{
				{Key: "projectMetadata.latest", Value: assetUpdates.ProjectMetadata},
				{Key: "lastRefreshedTime", Value: indexTime},
			}}}

			// TODO: check whether to remove the thumbnail cache when the thumbnail data is updated.
			if currentAsset.ProjectMetadata.Latest.ThumbnailURL != assetUpdates.ProjectMetadata.ThumbnailURL {
				log.Debug("image cache need to be reset",
					zap.String("old", currentAsset.ProjectMetadata.Latest.ThumbnailURL),
					zap.String("new", assetUpdates.ProjectMetadata.ThumbnailURL))
				updates = append(updates, bson.E{Key: "$unset", Value: bson.M{"thumbnailID": ""}})
			}

			_, err := s.assetCollection.UpdateOne(
				ctx,
				bson.M{"indexID": assetIndexID},
				updates,
			)
			if err != nil {
				return err
			}
		}
	}

	// loop the attached tokens for an asset and try to insert or update
	for _, token := range assetUpdates.Tokens {
		token.AssetID = id

		if token.IndexID == "" {
			token.IndexID = TokenIndexID(token.Blockchain, token.ContractAddress, token.ID)
		}

		if assetUpdates.Source == SourceFeralFile {
			token.Source = SourceFeralFile
		}

		tokenResult := s.tokenCollection.FindOne(ctx, bson.M{"indexID": token.IndexID})
		edition := token.Edition
		lastActivityTime := token.LastActivityTime
		if err := tokenResult.Err(); err != nil {
			if err == mongo.ErrNoDocuments {
				// If a token is not found, insert a new token
				log.InfoWithContext(ctx, "new token found", zap.String("token_id", token.ID))

				if token.LastActivityTime.IsZero() {
					// set LastActivityTime to default token minted time
					lastActivityTime = token.MintedAt
					token.LastActivityTime = lastActivityTime
				}

				if token.Owner != "" {
					token.OwnersArray = []string{token.Owner}
					token.Owners = map[string]int64{token.Owner: 1}
				}

				// Insert token
				_, err := s.tokenCollection.InsertOne(ctx, token)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			// Parse the existing token
			var currentToken Token
			if err := tokenResult.Decode(&currentToken); err != nil {
				return err
			}
			edition = currentToken.Edition
			lastActivityTime = currentToken.LastActivityTime

			// Check if token need to be updated
			if checkIfTokenNeedToUpdate(assetUpdates.Source, currentToken) {
				log.Debug("token data need to update", zap.String("token_id", token.ID))

				tokenUpdateSet := TokenUpdateSet{
					Fungible:          token.Fungible,
					Source:            token.Source,
					AssetID:           id,
					Edition:           token.Edition,
					EditionName:       token.EditionName,
					ContractAddress:   token.ContractAddress,
					LastRefreshedTime: indexTime,
				}

				if currentToken.MintedAt.IsZero() && !token.MintedAt.IsZero() {
					tokenUpdateSet.MintedAt = token.MintedAt
				}

				if !token.LastActivityTime.IsZero() {
					tokenUpdateSet.LastActivityTime = token.LastActivityTime
				}

				edition = tokenUpdateSet.Edition
				if !tokenUpdateSet.LastActivityTime.IsZero() {
					lastActivityTime = tokenUpdateSet.LastActivityTime
				}

				tokenUpdate := bson.M{"$set": structs.Map(tokenUpdateSet)}

				log.Debug("token data for updated", zap.String("token_id", token.ID), zap.Any("tokenUpdate", tokenUpdate))
				r, err := s.tokenCollection.UpdateOne(ctx, bson.M{"indexID": token.IndexID}, tokenUpdate)
				if err != nil {
					return err
				}
				if r.MatchedCount == 0 {
					log.WarnWithContext(ctx, "token is not updated", zap.String("token_id", token.ID))
				}
			}
		}

		// Update collection asset
		_, err := s.collectionAssetsCollection.UpdateOne(
			ctx,
			bson.M{"tokenIndexID": token.IndexID},
			bson.M{"$set": bson.M{
				"lastActivityTime": lastActivityTime,
				"edition":          edition,
			}},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// SwapToken marks the original token to burned and creates a new token record which inherits
// original blockchain information
func (s *MongodbIndexerStore) SwapToken(ctx context.Context, swap SwapUpdate) (string, error) {
	originalTokenIndexID := TokenIndexID(swap.OriginalBlockchain, swap.OriginalContractAddress, swap.OriginalTokenID)

	var newTokenIndexID string

	switch swap.NewBlockchain {
	case utils.EthereumBlockchain, utils.TezosBlockchain:
		newTokenIndexID = TokenIndexID(swap.NewBlockchain, swap.NewContractAddress, swap.NewTokenID)
	default:
		return "", fmt.Errorf("blockchain is not supported")
	}

	tokenResult := s.tokenCollection.FindOne(ctx, bson.M{
		"indexID": originalTokenIndexID,
	})
	if err := tokenResult.Err(); err != nil {
		return "", err
	}

	var originalToken Token
	if err := tokenResult.Decode(&originalToken); err != nil {
		return "", err
	}

	if originalToken.Burned && originalToken.SwappedTo != nil {
		if *originalToken.SwappedTo != newTokenIndexID {
			return "", fmt.Errorf("token has burned into different id")
		}
	}

	originalBaseTokenInfo := originalToken.BaseTokenInfo

	newToken := originalToken
	newToken.ID = swap.NewTokenID
	newToken.IndexID = newTokenIndexID
	newToken.Blockchain = swap.NewBlockchain
	newToken.ContractAddress = swap.NewContractAddress
	newToken.ContractType = swap.NewContractType
	newToken.Swapped = true
	newToken.Burned = false
	newToken.SwappedTo = nil
	newToken.SwappedFrom = &originalTokenIndexID
	newToken.OriginTokenInfo = append([]BaseTokenInfo{originalBaseTokenInfo}, newToken.OriginTokenInfo...)

	log.Debug("update tokens for swapping", zap.String("from", originalTokenIndexID), zap.String("to", newTokenIndexID))
	session, err := s.mongoClient.StartSession()
	if err != nil {
		return "", err
	}
	defer session.EndSession(ctx)

	result, err := session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		if _, err := s.tokenCollection.UpdateOne(sessCtx, bson.M{"indexID": originalTokenIndexID}, bson.M{
			"$set": bson.M{
				"burned":    true,
				"swappedTo": newTokenIndexID,
			},
		}); err != nil {
			return nil, err
		}

		r, err := s.tokenCollection.UpdateOne(sessCtx,
			bson.M{"indexID": newTokenIndexID},
			bson.M{"$set": newToken},
			options.Update().SetUpsert(true))
		if err != nil {
			return nil, err
		}

		if r.MatchedCount == 0 && r.UpsertedCount == 0 {
			return nil, ErrNoRecordUpdated
		}

		return nil, nil
	})

	if err != nil {
		log.ErrorWithContext(ctx, errors.New("swap token transaction failed"),
			zap.String("originalTokenIndexID", originalTokenIndexID),
			zap.String("newTokenIndexID", newTokenIndexID),
			zap.Error(err))
		return "", err
	}

	log.Debug("swap token transaction", zap.Any("transaction_result", result))

	// Update account tokens
	_, err = s.accountTokenCollection.UpdateMany(ctx,
		bson.M{"indexID": originalTokenIndexID},
		bson.M{"$set": bson.M{
			"indexID":           newTokenIndexID,
			"lastRefreshedTime": time.Now(),
			"lastActivityTime":  time.Now(),
		}})

	return newTokenIndexID, err
}

// FilterTokenIDsWithInconsistentProvenanceForOwner returns a list of token ids where the latest token is not the given owner
func (s *MongodbIndexerStore) FilterTokenIDsWithInconsistentProvenanceForOwner(ctx context.Context, indexIDs []string, owner string) ([]string, error) {
	var tokenIDs []string

	c, err := s.tokenCollection.Find(ctx,
		bson.M{
			"indexID":            bson.M{"$in": indexIDs},
			"fungible":           false,
			"provenance.0.owner": bson.M{"$ne": owner},
		},
		options.Find().SetProjection(bson.M{"indexID": 1, "_id": 0}),
	)
	if err != nil {
		return nil, err
	}

	for c.Next(ctx) {
		var v struct {
			IndexID string
		}
		if err := c.Decode(&v); err != nil {
			return nil, err
		}

		tokenIDs = append(tokenIDs, v.IndexID)
	}

	return tokenIDs, nil
}

// GetTokensByIndexIDs returns a list of tokens by a given list of index id
func (s *MongodbIndexerStore) GetTokensByIndexIDs(ctx context.Context, ids []string) ([]Token, error) {
	var tokens []Token

	c, err := s.tokenCollection.Find(ctx, bson.M{"indexID": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}

	if err := c.All(ctx, &tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

// getDetailedTokensByAggregation returns detail tokens by mongodb aggregation
func (s *MongodbIndexerStore) getDetailedTokensByAggregation(ctx context.Context, filterParameter FilterParameter, offset, size int64) ([]DetailedToken, error) {
	tokens := []DetailedToken{}
	cursor, err := s.getTokensByAggregation(ctx, filterParameter, offset, size)

	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

// getPageCounts return the page counts by item length and page size
func getPageCounts(itemLength, PageSize int) int {
	pageCounts := itemLength / PageSize
	if itemLength%PageSize != 0 {
		pageCounts++
	}
	return pageCounts
}

// GetDetailedTokens returns a list of tokens information based on ids
func (s *MongodbIndexerStore) GetDetailedTokens(ctx context.Context, filterParameter FilterParameter, offset, size int64) ([]DetailedToken, error) {
	tokens := []DetailedToken{}

	log.Debug("GetDetailedTokens",
		zap.Any("filterParameter", filterParameter),
		zap.Int64("offset", offset),
		zap.Int64("size", size))
	startTime := time.Now()
	if length := len(filterParameter.IDs); length > 0 {
		for i := 0; i < getPageCounts(length, QueryPageSize); i++ {
			log.Debug("doc page", zap.Int("page", i))
			start := i * QueryPageSize
			end := (i + 1) * QueryPageSize
			if end > length {
				end = length
			}

			pagedTokens, err := s.getDetailedTokensByAggregation(ctx,
				FilterParameter{
					IDs:            filterParameter.IDs[start:end],
					Source:         filterParameter.Source,
					BurnedIncluded: filterParameter.BurnedIncluded,
				},
				offset, size)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, pagedTokens...)
		}
	} else {
		return s.getDetailedTokensByAggregation(ctx, filterParameter, offset, size)
	}
	log.Debug("GetDetailedTokens End", zap.Duration("queryTime", time.Since(startTime)))

	return tokens, nil
}

// FilterBurnedIndexIDs filter out burned tokens from provided list
func (s *MongodbIndexerStore) FilterBurnedIndexIDs(ctx context.Context, indexIDs []string) ([]string, error) {
	burnedTokens := make(map[string]struct{})

	c, err := s.tokenCollection.Find(ctx, bson.M{
		"indexID": bson.M{"$in": indexIDs},
		"burned":  true,
	},
		options.Find().SetProjection(bson.M{"indexID": 1, "_id": 0}))
	if err != nil {
		return nil, err
	}
	defer c.Close(ctx)

	for c.Next(ctx) {
		var v struct {
			IndexID string
		}
		if err := c.Decode(&v); err != nil {
			return nil, err
		}

		burnedTokens[v.IndexID] = struct{}{}
	}

	// Filter out burned tokens from original list
	filteredTokens := make([]string, 0, len(indexIDs))
	for _, id := range indexIDs {
		if _, burned := burnedTokens[id]; !burned {
			filteredTokens = append(filteredTokens, id)
		}
	}

	// Check for errors during cursor iteration
	if err := c.Err(); err != nil {
		return nil, err
	}

	return filteredTokens, nil
}

// UpdateOwner updates owner for a specific non-fungible token
func (s *MongodbIndexerStore) UpdateOwner(ctx context.Context, indexID string, owner string, updatedAt time.Time) error {
	if owner == "" {
		log.WarnWithContext(ctx, "ignore update empty owner", zap.String("indexID", indexID))
		return nil
	}

	burned := IsBurnAddress(owner, s.environment)
	_, err := s.tokenCollection.UpdateOne(ctx, bson.M{
		"indexID":          indexID,
		"fungible":         false,
		"lastActivityTime": bson.M{"$lt": updatedAt},
	}, bson.M{
		"$set": bson.M{
			"owner":             owner,
			"owners":            map[string]int64{owner: 1},
			"ownersArray":       []string{owner},
			"lastActivityTime":  updatedAt,
			"lastRefreshedTime": time.Now(),
			"burned":            burned,
		},
	})

	return err
}

// UpdateTokenProvenance updates provenance for a specific token
func (s *MongodbIndexerStore) UpdateTokenProvenance(ctx context.Context, indexID string, provenances []Provenance) error {
	if len(provenances) == 0 {
		log.WarnWithContext(ctx, "ignore update empty provenance", zap.String("indexID", indexID))
		return nil
	}

	currentOwner := provenances[0].Owner
	tokenUpdates := bson.M{
		"owner":             currentOwner,
		"owners":            map[string]int64{currentOwner: 1},
		"ownersArray":       []string{currentOwner},
		"lastActivityTime":  provenances[0].Timestamp,
		"lastRefreshedTime": time.Now(),
		"provenance":        provenances,
	}

	lastProvenance := provenances[len(provenances)-1]
	if lastProvenance.Type == "mint" || lastProvenance.Type == "issue" {
		tokenUpdates["mintedAt"] = lastProvenance.Timestamp
	}

	// update provenance only for non-burned tokens
	_, err := s.tokenCollection.UpdateOne(ctx, bson.M{
		"indexID": indexID,
	}, bson.M{
		"$set": tokenUpdates,
	})

	return err
}

// UpdateTokenOwners updates owners for a specific token
func (s *MongodbIndexerStore) UpdateTokenOwners(ctx context.Context, indexID string, lastActivityTime time.Time, ownerBalances []OwnerBalance) error {
	if len(ownerBalances) == 0 {
		log.WarnWithContext(ctx, "ignore update empty provenance", zap.String("indexID", indexID))
		return nil
	}

	ownersArray := make([]string, 0, len(ownerBalances))
	owners := map[string]int64{}
	for _, ownerBalance := range ownerBalances {
		ownersArray = append(ownersArray, ownerBalance.Address)
		owners[ownerBalance.Address] = ownerBalance.Balance
	}

	tokenUpdates := bson.M{
		"owners":            owners,
		"ownersArray":       ownersArray,
		"provenance":        nil,
		"lastRefreshedTime": time.Now(),
	}

	if !lastActivityTime.IsZero() {
		tokenUpdates["lastActivityTime"] = lastActivityTime
	}

	// update provenance only for non-burned tokens
	_, err := s.tokenCollection.UpdateOne(ctx, bson.M{
		"indexID": indexID,
	}, bson.M{
		"$set": tokenUpdates,
	})

	return err
}

// PushProvenance push the latest provenance record for a token
func (s *MongodbIndexerStore) PushProvenance(ctx context.Context, indexID string, lockedTime time.Time, provenance Provenance) error {
	if provenance.FormerOwner == nil {
		return fmt.Errorf("invalid former owner")
	}

	u, err := s.tokenCollection.UpdateOne(ctx, bson.M{
		"indexID":           indexID,
		"lastRefreshedTime": lockedTime,
		"lastActivityTime":  bson.M{"$lt": provenance.Timestamp},
	}, bson.M{
		"$set": bson.M{
			"owner":             provenance.Owner,
			"owners":            map[string]int64{provenance.Owner: 1},
			"ownersArray":       []string{provenance.Owner},
			"lastActivityTime":  provenance.Timestamp,
			"lastRefreshedTime": time.Now(),
		},

		"$push": bson.M{
			"provenance": bson.M{
				"$each":     bson.A{provenance},
				"$position": 0,
			},
		},
	})

	if err != nil {
		return err
	}

	if u.ModifiedCount == 0 {
		return ErrNoRecordUpdated
	}

	return nil
}

// GetOwnedTokenIDsByOwner returns a list of tokens which belongs to an owner
func (s *MongodbIndexerStore) GetOwnedTokenIDsByOwner(ctx context.Context, owner string) ([]string, error) {
	tokens := make([]string, 0)

	c, err := s.accountTokenCollection.Find(ctx, bson.M{
		"ownerAccount": owner,
		"balance":      bson.M{"$gt": 0},
	},
		options.Find().SetProjection(bson.M{"indexID": 1, "_id": 0}))
	if err != nil {
		return nil, err
	}

	for c.Next(ctx) {
		var v struct {
			IndexID string
		}
		if err := c.Decode(&v); err != nil {
			return nil, err
		}

		tokens = append(tokens, v.IndexID)
	}

	return tokens, nil
}

// GetTokensByOwner returns a list of DetailedTokens which belong to an owner
func (s *MongodbIndexerStore) GetDetailedTokensByOwners(ctx context.Context, owners []string, filterParameter FilterParameter, offset, size int64) ([]DetailedToken, error) {
	tokens := make([]DetailedToken, 0, 10)

	type asset struct {
		ThumbnailID     string                   `bson:"thumbnailID"`
		IPFSPinned      bool                     `json:"ipfsPinned"`
		Attributes      *AssetAttributes         `json:"attributes" bson:"attributes,omitempty"` // manually inserted fields
		ProjectMetadata VersionedProjectMetadata `bson:"projectMetadata"`
	}

	assets := map[string]asset{}

	var c *mongo.Cursor
	var err error

	if len(owners) == 1 {
		c, err = s.getTokensByAggregationForOwner(ctx, owners[0], filterParameter, offset, size)
		if err != nil {
			return nil, err
		}
	} else {
		// DEPRECATED: this condition should not use anymore
		c, err = s.getTokensByAggregationByOwners(ctx, owners, filterParameter, offset, size)
		if err != nil {
			return nil, err
		}
	}

	for c.Next(ctx) {
		var token DetailedToken
		if err := c.Decode(&token); err != nil {
			return nil, err
		}

		a, assetExist := assets[token.AssetID]
		if !assetExist {
			assetResult := s.assetCollection.FindOne(ctx, bson.M{"id": token.AssetID})
			if err := assetResult.Err(); err != nil {
				return nil, err
			}

			if err := assetResult.Decode(&a); err != nil {
				return nil, err
			}

			assets[token.AssetID] = a
		}

		// FIXME: hardcoded values for backward compatibility
		token.ThumbnailID = a.ThumbnailID
		token.IPFSPinned = a.IPFSPinned
		token.ProjectMetadata = a.ProjectMetadata
		token.Attributes = a.Attributes

		tokens = append(tokens, token)
	}

	return tokens, nil
}

// getTokensByAggregationForOwner queries tokens for a specific owner by aggregation
func (s *MongodbIndexerStore) getTokensByAggregationForOwner(ctx context.Context, owner string, filterParameter FilterParameter, offset, size int64) (*mongo.Cursor, error) {
	matchQuery := bson.M{
		fmt.Sprintf("owners.%s", owner): bson.M{"$gte": 1},
		"ownersArray":                   bson.M{"$in": bson.A{owner}},
	}

	if !filterParameter.BurnedIncluded {
		matchQuery["burned"] = bson.M{"$ne": true}
	}

	pipelines := []bson.M{
		{
			"$match": matchQuery,
		},
		{"$sort": bson.D{{Key: "lastActivityTime", Value: -1}, {Key: "_id", Value: -1}}},
		{"$addFields": bson.M{"balance": fmt.Sprintf("$owners.%s", owner)}},
		// lookup performs a cross blockchain join between tokens and assets collections
		{
			"$lookup": bson.M{
				"from": "assets",
				"let": bson.M{
					"assetID": "$assetID",
				},
				"pipeline": bson.A{
					bson.M{
						"$match": bson.M{
							"$expr": bson.M{
								"$eq": bson.A{
									"$id",
									"$$assetID",
								},
							},
						},
					},
					bson.M{
						"$project": bson.M{
							"source": 1,
							"_id":    0,
						},
					},
				},
				"as": "asset",
			},
		},
	}

	if filterParameter.Source != "" {
		pipelines = append(pipelines, bson.M{"$match": bson.M{"asset.source": filterParameter.Source}})
	}

	pipelines = append(pipelines,
		bson.M{"$skip": offset},
		bson.M{"$limit": size},
	)

	return s.tokenCollection.Aggregate(ctx, pipelines)
}

// DEPRECATED: getTokensByAggregationByOwners queries tokens by aggregation which provides a more flexible query option by mongodb
func (s *MongodbIndexerStore) getTokensByAggregationByOwners(ctx context.Context, owners []string, filterParameter FilterParameter, offset, size int64) (*mongo.Cursor, error) {
	entryMatch := bson.M{"owner": bson.M{"$in": owners}}
	if !filterParameter.BurnedIncluded {
		entryMatch["burned"] = bson.M{"$ne": true}
	}

	pipelines := []bson.M{
		{
			"$match": entryMatch,
		},
		{"$sort": bson.D{{Key: "lastActivityTime", Value: -1}, {Key: "_id", Value: -1}}},
		// lookup performs a cross blockchain join between tokens and assets collections
		{
			"$lookup": bson.M{
				"from": "assets",
				"let": bson.M{
					"assetID": "$assetID",
				},
				"pipeline": bson.A{
					bson.M{
						"$match": bson.M{
							"$expr": bson.M{
								"$eq": bson.A{
									"$id",
									"$$assetID",
								},
							},
						},
					},
					bson.M{
						"$project": bson.M{
							"source": 1,
							"_id":    0,
						},
					},
				},
				"as": "asset",
			},
		},
		// unwind turns the
		{"$unwind": "$asset"},
	}

	if filterParameter.Source != "" {
		pipelines = append(pipelines, bson.M{"$match": bson.M{"asset.source": filterParameter.Source}})
	}

	pipelines = append(pipelines,
		bson.M{"$skip": offset},
		bson.M{"$limit": size},
	)

	return s.tokenCollection.Aggregate(ctx, pipelines)
}

// getTokensByAggregation queries tokens by aggregation which provides a more flexible query option by mongodb
func (s *MongodbIndexerStore) getTokensByAggregation(ctx context.Context, filterParameter FilterParameter, offset, size int64) (*mongo.Cursor, error) {
	matchQuery := bson.M{}

	if len(filterParameter.IDs) > 0 {
		matchQuery = bson.M{
			"indexID": bson.M{"$in": filterParameter.IDs},
		}
		if !filterParameter.BurnedIncluded {
			matchQuery["burned"] = bson.M{"$ne": true}
		}
	}

	pipelines := []bson.M{
		{
			"$match": matchQuery,
		},
		{"$sort": bson.D{{Key: "lastActivityTime", Value: -1}, {Key: "_id", Value: -1}}},
		// lookup performs a cross blockchain join between tokens and assets collections
		{
			"$lookup": bson.M{
				"from": "assets",
				"let": bson.M{
					"assetID": "$assetID",
				},
				"pipeline": bson.A{
					bson.M{
						"$match": bson.M{
							"$expr": bson.M{
								"$eq": bson.A{
									"$id",
									"$$assetID",
								},
							},
						},
					},
					bson.M{
						"$project": bson.M{
							"source":          1,
							"projectMetadata": 1,
							"attributes":      1,
							"thumbnailID":     1,
							"ipfsPinned":      1,
							"_id":             0,
						},
					},
				},
				"as": "asset",
			},
		},
		{"$unwind": "$asset"},
		{
			"$replaceRoot": bson.M{
				"newRoot": bson.M{
					"$mergeObjects": bson.A{"$$ROOT", "$asset"},
				},
			},
		},
		{"$project": bson.M{"asset": 0}},
	}

	if filterParameter.Source != "" {
		pipelines = append(pipelines, bson.M{"$match": bson.M{"source": filterParameter.Source}})
	}

	if len(matchQuery) == 0 {
		if size == 0 || size > 100 {
			size = 100
		}
		pipelines = append(pipelines,
			bson.M{"$skip": offset},
			bson.M{"$limit": size},
		)
	}

	return s.tokenCollection.Aggregate(ctx, pipelines)
}

// GetTokensByTextSearch returns a list of token those assets match have attributes that match the search text.
func (s *MongodbIndexerStore) GetTokensByTextSearch(ctx context.Context, searchText string, offset, size int64) ([]DetailedToken, error) {
	log.Debug("GetTokensByTextSearch",
		zap.String("searchText", searchText),
		zap.Int64("offset", offset),
		zap.Int64("size", size))

	pipeline := []bson.M{
		{"$match": bson.M{
			"projectMetadata.latest.source": SourceFeralFile, // FIXME: currently, we limit the source of query to feralfile
			"$or": bson.A{
				bson.M{"projectMetadata.latest.artistName": bson.M{"$regex": primitive.Regex{Pattern: searchText, Options: "i"}}},
				bson.M{"projectMetadata.latest.exhibitionTitle": bson.M{"$regex": primitive.Regex{Pattern: searchText, Options: "i"}}},
				bson.M{"projectMetadata.latest.title": bson.M{"$regex": primitive.Regex{Pattern: searchText, Options: "i"}}},
			},
		}},

		// group to generate the follow two items:
		// 1. a list of ids
		// 2. a map of asset id and the project information of an asset
		{"$group": bson.M{
			"_id": nil,
			"ids": bson.M{
				"$addToSet": "$id",
			},
			"assets": bson.M{"$push": bson.M{"k": "$id", "v": "$projectMetadata.latest"}},
		}},

		{"$addFields": bson.M{"assets": bson.M{"$arrayToObject": "$assets"}}},
	}

	assetCursor, err := s.assetCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var assetAggregation struct {
		IDs    []string
		Assets map[string]ProjectMetadata
	}

	for assetCursor.Next(ctx) {
		if err := assetCursor.Decode(&assetAggregation); err != nil {
			return nil, err
		}
	}

	tokens := make([]DetailedToken, 0)
	if len(assetAggregation.IDs) == 0 {
		return tokens, nil
	}

	tokenCursor, err := s.tokenCollection.Find(ctx, bson.M{"assetID": bson.M{"$in": assetAggregation.IDs}}, options.Find().SetLimit(size).SetSkip(offset))
	if err != nil {
		return nil, err
	}

	for tokenCursor.Next(ctx) {
		t := DetailedToken{}

		if err := tokenCursor.Decode(&t); err != nil {
			return nil, err
		}

		t.ProjectMetadata.Latest = assetAggregation.Assets[t.AssetID]
		tokens = append(tokens, t)
	}

	return tokens, nil
}

// GetIdentity returns an identity of an account
func (s *MongodbIndexerStore) GetIdentity(ctx context.Context, accountNumber string) (AccountIdentity, error) {
	var identity AccountIdentity

	r := s.identityCollection.FindOne(ctx,
		bson.M{"accountNumber": accountNumber},
	)
	if err := r.Err(); err != nil {
		if err != mongo.ErrNoDocuments {
			return identity, err
		}
	} else {
		if err := r.Decode(&identity); err != nil {
			return identity, err
		}
	}

	if identity.Name == "" {
		// fallback to check ff identities if not found
		r := s.ffIdentityCollection.FindOne(ctx,
			bson.M{"accountNumber": accountNumber},
		)
		if err := r.Err(); err != nil {
			if err == mongo.ErrNoDocuments {
				return identity, nil
			}
		} else {
			if err := r.Decode(&identity); err != nil {
				return identity, err
			}
		}
	}

	return identity, nil
}

// GetIdentities returns a list of identities by a list of account numbers
func (s *MongodbIndexerStore) GetIdentities(ctx context.Context, accountNumbers []string) (map[string]AccountIdentity, error) {
	identities := map[string]AccountIdentity{}

	{
		c, err := s.identityCollection.Find(ctx,
			bson.M{"accountNumber": bson.M{"$in": accountNumbers}},
		)
		if err != nil {
			return identities, err
		}

		for c.Next(ctx) {
			var identity AccountIdentity
			if err := c.Decode(&identity); err != nil {
				return identities, err
			}
			identities[identity.AccountNumber] = identity
		}
	}

	// FIXME: this is a quick fix and do not de-dup for accounts already found in previous query
	{
		c, err := s.ffIdentityCollection.Find(ctx,
			bson.M{"accountNumber": bson.M{"$in": accountNumbers}},
		)
		if err != nil {
			return identities, err
		}
		for c.Next(ctx) {
			var identity AccountIdentity
			if err := c.Decode(&identity); err != nil {
				return identities, err
			}
			// update identity for the with FF identity if the blockchain identity is not found
			if id, ok := identities[identity.AccountNumber]; !ok || id.Name == "" {
				identities[identity.AccountNumber] = identity
			}
		}
	}

	return identities, nil
}

// IndexIdentity saves an identity into indexer store
func (s *MongodbIndexerStore) IndexIdentity(ctx context.Context, identity AccountIdentity) error {
	identity.LastUpdatedTime = time.Now()

	r, err := s.identityCollection.UpdateOne(ctx,
		bson.M{"accountNumber": identity.AccountNumber},
		bson.M{"$set": identity},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return err
	}

	if r.MatchedCount == 0 && r.UpsertedCount == 0 {
		log.WarnWithContext(ctx, "identity is not added or updated", zap.String("account_number", identity.AccountNumber))
	}

	return nil
}

// IndexAccount indexes the account by inputs
func (s *MongodbIndexerStore) IndexAccount(ctx context.Context, account Account) error {
	if account.LastUpdatedTime.IsZero() {
		account.LastUpdatedTime = time.Now()
	}

	r, err := s.accountCollection.UpdateOne(ctx,
		bson.M{"account": account.Account},
		bson.M{"$set": account},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return err
	}

	if r.MatchedCount == 0 && r.UpsertedCount == 0 {
		log.WarnWithContext(ctx, "account is not added or updated", zap.String("account", account.Account))
	}

	return nil
}

// IndexAccountTokens indexes the account tokens by inputs
func (s *MongodbIndexerStore) IndexAccountTokens(ctx context.Context, owner string, accountTokens []AccountToken) error {
	margin := 15 * time.Second
	for _, accountToken := range accountTokens {
		log.Debug("update account token", zap.String("indexID", accountToken.IndexID), zap.Any("accountToken", accountToken))
		r, err := s.accountTokenCollection.UpdateOne(ctx,
			bson.M{"indexID": accountToken.IndexID, "ownerAccount": owner, "lastActivityTime": bson.M{"$lt": accountToken.LastActivityTime.Add(-margin)}},
			bson.M{"$set": accountToken},
			options.Update().SetUpsert(true),
		)

		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				// when a duplicated error happens, it means the account token
				// is in a state which is better than current event.
				log.WarnWithContext(ctx, "account token is in a future state", zap.String("indexID", accountToken.IndexID))
				continue
			}
			log.ErrorWithContext(ctx, errors.New("cannot index account token"), zap.String("indexID", accountToken.IndexID), zap.String("owner", owner), zap.Error(err))
			return err
		}
		if r.MatchedCount == 0 && r.UpsertedCount == 0 {
			// TODO: not sure when will this happen. Figure this our later
			log.WarnWithContext(ctx, "account token is not added or updated",
				zap.String("ownerAccount", owner), zap.String("indexID", accountToken.IndexID))
		}
	}

	return nil
}

// GetAccount returns an account by a given address
func (s *MongodbIndexerStore) GetAccount(ctx context.Context, owner string) (Account, error) {
	var account Account

	r := s.accountCollection.FindOne(ctx,
		bson.M{"account": owner},
	)
	if err := r.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return account, nil
		}
		return account, err
	}

	if err := r.Decode(&account); err != nil {
		return account, err
	}

	return account, nil
}

// GetLatestActivityTimeByIndexIDs returns a list of latest value of lastActivityTime for account tokens groups by indexID
func (s *MongodbIndexerStore) GetLatestActivityTimeByIndexIDs(ctx context.Context, indexIDs []string) (map[string]time.Time, error) {
	accountTokenLatestActivityTimes := map[string]time.Time{}

	c, err := s.accountTokenCollection.Aggregate(ctx, []bson.M{
		{"$match": bson.M{"indexID": bson.M{"$in": indexIDs}}},
		{"$sort": bson.D{{Key: "lastActivityTime", Value: 1}}},
		{
			"$group": bson.M{
				"_id":              "$indexID",
				"indexID":          bson.M{"$last": "$indexID"},
				"lastActivityTime": bson.M{"$last": "$lastActivityTime"},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	for c.Next(ctx) {
		var token AccountToken
		if err := c.Decode(&token); err != nil {
			return nil, err
		}

		accountTokenLatestActivityTimes[token.IndexID] = token.LastActivityTime
	}

	return accountTokenLatestActivityTimes, nil
}

// UpdateAccountTokenOwners updates all account owners for a specific token
func (s *MongodbIndexerStore) UpdateAccountTokenOwners(ctx context.Context, indexID string, ownerBalances []OwnerBalance) error {
	ownerList := make([]string, 0, len(ownerBalances))
	now := time.Now()

	tokenResult := s.tokenCollection.FindOne(ctx, bson.M{"indexID": indexID})
	if err := tokenResult.Err(); err != nil {
		return err
	}

	var token Token
	if err := tokenResult.Decode(&token); err != nil {
		return err
	}

	for _, ownerBalance := range ownerBalances {
		tokenUpdate := AccountToken{
			BaseTokenInfo:     token.BaseTokenInfo,
			IndexID:           indexID,
			OwnerAccount:      ownerBalance.Address,
			Balance:           ownerBalance.Balance,
			LastActivityTime:  ownerBalance.LastTime,
			LastRefreshedTime: now,
		}

		_, err := s.accountTokenCollection.UpdateOne(ctx,
			bson.M{"indexID": indexID, "ownerAccount": ownerBalance.Address},
			bson.M{"$set": tokenUpdate},
			options.Update().SetUpsert(true),
		)
		if err != nil {
			log.ErrorWithContext(ctx, errors.New("could not update balance "), zap.String("indexID", indexID), zap.String("owner", ownerBalance.Address), zap.Error(err))
			continue
		}

		ownerList = append(ownerList, ownerBalance.Address)
	}

	_, err := s.accountTokenCollection.UpdateMany(ctx, bson.M{
		"indexID": bson.M{"$eq": indexID}, "ownerAccount": bson.M{"$nin": ownerList},
	}, bson.M{
		"$set": bson.M{
			"balance":           0,
			"lastRefreshedTime": now,
		},
	})

	return err
}

// IndexDemoTokens copies  existent tokens in the db but change the blockchain name to "demo" and modify owner
// Before indexing new demo tokens, all of the old ones of the same owner need to be deleted.
func (s *MongodbIndexerStore) IndexDemoTokens(ctx context.Context, owner string, indexIDs []string) error {
	if err := s.DeleteDemoTokens(ctx, owner); err != nil {
		return err
	}

	for _, indexID := range indexIDs {
		demoIndexID := DemoTokenPrefix(indexID)

		r := s.tokenCollection.FindOne(ctx, bson.M{"isDemo": true, "indexID": demoIndexID})
		if err := r.Err(); err != nil {
			if err == mongo.ErrNoDocuments {
				// Create a new demo token if it does not exist
				r := s.tokenCollection.FindOne(ctx, bson.M{"indexID": indexID})
				if err := r.Err(); err != nil {
					return err
				}

				var token Token
				if err := r.Decode(&token); err != nil {
					return err
				}

				token.IndexID = demoIndexID
				token.IsDemo = true
				token.OwnersArray = []string{owner}
				token.Owners[owner] = 1
				if _, err := s.tokenCollection.InsertOne(ctx, token); err != nil {
					log.ErrorWithContext(ctx, errors.New("error while inserting demo tokens"), zap.String("indexID", demoIndexID), zap.Error(err))
					return err
				}
				log.Debug("demo token is indexed", zap.String("indexID", demoIndexID))
			} else {
				log.ErrorWithContext(ctx, errors.New("error while finding demoIndexID in the database"), zap.String("demoIndexID", demoIndexID), zap.Error(err))
				return err
			}
		} else {
			// Add a new owner to the demo tokens which already exist
			if _, err := s.tokenCollection.UpdateOne(ctx,
				bson.M{"isDemo": true, "indexID": demoIndexID, "ownersArray": bson.M{"$nin": bson.A{owner}}},
				bson.M{
					"$push": bson.M{"ownersArray": owner},
					"$set":  bson.M{fmt.Sprintf("owners.%s", owner): int64(1)},
				}); err != nil {
				return err
			}
			log.Debug("demo token is updated", zap.String("indexID", demoIndexID))
		}
	}

	return nil
}

// DeleteDemoTokens deletes demo tokens which exclusively belong to an owner and updates demo tokens if they are related to other owners
func (s *MongodbIndexerStore) DeleteDemoTokens(ctx context.Context, owner string) error {
	// delete demo tokens that exclusively belong to the owner
	_, err := s.tokenCollection.DeleteMany(ctx, bson.M{
		"isDemo":      true,
		"ownersArray": bson.M{"$eq": bson.A{owner}},
		"indexID":     bson.M{"$regex": "^demo"}},
	)

	if err != nil {
		return err
	}

	// remove the owners in the demo tokens if the tokens belong to other owners
	_, err = s.tokenCollection.UpdateMany(ctx,
		bson.M{
			"isDemo":      true,
			"ownersArray": bson.M{"$in": bson.A{owner}},
			"indexID":     bson.M{"$regex": "^demo"},
		},
		bson.M{
			"$pull": bson.M{
				"ownersArray": owner,
			},
			"$unset": bson.M{
				fmt.Sprintf("owners.%s", owner): bson.M{"$gte": 1},
			},
		})
	if err != nil {
		return err
	}

	return nil
}

// UpdateOwnerForFungibleToken adds a new owner to a fungible token
func (s *MongodbIndexerStore) UpdateOwnerForFungibleToken(ctx context.Context, indexID string, lockedTime time.Time, to string, total int64) error {
	r, err := s.tokenCollection.UpdateOne(ctx,
		bson.M{
			"indexID":           indexID,
			"lastRefreshedTime": lockedTime,
		},
		bson.M{
			"$addToSet": bson.M{"ownersArray": to},
			"$set":      bson.M{"owners." + to: total},
		},
	)

	if err != nil {
		return err
	}

	if r.MatchedCount == 0 {
		return ErrNoRecordUpdated
	}

	return nil
}

func (s *MongodbIndexerStore) GetTokenByIndexID(ctx context.Context, indexID string) (*Token, error) {
	tokens, err := s.GetTokensByIndexIDs(ctx, []string{indexID})
	if err != nil {
		return nil, err
	}

	if len(tokens) == 0 {
		return nil, nil
	}

	return &tokens[0], err
}

// MarkAccountTokenChanged sets the lastRefreshedTime to now
func (s *MongodbIndexerStore) MarkAccountTokenChanged(ctx context.Context, indexIDs []string) error {
	_, err := s.accountTokenCollection.UpdateMany(ctx, bson.M{
		"indexID": bson.M{"$in": indexIDs},
	}, bson.M{
		"$set": bson.M{"lastRefreshedTime": time.Now()},
	})

	if err != nil {
		log.ErrorWithContext(ctx, errors.New("cannot update account tokens"), zap.Error(err), zap.Any("indexIDs", indexIDs))
	}

	return err
}

// GetDetailedAccountTokensByOwners returns a list of DetailedToken by owner
func (s *MongodbIndexerStore) GetDetailedAccountTokensByOwners(ctx context.Context, owner []string, filterParameter FilterParameter, lastUpdatedAt time.Time, sortBy string, offset, size int64) ([]DetailedTokenV2, error) {
	var sortKey string
	if sortBy == "lastActivityTime" {
		sortKey = sortBy
	} else {
		sortKey = "lastRefreshedTime"
	}

	filter := bson.M{
		"ownerAccount":      bson.M{"$in": owner},
		"lastRefreshedTime": bson.M{"$gte": lastUpdatedAt},
	}

	findOptions := options.Find().SetSort(bson.D{{Key: sortKey, Value: -1}, {Key: "_id", Value: -1}}).SetLimit(QueryPageSize)

	accountTokens := []AccountToken{}
	detailTokens := []DetailedTokenV2{}
	page := 0
	for {
		queryOffset := int64(page * QueryPageSize)
		expectedSize := size
		// need to do manually offset for source since data was filtered
		if filterParameter.Source == "" {
			queryOffset = offset + queryOffset
		} else {
			expectedSize = offset + expectedSize
		}

		findOptions.SetSkip(queryOffset)
		cursor, err := s.accountTokenCollection.Find(ctx, filter, findOptions)
		if err != nil {
			return nil, err
		}

		indexIDs := make([]string, 0)
		for cursor.Next(ctx) {
			var token AccountToken

			if err := cursor.Decode(&token); err != nil {
				return nil, err
			}

			indexIDs = append(indexIDs, token.IndexID)
			accountTokens = append(accountTokens, token)
		}

		cursor.Close(ctx)

		if len(indexIDs) == 0 {
			break
		}

		filterParameter.IDs = indexIDs
		tokens, err := s.GetDetailedTokensV2(ctx, filterParameter, 0, int64(len(indexIDs)))

		if err != nil {
			return nil, err
		}

		detailTokens = append(detailTokens, tokens...)

		if len(detailTokens) >= int(expectedSize) {
			break
		}

		page++
	}

	detailedTokenMap := map[string]DetailedTokenV2{}
	results := []DetailedTokenV2{}

	for _, t := range detailTokens {
		detailedTokenMap[t.IndexID] = t
	}

	skipped := 0
	for _, a := range accountTokens {
		token, ok := detailedTokenMap[a.IndexID]

		if !ok {
			continue
		}

		token.Balance = a.Balance
		token.Owner = a.OwnerAccount
		token.LastRefreshedTime = a.LastRefreshedTime
		if !a.LastActivityTime.IsZero() {
			token.LastActivityTime = a.LastActivityTime
		}

		if filterParameter.Source != "" && skipped < int(offset) {
			skipped++
			continue
		}

		results = append(results, token)

		if len(results) == int(size) {
			break
		}
	}

	return results, nil
}

// CountDetailedAccountTokensByOwner count the number of DetailedToken by owner
func (s *MongodbIndexerStore) CountDetailedAccountTokensByOwner(ctx context.Context, owner string) (int64, error) {
	filter := bson.M{
		"ownerAccount": owner,
		"balance":      bson.M{"$gt": 0},
	}

	return s.accountTokenCollection.CountDocuments(ctx, filter)
}

// GetDetailedTokensV2 returns a list of tokens information based on ids
func (s *MongodbIndexerStore) GetDetailedTokensV2(ctx context.Context, filterParameter FilterParameter, offset, size int64) ([]DetailedTokenV2, error) {
	tokens := []DetailedTokenV2{}

	log.Debug("GetDetailedTokensV2",
		zap.Any("filterParameter", filterParameter),
		zap.Int64("offset", offset),
		zap.Int64("size", size))
	startTime := time.Now()
	if length := len(filterParameter.IDs); length > 0 {
		queryIDsEnd := int(offset + size)
		if queryIDsEnd > length {
			queryIDsEnd = length
		}
		queryIDs := filterParameter.IDs[offset:queryIDsEnd]
		queryLen := len(queryIDs)
		for i := 0; i < getPageCounts(queryLen, QueryPageSize); i++ {
			log.Debug("doc page", zap.Int("page", i))
			start := i * QueryPageSize
			end := (i + 1) * QueryPageSize
			if end > queryLen {
				end = queryLen
			}

			pagedTokens, err := s.getDetailedTokensV2InView(ctx,
				FilterParameter{
					IDs:            queryIDs[start:end],
					Source:         filterParameter.Source,
					BurnedIncluded: filterParameter.BurnedIncluded,
				},
				0,
				int64(end-start))
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, pagedTokens...)
		}
	} else {
		return s.getDetailedTokensV2InView(ctx, filterParameter, offset, size)
	}
	log.Debug("GetDetailedTokensV2 End", zap.Duration("queryTime", time.Since(startTime)))

	return tokens, nil
}

// getDetailedTokensV2InView returns detail tokens from mongodb custom view
func (s *MongodbIndexerStore) getDetailedTokensV2InView(ctx context.Context, filterParameter FilterParameter, offset, size int64) ([]DetailedTokenV2, error) {
	tokens := []DetailedTokenV2{}
	match := bson.M{"indexID": bson.M{"$in": filterParameter.IDs}}
	if !filterParameter.BurnedIncluded {
		match["burned"] = bson.M{"$ne": true}
	}

	pipelines := []bson.M{
		{"$match": match},
		{"$addFields": bson.M{"__order": bson.M{"$indexOfArray": bson.A{filterParameter.IDs, "$indexID"}}}},
		{"$sort": bson.M{"__order": 1}},
	}

	if filterParameter.Source != "" {
		pipelines = append(pipelines, bson.M{"$match": bson.M{"asset.source": filterParameter.Source}})
	}

	pipelines = append(pipelines,
		bson.M{"$skip": offset},
		bson.M{"$limit": size},
	)

	cursor, err := s.tokenAssetCollection.Aggregate(ctx, pipelines)

	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

// GetDetailedToken returns a token information based on indexID
func (s *MongodbIndexerStore) GetDetailedToken(ctx context.Context, indexID string, burnedIncluded bool) (DetailedToken, error) {
	filterParameter := FilterParameter{
		IDs:            []string{indexID},
		BurnedIncluded: burnedIncluded,
	}

	detailedTokens, err := s.GetDetailedTokens(ctx, filterParameter, 0, 1)
	if err != nil {
		return DetailedToken{}, err
	}

	if len(detailedTokens) == 0 {
		return DetailedToken{}, fmt.Errorf("token not found")
	}

	return detailedTokens[0], nil
}

// GetTotalBalanceOfOwnerAccounts sum balance of ownerAccounts
func (s *MongodbIndexerStore) GetTotalBalanceOfOwnerAccounts(ctx context.Context, addresses []string) (int, error) {
	cursor, err := s.accountTokenCollection.Aggregate(ctx, []bson.M{
		{"$match": bson.M{"ownerAccount": bson.M{"$in": addresses}}},
		{"$group": bson.M{"_id": nil, "total": bson.M{"$sum": "$balance"}}},
	})
	if err != nil {
		return 0, err
	}

	defer cursor.Close(ctx)

	var totalBalance TotalBalance

	if cursor.Next(ctx) {
		if err := cursor.Decode(&totalBalance); err != nil {
			return 0, err
		}
	}

	return totalBalance.Total, nil
}

// GetNullProvenanceTokensByIndexIDs returns indexIDs that have null provenance
func (s *MongodbIndexerStore) GetNullProvenanceTokensByIndexIDs(ctx context.Context, indexIDs []string) ([]string, error) {
	var nullProvenanceIDs []string
	var tokens []Token

	c, err := s.tokenCollection.Find(ctx, bson.M{
		"indexID":    bson.M{"$in": indexIDs},
		"fungible":   false,
		"provenance": nil,
	})

	if err != nil {
		return nil, err
	}

	if err := c.All(ctx, &tokens); err != nil {
		return nil, err
	}

	for _, token := range tokens {
		nullProvenanceIDs = append(nullProvenanceIDs, token.IndexID)
	}

	return nullProvenanceIDs, nil
}

// GetOwnerAccountsByIndexIDs Get Owner Accounts By IndexIDs
func (s *MongodbIndexerStore) GetOwnerAccountsByIndexIDs(ctx context.Context, indexIDs []string) ([]string, error) {
	filter := bson.M{
		"indexID": bson.M{
			"$in": indexIDs,
		},
	}

	cursor, err := s.accountTokenCollection.Find(
		ctx,
		filter,
	)
	if err != nil {
		return nil, err
	}

	var owners []string

	for cursor.Next(ctx) {
		var accountToken AccountToken

		if err := cursor.Decode(&accountToken); err != nil {
			return nil, err
		}

		owners = append(owners, accountToken.OwnerAccount)
	}

	return owners, nil
}

// CheckAddressOwnTokenByCriteria returns true if address owns token
func (s *MongodbIndexerStore) CheckAddressOwnTokenByCriteria(ctx context.Context, address string, criteria Criteria) (bool, error) {
	if criteria.IndexID != "" {
		return s.checkAddressOwnTokenHasIndexID(ctx, address, criteria.IndexID)
	}

	if criteria.Source != "" {
		return s.checkAddressOwnTokenInSource(ctx, address, criteria.Source)
	}

	return false, nil
}

// checkAddressOwnTokenHasIndexID returns true if address owns token
func (s *MongodbIndexerStore) checkAddressOwnTokenHasIndexID(ctx context.Context, address string, indexID string) (bool, error) {
	var accountToken AccountToken

	if err := s.accountTokenCollection.FindOne(ctx, bson.M{
		"ownerAccount": address,
		"indexID":      indexID,
		"balance": bson.M{
			"$gt": 0,
		},
	}).Decode(&accountToken); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// checkAddressOwnTokenInSource returns true if address owns token in source
func (s *MongodbIndexerStore) checkAddressOwnTokenInSource(ctx context.Context, address string, source string) (bool, error) {
	var indexIDs []string

	cursor, err := s.accountTokenCollection.Find(ctx, bson.M{
		"ownerAccount": address,
		"balance": bson.M{
			"$gt": 0,
		},
	})
	if err != nil {
		return false, err
	}

	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		t := AccountToken{}

		if err := cursor.Decode(&t); err != nil {
			return false, err
		}

		indexIDs = append(indexIDs, t.IndexID)
	}

	if len(indexIDs) == 0 {
		return false, nil
	}

	// check if any token has source
	for idxRange := range gopart.Partition(len(indexIDs), 25) {
		var token Token

		err = s.tokenCollection.FindOne(ctx, bson.M{
			"indexID": bson.M{
				"$in": indexIDs[idxRange.Low:idxRange.High],
			},
			"source": source,
		}).Decode(&token)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				continue
			}

			return false, err
		}

		return true, nil
	}

	return false, nil
}

// GetOwnersByBlockchainContracts returns owners by blockchain and contract
func (s *MongodbIndexerStore) GetOwnersByBlockchainContracts(ctx context.Context, blockchainContracts map[string][]string) ([]string, error) {
	var or []bson.M

	for k, v := range blockchainContracts {
		or = append(or, bson.M{
			"blockchain":      bson.M{"$eq": k},
			"contractAddress": bson.M{"$in": v},
		})
	}

	filter := bson.M{"$or": or}

	cursor, err := s.accountTokenCollection.Find(
		ctx,
		filter,
	)
	if err != nil {
		return nil, err
	}

	var owners []string
	temp := make(map[string]interface{})

	for cursor.Next(ctx) {
		var accountToken AccountToken

		if err := cursor.Decode(&accountToken); err != nil {
			return nil, err
		}

		_, ok := temp[accountToken.OwnerAccount]
		if !ok {
			temp[accountToken.OwnerAccount] = nil
			owners = append(owners, accountToken.OwnerAccount)
		}
	}

	return owners, nil
}

// IndexCollection index new collection
func (s *MongodbIndexerStore) IndexCollection(ctx context.Context, collection Collection) error {
	if collection.LastUpdatedTime.IsZero() {
		collection.LastUpdatedTime = time.Now()
	}

	r, err := s.collectionsCollection.UpdateOne(ctx,
		bson.M{"id": collection.ID},
		bson.M{"$set": collection},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return err
	}

	if r.MatchedCount == 0 && r.UpsertedCount == 0 {
		log.WarnWithContext(ctx, "collection is not added or updated", zap.String("collection", collection.ID))
	}

	return nil
}

// IndexCollectionAsset index new collection tokens
func (s *MongodbIndexerStore) IndexCollectionAsset(ctx context.Context, collectionID string, collectionAssets []CollectionAsset) error {
	for _, c := range collectionAssets {
		log.Debug("update collection asset", zap.String("asset", c.TokenIndexID), zap.Any("accountToken", c))
		r, err := s.collectionAssetsCollection.UpdateOne(ctx,
			bson.M{"collectionID": c.CollectionID, "tokenIndexID": c.TokenIndexID},
			bson.M{"$set": c},
			options.Update().SetUpsert(true),
		)

		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				// when a duplicated error happens, it means the account token
				// is in a state which is better than current event.
				log.WarnWithContext(ctx, "collection token is in a future state", zap.String("indexID", c.TokenIndexID))
				continue
			}
			log.ErrorWithContext(ctx, errors.New("cannot index collection token"), zap.String("indexID", c.TokenIndexID), zap.String("collectionID", collectionID), zap.Error(err))
			return err
		}
		if r.MatchedCount == 0 && r.UpsertedCount == 0 {
			log.WarnWithContext(ctx, "collection token is not added or updated",
				zap.String("collectionID", collectionID), zap.String("indexID", c.TokenIndexID))
		}
	}

	return nil
}

// DeleteDeprecatedCollectionAsset removes old tokens not belong the collection anymore
func (s *MongodbIndexerStore) DeleteDeprecatedCollectionAsset(ctx context.Context, collectionID, runID string) error {
	_, err := s.collectionAssetsCollection.DeleteMany(ctx,
		bson.M{"collectionID": collectionID, "runID": bson.M{"$ne": runID}},
	)

	return err
}

// DeleteCollection removes the collection and related assets
func (s *MongodbIndexerStore) DeleteCollection(ctx context.Context, collectionID string) error {
	// Start a session to ensure atomicity
	session, err := s.mongoClient.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	// Run the delete operation in a transaction to ensure atomicity
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		if _, err := s.collectionsCollection.DeleteOne(sessCtx,
			bson.M{"id": collectionID},
		); err != nil {
			return nil, err
		}

		if _, err := s.collectionAssetsCollection.DeleteMany(sessCtx,
			bson.M{"collectionID": collectionID},
		); err != nil {
			return nil, err
		}

		return nil, nil
	})

	return err
}

// ReplaceCollectionCreator updates all occurrences of oldCreator to newCreator
// in the creators array of matching collections and updates the lastUpdatedTime.
func (s *MongodbIndexerStore) ReplaceCollectionCreator(ctx context.Context, oldCreator, newCreator string) error {
	// Define the filter to find documents where creators contains oldCreator
	filter := bson.M{"creators": oldCreator}

	// Define the update to replace oldCreator with newCreator in creators array
	// and set lastUpdatedTime to current time
	update := bson.M{
		"$set": bson.M{
			"creators.$[elem]": newCreator,
			"lastUpdatedTime":  time.Now(),
		},
	}

	// Define array filters to identify elements in creators array to update
	opts := options.Update().SetArrayFilters(options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"elem": oldCreator},
		},
	})

	// Perform the update operation on all matching documents
	_, err := s.collectionsCollection.UpdateMany(ctx, filter, update, opts)
	return err
}

// UpdateCollectionCreators sync the creators to the creators array
// of the collection identified by collectionID, if not already present, and updates lastUpdatedTime.
func (s *MongodbIndexerStore) UpdateCollectionCreators(ctx context.Context, collectionID string, creators []string) error {
	// Define the update operation
	update := bson.M{
		"$set": bson.M{
			"creators":        creators,
			"lastUpdatedTime": time.Now(),
		},
	}

	// Perform the update on the collection with the specified ID
	_, err := s.collectionsCollection.UpdateOne(ctx, bson.M{"id": collectionID}, update)
	return err
}

// GetCollectionLastUpdateTimeeForOwner returns collection last refreshed time for an owner
func (s *MongodbIndexerStore) GetCollectionLastUpdatedTimeForCreator(ctx context.Context, creator string) (time.Time, error) {
	findOptions := options.FindOne().SetSort(bson.D{{Key: "lastUpdatedTime", Value: -1}})
	r := s.collectionsCollection.FindOne(ctx, bson.M{
		"creator": creator,
	}, findOptions)

	if err := r.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			// If a token is not found, return zero time
			return time.Time{}, nil
		}

		return time.Time{}, err
	}

	var collection Collection
	if err := r.Decode(&collection); err != nil {
		return time.Time{}, err
	}

	return collection.LastUpdatedTime, nil
}

// GetCollectionLastUpdatedTime returns collection last refreshed time by collectionID
func (s *MongodbIndexerStore) GetCollectionLastUpdatedTime(ctx context.Context, collectionID string) (time.Time, error) {
	r := s.collectionsCollection.FindOne(ctx, bson.M{
		"id": collectionID,
	})

	if err := r.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			// If a token is not found, return zero time
			return time.Time{}, nil
		}

		return time.Time{}, err
	}

	var collection Collection
	if err := r.Decode(&collection); err != nil {
		return time.Time{}, err
	}

	return collection.LastUpdatedTime, nil
}

// GetCollectionByID returns the collection by given id
func (s *MongodbIndexerStore) GetCollectionByID(ctx context.Context, id string) (*Collection, error) {
	r := s.collectionsCollection.FindOne(ctx, bson.M{
		"id": id,
	})

	if err := r.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		return nil, err
	}

	var collection Collection
	if err := r.Decode(&collection); err != nil {
		return nil, err
	}

	return &collection, nil
}

// GetCollectionsByOwners returns list of collections for owners
func (s *MongodbIndexerStore) GetCollectionsByCreators(ctx context.Context, creators []string, offset, size int64) ([]Collection, error) {
	filter := bson.M{
		"creators": bson.M{"$in": creators},
	}
	findOptions := options.Find().SetSort(bson.D{{Key: "lastActivityTime", Value: -1}, {Key: "_id", Value: -1}})

	collections := []Collection{}
	page := 0
	for {
		isLastPage := false

		queryOffset := offset + int64(page*QueryPageSize)
		queryLimit := int64(QueryPageSize)

		if queryOffset+QueryPageSize > offset+size {
			queryLimit = offset + size - queryOffset
			isLastPage = true
		}

		findOptions.SetSkip(queryOffset)
		findOptions.SetLimit(queryLimit)
		cursor, err := s.collectionsCollection.Find(ctx, filter, findOptions)
		if err != nil {
			return nil, err
		}

		for cursor.Next(ctx) {
			var collection Collection

			if err := cursor.Decode(&collection); err != nil {
				return nil, err
			}

			collections = append(collections, collection)
		}

		cursor.Close(ctx)

		if len(collections) < int(queryLimit) || isLastPage {
			break
		}

		page++
	}

	return collections, nil
}

// GetDetailedTokensByCollectionID returns list of tokens by the collectionID
func (s *MongodbIndexerStore) GetDetailedTokensByCollectionID(ctx context.Context, collectionID string, sortBy string, offset, size int64) ([]DetailedTokenV2, error) {
	var sort bson.D
	if sortBy == "lastActivityTime" {
		sort = bson.D{{Key: "lastActivityTime", Value: -1}, {Key: "_id", Value: -1}}
	} else {
		sort = bson.D{{Key: "edition", Value: 1}, {Key: "_id", Value: -1}}
	}

	var tokens []CollectionAsset
	findOptions := options.Find().
		SetSort(sort).
		SetLimit(size).
		SetSkip(offset)
	c, err := s.collectionAssetsCollection.Find(ctx, bson.M{
		"collectionID": collectionID,
	}, findOptions)

	if err != nil {
		return nil, err
	}

	if err := c.All(ctx, &tokens); err != nil {
		return nil, err
	}

	indexIDs := []string{}
	for _, token := range tokens {
		indexIDs = append(indexIDs, token.TokenIndexID)
	}

	if len(indexIDs) == 0 {
		return []DetailedTokenV2{}, nil
	}

	filterParameter := FilterParameter{IDs: indexIDs}
	return s.GetDetailedTokensV2(ctx, filterParameter, 0, int64(len(indexIDs)))
}

// fields that may not appear in metadata or values maps
var reserved = map[string]struct{}{
	"_id":       {},
	"id":        {},
	"metadata":  {},
	"shares":    {},
	"timestamp": {},
	"values":    {},
	"uniqueID":  {},
}

// WriteTimeSeriesData - validate and store a time series record
func (s *MongodbIndexerStore) WriteTimeSeriesData(
	ctx context.Context,
	records []GenericSalesTimeSeries,
) error {
	insertsMap := make(map[string]interface{})
	for _, r := range records {
		timestamp, err := time.Parse(time.RFC3339Nano, r.Timestamp)
		if nil != err {
			log.ErrorWithContext(ctx, errors.New("error parsing timestamp"),
				zap.String("timestamp", r.Timestamp),
				zap.Error(err),
			)
			return err
		}

		var saleTokenUniqueIDs []string
		for k, v := range r.Metadata {
			// ensure no reserved fields in metadata
			if _, ok := reserved[k]; ok {
				log.WarnWithContext(ctx,
					"reserved metadata field name",
					zap.String("key", k),
					zap.Any("value", v),
				)
				return fmt.Errorf("reserved field name: metadata.%s", k)
			}
			if k == "bundleTokenInfo" {
				interfaces, ok := v.([]interface{})
				if !ok {
					return fmt.Errorf("wrong format: metadata.%s is not a slice", k)
				}
				for i, inter := range interfaces {
					m, ok := inter.(map[string]interface{})
					if !ok {
						return fmt.Errorf("wrong format: metadata.%s[%d] is not a map[string]interface{}", k, i)
					}
					var ca string
					if m["contractAddress"] != nil {
						ca, ok = m["contractAddress"].(string)
						if !ok {
							return fmt.Errorf("wrong format: metadata.%s[%d][%s] is not a string", k, i, `"contractAddress"`)
						}
					}
					saleTokenUniqueIDs = append(saleTokenUniqueIDs,
						fmt.Sprintf("%s-%s-%s",
							r.Metadata["blockchain"].(string),
							ca,
							m["tokenID"].(string)))
				}
			}
		}

		var transactionIDs []string
		txIDs, ok := r.Metadata["transactionIDs"].([]interface{})
		if !ok {
			return fmt.Errorf("wrong format: metadata.transactionIDs is not a slice")
		}

		for i, v := range txIDs {
			txID, ok := v.(string)
			if !ok {
				return fmt.Errorf("wrong format: metadata.transactionIDs[%d] is not a string", i)
			}
			transactionIDs = append(transactionIDs, txID)
		}

		sort.Strings(saleTokenUniqueIDs)
		sort.Strings(transactionIDs)

		uniqueID := HexSha1(fmt.Sprintf("%s|%s",
			strings.Join(transactionIDs, ","),
			strings.Join(saleTokenUniqueIDs, ","),
		))

		// skip duplicate
		if _, existed := insertsMap[uniqueID]; existed {
			continue
		}

		r.Metadata["uniqueID"] = uniqueID

		// root of the BSON document
		doc := bson.M{
			"timestamp": timestamp,
			"metadata":  r.Metadata,
		}

		// ensure no reserved fields in values and convert
		for k, v := range r.Values {
			if _, ok := reserved[k]; ok {
				log.WarnWithContext(ctx,
					"reserved values field name",
					zap.String("key", k),
					zap.String("value", v),
				)
				return fmt.Errorf("reserved field name: values.%s", k)
			}

			doc[k], err = primitive.ParseDecimal128(v)
			if err != nil {
				log.WarnWithContext(ctx,
					"invalid Decimal128 in values field",
					zap.String("key", k),
					zap.String("value", v),
					zap.Error(err),
				)
				return fmt.Errorf("Decimal128 error: %s on: values.%s = %q", err, k, v)
			}
		}

		// ensure no reserved fields in shares and convert
		sv := bson.M{}
		for k, v := range r.Shares {
			if _, ok := reserved[k]; ok {
				log.WarnWithContext(ctx,
					"reserved shares field name",
					zap.String("key", k),
					zap.String("value", v),
				)
				return fmt.Errorf("reserved field name: shares.%s", k)
			}

			sv[k], err = primitive.ParseDecimal128(v)
			if err != nil {
				log.WarnWithContext(ctx,
					"invalid Decimal128 in shares field",
					zap.String("key", k),
					zap.String("value", v),
					zap.Error(err),
				)
				return fmt.Errorf("Decimal128 error: %s on: shares.%s = %q", err, k, v)
			}
		}
		doc["shares"] = sv

		filter := bson.M{
			"metadata.uniqueID": uniqueID,
		}

		log.Debug("deleting documents",
			zap.String("uniqueID", uniqueID),
			zap.Error(err))

		result, err := s.salesTimeSeriesCollection.DeleteMany(ctx, filter)
		if err != nil {
			log.ErrorWithContext(ctx, errors.New("error deleting documents"),
				zap.String("uniqueID", uniqueID),
				zap.Error(err))
			return err
		}
		if result.DeletedCount > 0 {
			log.InfoWithContext(ctx, "deleted duplicated documents",
				zap.Int64("deletedCount", result.DeletedCount),
				zap.Any("record", r.Metadata))
		}
		insertsMap[uniqueID] = doc
	}

	var inserts []interface{}
	for _, doc := range insertsMap {
		inserts = append(inserts, doc)
	}

	if len(inserts) > 0 {
		_, err := s.salesTimeSeriesCollection.InsertMany(ctx, inserts)
		if err != nil {
			log.ErrorWithContext(ctx, errors.New("error inserting documents"), zap.Error(err))
			return err
		}
	}

	return nil
}

func (s *MongodbIndexerStore) WriteHistoricalExchangeRate(ctx context.Context, records []coinbase.HistoricalExchangeRate) error {
	var operations []mongo.WriteModel

	for _, r := range records {
		filter := bson.M{"timestamp": r.Time, "currencyPair": r.CurrencyPair}
		update := bson.M{"$set": bson.M{
			"timestamp":    r.Time,
			"price":        r.Open,
			"currencyPair": r.CurrencyPair,
		}}
		model := mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true)
		operations = append(operations, model)
	}

	if len(operations) > 0 {
		_, err := s.historicalExchangeRatesCollection.BulkWrite(ctx, operations)
		if err != nil {
			log.ErrorWithContext(ctx, errors.New("error in bulk write operation"), zap.Error(err))
			return err
		}
	}

	return nil
}

func (s *MongodbIndexerStore) GetHistoricalExchangeRate(ctx context.Context, filter HistoricalExchangeRateFilter) (ExchangeRate, error) {
	var closestExchangeRate ExchangeRate
	var lowerExchangeRate ExchangeRate
	var greaterExchangeRate ExchangeRate

	requestingTimestamp := filter.Timestamp.UTC()

	// Find the closest lower timestamp
	lowerFilterMap := bson.M{
		"currencyPair": filter.CurrencyPair,
		"timestamp": bson.M{
			"$lte": requestingTimestamp,
		},
	}
	lowerFindOptions := options.FindOne()
	lowerFindOptions.SetSort(bson.D{{Key: "timestamp", Value: -1}})

	err := s.historicalExchangeRatesCollection.FindOne(ctx, lowerFilterMap, lowerFindOptions).Decode(&lowerExchangeRate)
	if err != nil && err != mongo.ErrNoDocuments {
		return closestExchangeRate, err
	}

	if requestingTimestamp.Equal(lowerExchangeRate.Timestamp) {
		return lowerExchangeRate, nil
	}

	// Find the closest greater timestamp
	greaterFilterMap := bson.M{
		"currencyPair": filter.CurrencyPair,
		"timestamp": bson.M{
			"$gte": requestingTimestamp,
		},
	}
	greaterFindOptions := options.FindOne()
	greaterFindOptions.SetSort(bson.D{{Key: "timestamp", Value: 1}})

	err = s.historicalExchangeRatesCollection.FindOne(ctx, greaterFilterMap, greaterFindOptions).Decode(&greaterExchangeRate)
	if err != nil && err != mongo.ErrNoDocuments {
		return closestExchangeRate, err
	}

	// Compare the two results to find the closest timestamp
	if lowerExchangeRate.Timestamp.IsZero() {
		closestExchangeRate = greaterExchangeRate
	} else if greaterExchangeRate.Timestamp.IsZero() {
		closestExchangeRate = lowerExchangeRate
	} else {
		lowerDiff := requestingTimestamp.Sub(lowerExchangeRate.Timestamp)
		greaterDiff := greaterExchangeRate.Timestamp.Sub(requestingTimestamp)
		if lowerDiff <= greaterDiff {
			closestExchangeRate = lowerExchangeRate
		} else {
			closestExchangeRate = greaterExchangeRate
		}
	}

	return closestExchangeRate, nil
}

// SaleTimeSeriesDataExists - check if a sale time series data exists for a transaction hash and blockchain
func (s *MongodbIndexerStore) SaleTimeSeriesDataExists(ctx context.Context, txID, blockchain string) (bool, error) {
	count, err := s.salesTimeSeriesCollection.CountDocuments(ctx, bson.M{
		"metadata.transactionIDs": txID,
		"metadata.blockchain":     blockchain,
	})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *MongodbIndexerStore) GetSaleTimeSeriesData(ctx context.Context, filter SalesFilterParameter) ([]SaleTimeSeries, error) {
	var saleTimeSeries []SaleTimeSeries

	match := bson.M{}
	if len(filter.Addresses) > 0 {
		addressFilter := bson.A{}
		for _, a := range filter.Addresses {
			addressFilter = append(addressFilter, bson.M{fmt.Sprintf("shares.%s", a): bson.M{"$nin": bson.A{nil, ""}}})
		}
		match["$or"] = addressFilter
	}
	if filter.Marketplace != "" {
		match["metadata.marketplace"] = filter.Marketplace
	}

	timestampFilter := bson.M{}
	if filter.From != nil {
		timestampFilter["$gte"] = filter.From
	}
	if filter.To != nil {
		timestampFilter["$lte"] = filter.To
	}
	if len(timestampFilter) > 0 {
		match["timestamp"] = timestampFilter
	}

	sort := -1
	if filter.SortASC {
		sort = 1
	}
	pipelines := []bson.M{
		{"$match": match},
		{"$sort": bson.D{{Key: "timestamp", Value: sort}, {Key: "_id", Value: sort}}},
	}

	pipelines = append(pipelines,
		bson.M{"$skip": filter.Offset},
		bson.M{"$limit": filter.Limit},
	)

	cursor, err := s.salesTimeSeriesCollection.Aggregate(ctx, pipelines)

	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &saleTimeSeries); err != nil {
		return nil, err
	}

	return saleTimeSeries, nil
}

// AggregateSaleRevenues - get sale revenue group by currency belong to an address
func (s *MongodbIndexerStore) AggregateSaleRevenues(ctx context.Context, filter SalesFilterParameter) (map[string]primitive.Decimal128, error) {
	revenues := []struct {
		Currency string               `bson:"currency"`
		Total    primitive.Decimal128 `bson:"total"`
	}{}

	addressFilter := bson.A{}
	projectRevenueFields := bson.A{}
	for _, a := range filter.Addresses {
		addressFilter = append(addressFilter, bson.M{fmt.Sprintf("shares.%s", a): bson.M{"$nin": bson.A{nil, ""}}})
		projectRevenueFields = append(projectRevenueFields, bson.M{"$ifNull": bson.A{fmt.Sprintf("$shares.%s", a), 0}})
	}

	match := bson.M{"$or": addressFilter}
	if filter.Marketplace != "" {
		match["metadata.marketplace"] = filter.Marketplace
	}

	timestampFilter := bson.M{}
	if filter.From != nil {
		timestampFilter["$gte"] = filter.From
	}
	if filter.To != nil {
		timestampFilter["$lte"] = filter.To
	}
	if len(timestampFilter) > 0 {
		match["timestamp"] = timestampFilter
	}

	pipelines := []bson.M{
		{"$match": match},
		{"$project": bson.M{
			"revenue": bson.M{
				"$add": projectRevenueFields,
			},
			"metadata.revenueCurrency": 1,
		}},
		{"$group": bson.M{
			"_id":      "$metadata.revenueCurrency",
			"currency": bson.M{"$last": "$metadata.revenueCurrency"},
			"total":    bson.M{"$sum": "$revenue"},
		}},
	}

	cursor, err := s.salesTimeSeriesCollection.Aggregate(ctx, pipelines)

	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &revenues); err != nil {
		return nil, err
	}

	resultMap := make(map[string]primitive.Decimal128)
	for _, r := range revenues {
		resultMap[r.Currency] = r.Total
	}

	return resultMap, nil
}

func (s *MongodbIndexerStore) GetExchangeRateLastTime(ctx context.Context) (time.Time, error) {
	findOptions := options.FindOne().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	r := s.historicalExchangeRatesCollection.FindOne(ctx, bson.M{}, findOptions)

	if err := r.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			// If a document is not found, return zero time
			return time.Time{}, nil
		}

		return time.Time{}, err
	}

	var rate ExchangeRate
	if err := r.Decode(&rate); err != nil {
		return time.Time{}, err
	}

	return rate.Timestamp, nil
}

func (s *MongodbIndexerStore) UpdateAssetConfiguration(
	ctx context.Context,
	indexID string,
	configuration *AssetConfiguration) (int64, error) {
	if nil == configuration {
		return 0, errors.New("configuration is nil")
	}

	// Convert to JSON and back to map to utilize omitempty tag
	jsonBytes, err := json.Marshal(configuration)
	if err != nil {
		return 0, err
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &configMap); err != nil {
		return 0, err
	}

	// Create update document with only non-nil fields
	updateFields := bson.M{}
	flattenMap(configMap, "attributes.configuration", updateFields)

	// If no fields to update, just return early
	if len(updateFields) == 0 {
		return 0, nil
	}

	r, err := s.assetCollection.UpdateOne(
		ctx,
		bson.M{"indexID": indexID},
		bson.M{"$set": updateFields},
	)
	return r.ModifiedCount, err
}
