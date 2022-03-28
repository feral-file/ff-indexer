package indexer

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	assetCollectionName    = "assets"
	tokenCollectionName    = "tokens"
	identityCollectionName = "identities"
)

var ErrNoRecordUpdated = fmt.Errorf("no record updated")

type IndexerStore interface {
	IndexAsset(ctx context.Context, id string, assetUpdates AssetUpdates) error
	SwapToken(ctx context.Context, swapUpdate SwapUpdate) (string, error)
	UpdateTokenProvenance(ctx context.Context, indexID string, provenances []Provenance) error
	PushProvenance(ctx context.Context, indexID string, lockedTime time.Time, provenance Provenance) error

	GetTokensByIndexIDs(ctx context.Context, indexIDs []string) ([]Token, error)
	GetOutdatedTokens(ctx context.Context, size int64) ([]Token, error)
	GetTokenIDsByOwner(ctx context.Context, owner string) ([]string, error)
	GetTokenIDsByOwners(ctx context.Context, owners []string) ([]string, error)

	GetDetailedTokens(ctx context.Context, filterParameter FilterParameter, offset, size int64) ([]DetailedToken, error)
	GetDetailedTokensByOwners(ctx context.Context, owner []string, offset, size int64) ([]DetailedToken, error)

	GetTokensByTextSearch(ctx context.Context, searchText string, offset, size int64) ([]DetailedToken, error)

	GetIdentity(ctx context.Context, accountNumber string) (AccountIdentity, error)
	GetIdentities(ctx context.Context, accountNumbers []string) (map[string]AccountIdentity, error)
	IndexIdentity(ctx context.Context, identity AccountIdentity) error
}

type FilterParameter struct {
	IDs []string
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

	return &MongodbIndexerStore{
		dbName:             dbName,
		mongoClient:        mongoClient,
		tokenCollection:    tokenCollection,
		assetCollection:    assetCollection,
		identityCollection: identityCollection,
	}, nil
}

type MongodbIndexerStore struct {
	dbName             string
	mongoClient        *mongo.Client
	tokenCollection    *mongo.Collection
	assetCollection    *mongo.Collection
	identityCollection *mongo.Collection
}

// IndexAsset creates an asset and its corresponded tokens by inputs
func (s *MongodbIndexerStore) IndexAsset(ctx context.Context, id string, assetUpdates AssetUpdates) error {
	assetCreated := false

	indexID := fmt.Sprintf("%s-%s", strings.ToLower(assetUpdates.Source), id)

	r := s.assetCollection.FindOne(ctx, bson.M{"indexID": indexID})
	if err := r.Err(); err != nil {
		if r.Err() == mongo.ErrNoDocuments {
			// Create a new asset if it is not added
			if _, err := s.assetCollection.InsertOne(ctx, bson.M{
				"id":                 id,
				"indexID":            indexID,
				"source":             assetUpdates.Source,
				"blockchainMetadata": assetUpdates.BlockchainMetadata,
				"projectMetadata": bson.M{
					"origin": assetUpdates.ProjectMetadata,
					"latest": assetUpdates.ProjectMetadata,
				},
			}); err != nil {
				return err
			}
			assetCreated = true
		} else {
			return err
		}
	}

	if !assetCreated {
		s.assetCollection.UpdateOne(
			ctx,
			bson.M{"indexID": indexID},
			bson.D{{"$set", bson.D{{"projectMetadata.latest", assetUpdates.ProjectMetadata}}}},
		)
	}

	for _, token := range assetUpdates.Tokens {
		token.AssetID = id

		if token.IndexID == "" {
			token.IndexID = TokenIndexID(token.Blockchain, token.ContractAddress, token.ID)
		}

		tokenResult := s.tokenCollection.FindOne(ctx, bson.M{"indexID": token.IndexID})
		if err := tokenResult.Err(); err != nil {
			if err == mongo.ErrNoDocuments {
				// insert a new token entry if it is not found
				token.LastActivityTime = token.MintAt
				logrus.WithField("token_id", token.ID).Warn("token is not found")
				_, err := s.tokenCollection.InsertOne(ctx, token)
				if err != nil {
					return err
				}
				continue
			} else {
				return err
			}
		}

		var currentToken Token
		if err := tokenResult.Decode(&currentToken); err != nil {
			return err
		}

		if token.MintAt.Sub(token.LastActivityTime) > 0 {
			token.LastActivityTime = token.MintAt
		}

		// ignore updates for swapped and burned token
		if currentToken.Swapped || currentToken.Burned {
			continue
		}

		logrus.WithField("token_id", token.ID).WithField("token", token).Debug("token data for updated")
		r, err := s.tokenCollection.UpdateOne(ctx,
			bson.M{"indexID": token.IndexID, "swapped": bson.M{"$ne": true}, "burned": bson.M{"$ne": true}},
			bson.M{"$set": token}, options.Update().SetUpsert(true))
		if err != nil {
			return err
		}
		if r.MatchedCount == 0 && r.UpsertedCount == 0 {
			logrus.WithField("token_id", token.ID).Warn("token is not added or updated")
		}
	}
	return nil
}

