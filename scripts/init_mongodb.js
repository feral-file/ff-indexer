db.createUser({
  user: 'nft_indexer',
  pwd: passwordPrompt(),
  roles: [{
      role: 'dbAdmin',
      db: 'nft_indexer'
    },
    {
      role: 'readWrite',
      db: 'nft_indexer'
    }
  ]
});

db.createUser({
  user: 'cache',
  pwd: passwordPrompt(),
  roles: [{
      role: 'dbAdmin',
      db: 'cache'
    },
    {
      role: 'readWrite',
      db: 'cache'
    }
  ]
});

// tokens index
db.tokens.createIndex({
  indexID: 1
}, {
  unique: true,
  sparse: true
});
db.tokens.createIndex({
  id: 1
}, );
db.tokens.createIndex({
  owner: 1
}, );
db.tokens.createIndex({
  owners: 1
}, );
db.tokens.createIndex({
  ownersArray: 1
}, );
db.tokens.createIndex({
  blockchain: 1
}, );
db.tokens.createIndex({
  assetID: 1
}, );
db.tokens.createIndex({
  lastRefreshedTime: 1
}, );
db.tokens.createIndex({
  lastRefreshedTime: -1
}, );
db.tokens.createIndex({
  contractAddress: 1
}, );
db.tokens.createIndex({
  swapped: 1
}, );

// assets index
db.assets.createIndex({
  id: 1
}, );
db.assets.createIndex({
  indexID: 1
}, {
  unique: true,
  sparse: true
});
db.assets.createIndex({
  "source": 1
});
db.assets.createIndex({
  "thumbnailID": 1
});
db.assets.createIndex({
  "thumbnailLastCheck": 1
});
db.assets.createIndex({
  "thumbnailFailedReason": 1
});
db.assets.createIndex({
  "projectMetadata.latest.source": 1
});
db.assets.createIndexes([{
  "projectMetadata.latest.thumbnailURL": 1
}, {
  "projectMetadata.latest.galleryThumbnailURL": 1
}, {
  "projectMetadata.latest.previewURL": 1
}]);
db.assets.createIndex({
  "projectMetadata.latest.lastUpdatedAt": 1
});

// accounts index
db.accounts.createIndex({
  account: 1
}, {
  unique: true,
  sparse: true
});

// account_tokens index
db.account_tokens.createIndex({
  indexID: 1,
  ownerAccount: 1
}, {
  unique: true,
  sparse: true
});
db.account_tokens.createIndexes([{
  "ownerAccount": 1,
  "lastActivityTime": -1
}, {
  "ownerAccount": 1
}, {
  "blockchain": 1
}, {
  "ownerAccount": 1,
  "runID": 1
}, {
  "lastActivityTime": -1
}, {
  "lastRefreshedTime": -1
}, {
  "pendingTxs": 1
}]);


// identities index
db.identities.createIndex({
  accountNumber: 1,
  blockchain: 1
}, {
  unique: true,
  sparse: true
});

db.createView("token_assets", "tokens", [{
    "$lookup": {
      "from": "assets",
      "localField": "assetID",
      "foreignField": "id",
      "as": "asset"
    }
  },
  {
    "$unwind": "$asset"
  },
  {
    "$addFields": {
      "asset.metadata.project": "$asset.projectMetadata"
    }
  },
  {
    "$project": {
      "asset._id": 0,
      "asset.projectMetadata": 0,
      "ownersArray": 0,
      "_id": 0
    }
  }
]);

db.collections.createIndexes([{
  "creator": 1,
  "lastActivityTime": -1
}, {
  "creator": 1
}, {
  "id": 1
}, {
  "lastActivityTime": -1
}, {
  "lastUpdatedTime": -1
}]);


db.collection_assets.createIndexes([{
  "collectionID": 1
}, {
  "lastActivityTime": -1
}, {
  "edition": 1
}]);

// time series data for sales info
db.createCollection(
  "sales_time_series",
  {
    timeseries: {
      timeField: "timestamp",
      metaField: "metadata",
      granularity: "seconds"
    }
  }
);
