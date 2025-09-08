# Meilisearch Token Streaming Integration

This document describes the Meilisearch integration for streaming NFT token data and ownership information, enabling users to search for any token in their collection across ERC721, ERC1155, and FA2 standards.

## Overview

The Meilisearch streaming system is designed to:
- Stream token data and current ownership to a Meilisearch server
- Support search across multiple blockchain standards (ERC721, ERC1155, FA2)
- Allow users to query tokens by passing their addresses
- Provide the latest metadata for each token
- Use Cadence workflows for scalable, fault-tolerant processing

## Architecture

### Components

1. **Data Models** (`meilisearch_structs.go`)
   - `MeilisearchTokenDocument`: The document structure indexed in Meilisearch
   - `MeilisearchStreamConfig`: Configuration for Meilisearch operations
   - `MeilisearchStreamRequest`: Request structure for streaming operations

2. **Activities** (`background/worker/activities_meilisearch.go`)
   - `CreateOrUpdateMeilisearchIndex`: Creates/updates Meilisearch index with search-optimized settings
   - `BatchIndexTokensToMeilisearch`: Indexes batches of tokens to Meilisearch
   - `GetTokensForAddresses`: Retrieves tokens for given addresses
   - `CountTokensForAddresses`: Counts total tokens for progress tracking

### Search Configuration

The Meilisearch index is configured with a focus on meaningful, human-readable search:

**Searchable Attributes** (what users can search):
- `title`: Token/artwork title
- `description`: Token/artwork description  
- `artistName`: Artist or creator name
- `collectionName`: Collection or series name
- `tags`: Descriptive tags or categories
- `medium`: Artistic medium (digital, photography, etc.)

**Design Philosophy**: Technical identifiers like token IDs, contract addresses, owner addresses, MIME types, and cryptographic hashes are intentionally excluded from searchable attributes. These provide no meaningful search value to users but remain available for precise filtering.

3. **Workflows** (`background/worker/workflows-meilisearch.go`)
   - `StreamTokensToMeilisearchWorkflow`: Main parent workflow orchestrating the streaming process
   - `ProcessTokenBatchToMeilisearchWorkflow`: Child workflow processing token batches
   - `RefreshTokensInMeilisearchWorkflow`: Workflow for refreshing specific tokens

4. **Registration**
   - Background worker (`services/nft-indexer-background/main.go`) registers Meilisearch workflows and activities:
     - Workflows: `StreamTokensToMeilisearchWorkflow`, `ProcessTokenBatchToMeilisearchWorkflow`, `RefreshTokensInMeilisearchWorkflow`, `DeleteBurnedTokensFromMeilisearchWorkflow`
     - Activities: `CreateOrUpdateMeilisearchIndex`, `BatchIndexTokensToMeilisearch`, `GetTokensForAddresses`, `CountTokensForAddresses`, `DeleteBurnedTokensFromMeilisearch`, `WaitForMeilisearchTask`, `GetDetailedTokensV2`
   - Provenance worker (`services/nft-provenance-indexer/main.go`) registers `UpdateTokenOwnershipInMeilisearch` so ownership refresh updates Meilisearch.

## Usage

The HTTP API routes for Meilisearch have been removed. Invoke workflows directly via Cadence CLI.

### Start streaming tokens (by addresses)

```bash
cadence --do <domain> --address <host:port> workflow start \
  --tasklist nft-indexer \
  --workflow_type StreamTokensToMeilisearchWorkflow \
  --input '{
    "addresses": [
      "0x1234567890123456789012345678901234567890",
      "tz1LPJ34B1Z8XsxtgoCv5NRBTHTXoeG49A9h"
    ],
    "config": {
      "batchSize": 100,
      "maxConcurrency": 5,
      "updateExisting": true
    },
    "includeHistory": false
  }'
```

### Refresh specific tokens in Meilisearch

```bash
cadence --do <domain> --address <host:port> workflow start \
  --tasklist nft-indexer \
  --workflow_type RefreshTokensInMeilisearchWorkflow \
  --input '["ethereum-...-1", "tezos-...-1"]'
```

## Meilisearch Document Structure

Each token is indexed as a document with the following structure:

```json
{
  "indexID": "ethereum-0x1234567890123456789012345678901234567890-1",
  "tokenID": "1",
  "contractAddress": "0x1234567890123456789012345678901234567890",
  "blockchain": "ethereum",
  "contractType": "ERC721",
  "ownerAddresses": ["0x5678901234567890123456789012345678901234"],
  "ownerBalances": {"0x5678901234567890123456789012345678901234": 1},
  "totalSupply": 1,
  "fungible": false,
  "title": "Cool NFT #1",
  "description": "An amazing NFT",
  "artistName": "Artist Name",
  "artistID": "artist123",
  "medium": "image",
  "mimeType": "image/png",
  "assetURL": "https://example.com/nft.png",
  "thumbnailURL": "https://example.com/thumbnail.png",
  "previewURL": "https://example.com/preview.png",
  "attributes": {"trait1": "value1", "trait2": "value2"},
  "tags": ["art", "collectible"],
  "edition": 1,
  "maxEdition": 100,
  "mintedAt": "2024-01-01T00:00:00Z",
  "lastActivityTime": "2024-01-01T00:00:00Z",
  "lastRefreshedTime": "2024-01-01T00:00:00Z",
  "indexedAt": "2024-01-01T00:00:00Z",
  "burned": false,
  "swapped": false,
  "searchText": "Cool NFT #1 An amazing NFT Artist Name..."
}
```

