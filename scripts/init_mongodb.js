// Generated 2025-09-24T03:36:00.000Z from DB: nft_indexer
// Replays collections (with options), views, and indexes (no data)
use('nft_indexer');

// Collection: historical_exchange_rates
if (!db.getCollectionNames().includes('historical_exchange_rates')) {
  db.createCollection('historical_exchange_rates', {});
}

// Collection: block_caches
if (!db.getCollectionNames().includes('block_caches')) {
  db.createCollection('block_caches', {});
}

// Collection: collection_assets
if (!db.getCollectionNames().includes('collection_assets')) {
  db.createCollection('collection_assets', {});
}

// Collection: sales_time_series
if (!db.getCollectionNames().includes('sales_time_series')) {
  db.createCollection('sales_time_series', {
    timeseries: {
      timeField: 'timestamp',
      metaField: 'metadata',
      granularity: 'seconds',
      bucketMaxSpanSeconds: 3600,
    },
  });
}

// Collection: accounts
if (!db.getCollectionNames().includes('accounts')) {
  db.createCollection('accounts', {});
}

// Collection: tokens
if (!db.getCollectionNames().includes('tokens')) {
  db.createCollection('tokens', {});
}

// Collection: ff_identities
if (!db.getCollectionNames().includes('ff_identities')) {
  db.createCollection('ff_identities', {});
}

// Collection: identities
if (!db.getCollectionNames().includes('identities')) {
  db.createCollection('identities', {});
}

// Collection: assets
if (!db.getCollectionNames().includes('assets')) {
  db.createCollection('assets', {});
}

// Collection: collections
if (!db.getCollectionNames().includes('collections')) {
  db.createCollection('collections', {});
}

// Collection: token_feedbacks
if (!db.getCollectionNames().includes('token_feedbacks')) {
  db.createCollection('token_feedbacks', {});
}

// Collection: account_tokens
if (!db.getCollectionNames().includes('account_tokens')) {
  db.createCollection('account_tokens', {});
}

// Collection: asset_static_preview_url
if (!db.getCollectionNames().includes('asset_static_preview_url')) {
  db.createCollection('asset_static_preview_url', {});
}

// View: token_assets
if (!db.getCollectionInfos({ name: 'token_assets' }).length) {
  db.createCollection('token_assets', {
    viewOn: 'tokens',
    pipeline: [
      {
        $lookup: {
          from: 'assets',
          localField: 'assetID',
          foreignField: 'id',
          as: 'asset',
        },
      },
      { $unwind: '$asset' },
      {
        $lookup: {
          from: 'asset_static_preview_url',
          localField: 'assetID',
          foreignField: 'assetID',
          as: 'staticPreviewURL',
        },
      },
      {
        $unwind: {
          path: '$staticPreviewURL',
          preserveNullAndEmptyArrays: true,
        },
      },
      {
        $addFields: {
          'asset.metadata.project': '$asset.projectMetadata',
          'asset.staticPreviewURLLandscape': '$staticPreviewURL.landscapeURL',
          'asset.staticPreviewURLPortrait': '$staticPreviewURL.portraitURL',
        },
      },
      {
        $project: {
          'asset._id': 0,
          'asset.projectMetadata': 0,
          ownersArray: 0,
          _id: 0,
          staticPreviewURL: 0,
        },
      },
    ],
  });
}

// Indexes for historical_exchange_rates
db.getCollection('historical_exchange_rates').createIndex(
  { timestamp: 1, currencyPair: 1 },
  { name: 'timestamp_1_currencyPair_1', unique: true }
);

// Indexes for collection_assets
db.getCollection('collection_assets').createIndex(
  { collectionID: 1 },
  { name: 'collectionID_1' }
);
db.getCollection('collection_assets').createIndex(
  { edition: 1 },
  { name: 'edition_1' }
);
db.getCollection('collection_assets').createIndex(
  { lastActivityTime: -1 },
  { name: 'lastActivityTime_-1' }
);
db.getCollection('collection_assets').createIndex(
  { tokenIndexID: 1 },
  { name: 'tokenIndexID_1' }
);

// Indexes for sales_time_series
db.getCollection('sales_time_series').createIndex(
  { 'metadata.blockchain': 1 },
  { name: 'metadata.blockchain_1' }
);
db.getCollection('sales_time_series').createIndex(
  { 'metadata.bundleTokenInfo.contractAddress': 1 },
  { name: 'metadata.bundleTokenInfo.contractAddress_1' }
);
db.getCollection('sales_time_series').createIndex(
  { 'metadata.bundleTokenInfo.tokenID': 1 },
  { name: 'metadata.bundleTokenInfo.tokenID_1' }
);
db.getCollection('sales_time_series').createIndex(
  { 'metadata.bundleTokenInfo.buyerAddress': 1 },
  { name: 'metadata.bundleTokenInfo.buyerAddress_1' }
);
db.getCollection('sales_time_series').createIndex(
  { 'metadata.bundleTokenInfo.sellerAddress': 1 },
  { name: 'metadata.bundleTokenInfo.sellerAddress_1' }
);
db.getCollection('sales_time_series').createIndex(
  { 'metadata.marketplace': 1 },
  { name: 'metadata.marketplace_1' }
);
db.getCollection('sales_time_series').createIndex(
  { 'metadata.saleType': 1 },
  { name: 'metadata.saleType_1' }
);
db.getCollection('sales_time_series').createIndex(
  { 'metadata.transactionIDs': 1 },
  { name: 'metadata.transactionIDs_1' }
);

