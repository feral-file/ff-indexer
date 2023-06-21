package indexer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bitmark-inc/nft-indexer/log"
	"github.com/fatih/structs"
	"github.com/meirf/gopart"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	QueryPageSize       = 25
	UnsignedFxhashCID   = "QmYwSwa5hP4346GqD7hAjutwJSmeYTdiLQ7Wec2C7Cez1D"
	UnresolvedFxhashURL = "https://gateway.fxhash.xyz/ipfs//"
)

const (
	assetCollectionName          = "assets"
	tokenCollectionName          = "tokens"
	identityCollectionName       = "identities"
	ffIdentityCollectionName     = "ff_identities"
	accountCollectionName        = "accounts"
	accountTokenCollectionName   = "account_tokens"
	tokenFeedbackCollectionName  = "token_feedbacks"
	tokenAssetViewCollectionName = "token_assets"
)

var ErrNoRecordUpdated = fmt.Errorf("no record updated")

type Store interface {
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

	GetPendingAccountTokens(ctx context.Context) ([]AccountToken, error)
	AddPendingTxToAccountToken(ctx context.Context, ownerAccount, indexID, pendingTx, blockchain, ID string) error
	UpdatePendingTxsToAccountToken(ctx context.Context, ownerAccount, indexID string, pendingTxs []string, lastPendingTimes []time.Time) error

	IndexAccount(ctx context.Context, account Account) error
	IndexAccountTokens(ctx context.Context, owner string, accountTokens []AccountToken) error
	GetAccount(ctx context.Context, owner string) (Account, error)
	GetAccountTokensByIndexIDs(ctx context.Context, indexIDs []string) ([]AccountToken, error)
	UpdateAccountTokenOwners(ctx context.Context, indexID string, tokenBalances []OwnerBalance) error
	GetDetailedAccountTokensByOwner(ctx context.Context, account string, filterParameter FilterParameter, offset, size int64) ([]DetailedToken, error)
	IndexDemoTokens(ctx context.Context, owner string, indexIDs []string) error
	DeleteDemoTokens(ctx context.Context, owner string) error

	UpdateOwnerForFungibleToken(ctx context.Context, indexID string, lockedTime time.Time, to string, total int64) error

	GetAbsentMimeTypeTokens(ctx context.Context, limit int) ([]AbsentMIMETypeToken, error)
	UpdateTokenFeedback(ctx context.Context, tokenFeedbacks []TokenFeedbackUpdate, userDID string) error
	GetGrouppedTokenFeedbacks(ctx context.Context) ([]GrouppedTokenFeedback, error)
	UpdateTokenSugesstedMIMEType(ctx context.Context, indexID, mimeType string) error
	GetPresignedThumbnailTokens(ctx context.Context) ([]Token, error)

	MarkAccountTokenChanged(ctx context.Context, indexIDs []string) error

	GetDetailedTokensV2(ctx context.Context, filterParameter FilterParameter, offset, size int64) ([]DetailedTokenV2, error)
	GetDetailedAccountTokensByOwners(ctx context.Context, owner []string, filterParameter FilterParameter, lastUpdatedAt time.Time, sortBy string, offset, size int64) ([]DetailedTokenV2, error)

	GetDetailedToken(ctx context.Context, indexID string) (DetailedToken, error)
	GetTotalBalanceOfOwnerAccounts(ctx context.Context, addresses []string) (int, error)

	GetNullProvenanceTokensByIndexIDs(ctx context.Context, indexIDs []string) ([]string, error)

	GetOwnerAccountsByIndexIDs(ctx context.Context, indexIDs []string) ([]string, error)

	CheckAddressOwnTokenByCriteria(ctx context.Context, address string, criteria Criteria) (bool, error)
	GetOwnersByBlockchainContracts(context.Context, map[string][]string) ([]string, error)
}

