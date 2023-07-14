package cache

import (
	"context"

	log "github.com/bitmark-inc/autonomy-logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type CacheStore interface {
	SaveData(ctx context.Context, cacheKey string, value interface{}) error
	GetData(ctx context.Context, cacheKey string) (interface{}, error)
}

type MongoDBCacheStore struct {
	dbName               string
	mongoClient          *mongo.Client
	blockCacheCollection *mongo.Collection
}

const (
	blockCacheCollectionName = "block_caches"
)

func NewMongoDBCacheStore(ctx context.Context, mongodbURI, dbName string) (*MongoDBCacheStore, error) {
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongodbURI))
	if err != nil {
		return nil, err
	}

	db := mongoClient.Database(dbName)
	blockCacheCollection := db.Collection(blockCacheCollectionName)

	return &MongoDBCacheStore{
		dbName:               dbName,
		mongoClient:          mongoClient,
		blockCacheCollection: blockCacheCollection,
	}, nil
}

// SaveData insert or update the the value for the cacheKey
func (s *MongoDBCacheStore) SaveData(ctx context.Context, cacheKey string, value interface{}) error {
	r, err := s.blockCacheCollection.UpdateOne(ctx,
		bson.M{"key": cacheKey},
		bson.M{"$set": bson.M{"key": cacheKey, "data": value}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return err
	}

	if r.MatchedCount == 0 && r.UpsertedCount == 0 {
		log.Warn("cache is not added or updated", zap.String("key", cacheKey))
	}

	return nil
}

// GetData get the data by cacheKey
func (s *MongoDBCacheStore) GetData(ctx context.Context, cacheKey string) (interface{}, error) {
	var info struct {
		Key  string      `bson:"key"`
		Data interface{} `bson:"data"`
	}

	r := s.blockCacheCollection.FindOne(ctx,
		bson.M{
			"key": cacheKey,
		},
	)
	if err := r.Err(); err != nil {
		return nil, err
	}

	if err := r.Decode(&info); err != nil {
		return nil, err
	}

	return info.Data, nil
}
