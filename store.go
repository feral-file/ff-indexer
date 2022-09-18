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

	UpdateOwner(ctx context.Context, indexID, owner string, updatedAt time.Time) error
	UpdateTokenProvenance(ctx context.Context, indexID string, provenances []Provenance) error
	UpdateMaintainedTokenProvenance(ctx context.Context, indexID string, provenances MaintainedProvenance) error
	UpdateTokenOwners(ctx context.Context, indexID string, lastActivityTime time.Time, owners map[string]int64) error
	PushProvenance(ctx context.Context, indexID string, lockedTime time.Time, provenance Provenance) error

	GetTokensByIndexIDs(ctx context.Context, indexIDs []string) ([]Token, error)
	GetOutdatedTokensByOwner(ctx context.Context, owner string) ([]Token, error)
	GetTokenIDsByOwner(ctx context.Context, owner string) ([]string, error)
	GetTokenIDsByOwners(ctx context.Context, owners []string) ([]string, error)

	GetDetailedTokens(ctx context.Context, filterParameter FilterParameter, offset, size int64) ([]DetailedToken, error)
	GetDetailedTokensByOwners(ctx context.Context, owner []string, filterParameter FilterParameter, offset, size int64) ([]DetailedToken, error)

	GetTokensByTextSearch(ctx context.Context, searchText string, offset, size int64) ([]DetailedToken, error)

	GetIdentity(ctx context.Context, accountNumber string) (AccountIdentity, error)
	GetIdentities(ctx context.Context, accountNumbers []string) (map[string]AccountIdentity, error)
	IndexIdentity(ctx context.Context, identity AccountIdentity) error
}