type FilterParameter struct {
	Source string
	IDs    []string
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

func NewMongodbIndexerStore(ctx context.Context, mongodbURI, dbName string) (*MongodbIndexerStore, error) {
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
	tokenFeedbackCollection := db.Collection(tokenFeedbackCollectionName)
	tokenAssetCollection := db.Collection(tokenAssetViewCollectionName)

	return &MongodbIndexerStore{
		dbName:                  dbName,
		mongoClient:             mongoClient,
		tokenCollection:         tokenCollection,
		assetCollection:         assetCollection,
		identityCollection:      identityCollection,
		ffIdentityCollection:    ffIdentityCollection,
		accountCollection:       accountCollection,
		accountTokenCollection:  accountTokenCollection,
		tokenFeedbackCollection: tokenFeedbackCollection,
		tokenAssetCollection:    tokenAssetCollection,
	}, nil
}

type MongodbIndexerStore struct {
	dbName                  string
	mongoClient             *mongo.Client
	tokenCollection         *mongo.Collection
	assetCollection         *mongo.Collection
	identityCollection      *mongo.Collection
	ffIdentityCollection    *mongo.Collection
	accountCollection       *mongo.Collection
	accountTokenCollection  *mongo.Collection
	tokenFeedbackCollection *mongo.Collection
	tokenAssetCollection    *mongo.Collection
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
	LastRefreshedTime time.Time `structs:"lastRefreshedTime"`
	LastActivityTime  time.Time `structs:"lastActivityTime,omitempty"`
}

// checkIfTokenNeedToUpdate returns true if the new token data is suppose to be
// better than existent one.
func checkIfTokenNeedToUpdate(assetSource string, currentToken, newToken Token) bool {
	// ignore updates for swapped and burned token
	if currentToken.Swapped || currentToken.Burned {
		return false
	}

	// check if we need to update an existent token
	if assetSource == SourceFeralFile {
		return true
	}

	// assetSource is not feral file
	// only update if token source is not feral file and token balance is greater than zero.
	if newToken.Balance > 0 && currentToken.Source != SourceFeralFile {
		return true
	}

	return false
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
		if err := tokenResult.Err(); err != nil {
			if err == mongo.ErrNoDocuments {
				// If a token is not found, insert a new token
				log.Info("new token found", zap.String("token_id", token.ID))

				if token.LastActivityTime.IsZero() {
					// set LastActivityTime to default token minted time
					token.LastActivityTime = token.MintedAt
				}
				token.OwnersArray = []string{token.Owner}
				token.Owners = map[string]int64{token.Owner: 1}
				_, err := s.tokenCollection.InsertOne(ctx, token)
				if err != nil {
					return err
				}
				continue
			}

			return err
		}

		var currentToken Token
		if err := tokenResult.Decode(&currentToken); err != nil {
			return err
		}

		if checkIfTokenNeedToUpdate(assetUpdates.Source, currentToken, token) {
			tokenUpdateSet := TokenUpdateSet{
				Fungible:          token.Fungible,
				Source:            token.Source,
				AssetID:           id,
				Edition:           token.Edition,
				EditionName:       token.EditionName,
				ContractAddress:   token.ContractAddress,
				LastRefreshedTime: indexTime,
			}

			if !token.LastActivityTime.IsZero() {
				tokenUpdateSet.LastActivityTime = token.LastActivityTime
			}

			tokenUpdate := bson.M{"$set": structs.Map(tokenUpdateSet)}

			log.Debug("token data for updated", zap.String("token_id", token.ID), zap.Any("tokenUpdate", tokenUpdate))
			r, err := s.tokenCollection.UpdateOne(ctx, bson.M{"indexID": token.IndexID}, tokenUpdate)
			if err != nil {
				return err
			}
			if r.MatchedCount == 0 {
				log.Warn("token is not updated", zap.String("token_id", token.ID))
			}
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
	case EthereumBlockchain, TezosBlockchain:
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
		// return burned token if the SwappedTo is identical to newTokenIndexID
		if *originalToken.SwappedTo == newTokenIndexID {
			return newTokenIndexID, nil
		}
		return "", fmt.Errorf("token has burned into different id")
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
		if _, err := s.tokenCollection.UpdateOne(ctx, bson.M{"indexID": originalTokenIndexID}, bson.M{
			"$set": bson.M{
				"burned":    true,
				"swappedTo": newTokenIndexID,
			},
		}); err != nil {
			return nil, err
		}

		r, err := s.tokenCollection.UpdateOne(ctx,
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
		log.Error("swap token transaction failed",
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
			"burned":             bson.M{"$ne": true},
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
	if (itemLength % PageSize) != 0 {
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
				FilterParameter{IDs: filterParameter.IDs[start:end], Source: filterParameter.Source},
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

// UpdateOwner updates owner for a specific non-fungible token
func (s *MongodbIndexerStore) UpdateOwner(ctx context.Context, indexID string, owner string, updatedAt time.Time) error {
	if owner == "" {
		log.Warn("ignore update empty owner", zap.String("indexID", indexID))
		return nil
	}

	// update provenance only for non-burned tokens
	_, err := s.tokenCollection.UpdateOne(ctx, bson.M{
		"indexID":          indexID,
		"fungible":         false,
		"burned":           bson.M{"$ne": true},
		"lastActivityTime": bson.M{"$lt": updatedAt},
	}, bson.M{
		"$set": bson.M{
			"owner":             owner,
			"owners":            map[string]int64{owner: 1},
			"ownersArray":       []string{owner},
			"lastActivityTime":  updatedAt,
			"lastRefreshedTime": time.Now(),
		},
	})

	return err
}

// UpdateTokenProvenance updates provenance for a specific token
func (s *MongodbIndexerStore) UpdateTokenProvenance(ctx context.Context, indexID string, provenances []Provenance) error {
	if len(provenances) == 0 {
		log.Warn("ignore update empty provenance", zap.String("indexID", indexID))
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
		"burned":  bson.M{"$ne": true},
	}, bson.M{
		"$set": tokenUpdates,
	})

	return err
}

// UpdateTokenOwners updates owners for a specific token
func (s *MongodbIndexerStore) UpdateTokenOwners(ctx context.Context, indexID string, lastActivityTime time.Time, ownerBalances []OwnerBalance) error {
	if len(ownerBalances) == 0 {
		log.Warn("ignore update empty provenance", zap.String("indexID", indexID))
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
		"burned":  bson.M{"$ne": true},
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
		"burned":                        bson.M{"$ne": true},
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
	pipelines := []bson.M{
		{
			"$match": bson.M{
				"owner":  bson.M{"$in": owners},
				"burned": bson.M{"$ne": true},
			},
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
			"burned":  bson.M{"$ne": true},
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
		log.Warn("identity is not added or updated", zap.String("account_number", identity.AccountNumber))
	}

	return nil
}

// AddPendingTxToAccountToken add pendingTx to a specific account token if this pendingTx does not exist
func (s *MongodbIndexerStore) AddPendingTxToAccountToken(ctx context.Context, ownerAccount, indexID, pendingTx, blockchain, ID string) error {
	r, err := s.accountTokenCollection.UpdateOne(ctx,
		bson.M{
			"indexID":      indexID,
			"ownerAccount": ownerAccount,
			"pendingTxs":   bson.M{"$nin": bson.A{pendingTx}},
		},
		bson.M{
			"$push": bson.M{
				"pendingTxs":      pendingTx,
				"lastPendingTime": time.Now(),
			},
			"$set": bson.M{
				"blockchain": blockchain,
				"id":         ID,
			},
		},
	)

	if err != nil {
		log.Error("cannot add pendingTx to account token",
			zap.String("ownerAccount", ownerAccount),
			zap.String("indexID", indexID),
			zap.String("pendingTx", pendingTx),
			zap.Error(err))
		return err
	}

	if r.MatchedCount == 0 || r.ModifiedCount == 0 {
		// 1. We don't have this account token OR
		// 2. This account token already has the pendingTx.
		// We try to insert this account token. If an error happen, it has a
		// high chance to be 2. We log down it with error for tracking.
		_, err := s.accountTokenCollection.InsertOne(ctx,
			bson.M{
				"indexID":         indexID,
				"ownerAccount":    ownerAccount,
				"blockchain":      blockchain,
				"id":              ID,
				"lastPendingTime": bson.A{time.Now()},
				"pendingTxs":      bson.A{pendingTx},
			},
		)
		if err != nil {
			log.Warn("cannot insert a new account token",
				zap.Error(err),
				zap.String("ownerAccount", ownerAccount),
				zap.String("indexID", indexID),
				zap.String("pendingTx", pendingTx))
		}
	}

	return nil
}

// UpdatePendingTxsToAccountToken updates the the pending txs of an account token
func (s *MongodbIndexerStore) UpdatePendingTxsToAccountToken(ctx context.Context, ownerAccount, indexID string, pendingTxs []string, lastPendingTimes []time.Time) error {
	r, err := s.accountTokenCollection.UpdateOne(ctx,
		bson.M{
			"indexID":      indexID,
			"ownerAccount": ownerAccount,
		},
		bson.M{
			"$set": bson.M{
				"pendingTxs":      pendingTxs,
				"lastPendingTime": lastPendingTimes,
			},
		},
	)

	if err != nil {
		log.Error("cannot update pendingTxs and lastPendingTime to account token",
			zap.String("ownerAccount", ownerAccount),
			zap.String("indexID", indexID),
			zap.Error(err))
		return err
	}

	if r.ModifiedCount == 0 {
		log.Warn("account token does not update",
			zap.Error(err),
			zap.String("ownerAccount", ownerAccount),
			zap.String("indexID", indexID),
		)
	}

	return nil
}

// GetPendingAccountTokens gets all pending account tokens in the db
func (s *MongodbIndexerStore) GetPendingAccountTokens(ctx context.Context) ([]AccountToken, error) {
	cursor, err := s.accountTokenCollection.Find(ctx, bson.M{"pendingTxs": bson.M{"$nin": bson.A{nil, bson.A{}}}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	pendingAccountTokens := []AccountToken{}
	for cursor.Next(ctx) {
		var accountToken AccountToken

		if err := cursor.Decode(&accountToken); err != nil {
			var raw interface{}
			if err := cursor.Decode(&raw); err != nil {
				log.Error("fail to decode account token into raw interface", zap.Error(err))
			}
			log.Error("fail to decode account token", zap.Error(err), zap.Any("raw", raw))
			continue
		}

		pendingAccountTokens = append(pendingAccountTokens, accountToken)
	}
	return pendingAccountTokens, nil
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
		log.Warn("account is not added or updated", zap.String("account", account.Account))
	}

	return nil
}

// IndexAccountTokens indexes the account tokens by inputs
func (s *MongodbIndexerStore) IndexAccountTokens(ctx context.Context, owner string, accountTokens []AccountToken) error {
	margin := 15 * time.Second
	for _, accountToken := range accountTokens {
		log.Debug("account token is in a future state", zap.String("indexID", accountToken.IndexID), zap.Any("accountToken", accountToken))
		r, err := s.accountTokenCollection.UpdateOne(ctx,
			bson.M{"indexID": accountToken.IndexID, "ownerAccount": owner, "lastActivityTime": bson.M{"$lt": accountToken.LastActivityTime.Add(-margin)}},
			bson.M{"$set": accountToken},
			options.Update().SetUpsert(true),
		)

		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				// when a duplicated error happens, it means the account token
				// is in a state which is better than current event.
				log.Warn("account token is in a future state", zap.String("indexID", accountToken.IndexID))
				continue
			}
			log.Error("cannot index account token", zap.String("indexID", accountToken.IndexID), zap.String("owner", owner), zap.Error(err))
			return err
		}
		if r.MatchedCount == 0 && r.UpsertedCount == 0 {
			// TODO: not sure when will this happen. Figure this our later
			log.Warn("account token is not added or updated", zap.String("token_id", accountToken.ID))
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

// GetAccountTokensByIndexIDs returns a list of account tokens by a given list of index id
func (s *MongodbIndexerStore) GetAccountTokensByIndexIDs(ctx context.Context, indexIDs []string) ([]AccountToken, error) {
	tokens := make([]AccountToken, 0)

	c, err := s.accountTokenCollection.Aggregate(ctx, []bson.M{
		{"$match": bson.M{"indexID": bson.M{"$in": indexIDs}}},
		{"$sort": bson.D{{Key: "lastActivityTime", Value: 1}}},
		{
			"$group": bson.M{
				"_id":    "$indexID",
				"detail": bson.M{"$first": "$$ROOT"},
			},
		},
		{"$replaceRoot": bson.M{"newRoot": "$detail"}},
	})

	if err != nil {
		return nil, err
	}

	for c.Next(ctx) {
		var token AccountToken
		if err := c.Decode(&token); err != nil {
			return nil, err
		}

		tokens = append(tokens, token)
	}

	return tokens, nil
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
			log.Error("could not update balance ", zap.String("indexID", indexID), zap.String("owner", ownerBalance.Address), zap.Error(err))
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

// GetDetailedAccountTokensByOwner returns a list of DetailedToken by account owner
func (s *MongodbIndexerStore) GetDetailedAccountTokensByOwner(ctx context.Context, account string, filterParameter FilterParameter, offset, size int64) ([]DetailedToken, error) {
	findOptions := options.Find().SetSort(bson.D{{Key: "lastActivityTime", Value: -1}, {Key: "_id", Value: -1}}).SetLimit(size).SetSkip(offset)

	log.Debug("GetDetailedAccountTokensByOwner",
		zap.Any("filterParameter", filterParameter),
		zap.Int64("offset", offset),
		zap.Int64("size", size))

	cursor, err := s.accountTokenCollection.Find(ctx, bson.M{
		"ownerAccount": account,
		"balance":      bson.M{"$gt": 0}}, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	indexIDs := make([]string, 0)
	accountTokenMap := map[string]AccountToken{}
	for cursor.Next(ctx) {
		var token AccountToken

		if err := cursor.Decode(&token); err != nil {
			return nil, err
		}

		indexIDs = append(indexIDs, token.IndexID)
		accountTokenMap[token.IndexID] = token
	}

	if len(indexIDs) == 0 {
		return []DetailedToken{}, nil
	}

	filterParameter.IDs = indexIDs
	assets, err := s.GetDetailedTokens(ctx, filterParameter, offset, size)

	if err != nil {
		return nil, err
	}

	for i := range assets {
		asset := &assets[i]

		asset.Balance = accountTokenMap[asset.IndexID].Balance
		asset.Owner = accountTokenMap[asset.IndexID].OwnerAccount
	}

	return assets, nil
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
					log.Error("error while inserting demo tokens", zap.String("indexID", demoIndexID), zap.Error(err))
					return err
				}
				log.Debug("demo token is indexed", zap.String("indexID", demoIndexID))
			} else {
				log.Error("error while finding demoIndexID in the database", zap.String("demoIndexID", demoIndexID), zap.Error(err))
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

// GetAbsentMimeTypeTokens returns list up random limit tokens that mimeType is absent
func (s *MongodbIndexerStore) GetAbsentMimeTypeTokens(ctx context.Context, limit int) ([]AbsentMIMETypeToken, error) {
	compactedToken := []AbsentMIMETypeToken{}

	var tokens []struct {
		IndexID         string                   `bson:"indexID"`
		ProjectMetadata VersionedProjectMetadata `json:"projectMetadata" bson:"projectMetadata"`
	}
	c, err := s.assetCollection.Aggregate(ctx, []bson.M{
		{
			"$match": bson.D{{Key: "$or", Value: []interface{}{
				bson.D{{Key: "projectMetadata.latest.mimeType", Value: ""}},
				bson.D{{Key: "projectMetadata.latest.mimeType", Value: bson.M{"$exists": false}}},
			}}},
		},
		{"$sample": bson.M{"size": limit}},
	})
	if err != nil {
		return nil, err
	}

	if err := c.All(ctx, &tokens); err != nil {
		return nil, err
	}

	for _, token := range tokens {
		compactedToken = append(compactedToken, AbsentMIMETypeToken{
			IndexID:    token.IndexID,
			PreviewURL: token.ProjectMetadata.Latest.PreviewURL,
		})
	}

	return compactedToken, nil
}

// UpdateTokenFeedback inserts or updates list of token feedback by a user.
func (s *MongodbIndexerStore) UpdateTokenFeedback(ctx context.Context, tokenFeedbacks []TokenFeedbackUpdate, userDID string) error {
	r := s.tokenFeedbackCollection.FindOne(ctx, bson.M{"did": userDID}, options.FindOne().SetSort(bson.M{"lastUpdatedTime": -1}))

	if err := r.Err(); err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		}
	}

	if r.Err() == nil {
		var lastTokenFeedback TokenFeedback

		if err := r.Decode(&lastTokenFeedback); err != nil {
			return err
		}

		delay := time.Hour

		if lastTokenFeedback.LastUpdatedTime.Unix() > time.Now().Add(-delay).Unix() {
			log.Debug("feedback submit too frequently",
				zap.Int64("lastUpdatedTime", lastTokenFeedback.LastUpdatedTime.Unix()),
				zap.Int64("now", time.Now().Add(-delay).Unix()),
				zap.String("account", userDID),
			)
			return fmt.Errorf("feedback submit too frequently")
		}
	}

	for _, token := range tokenFeedbacks {
		tokenFeedback := TokenFeedback{
			IndexID:         token.IndexID,
			MimeType:        token.MimeType,
			DID:             userDID,
			LastUpdatedTime: time.Now(),
		}

		r, err := s.tokenFeedbackCollection.UpdateOne(ctx,
			bson.M{"indexID": token.IndexID, "did": userDID},
			bson.M{"$set": tokenFeedback},
			options.Update().SetUpsert(true),
		)

		if err != nil {
			return err
		}

		if r.ModifiedCount == 0 && r.UpsertedCount == 0 {
			log.Warn("token feedback is not added or updated",
				zap.String("index_id", tokenFeedback.IndexID),
				zap.String("did", tokenFeedback.DID),
			)
		}
	}

	return nil
}

// GetGrouppedTokenFeedbacks returns token feedbacks that group by indexID & mimeTypes.
func (s *MongodbIndexerStore) GetGrouppedTokenFeedbacks(ctx context.Context) ([]GrouppedTokenFeedback, error) {
	tokenFeedbacks := make([]GrouppedTokenFeedback, 0)

	c, err := s.tokenFeedbackCollection.Aggregate(ctx, []bson.M{
		{"$sort": bson.D{{Key: "lastActivityTime", Value: 1}}},
		{
			"$group": bson.M{
				"_id":   bson.M{"indexID": "$indexID", "mimeType": "$mimeType"},
				"count": bson.M{"$sum": 1},
			},
		},
		{
			"$group": bson.M{
				"_id": "$_id.indexID",
				"mimeTypes": bson.M{
					"$push": bson.M{
						"mimeType": "$_id.mimeType",
						"count":    "$count",
					},
				},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	for c.Next(ctx) {
		var tokenFeedback GrouppedTokenFeedback
		if err := c.Decode(&tokenFeedback); err != nil {
			return nil, err
		}

		tokenFeedbacks = append(tokenFeedbacks, tokenFeedback)
	}

	return tokenFeedbacks, nil
}

func (s *MongodbIndexerStore) UpdateTokenSugesstedMIMEType(ctx context.Context, indexID, mimeType string) error {
	r := s.assetCollection.FindOne(ctx, bson.M{"indexID": indexID})
	if err := r.Err(); err != nil {
		return err
	}

	updates := bson.D{{Key: "$set", Value: bson.D{
		{Key: "projectMetadata.latest.suggestionMimeType", Value: mimeType},
		{Key: "lastRefreshedTime", Value: time.Now()},
	}}}

	_, err := s.assetCollection.UpdateOne(
		ctx,
		bson.M{"indexID": indexID},
		updates,
	)

	return err
}

// GetPresignedThumbnailTokens gets tokens that have presigned thumbnail
func (s *MongodbIndexerStore) GetPresignedThumbnailTokens(ctx context.Context) ([]Token, error) {
	tokens := []Token{}
	pattern := fmt.Sprintf("%s|%s|^$", UnsignedFxhashCID, UnresolvedFxhashURL)

	cursor, err := s.assetCollection.Find(ctx, bson.M{
		"source":                              SourceTZKT,
		"projectMetadata.latest.source":       "fxhash",
		"projectMetadata.latest.thumbnailURL": bson.M{"$regex": pattern},
	})
	if err != nil {
		return nil, err
	}

	for cursor.Next(ctx) {
		var currentAsset struct {
			ProjectMetadata VersionedProjectMetadata `json:"projectMetadata" bson:"projectMetadata"`
		}

		if err := cursor.Decode(&currentAsset); err != nil {
			return nil, err
		}

		r := s.tokenCollection.FindOne(ctx, bson.M{"assetID": currentAsset.ProjectMetadata.Latest.AssetID})
		if r.Err() != nil {
			log.Error("cannot find asset ID", zap.String("assetID", currentAsset.ProjectMetadata.Latest.AssetID), zap.Error(r.Err()))
			continue
		}

		var token Token

		if err := r.Decode(&token); err != nil {
			log.Error("cannot decode token", zap.Error(err))
			continue
		}

		tokens = append(tokens, token)
	}

	return tokens, nil
}

// MarkAccountTokenChanged sets the lastRefreshedTime to now
func (s *MongodbIndexerStore) MarkAccountTokenChanged(ctx context.Context, indexIDs []string) error {
	_, err := s.accountTokenCollection.UpdateMany(ctx, bson.M{
		"indexID": bson.M{"$in": indexIDs},
	}, bson.M{
		"$set": bson.M{"lastRefreshedTime": time.Now()},
	})

	if err != nil {
		log.Error("cannot update account tokens", zap.Error(err), zap.Any("indexIDs", indexIDs))
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
				FilterParameter{IDs: queryIDs[start:end], Source: filterParameter.Source}, 0, int64(end-start))
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

	pipelines := []bson.M{
		{"$match": bson.M{"indexID": bson.M{"$in": filterParameter.IDs}, "burned": bson.M{"$ne": true}}},
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
func (s *MongodbIndexerStore) GetDetailedToken(ctx context.Context, indexID string) (DetailedToken, error) {
	filterParameter := FilterParameter{
		IDs: []string{indexID},
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