// Indexes for accounts
db.getCollection('accounts').createIndex(
  { account: 1 },
  { name: 'account_1', unique: true, sparse: true }
);

// Indexes for tokens
db.getCollection('tokens').createIndex(
  { indexID: 1 },
  { name: 'indexID_1', unique: true, sparse: true }
);
db.getCollection('tokens').createIndex({ id: 1 }, { name: 'id_1' });
db.getCollection('tokens').createIndex(
  { ownersArray: 1 },
  { name: 'ownersArray_1' }
);
db.getCollection('tokens').createIndex({ swapped: 1 }, { name: 'swapped_1' });
db.getCollection('tokens').createIndex(
  { contractAddress: 1 },
  { name: 'contractAddress_1' }
);
db.getCollection('tokens').createIndex({ owner: 1 }, { name: 'owner_1' });
db.getCollection('tokens').createIndex({ owners: 1 }, { name: 'owners_1' });
db.getCollection('tokens').createIndex(
  { lastRefreshedTime: 1 },
  { name: 'lastRefreshedTime_1' }
);
db.getCollection('tokens').createIndex(
  { blockchain: 1 },
  { name: 'blockchain_1' }
);
db.getCollection('tokens').createIndex({ isDemo: 1 }, { name: 'isDemo_1' });
db.getCollection('tokens').createIndex({ assetID: 1 }, { name: 'assetID_1' });
db.getCollection('tokens').createIndex(
  { lastRefreshedTime: -1 },
  { name: 'lastRefreshedTime_-1' }
);
db.getCollection('tokens').createIndex(
  { LastRefreshedTime: -1 },
  { name: 'LastRefreshedTime_-1' }
);
db.getCollection('tokens').createIndex({ fungible: 1 }, { name: 'fungible_1' });
db.getCollection('tokens').createIndex({ source: 1 }, { name: 'source_1' });

// Indexes for ff_identities
db.getCollection('ff_identities').createIndex(
  { accountNumber: 1 },
  { name: 'accountNumber_1', unique: true }
);

// Indexes for identities
db.getCollection('identities').createIndex(
  { accountNumber: 1 },
  { name: 'accountNumber_1', unique: true }
);
db.getCollection('identities').createIndex(
  { accountNumber: 1, blockchain: 1 },
  { name: 'accountNumber_1_blockchain_1', unique: true, sparse: true }
);

