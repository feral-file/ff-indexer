# Development Guide

This document provides comprehensive information for developers working on the FF-Indexer project.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Project Structure](#project-structure)
- [Services Overview](#services-overview)
- [Local Development Setup](#local-development-setup)
- [Configuration](#configuration)
- [Building and Running](#building-and-running)
- [Testing](#testing)
- [Docker Development](#docker-development)
- [Makefile Usage](#makefile-usage)
- [Architecture Details](#architecture-details)
- [Database Schema](#database-schema)
- [API Development](#api-development)
- [Workflow Development](#workflow-development)
- [Monitoring and Logging](#monitoring-and-logging)
- [Deployment](#deployment)

## Prerequisites

### System Requirements

- **Go**: Version 1.21 or higher
- **MongoDB**: Version 7.0 or higher
- **PostgreSQL**: Version 13 or higher
- **Make**: GNU Make or compatible

### External Dependencies

- **Cadence/Temporal**: Workflow orchestration engine
- **Ethereum RPC**: Access to Ethereum network (Infura, Alchemy, etc.)
- **Tezos RPC**: Access to Tezos network
- **AWS Services**: S3, Parameter Store, Managed Blockchain Query
- **External APIs**:
  - OpenSea API
  - TZKT API
  - FxHash API
  - Various IPFS gateways

### Development Tools

- **Protocol Buffers**: For gRPC code generation
- **gqlgen**: For GraphQL code generation
- **golangci-lint**: For code linting

## Project Structure

```
ff-indexer/
├── background/worker/          # Background workflow activities and workflows
├── cache/                      # Cache store implementation
├── cadence/                    # Cadence/Temporal client setup
├── contracts/                  # Smart contract bindings
├── externals/                  # External API clients
│   ├── coinbase/              # Coinbase API client
│   ├── ens/                   # Ethereum Name Service client
│   ├── etherscan/             # Etherscan API client
│   ├── fxhash/                # FxHash API client
│   ├── objkt/                 # Objkt API client
│   ├── opensea/               # OpenSea API client
│   └── tezos-domain/          # Tezos domain client
├── protos/                    # Protocol buffer definitions
├── scripts/                   # Database and deployment scripts
├── sdk/                       # SDKs for REST and GRPC communication
├── services/                  # Microservices
│   ├── api-gateway/           # REST and GraphQL API
│   ├── event-processor/       # Event processing service
│   ├── ethereum-event-emitter/# Ethereum blockchain monitor
│   ├── grpc-gateway/          # gRPC service
│   ├── image-indexer/         # Image processing service
│   ├── provenance-indexer/    # Provenance tracking service
│   ├── tezos-event-emitter/   # Tezos blockchain monitor
│   └── workflow-runner/       # Cadence workflow worker
├── traceutils/                # Tracing and monitoring utilities
├── constants.go               # Global constants
├── engine.go                  # Core indexing engine
├── indexer*.go                # Blockchain-specific indexers
├── store.go                   # Database interfaces
├── structs.go                 # Core data structures
└── utils*.go                  # Utility functions
```

## Services Overview

### API Gateway (`services/api-gateway/`)

**Responsibility**: Provides REST and GraphQL endpoints for external clients.

**Key Features**:
- RESTful API for NFT queries and indexing
- GraphQL API with comprehensive schema
- API token authentication
- Rate limiting and CORS handling
- Health check endpoints

**Dependencies**:
- MongoDB (indexer store)
- MongoDB (cache store)
- Cadence worker client
- Ethereum client
- ENS client
- Tezos domain client

**Configuration**: `services/api-gateway/config.yaml.sample`

### Event Processor (`services/event-processor/`)

**Responsibility**: Processes blockchain events and coordinates downstream actions.

**Key Features**:
- Multi-stage event processing pipeline
- NFT transfer, mint, burn event handling
- Series registry event processing
- Notification system integration
- gRPC server for receiving events

**Event Processing Stages**:
1. **Stage 1**: Update latest owner
2. **Stage 2**: Trigger full token updates and provenance
3. **Stage 3**: Send notifications for ownership changes
4. **Stage 4**: Send events to feed server
5. **Stage 5**: Index token sales

**Dependencies**:
- PostgreSQL (event queue)
- MongoDB (indexer store)
- gRPC gateway client
- Cadence worker client
- Notification service

### Event Emitters

#### Ethereum Event Emitter (`services/ethereum-event-emitter/`)

**Responsibility**: Monitors Ethereum blockchain for NFT-related events.

**Key Features**:
- WebSocket connection to Ethereum node
- ERC-721 and ERC-1155 transfer event monitoring
- Series registry contract event monitoring
- Automatic reconnection and error handling
- State persistence via AWS Parameter Store

#### Tezos Event Emitter (`services/tezos-event-emitter/`)

**Responsibility**: Monitors Tezos blockchain for NFT-related events.

**Key Features**:
- TZKT WebSocket API integration
- Token transfer monitoring
- BigMap updates for metadata changes
- Historical event processing from last stopped block

### Workflow Runner (`services/workflow-runner/`)

**Responsibility**: Executes background indexing workflows using Cadence.

**Key Workflows**:
- `IndexETHTokenWorkflow`: Index Ethereum tokens by owner
- `IndexTezosTokenWorkflow`: Index Tezos tokens by owner
- `IndexTokenWorkflow`: Generic token indexing
- `IndexEthereumTokenSale`: Process Ethereum token sales
- `IndexTezosTokenSale`: Process Tezos token sales
- `CrawlHistoricalExchangeRate`: Fetch historical exchange rates

**Key Activities**:
- `IndexToken`: Core token indexing logic
- `CacheArtifact`: IPFS artifact caching
- `RefreshTokenProvenance`: Update token provenance
- `GetTokenByIndexID`: Retrieve token data
- `IndexAccountTokens`: Update account token balances

### GRPC Gateway (`services/grpc-gateway/`)

**Responsibility**: Provides gRPC interface for internal service communication.

**Key Methods**:
- `GetTokenByIndexID`: Retrieve token by index ID
- `PushProvenance`: Update token provenance
- `UpdateOwner`: Update token ownership
- `IndexAccountTokens`: Index account tokens
- `GetSaleTimeSeries`: Retrieve sales data
- `GetHistoricalExchangeRate`: Get exchange rate data

### Image Indexer (`services/image-indexer/`)

**Responsibility**: Downloads, processes, and stores NFT images.

**Key Features**:
- Chrome/Chromium-based image rendering
- Thumbnail generation (multiple sizes)
- Cloudflare Images integration
- Image processing with FFmpeg
- PostgreSQL metadata storage
- Retry mechanism for failed downloads

**Image Processing Pipeline**:
1. Download source image/video from IPFS or HTTP
2. Generate thumbnails using Chrome headless
3. Upload to Cloudflare Images
4. Store metadata in PostgreSQL
5. Update MongoDB with image URLs

### Provenance Indexer (`services/provenance-indexer/`)

**Responsibility**: Tracks and indexes NFT ownership history and provenance.

**Key Features**:
- Historical ownership tracking
- Transaction provenance
- Cross-blockchain provenance support
- Integration with external data sources

## Local Development Setup
TBD


## Configuration

Each service has its own configuration file. Sample configurations are provided in each service directory for reference:

- `services/api-gateway/config.yaml.sample`
- `services/event-processor/config.yaml.sample` 
- `services/workflow-runner/config.yaml.sample`
- `services/grpc-gateway/config.yaml.sample`
- `services/image-indexer/config.yaml.sample`
- `services/provenance-indexer/config.yaml.sample`
- `services/ethereum-event-emitter/config.yaml.sample`
- `services/tezos-event-emitter/config.yaml.sample`

Copy the sample files and update them with your specific configuration values for database connections, API keys, and external service endpoints.

## Building and Running

### Using Make

The project includes comprehensive Makefile targets:

```bash
# Build all services
make build

# Build specific service
make api-gateway
make workflow-runner
make event-processor

# Run services locally
make run-api-gateway
make run-workflow-runner
make run-event-processor

# Generate code
make generate-api-gateway-graphql
make generate-event-processor-grpc
make generate-gateway-grpc

# Clean build artifacts
make clean
```

### Manual Building

```bash
# Build API Gateway
go build -o bin/api-gateway ./services/api-gateway

# Build with specific tags
go build -tags development -o bin/api-gateway ./services/api-gateway

# Cross-compilation for Linux
GOOS=linux GOARCH=amd64 go build -o bin/api-gateway-linux ./services/api-gateway
```

### Running Services

**Development Mode**:
```bash
# Run with custom config
./bin/api-gateway -c custom-config.yaml

# Run with environment variables
NFT_INDEXER_DEBUG=true ./bin/api-gateway
```

**Production Mode**:
```bash
# Run with production config
./bin/api-gateway -c production-config.yaml

# Run as systemd service
sudo systemctl start ff-indexer-api-gateway
```

## Testing

### Unit Tests

```bash
# Run all tests
make test

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./services/api-gateway/...

# Run tests with race detection
go test -race ./...

# Run tests with verbose output
go test -v ./services/event-processor/...
```

### Integration Tests

```bash
# Run integration tests (requires external services)
go test -tags integration ./...

# Run specific integration test
go test -tags integration ./services/api-gateway/ -run TestAPIIntegration
```

### Benchmark Tests

```bash
# Run benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkIndexToken ./background/worker/
```

### Test Database Setup

```bash
# Set up test databases
docker run -d --name test-mongodb -p 27018:27017 mongo:7.0
docker run -d --name test-postgres -p 5433:5432 -e POSTGRES_PASSWORD=test postgres:12

# Use test configuration
export NFT_INDEXER_STORE_DB_URI="mongodb://localhost:27018/"
export NFT_INDEXER_STORE_DB_NAME="nft_indexer_test"
```

## Docker Development

### Building Docker Images

```bash
# Build API Gateway image
make build-api-gateway dist=dev

# Build all images
make image dist=dev GITHUB_USER=your-user GITHUB_TOKEN=your-token

# Build with custom arguments
docker build -f Dockerfile-api-gateway \
  --build-arg GITHUB_USER=your_user \
  --build-arg GITHUB_TOKEN=your_token \
  -t ff-indexer:api-dev .
```

### Docker Compose Development
TBD

## Makefile Usage

### Build Targets

| Target | Description |
|--------|-------------|
| `make build` | Build all services |
| `make api-gateway` | Build API Gateway |
| `make workflow-runner` | Build Workflow Runner |
| `make event-processor` | Build Event Processor |
| `make grpc-gateway` | Build gRPC Gateway |
| `make image-indexer` | Build Image Indexer |
| `make provenance-indexer` | Build Provenance Indexer |
| `make ethereum-event-emitter` | Build Ethereum Event Emitter |
| `make tezos-event-emitter` | Build Tezos Event Emitter |

### Run Targets

| Target | Description |
|--------|-------------|
| `make run-api-gateway` | Run API Gateway locally |
| `make run-workflow-runner` | Run Workflow Runner locally |
| `make run-event-processor` | Run Event Processor locally |
| `make run-grpc-gateway` | Run gRPC Gateway locally |
| `make run-image-indexer` | Run Image Indexer locally |
| `make run-provenance-indexer` | Run Provenance Indexer locally |
| `make run-ethereum-event-emitter` | Run Ethereum Event Emitter locally |
| `make run-tezos-event-emitter` | Run Tezos Event Emitter locally |

### Docker Targets

| Target | Description |
|--------|-------------|
| `make build-api-gateway dist=<version>` | Build API Gateway Docker image |
| `make build-workflow-runner dist=<version>` | Build Workflow Runner Docker image |
| `make build-event-processor dist=<version>` | Build Event Processor Docker image |
| `make image dist=<version>` | Build all Docker images |
| `make push dist=<version>` | Push images to registry |

### Code Generation Targets

| Target | Description |
|--------|-------------|
| `make generate-api-gateway-graphql` | Generate GraphQL code |
| `make generate-event-processor-grpc` | Generate Event Processor gRPC code |
| `make generate-gateway-grpc` | Generate Gateway gRPC code |
| `make build-rebuild` | Regenerate all code and build |

### Utility Targets

| Target | Description |
|--------|-------------|
| `make test` | Run all tests |
| `make vet` | Run Go vet linter |
| `make clean` | Clean build artifacts |
| `make complete-clean` | Clean all caches |
| `make config` | Copy sample config files |
| `make help` | Show available targets |

### Dependency Management

| Target | Description |
|--------|-------------|
| `make list-updates` | List available dependency updates |
| `make update-patch-level` | Update patch-level dependencies |
| `make update-full` | Update all dependencies |
| `make update-picker` | Interactive dependency updates |

## Architecture Details

### Data Flow

1. **Event Detection**: Event emitters monitor blockchain networks
2. **Event Processing**: Events are queued and processed through multiple stages
3. **Workflow Execution**: Background workflows handle complex indexing tasks
4. **Data Storage**: Processed data is stored in MongoDB and PostgreSQL
5. **API Access**: Clients access data through REST and GraphQL APIs

### Communication Patterns

- **Event-Driven**: Services communicate through events and message queues
- **Workflow Orchestration**: Complex operations use Cadence workflows
- **gRPC**: Internal service-to-service communication
- **REST/GraphQL**: External client communication
- **WebSocket**: Real-time blockchain event monitoring

### Scalability Considerations

- **Horizontal Scaling**: Services can be scaled independently
- **Database Sharding**: MongoDB collections can be sharded by blockchain
- **Caching**: Multiple caching layers (Redis, Cloudflare, IPFS)
- **Rate Limiting**: API endpoints have configurable rate limits
- **Circuit Breakers**: External API calls use circuit breaker patterns

## Database Schema
TBD

## API Development

### GraphQL Schema Development

The GraphQL schema is defined in `services/api-gateway/graph/schema.graphqls`:

```graphql
type Token {
  id: String!
  blockchain: String!
  fungible: Boolean!
  contractType: String!
  contractAddress: String!

  edition: Int64!
  editionName: String!
  mintAt: Time
  mintedAt: Time
  balance: Int64!
  owner: String!
  owners: [Owner!]
  originTokenInfo: [BaseTokenInfo!]

  ...
}

type Query {
  tokens(
    owners: [String!]! = []
    ids: [String!]! = []
    collectionID: String! = ""
    source: String! = ""
    lastUpdatedAt: Time
    burnedIncluded: Boolean! = false
    sortBy: String
    offset: Int64! = 0
    size: Int64! = 50
  ): [Token!]!
  identity(account: String!): Identity
  ethBlockTime(blockHash: String!): BlockTime
  collections(
    creators: [String!]! = []
    offset: Int64! = 0
    size: Int64! = 50
  ): [Collection!]!
  collection(id: String!): Collection
}

type Mutation {
  indexHistory(indexID: String!): Boolean!
  indexCollection(creators: [String!]!): Boolean!
}
```

Generate GraphQL code:
```bash
cd services/api-gateway/graph
go run github.com/99designs/gqlgen generate
```

### REST API Development

REST endpoints are defined in `services/api-gateway/routes.go`:

```go
func (s *Server) SetupRoute() {
    // NFT endpoints
    s.route.GET("/nft", s.ListNFTs)
    s.route.GET("/nft/search", s.SearchNFTs)
    s.route.POST("/nft/index", s.IndexNFTs)
    s.route.POST("/nft/:token_id/provenance", s.RefreshProvenance)
    
    // Collection endpoints
    v2Collections := s.route.Group("/v2/collections")
    v2Collections.GET("", s.GetCollectionsByCreators)
    v2Collections.GET("/:collection_id", s.GetCollectionByID)
}
```

### Adding New Endpoints

1. **Define the route**:
```go
s.route.GET("/nft/analytics", s.GetNFTAnalytics)
```

2. **Implement the handler**:
```go
func (s *Server) GetNFTAnalytics(c *gin.Context) {
    // Implementation
}
```

3. **Add tests**:
```go
func TestGetNFTAnalytics(t *testing.T) {
    // Test implementation
}
```

## Workflow Development

### Creating New Workflows

Workflows are defined in `background/worker/workflows_*.go`:

```go
func (w *Worker) MyNewWorkflow(ctx workflow.Context, input MyWorkflowInput) error {
    ctx = ContextRegularActivity(ctx, TaskListName)
    
    // Execute activities
    var result MyActivityResult
    err := workflow.ExecuteActivity(
        ContextRetryActivity(ctx, ""),
        w.MyActivity,
        input.Param1,
        input.Param2,
    ).Get(ctx, &result)
    
    if err != nil {
        return err
    }
    
    // Continue workflow logic
    return nil
}
```

### Creating New Activities

Activities are defined in `background/worker/activities_*.go`:

```go
func (w *Worker) MyActivity(ctx context.Context, param1 string, param2 int) (MyActivityResult, error) {
    // Activity implementation
    result := MyActivityResult{
        // Set result fields
    }
    
    return result, nil
}
```

### Registering Workflows and Activities

In `services/workflow-runner/main.go`:

```go
// Register workflows
workflow.Register(worker.MyNewWorkflow)

// Register activities
activity.Register(worker.MyActivity)
```

### Workflow Best Practices

1. **Deterministic Execution**: Workflows must be deterministic
2. **Activity Timeouts**: Set appropriate timeouts for activities
3. **Error Handling**: Handle errors gracefully with retries
4. **Versioning**: Use workflow versioning for backward compatibility
5. **Testing**: Write unit tests for workflows and activities

## Monitoring and Logging

### Logging

The project uses structured logging with Zap:

```go
import (
    log "github.com/bitmark-inc/autonomy-logger"
    "go.uber.org/zap"
)

// Info logging
log.InfoWithContext(ctx, "Processing token", 
    zap.String("tokenID", tokenID),
    zap.String("blockchain", blockchain))

// Error logging
log.ErrorWithContext(ctx, errors.New("failed to process token"),
    zap.Error(err),
    zap.String("tokenID", tokenID))

// Debug logging
log.Debug("Debug information", zap.Any("data", data))
```

### Metrics and Monitoring

**Health Checks**:
```bash
# API Gateway health
curl http://localhost:8080/healthz

# gRPC Gateway health
grpcurl -plaintext localhost:8888 grpc.health.v1.Health/Check
```

**Sentry Integration**:
```go
if err := log.Initialize(viper.GetBool("debug"), &sentry.ClientOptions{
    Dsn:         viper.GetString("sentry.dsn"),
    Environment: environment,
}); err != nil {
    panic(err)
}
```

### Tracing

HTTP request tracing is available in `traceutils/`:

```go
import "github.com/feral-file/ff-indexer/traceutils"

// Add tracing to HTTP handlers
handler := traceutils.TraceHTTP(originalHandler)
```

## Deployment

### Docker Deployment

**Build images**:
```bash
make image dist=v1.0.0 GITHUB_USER=your_user GITHUB_TOKEN=your_token
```

**Push to registry**:
```bash
make push dist=v1.0.0
```

**Deploy with Docker Compose**:
```yaml
version: '3.8'
services:
  api-gateway:
    image: your-registry/ff-indexer:api-v1.0.0
    ports:
      - "8080:8080"
    environment:
      - NFT_INDEXER_ENVIRONMENT=production
    volumes:
      - ./config/api-gateway.yaml:/config.yaml
```

### Kubernetes Deployment

Create Kubernetes manifests:

```yaml
# api-gateway-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api-gateway
  template:
    metadata:
      labels:
        app: api-gateway
    spec:
      containers:
      - name: api-gateway
        image: your-registry/ff-indexer:api-v1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: NFT_INDEXER_ENVIRONMENT
          value: "production"
        volumeMounts:
        - name: config
          mountPath: /config.yaml
          subPath: config.yaml
      volumes:
      - name: config
        configMap:
          name: api-gateway-config
```

### Production Considerations

1. **Resource Limits**: Set appropriate CPU and memory limits
2. **Health Checks**: Configure liveness and readiness probes
3. **Secrets Management**: Use Kubernetes secrets or external secret managers
4. **Load Balancing**: Use appropriate load balancing strategies
5. **Monitoring**: Set up comprehensive monitoring and alerting
6. **Backup**: Implement database backup strategies
7. **Security**: Use security scanning and vulnerability management

### Debugging
TBD

### Log Analysis
TBD

This development guide provides comprehensive information for working with the FF-Indexer project. For specific issues not covered here, please refer to the project's issue tracker or create a new issue with detailed information about your problem.
