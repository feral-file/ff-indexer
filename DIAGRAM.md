# System Architecture Diagram

This document provides a visual representation of the FF-Indexer system architecture, showing how all components interact and integrate with each other.

## System Architecture

```mermaid
flowchart TD
    %% External Sources
    subgraph EXT["External Sources"]
        BLOCKCHAIN[Blockchain Networks<br/>Ethereum, Tezos, Bitmark]
        MARKETPLACES[Marketplace APIs<br/>OpenSea, OBJKT, etc.]
        IPFS[IPFS Network<br/>Metadata & Images]
    end

    %% Event Detection
    subgraph EVENTS["Event Detection"]
        EEE[Ethereum Event Emitter]
        TEE[Tezos Event Emitter]
    end

    %% Core Processing
    subgraph CORE["Core Processing"]
        EP[Event Processor]
        WR[Workflow Runner]
        AG[API Gateway]
        GG[gRPC Gateway]
    end

    %% Background Services
    subgraph BG["Background Services"]
        II[Image Indexer]
        PI[Provenance Indexer]
    end

    %% Storage
    subgraph DATA["Storage"]
        MONGO[(MongoDB<br/>Token Data)]
        PG_EVENTS[(PostgreSQL<br/>Event Queue)]
        PG_IMAGES[(PostgreSQL<br/>Image Metadata)]
        CACHE[Cache Store]
    end

    %% Infrastructure
    subgraph INFRA["Infrastructure"]
        CADENCE[Cadence/Temporal<br/>Workflow Engine]
        CLOUDFLARE[Cloudflare Images<br/>CDN Storage]
    end

    %% Clients
    subgraph CLIENTS["Client Applications"]
        WEB[Web Applications]
        MOBILE[Mobile Applications]
        API_CLIENTS[API Clients]
    end

    %% Main Event Flow
    BLOCKCHAIN --> EEE
    BLOCKCHAIN --> TEE
    EEE --> EP
    TEE --> EP
    EP --> PG_EVENTS
    EP --> WR
    
    %% Workflow Runner (uses Cadence)
    WR <--> CADENCE
    WR --> MONGO
    MARKETPLACES --> WR
    IPFS --> WR
    
    %% Provenance Indexer (uses Cadence, different task list)
    PI <--> CADENCE
    PI --> MONGO
    
    %% Image Indexer (standalone, no Cadence)
    II --> MONGO
    II --> PG_IMAGES
    II --> CLOUDFLARE
    IPFS --> II
    
    %% API Layer
    AG --> MONGO
    AG --> CACHE
    
    %% gRPC Gateway (internal service communication)
    EP --> GG
    GG --> MONGO
    
    %% Client Access
    CLIENTS --> AG
    
    %% Styling
    classDef external fill:#e3f2fd,stroke:#1976d2,stroke-width:2px
    classDef processing fill:#e8f5e8,stroke:#388e3c,stroke-width:2px
    classDef storage fill:#fff3e0,stroke:#f57c00,stroke-width:2px
    classDef infrastructure fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    classDef client fill:#fce4ec,stroke:#c2185b,stroke-width:2px
    
    class BLOCKCHAIN,MARKETPLACES,IPFS external
    class EEE,TEE,EP,WR,AG,GG,II,PI processing
    class MONGO,PG_EVENTS,PG_IMAGES,CACHE storage
    class CADENCE,CLOUDFLARE infrastructure
    class WEB,MOBILE,API_CLIENTS client
```

## How It Works

The FF-Indexer system processes NFT data through several layers:

**External Sources**: Blockchain networks emit NFT events, while marketplace APIs and IPFS provide additional metadata and images.

**Event Detection**: Event emitters monitor blockchain networks and capture NFT events in real-time.

**Core Processing**: 
- **Event Processor** handles incoming events, stores them in PostgreSQL event queue, and triggers workflows
- **Workflow Runner** coordinates background indexing tasks using Cadence workflows
- **API Gateway** serves data directly to external clients
- **gRPC Gateway** handles internal service-to-service communication

**Background Services**: 
- **Provenance Indexer** tracks ownership history using Cadence workflows (different task list from Workflow Runner)
- **Image Indexer** operates independently (no Cadence), processes images from IPFS, stores metadata in PostgreSQL, and uploads to Cloudflare

**Storage**: 
- **MongoDB** stores main token-related data (tokens, assets, artists, sales)
- **PostgreSQL Event Queue** stores blockchain events for processing
- **PostgreSQL Image Metadata** stores image processing information
- **Cache Store** improves API performance

**Infrastructure**: 
- **Cadence/Temporal** manages workflow orchestration for Workflow Runner and Provenance Indexer
- **Cloudflare Images** provides CDN storage for processed images

**Client Applications**: Web applications, mobile apps, and API clients consume the indexed NFT data through the API Gateway.