## Meilisearch Index Settings

The system automatically configures the Meilisearch index with optimized settings:

### Searchable Attributes
- `title`, `description`, `artistName`, `collectionName`
- `searchText`, `tags`, `medium`, `mimeType`

### Filterable Attributes
- `blockchain`, `contractType`, `contractAddress`, `ownerAddresses`
- `fungible`, `burned`, `swapped`, `medium`, `mimeType`, `source`
- `artistID`, `mintedAt`, `lastActivityTime`

### Sortable Attributes
- `mintedAt`, `lastActivityTime`, `lastRefreshedTime`, `indexedAt`
- `edition`, `basePrice`

## Workflow Features

### Parallelism and Performance
- **Concurrent Processing**: Configurable concurrency level (default: 5)
- **Batch Processing**: Configurable batch size (default: 100)
- **Sub-batch Processing**: Further divides large batches for optimal performance
- **Channel-based Communication**: Uses Cadence channels for efficient parallel execution

### Fault Tolerance
- **Retry Logic**: Configurable retry attempts with exponential backoff
- **Error Tracking**: Comprehensive error reporting and categorization
- **Graceful Degradation**: Continues processing even if some batches fail
- **Progress Tracking**: Real-time progress monitoring and statistics

### Scalability
- **Child Workflows**: Uses child workflows to distribute load
- **Continue-as-New**: Prevents workflow history from growing too large
- **Resource Management**: Limits concurrent operations to prevent resource exhaustion

## Configuration Options

### Environment Variables / Config File

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `meilisearch.endpoint` | string | http://localhost:7700 | Meilisearch server endpoint |
| `meilisearch.api_key` | string | - | Meilisearch API key (optional) |
| `meilisearch.index_name` | string | nft-tokens | Index name for tokens |
| `meilisearch.delete_burned` | bool | deprecated | Burned tokens are always excluded and removed |

### MeilisearchStreamConfig (Request Parameters)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `batchSize` | int | 100 | Number of tokens per batch |
| `maxConcurrency` | int | 5 | Maximum concurrent operations |
| `retryAttempts` | int | 3 | Number of retry attempts |
| `retryDelay` | int | 10 | Delay between retries (seconds) |
| `updateExisting` | bool | false | Whether to update existing documents |
| `deleteBurned` | bool | false | Whether to delete burned tokens |

## Monitoring and Observability

The system provides comprehensive logging and metrics:

### Workflow Logs
- Workflow start/completion events
- Batch processing progress
- Error events with context
- Performance metrics

### Result Structure
```json
{
  "totalTokensProcessed": 1000,
  "totalTokensIndexed": 950,
  "totalTokensSkipped": 30,
  "totalTokensErrored": 20,
  "processingTime": "5m30s",
  "batchResults": [...],
  "errors": [...]
}
```

## Search Examples

Once tokens are indexed in Meilisearch, you can perform powerful searches:

### 1. Search by Owner
```json
{
  "filter": "ownerAddresses = '0x1234567890123456789012345678901234567890'"
}
```

### 2. Search by Multiple Owners
```json
{
  "filter": "ownerAddresses IN ['0x123...', '0x456...', 'tz1...']"
}
```

### 3. Full-text Search
```json
{
  "q": "cool art collectible",
  "attributesToSearchOn": ["title", "description", "searchText"]
}
```

### 4. Combined Filters
```json
{
  "q": "art",
  "filter": "ownerAddresses = '0x123...' AND blockchain = 'ethereum' AND burned = false",
  "sort": ["lastActivityTime:desc"]
}
```

## Best Practices

1. **Batch Size**: Use appropriate batch sizes (50-200) based on your Meilisearch server capacity
2. **Concurrency**: Start with low concurrency (3-5) and increase based on performance
3. **Rate Limiting**: Implement rate limiting on the API endpoints
4. **Index Management**: Regularly monitor index size and performance
5. **Error Handling**: Monitor error rates and implement alerting
6. **Data Freshness**: Set up periodic refresh workflows for active tokens

## Troubleshooting

### Common Issues

1. **Connection Errors**: Check Meilisearch endpoint and API key
2. **Rate Limiting**: Reduce batch size and concurrency
3. **Memory Issues**: Increase Meilisearch server resources
4. **Index Corruption**: Recreate the index with fresh data
5. **Workflow Timeouts**: Increase workflow timeout for large datasets

## Burn and Burn-Address Handling

- Burned tokens are not indexed and are removed from Meilisearch on ownership refresh.
- Non-fungible tokens owned by a burn address are skipped. Fungible tokens whose only owner is a burn address are skipped.
- Ownership changes trigger `RefreshTokenOwnershipWorkflow`, which updates Meilisearch via `UpdateTokenOwnershipInMeilisearch` and deletes docs when applicable.

## Index Creation

The index is created if missing with primary key `indexID` before settings are applied. This ensures idempotent upserts when streaming.