// SwapToken marks the original token to burned and creates a new token record which inherits
// original blockchain information
func (s *MongodbIndexerStore) SwapToken(ctx context.Context, swap SwapUpdate) (string, error) {

	originalBlockchainAlias, ok := BlockchianAlias[swap.OriginalBlockchain]
	if !ok {
		return "", fmt.Errorf("original blockchain is not supported")
	}
	originalTokenIndexID := fmt.Sprintf("%s-%s-%s", originalBlockchainAlias, swap.OriginalContractAddress, swap.OriginalTokenID)

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

	if originalToken.Burned {
		return "", fmt.Errorf("token has burned")
	}
	originalBaseTokenInfo := originalToken.BaseTokenInfo

	newBlockchainAlias, ok := BlockchianAlias[swap.NewBlockchain]
	if !ok {
		return "", fmt.Errorf("blockchain is not supported")
	}

	newTokenIndexID := fmt.Sprintf("%s-%s-%s", newBlockchainAlias, swap.NewContractAddress, swap.NewTokenID)

	switch swap.NewBlockchain {
	case EthereumBlockchain:
		tokenHexID, ok := big.NewInt(0).SetString(swap.NewTokenID, 10)
		if !ok {
			return "", fmt.Errorf("invalid token id for swapping")
		}

		newTokenIndexID = fmt.Sprintf("%s-%s-%s", newBlockchainAlias, swap.NewContractAddress, tokenHexID.Text(16))
	default:
	}

	newToken := originalToken
	newToken.ID = swap.NewTokenID
	newToken.IndexID = newTokenIndexID
	newToken.Blockchain = swap.NewBlockchain
	newToken.ContractAddress = swap.NewContractAddress
	newToken.ContractType = swap.NewContractType
	newToken.Swapped = true
	newToken.OriginTokenInfo = append([]BaseTokenInfo{originalBaseTokenInfo}, newToken.OriginTokenInfo...)

	logrus.WithField("from", originalTokenIndexID).WithField("to", newTokenIndexID).Debug("update tokens for swapping")
	session, err := s.mongoClient.StartSession()
	if err != nil {
		return "", err
	}
	defer session.EndSession(ctx)

	result, err := session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		if _, err := s.tokenCollection.UpdateOne(ctx, bson.M{"indexID": originalTokenIndexID}, bson.M{
			"$set": bson.M{
				"burned": true,
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

		if r.ModifiedCount == 0 && r.UpsertedCount == 0 {
			return nil, ErrNoRecordUpdated
		}

		return nil, nil
	})

	logrus.WithField("transaction_result", result).Debug("swap token transaction")

	return newTokenIndexID, err
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

// GetOutdatedTokens returns a list of outdated tokens
func (s *MongodbIndexerStore) GetOutdatedTokens(ctx context.Context, size int64) ([]Token, error) {
	var tokens []Token

	cursor, err := s.tokenCollection.Find(ctx, bson.M{
		"blockchain": "bitmark",
		"burned":     bson.M{"$ne": true},
		"$or": bson.A{
			bson.M{"lastRefreshedTime": bson.M{"$exists": false}},
			bson.M{"lastRefreshedTime": bson.M{"$lt": time.Now().Add(-time.Hour)}},
		},
	}, options.Find().SetSort(bson.M{"lastRefreshedTime": 1}).SetLimit(size))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

// GetDetailedTokens returns a list of tokens information based on id
func (s *MongodbIndexerStore) GetDetailedTokens(ctx context.Context, filterParameter FilterParameter, offset, size int64) ([]DetailedToken, error) {
	tokens := []DetailedToken{}

	tokenFilter := bson.M{}

	if len(filterParameter.IDs) > 0 {
		tokenFilter["id"] = bson.M{"$in": filterParameter.IDs}
	}

	logrus.
		WithField("filterParameter", filterParameter).
		WithField("offset", offset).
		WithField("size", size).
		Debug("GetDetailedTokens")

	cursor, err := s.tokenCollection.Find(ctx, tokenFilter, options.Find().SetLimit(size).SetSkip(offset))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	assets := map[string]struct {
		ThumbnailID     string                   `bson:"thumbnailID"`
		ProjectMetadata VersionedProjectMetadata `json:"projectMetadata" bson:"projectMetadata"`
	}{}
	for cursor.Next(ctx) {
		var token Token

		if err := cursor.Decode(&token); err != nil {
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
		a.ProjectMetadata.Latest.FirstMintedAt = "0001-01-01T00:00:00.000Z"
		a.ProjectMetadata.Origin.FirstMintedAt = "0001-01-01T00:00:00.000Z"
		tokens = append(tokens, DetailedToken{
			Token:           token,
			ThumbnailID:     a.ThumbnailID,
			ProjectMetadata: a.ProjectMetadata,
		})
	}
	return tokens, nil
}

// UpdateTokenProvenance updates provenance for a specific token
func (s *MongodbIndexerStore) UpdateTokenProvenance(ctx context.Context, indexID string, provenances []Provenance) error {
	if len(provenances) == 0 {
		logrus.WithField("indexID", indexID).Warn("ignore update empty provenance")
		return nil
	}

	_, err := s.tokenCollection.UpdateOne(ctx, bson.M{
		"indexID": indexID,
	}, bson.M{
		"$set": bson.M{
			"owner":             provenances[0].Owner,
			"lastActivityTime":  provenances[0].Timestamp,
			"lastRefreshedTime": time.Now(),
			"provenance":        provenances,
		},
	})

	return err
}

// PushProvenance push the latest provenance record for a token
func (s *MongodbIndexerStore) PushProvenance(ctx context.Context, indexID string, lockedTime time.Time, provenance Provenance) error {
	u, err := s.tokenCollection.UpdateOne(ctx, bson.M{
		"indexID":                indexID,
		"lastRefreshedTime":      lockedTime,
		"provenance.0.timestamp": bson.M{"$lt": provenance.Timestamp},
		"provenance.0.owner":     provenance.FromOwner,
	}, bson.M{
		"$set": bson.M{
			"owner":             provenance.Owner,
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

	if u.ModifiedCount == 0 {
		return ErrNoRecordUpdated
	}

	return err
}

// GetTokenIDsByOwner returns a list of tokens which belongs to an owner
func (s *MongodbIndexerStore) GetTokenIDsByOwner(ctx context.Context, owner string) ([]string, error) {
	return s.GetTokenIDsByOwners(ctx, []string{owner})
}

// GetTokenIDsByOwners returns a list of tokens which belongs to a list of owner
func (s *MongodbIndexerStore) GetTokenIDsByOwners(ctx context.Context, owners []string) ([]string, error) {
	tokens := make([]string, 0)

	c, err := s.tokenCollection.Find(ctx,
		bson.M{
			"owner": bson.M{"$in": owners}, "burned": bson.M{"$ne": true},
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
func (s *MongodbIndexerStore) GetDetailedTokensByOwners(ctx context.Context, owners []string, offset, size int64) ([]DetailedToken, error) {
	tokens := make([]DetailedToken, 0)

	type asset struct {
		ThumbnailID     string                   `bson:"thumbnailID"`
		ProjectMetadata VersionedProjectMetadata `bson:"projectMetadata"`
	}

	assets := map[string]asset{}
	c, err := s.tokenCollection.Find(ctx, bson.M{"owner": bson.M{"$in": owners}, "burned": bson.M{"$ne": true}},
		options.Find().SetSort(bson.D{{"lastActivityTime", -1}, {"_id", -1}}).SetLimit(size).SetSkip(offset),
	)
	if err != nil {
		return nil, err
	}

	if err := c.All(ctx, &tokens); err != nil {
		return nil, err
	}
	for i, t := range tokens {
		a, assetExist := assets[t.AssetID]
		if !assetExist {
			assetResult := s.assetCollection.FindOne(ctx, bson.M{"id": t.AssetID})
			if err := assetResult.Err(); err != nil {
				return nil, err
			}

			if err := assetResult.Decode(&a); err != nil {
				return nil, err
			}

			assets[t.AssetID] = a
		}

		// FIXME: hardcoded values for backward compatibility
		a.ProjectMetadata.Latest.FirstMintedAt = "0001-01-01T00:00:00.000Z"
		a.ProjectMetadata.Origin.FirstMintedAt = "0001-01-01T00:00:00.000Z"
		tokens[i].ThumbnailID = a.ThumbnailID
		tokens[i].ProjectMetadata = a.ProjectMetadata
	}

	return tokens, nil
}

// GetTokensByTextSearch returns a list of token those assets match have attributes that match the search text.
func (s *MongodbIndexerStore) GetTokensByTextSearch(ctx context.Context, searchText string, offset, size int64) ([]DetailedToken, error) {
	logrus.WithField("searchText", searchText).
		WithField("offset", offset).
		WithField("size", size).
		Debug("GetTokensByTextSearch")

	pipeline := []bson.M{
		{"$match": bson.M{
			"projectMetadata.latest.source": "feralfile", // FIXME: currently, we limit the source of query to feralfile
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
		if err == mongo.ErrNoDocuments {
			return identity, nil
		} else {
			return identity, err
		}
	}

	if err := r.Decode(&identity); err != nil {
		return identity, err
	}

	return identity, nil
}

// GetIdentities returns a list of identities by a list of account numbers
func (s *MongodbIndexerStore) GetIdentities(ctx context.Context, accountNumbers []string) (map[string]AccountIdentity, error) {
	identities := map[string]AccountIdentity{}

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
		logrus.WithField("account_number", identity.AccountNumber).Warn("identity is not added or updated")
	}

	return nil
}