// Indexes for assets
db.getCollection('assets').createIndex(
  { 'projectMetadata.latest.medium': 1 },
  { name: 'projectMetadata.latest.medium_1' }
);
db.getCollection('assets').createIndex({ id: 1 }, { name: 'id_1' });
db.getCollection('assets').createIndex(
  { 'projectMetadata.latest.mimeType': 1 },
  { name: 'projectMetadata.latest.mimeType_1' }
);
db.getCollection('assets').createIndex(
  { 'projectMetadata.latest.source': 1 },
  { name: 'projectMetadata.latest.source_1' }
);
db.getCollection('assets').createIndex(
  { 'projectMetadata.latest.lastUpdatedAt': 1 },
  { name: 'projectMetadata.latest.lastUpdatedAt_1' }
);
db.getCollection('assets').createIndex(
  { indexID: 1 },
  { name: 'indexID_1', unique: true, sparse: true }
);
db.getCollection('assets').createIndex(
  { thumbnailID: 1 },
  { name: 'thumbnailID_1' }
);
db.getCollection('assets').createIndex(
  { thumbnailLastCheck: 1 },
  { name: 'thumbnailLastCheck_1' }
);
db.getCollection('assets').createIndex({ source: 1 }, { name: 'source_1' });
db.getCollection('assets').createIndex(
  { 'projectMetadata.latest.thumbnailURL': 1 },
  { name: 'projectMetadata.latest.thumbnailURL_1' }
);
db.getCollection('assets').createIndex(
  { 'projectMetadata.latest.lastUpdatedAt': -1 },
  { name: 'projectMetadata.latest.lastUpdatedAt_-1' }
);
db.getCollection('assets').createIndex(
  { thumbnailLastCheck: -1 },
  { name: 'thumbnailLastCheck_-1' }
);
db.getCollection('assets').createIndex(
  { thumbnailFailedReason: 1 },
  { name: 'thumbnailFailedReason_1' }
);
db.getCollection('assets').createIndex(
  { 'projectMetadata.latest.galleryThumbnailURL': 1 },
  { name: 'projectMetadata.latest.galleryThumbnailURL_1' }
);
db.getCollection('assets').createIndex(
  { 'projectMetadata.latest.previewURL': 1 },
  { name: 'projectMetadata.latest.previewURL_1' }
);
db.getCollection('assets').createIndex(
  { lastRefreshedTime: -1 },
  { name: 'lastRefreshedTime_-1' }
);
db.getCollection('assets').createIndex(
  { 'projectMetadata.latest.title': 1 },
  { name: 'projectMetadata.latest.title_1' }
);
db.getCollection('assets').createIndex(
  { 'projectMetadata.origin.artistID': 1 },
  { name: 'projectMetadata.origin.artistID_1' }
);
db.getCollection('assets').createIndex(
  { 'projectMetadata.latest.artistID': 1 },
  { name: 'projectMetadata.latest.artistID_1' }
);
db.getCollection('assets').createIndex(
  { 'projectMetadata.latest.description': 1 },
  { name: 'projectMetadata.latest.description_1' }
);
db.getCollection('assets').createIndex(
  { 'projectMetadata.latest.artists.id': 1 },
  { name: 'projectMetadata.latest.artists.id_1' }
);
db.getCollection('assets').createIndex(
  { 'projectMetadata.latest.artists.url': 1 },
  { name: 'projectMetadata.latest.artists.url_1' }
);
db.getCollection('assets').createIndex(
  { 'projectMetadata.origin.assetID': 1 },
  { name: 'projectMetadata.origin.assetID_1' }
);
db.getCollection('assets').createIndex(
  { 'projectMetadata.latest.assetID': 1 },
  { name: 'projectMetadata.latest.assetID_1' }
);
db.getCollection('assets').createIndex(
  { 'projectMetadata.origin.title': 1 },
  { name: 'projectMetadata.origin.title_1', sparse: true }
);
db.getCollection('assets').createIndex(
  { 'projectMetadata.origin.artistName': 1 },
  { name: 'projectMetadata.origin.artistName_1'}
);
db.getCollection('assets').createIndex(
  { 'projectMetadata.latest.artistName': 1 },
  { name: 'projectMetadata.latest.artistName_1' }
);

// Indexes for collections
db.getCollection('collections').createIndex(
  { creators: 1, lastActivityTime: -1 },
  { name: 'creators_1_lastActivityTime_-1' }
);
db.getCollection('collections').createIndex(
  { creators: 1 },
  { name: 'creators_1' }
);
db.getCollection('collections').createIndex({ id: 1 }, { name: 'id_1' });
db.getCollection('collections').createIndex(
  { lastActivityTime: -1 },
  { name: 'lastActivityTime_-1' }
);
db.getCollection('collections').createIndex(
  { lastUpdatedTime: -1 },
  { name: 'lastUpdatedTime_-1' }
);

// Indexes for account_tokens
db.getCollection('account_tokens').createIndex(
  { lastRefreshedTime: -1 },
  { name: 'lastRefreshedTime_-1' }
);
db.getCollection('account_tokens').createIndex(
  { ownerAccount: 1, lastActivityTime: -1 },
  { name: 'ownerAccount_1_lastActivityTime_-1' }
);
db.getCollection('account_tokens').createIndex(
  { indexID: 1, ownerAccount: 1 },
  { name: 'indexID_1_ownerAccount_1', unique: true, sparse: true }
);
db.getCollection('account_tokens').createIndex(
  { ownerAccount: 1, runID: 1 },
  { name: 'ownerAccount_1_runID_1' }
);
db.getCollection('account_tokens').createIndex(
  { ownerAccount: 1 },
  { name: 'ownerAccount_1' }
);
db.getCollection('account_tokens').createIndex(
  { blockchain: 1 },
  { name: 'blockchain_1' }
);
db.getCollection('account_tokens').createIndex(
  { lastActivityTime: -1 },
  { name: 'lastActivityTime_-1' }
);
db.getCollection('account_tokens').createIndex(
  { indexID: 1, ownerAccount: 1, lastActivityTime: -1 },
  { name: 'indexID_1_ownerAccount_1_lastActivityTime_-1' }
);
db.getCollection('account_tokens').createIndex(
  { contractAddress: 1 },
  { name: 'contractAddress_1' }
);
db.getCollection('account_tokens').createIndex(
  { lastUpdatedAt: 1 },
  { name: 'lastUpdatedAt_1' }
);
db.getCollection('account_tokens').createIndex(
  { lastPendingTime: -1 },
  { name: 'lastPendingTime_-1' }
);
db.getCollection('account_tokens').createIndex(
  { pendingTxs: 1 },
  { name: 'pendingTxs_1' }
);

// Indexes for asset_static_preview_url
db.getCollection('asset_static_preview_url').createIndex(
  { assetID: 1 },
  { name: 'assetID_1', unique: true }
);
