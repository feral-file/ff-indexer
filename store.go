package main

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	assetCollectionName = "assets"
	tokenCollectionName = "tokens"
)

const (
	blockchainTypeBitmark = "bitmark"
	blockchainTypeERC721  = "erc721"
)

type IndexerStore interface {
	IndexAsset(ctx context.Context, id string, assetUpdates AssetUpdates) error
	GetTokens(ctx context.Context, ids []string) ([]TokenInfo, error)
	GetTokensByOwner(ctx context.Context, owner string) ([]TokenInfo, error)
}

func NewMongodbIndexerStore(ctx context.Context, mongodbURI, dbName string) (*MongodbIndexerStore, error) {
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongodbURI))
	if err != nil {
		return nil, err
	}

	db := mongoClient.Database(dbName)
	tokenCollection := db.Collection(tokenCollectionName)
	assetCollection := db.Collection(assetCollectionName)

	return &MongodbIndexerStore{
		dbName:          dbName,
		mongoClient:     mongoClient,
		tokenCollection: tokenCollection,
		assetCollection: assetCollection,
	}, nil
}

type MongodbIndexerStore struct {
	dbName          string
	mongoClient     *mongo.Client
	tokenCollection *mongo.Collection
	assetCollection *mongo.Collection
}

// IndexAsset creates an asset and its corresponded tokens by inputs
func (s *MongodbIndexerStore) IndexAsset(ctx context.Context, id string, assetUpdates AssetUpdates) error {
	assetCreated := false
	r := s.assetCollection.FindOne(ctx, bson.M{"id": id})
	if err := r.Err(); err != nil {
		if r.Err() == mongo.ErrNoDocuments {
			// Create a new asset if it is not added
			if _, err := s.assetCollection.InsertOne(ctx, bson.M{
				"id":                 id,
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
			bson.M{"id": id},
			bson.D{{"$set", bson.D{{"projectMetadata.latest", assetUpdates.ProjectMetadata}}}},
		)
	}

	for _, token := range assetUpdates.Tokens {
		token.AssetID = id
		r, err := s.tokenCollection.UpdateOne(ctx, bson.M{"id": token.ID}, bson.M{"$set": token}, options.Update().SetUpsert(true))
		if err != nil {
			return err
		}
		if r.MatchedCount == 0 && r.UpsertedCount == 0 {
			return fmt.Errorf("token is not added")
		}
	}
	return nil
}

// GetTokens returns a list of tokens
func (s *MongodbIndexerStore) GetTokens(ctx context.Context, ids []string) ([]TokenInfo, error) {
	tokens := make([]TokenInfo, 0, len(ids))

	type asset struct {
		ProjectMetadata VersionedProjectMetadata `json:"projectMetadata"`
	}

	assets := map[string]asset{}
	for _, id := range ids {

		tokenResult := s.tokenCollection.FindOne(ctx, bson.M{"id": id})
		if err := tokenResult.Err(); err != nil {
			return nil, err
		}

		var token Token
		if err := tokenResult.Decode(&token); err != nil {
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

		tokens = append(tokens, TokenInfo{
			Token:           token,
			ProjectMetadata: a.ProjectMetadata,
		})
	}
	return tokens, nil
}

// GetTokensByOwner returns a list of tokens which belongs to an owner
func (s *MongodbIndexerStore) GetTokensByOwner(ctx context.Context, owner string) ([]TokenInfo, error) {
	tokens := make([]TokenInfo, 0)

	type asset struct {
		ProjectMetadata VersionedProjectMetadata `json:"projectMetadata"`
	}

	assets := map[string]asset{}
	c, err := s.tokenCollection.Find(ctx, bson.M{"owner": owner})
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
		tokens[i].ProjectMetadata = a.ProjectMetadata
	}

	return tokens, nil
}
