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
})

db.tokens.createIndex({
  indexID: 1
}, {
  unique: true,
  sparse: true
})
db.tokens.createIndex({
  id: 1
}, )
db.tokens.createIndex({
  owner: 1
}, )
db.tokens.createIndex({
  owners: 1
}, )
db.tokens.createIndex({
  ownersArray: 1
}, )
db.tokens.createIndex({
  blockchain: 1
}, )
db.tokens.createIndex({
  lastRefreshedTime: 1
}, )
db.tokens.createIndex({
  contractAddress: 1
}, )
db.tokens.createIndex({
  swapped: 1
}, )
db.assets.createIndex({
  id: 1
}, )
db.assets.createIndex({
  indexID: 1
}, {
  unique: true,
  sparse: true
})
db.assets.createIndex({
  "source": 1
})
db.assets.createIndex({
  "thumbnailID": 1
})
db.assets.createIndex({
  "thumbnailLastCheck": 1
})
db.assets.createIndex({
  "thumbnailFailedReason": 1
})
db.assets.createIndex({
  "projectMetadata.latest.source": 1
})
db.assets.createIndex({
  "projectMetadata.latest.thumbnailURL": 1
})
db.assets.createIndex({
  "projectMetadata.latest.lastUpdatedAt": 1
})

db.accounts.createIndex({
  account: 1
}, {
  unique: true,
  sparse: true
})

db.account_tokens.createIndex({
  indexID: 1,
  ownerAccount: 1
}, {
  unique: true,
  sparse: true
})

db.account_tokens.createIndexes([{
  "lastRefreshedTime": -1
}, {
  "lastActivityTime": -1
}])

db.identities.createIndex({
  accountNumber: 1,
  blockchain: 1
}, {
  unique: true,
  sparse: true
})

db.createView("token_assets", "tokens", [
  {
    "$lookup": {
      "from": "assets",
      "localField": "assetID",
      "foreignField": "id",
      "as": "asset"
    }
  },
  { "$unwind": "$asset" },
  {
    "$addFields": {
    "asset.metadata.project": "$asset.projectMetadata"
    }
  },
  {
    "$project": {
      "asset._id": 0,
      "asset.projectMetadata": 0,
      "owners": 0,
      "ownersArray": 0,
      "_id": 0
    }
  }
])