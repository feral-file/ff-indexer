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
  "projectMetadata.latest.source": 1
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

db.identities.createIndex({
  accountNumber: 1,
  blockchain: 1
}, {
  unique: true,
  sparse: true
})