type FilterParameter struct {
	Source string
	IDs    []string
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
	dbName               string
	mongoClient          *mongo.Client
	tokenCollection      *mongo.Collection
	assetCollection      *mongo.Collection
	identityCollection   *mongo.Collection
	provenanceCollection *mongo.Collection
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

	// update an existent asset
	if !assetCreated {
		var a struct {
			Source          string                   `json:"source" bson:"source"`
			ProjectMetadata VersionedProjectMetadata `json:"projectMetadata" bson:"projectMetadata"`
		}

		if err := r.Decode(&a); err != nil {
			return err
		}

		// igonre update when the original source is feralfile but the incoming source is not
		if !(a.Source == SourceFeralFile && assetUpdates.Source != SourceFeralFile) {
			// TODO: check whether to remove the thumbnail cache when the thumbnail data is updated.
			updates := bson.D{{"$set", bson.D{{"projectMetadata.latest", assetUpdates.ProjectMetadata}}}}
			if a.ProjectMetadata.Latest.ThumbnailURL != assetUpdates.ProjectMetadata.ThumbnailURL {
				logrus.
					WithField("old", a.ProjectMetadata.Latest.ThumbnailURL).
					WithField("new", assetUpdates.ProjectMetadata.ThumbnailURL).
					Debug("image cache need to be reset")
				updates = append(updates, bson.E{"$unset", bson.M{"thumbnailID": ""}})
			}

			s.assetCollection.UpdateOne(
				ctx,
				bson.M{"indexID": indexID},
				updates,
			)
		}
	}

	for _, token := range assetUpdates.Tokens {
		token.AssetID = id

		if token.IndexID == "" {
			token.IndexID = TokenIndexID(token.Blockchain, token.ContractAddress, token.ID)
		}

		tokenResult := s.tokenCollection.FindOne(ctx, bson.M{"indexID": token.IndexID})
		if err := tokenResult.Err(); err != nil {
			if err == mongo.ErrNoDocuments {
				// If a token is not found, insert a new token
				logrus.WithField("token_id", token.ID).Warn("token is not found")

				token.LastActivityTime = token.MintAt // set LastActivityTime to default token minted time
				token.OwnersArray = []string{token.Owner}
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

		// ignore updates for swapped and burned token
		if currentToken.Swapped || currentToken.Burned {
			continue
		}

		if token.Balance == 0 {
			logrus.WithField("token_id", token.ID).Warn("ignore zero balance update")
			continue
		}

		updateSet := bson.M{"owner": token.Owner, "fungible": token.Fungible}
		var addToSet bson.M
		if token.Fungible {
			updateSet[fmt.Sprintf("owners.%s", token.Owner)] = token.Balance
			addToSet = bson.M{"ownersArray": token.Owner}
		} else {
			updateSet["owners"] = map[string]int64{token.Owner: token.Balance}
			updateSet["ownersArray"] = []string{token.Owner}
		}

		tokenUpdate := bson.M{"$set": updateSet}
		if addToSet != nil {
			tokenUpdate["$addToSet"] = addToSet
		}

		logrus.WithField("token_id", token.ID).WithField("token", token).Debug("token data for updated")
		r, err := s.tokenCollection.UpdateOne(ctx,
			bson.M{"indexID": token.IndexID, "swapped": bson.M{"$ne": true}, "burned": bson.M{"$ne": true}},
			tokenUpdate, options.Update().SetUpsert(true))
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
	originalTokenIndexID := TokenIndexID(swap.OriginalBlockchain, swap.OriginalContractAddress, swap.OriginalTokenID)

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
		return "", fmt.Errorf("token has burned")
	}
	originalBaseTokenInfo := originalToken.BaseTokenInfo

	var newTokenIndexID string

	switch swap.NewBlockchain {
	case EthereumBlockchain:
		tokenHexID, ok := big.NewInt(0).SetString(swap.NewTokenID, 10)
		if !ok {
			return "", fmt.Errorf("invalid token id for swapping")
		}

		newTokenIndexID = TokenIndexID(EthereumBlockchain, swap.NewContractAddress, tokenHexID.Text(16))
	default:
		return "", fmt.Errorf("blockchain is not supported")
	}

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

	logrus.WithField("from", originalTokenIndexID).WithField("to", newTokenIndexID).Debug("update tokens for swapping")
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

// GetOutdatedTokensByOwner returns a list of outdated tokens for a specific owner
func (s *MongodbIndexerStore) GetOutdatedTokensByOwner(ctx context.Context, owner string) ([]Token, error) {
	var tokens []Token

	cursor, err := s.tokenCollection.Find(ctx, bson.M{
		fmt.Sprintf("owners.%s", owner): bson.M{"$gte": 1},
		"ownersArray":                   bson.M{"$in": bson.A{owner}},

		"burned":  bson.M{"$ne": true},
		"is_demo": bson.M{"$ne": true},
		"$or": bson.A{
			bson.M{"lastRefreshedTime": bson.M{"$exists": false}},
			bson.M{"lastRefreshedTime": bson.M{"$lt": time.Now().Add(-time.Hour)}},
		},
	}, options.Find().SetProjection(bson.M{"indexID": 1, "_id": 0}).SetSort(bson.M{"lastRefreshedTime": 1}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

// GetDetailedTokens returns a list of tokens information based on ids
func (s *MongodbIndexerStore) GetDetailedTokens(ctx context.Context, filterParameter FilterParameter, offset, size int64) ([]DetailedToken, error) {
	tokens := []DetailedToken{}

	tokenFilter := bson.M{}
	findOptions := options.Find().SetSort(bson.M{"_id": 1})

	if len(filterParameter.IDs) > 0 {
		tokenFilter["indexID"] = bson.M{"$in": filterParameter.IDs}
	} else {
		// set query limit and skip if it is about to query all tokens
		findOptions.SetLimit(size).SetSkip(offset)
	}

	logrus.
		WithField("filterParameter", filterParameter).
		WithField("offset", offset).
		WithField("size", size).
		Debug("GetDetailedTokens")

	cursor, err := s.tokenCollection.Find(ctx, tokenFilter, findOptions)
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

// FIXME: update using multiple owner pattern
// UpdateOwner updates owner for a specific token (single owner)
func (s *MongodbIndexerStore) UpdateOwner(ctx context.Context, indexID string, owner string, updatedAt time.Time) error {
	if owner == "" {
		logrus.WithField("indexID", indexID).Warn("ignore update empty owner")
		return nil
	}

	// update provenance only for non-burned tokens
	_, err := s.tokenCollection.UpdateOne(ctx, bson.M{
		"indexID":          indexID,
		"burned":           bson.M{"$ne": true},
		"lastActivityTime": bson.M{"$lte": updatedAt},
	}, bson.M{
		"$set": bson.M{
			"owner":             owner,
			"lastActivityTime":  updatedAt,
			"lastRefreshedTime": time.Now(),
		},
	})

	return err
}

// UpdateTokenProvenance updates provenance for a specific token
func (s *MongodbIndexerStore) UpdateTokenProvenance(ctx context.Context, indexID string, provenances []Provenance) error {
	if len(provenances) == 0 {
		logrus.WithField("indexID", indexID).Warn("ignore update empty provenance")
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

// UpdateMaintainedTokenProvenance updates provenance for a specific token
func (s *MongodbIndexerStore) UpdateMaintainedTokenProvenance(ctx context.Context, indexID string, provenances MaintainedProvenance) error {

	tokenUpdates := bson.M{
		"IndexID":    indexID,
		"Provenance": provenances.Provenance,
		"Owners":     provenances.Owners,
		"Fungible":   provenances.Fungible,
		"AssetID":    provenances.AssetID,
	}

	_, err := s.provenanceCollection.UpdateOne(ctx, bson.M{
		"indexID": indexID,
	}, bson.M{
		"$set": tokenUpdates,
	})

	return err
}

// UpdateTokenOwners updates owners for a specific token
func (s *MongodbIndexerStore) UpdateTokenOwners(ctx context.Context, indexID string, lastActivityTime time.Time, owners map[string]int64) error {
	if len(owners) == 0 {
		logrus.WithField("indexID", indexID).Warn("ignore update empty provenance")
		return nil
	}

	ownersArray := make([]string, 0, len(owners))
	for owner := range owners {
		ownersArray = append(ownersArray, owner)
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
	formerOwner := *provenance.FormerOwner

	u, err := s.tokenCollection.UpdateOne(ctx, bson.M{
		"indexID":                indexID,
		"lastRefreshedTime":      lockedTime,
		"provenance.0.timestamp": bson.M{"$lt": provenance.Timestamp},
		"provenance.0.owner":     formerOwner,
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
func (s *MongodbIndexerStore) GetDetailedTokensByOwners(ctx context.Context, owners []string, filterParameter FilterParameter, offset, size int64) ([]DetailedToken, error) {
	tokens := make([]DetailedToken, 0, 10)

	type asset struct {
		ThumbnailID     string                   `bson:"thumbnailID"`
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
		c, err = s.getTokensByAggregation(ctx, owners, filterParameter, offset, size)
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
		a.ProjectMetadata.Latest.FirstMintedAt = "0001-01-01T00:00:00.000Z"
		a.ProjectMetadata.Origin.FirstMintedAt = "0001-01-01T00:00:00.000Z"
		token.ThumbnailID = a.ThumbnailID
		token.ProjectMetadata = a.ProjectMetadata

		tokens = append(tokens, token)
	}

	return tokens, nil
}

// getTokensByAggregation queries tokens by aggregation which provides a more flexible query option by mongodb
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
		{"$sort": bson.D{{"lastActivityTime", -1}, {"_id", -1}}},
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

// getTokensByAggregation queries tokens by aggregation which provides a more flexible query option by mongodb
func (s *MongodbIndexerStore) getTokensByAggregation(ctx context.Context, owners []string, filterParameter FilterParameter, offset, size int64) (*mongo.Cursor, error) {
	pipelines := []bson.M{
		{
			"$match": bson.M{
				"owner":  bson.M{"$in": owners},
				"burned": bson.M{"$ne": true},
			},
		},
		{"$sort": bson.D{{"lastActivityTime", -1}, {"_id", -1}}},
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

// GetTokensByTextSearch returns a list of token those assets match have attributes that match the search text.
func (s *MongodbIndexerStore) GetTokensByTextSearch(ctx context.Context, searchText string, offset, size int64) ([]DetailedToken, error) {
	logrus.WithField("searchText", searchText).
		WithField("offset", offset).
		WithField("size", size).
		Debug("GetTokensByTextSearch")

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
