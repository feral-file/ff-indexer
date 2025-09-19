# FF-Indexer

A comprehensive NFT indexing system that tracks, processes, and provides access to NFT data across multiple blockchains including Ethereum, Tezos, and Bitmark.

## Project Status

⚠️ **This project is currently in active development and may contain bugs.** While the core functionality is operational, some features may be unstable or incomplete. Please refer to the [GitHub Issues](../../issues) for known problems and ongoing development tasks.

## Overview

FF-Indexer is a microservices-based system designed to index NFT data from various blockchain networks and external APIs. It provides real-time event processing, comprehensive metadata indexing, image processing, and a GraphQL API for querying NFT information.

## What It Solves

- **Multi-blockchain NFT Indexing**: Supports Ethereum, Tezos, and Bitmark blockchains
- **Real-time Event Processing**: Captures and processes NFT transfer, mint, and burn events as they happen
- **Cross-Marketplace Sales Data**: Aggregates NFT sales data from multiple marketplaces including OpenSea, Blur, X2Y2, OBJKT, FxHash, etc.,
- **Comprehensive Metadata Management**: Fetches and stores NFT metadata from IPFS and external sources
- **Image Processing**: Downloads, processes, and generates thumbnails for NFT images
- **Provenance Tracking**: Maintains complete ownership history and transaction records
- **Exchange Rate Tracking**: Historical price data for various cryptocurrency pairs
- **GraphQL API**: Flexible querying interface for NFT data

## Architecture

The system consists of several microservices:

- **API Gateway**: REST and GraphQL API endpoints
- **Event Processor**: Processes blockchain events and coordinates workflows
- **Event Emitters**: Monitor blockchain networks for NFT events (Ethereum & Tezos)
- **Workflow Runner**: Executes background indexing workflows using Cadence
- **GRPC Gateway**: Internal service communication
- **Image Indexer**: Processes and stores NFT images
- **Provenance Indexer**: Tracks ownership history and provenance

## Quick Start

### Prerequisites

- Go 1.21+
- MongoDB
- PostgreSQL (for image indexer)
- Docker & Docker Compose
- Cadence (Temporal workflow engine)

### Using Docker Compose

```bash
# Start all services
docker compose up --build -d
```

### Manual Setup

1. **Clone and build**:
```bash
git clone git@github.com:feral-file/ff-indexer.git
cd ff-indexer
make build
```

2. **Configure services**:
```bash
# Copy sample configs for each service
cp services/api-gateway/config.yaml.sample services/api-gateway/config.yaml
# Repeat for other services and update with your settings
```

3. **Run individual services**:
```bash
make run-api-gateway
make run-event-processor
make run-workflow-runner
# etc.
```

## Key Features

- **Multi-blockchain Support**: Ethereum, Tezos, and Bitmark
- **Real-time Processing**: Event-driven architecture with WebSocket connections
- **Scalable Architecture**: Microservices with workflow orchestration
- **Image Processing**: Automatic thumbnail generation and CDN integration
- **GraphQL API**: Flexible data querying with strong typing
- **Comprehensive Testing**: Unit tests and integration testing
- **Docker Support**: Full containerization for easy deployment

## Documentation

- [Development Guide](DEVELOPMENT.md) - Detailed setup, architecture, and development information
- [System Diagrams](DIAGRAM.md) - Architecture diagrams and component interactions
- API Documentation - TBD

## Configuration

Each service has its own configuration file. Sample configurations are provided in each service directory:

- `services/api-gateway/config.yaml.sample`
- `services/event-processor/config.yaml.sample` 
- `services/workflow-runner/config.yaml.sample`
- And more...

Key configuration areas:
- Database connections (MongoDB, PostgreSQL)
- Blockchain RPC endpoints
- External API keys (OpenSea, TZKT, etc.)
- IPFS gateways
- AWS services integration

## API Usage

The system provides both REST and GraphQL endpoints:

**REST API** (v2 endpoints):
```bash
# Get account NFTs
GET /v2/nft?owner=<address>

# Query NFTs with filters
POST /v2/nft/query

# Get collections by creators
GET /v2/collections?creators=<creator-addresses>
```

**GraphQL**:
```bash
# Access GraphQL playground
GET /v2/graphiql
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite: `make test`
6. Run linter: `make vet`
7. Submit a pull request

### Code Style

- Follow Go conventions and best practices
- Use meaningful variable and function names
- Add comments for complex logic
- Write unit tests for new features

## Testing

```bash
# Run all tests
make test

# Run linter
make vet

# Run specific service tests
go test ./services/api-gateway/...
```

## Known Issues

Please refer to the [GitHub Issues](../../issues) for current known problems, bug reports, and feature requests.

## License

This project is licensed under the Mozilla Public License Version 2.0. See the [LICENSE](LICENSE) file for details.

## Support

For issues and questions:
1. Check existing [GitHub Issues](../../issues)
2. Create a new issue with detailed description
3. Include relevant logs and configuration (without sensitive data)

## Acknowledgments

Built with:
- [Cadence](https://cadenceworkflow.io/) - Workflow orchestration
- [MongoDB](https://www.mongodb.com/) - Primary data storage
- [Gin](https://gin-gonic.com/) - HTTP web framework
- [gqlgen](https://gqlgen.com/) - GraphQL server generation
- [TZKT](https://tzkt.io/) - Tezos blockchain data
- [OpenSea API](https://docs.opensea.io/) - Ethereum NFT data
