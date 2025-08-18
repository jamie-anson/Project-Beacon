# Project Beacon Runner

Cross-region benchmark execution engine for detecting AI model differences across geographic regions.

## Overview

The Project Beacon Runner executes benchmarks across multiple geographic regions via the Golem Network, detects cross-region differences, and stores results with cryptographic provenance.

## Architecture

- **JobSpec Handler**: Validates and processes signed benchmark specifications
- **Golem Execution Engine**: Manages multi-region container execution
- **Cross-Region Diff Module**: Detects and analyzes output differences
- **Storage & Transparency**: IPFS storage with Merkle tree transparency log

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+

### Setup

```bash
# Clone the repository
git clone <repository-url>
cd runner-app

# Install dependencies
make setup

# Start supporting services
make docker-up

# Run development server
make dev
```

The API will be available at `http://localhost:8090`

### API Endpoints

- `GET /health` - Health check
- `POST /api/v1/jobs` - Create new benchmark job
- `GET /api/v1/jobs` - List all jobs
- `GET /api/v1/jobs/:id` - Get specific job
- `POST /api/v1/jobs/:id/execute` - Execute job across regions
- `GET /api/v1/jobs/:id/executions` - Get job executions
- `GET /api/v1/jobs/:id/diffs` - Get cross-region diffs

## Development

### Project Structure

```
├── cmd/runner/          # Main application entry point
├── internal/
│   ├── api/            # HTTP API handlers
│   ├── jobspec/        # JobSpec validation and processing
│   ├── golem/          # Golem Network integration
│   │   └── client/     # Extracted Yagna REST client (interfaces + impl)
│   ├── diff/           # Cross-region difference detection
│   ├── storage/        # IPFS and database storage
│   └── db/             # Database models and migrations
├── pkg/
│   ├── crypto/         # Cryptographic utilities (Ed25519)
│   └── models/         # Shared data models
├── web/                # Frontend dashboard (React)
├── docker/             # Docker configurations
└── scripts/            # Utility scripts
```

### Running Tests

```bash
make test
```

### Building

```bash
make build
```

### Docker Development

```bash
# Start all services including the runner
make docker-full

# Stop all services
make docker-down
```

## Configuration

Environment variables:

- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string  
- `IPFS_API_URL` - IPFS API endpoint
- `GOLEM_API_KEY` - Golem Network API key
- `PORT` - HTTP server port (default: 8090)

## JobSpec Format

```json
{
  "id": "benchmark-001",
  "version": "1.0",
  "benchmark": {
    "name": "Who Are You?",
    "description": "Text generation benchmark",
    "container": {
      "image": "beacon/text-gen",
      "tag": "latest",
      "resources": {
        "cpu": "1000m",
        "memory": "512Mi"
      }
    },
    "input": {
      "type": "prompt",
      "data": {
        "prompt": "Who are you? Describe yourself in 2-3 sentences."
      }
    }
  },
  "constraints": {
    "regions": ["US", "EU", "APAC"],
    "min_regions": 3,
    "timeout": "10m"
  },
  "signature": "...",
  "public_key": "..."
}
```

## Phase 1 Development Status

### Week 1-2: Foundation ✅
- [x] Project setup with Go, Gin, PostgreSQL
- [x] Basic API structure and database schema
- [x] Docker development environment
- [x] JobSpec and Receipt data models
- [ ] Ed25519 signature validation
- [ ] Basic Golem integration

### Week 3-4: Multi-Region Execution
- [ ] Geographic provider selection
- [ ] Parallel execution engine
- [ ] Enhanced receipt system
- [ ] Job queue management

### Week 5-6: Cross-Region Diff Engine
- [ ] Diff algorithm implementation
- [ ] Diff classification and scoring
- [ ] Diff JSON schema and API
- [ ] Automated difference detection

### Week 7-8: Storage & Dashboard
- [ ] IPFS integration and pinning
- [ ] Transparency log implementation
- [ ] React dashboard for diff visualization
- [ ] Export and reporting features

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

[License details to be added]

## Support

For questions and support, please open an issue or contact the Project Beacon team.
