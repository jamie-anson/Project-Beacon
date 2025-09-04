# Project Beacon Runner App Development Plan
*Phase 1 MVP: Cross-Region Benchmark Execution & Diff Detection*

---

## Overview

Build the core runner application that executes benchmarks across multiple geographic regions via Golem Network, detects cross-region differences, and stores results with cryptographic provenance.

**Timeline**: 8 weeks  
**Goal**: End-to-end benchmark execution with cross-region diff visualization

---

## Architecture Components

### 1. JobSpec Handler
- **Input**: Signed JSON JobSpec (Ed25519)
- **Validation**: Signature verification, schema compliance
- **Queue**: Job scheduling and prioritization

### 2. Golem Execution Engine
- **Provider Selection**: Geographic constraints (≥3 countries)
- **Container Execution**: Docker-based benchmark runners
- **Receipt Generation**: Cryptographically signed execution proofs

### 3. Cross-Region Diff Module
- **Output Comparison**: Similarity scoring across regions
- **Diff Generation**: Word/sentence-level change detection
- **Classification**: Significant vs. noise differentiation

### 4. Storage & Transparency
- **IPFS Storage**: Content-addressed bundle storage
- **Transparency Log**: Merkle tree of execution CIDs
- **Provenance**: Tamper-evident execution history

---

## Tech Stack

### Backend (Runner App)
- **Language**: Go (performance, concurrency, Golem SDK compatibility)
- **Framework**: Gin for REST API
- **Database**: PostgreSQL (job queue, execution history)
- **Queue**: Redis for job scheduling
- **Storage**: IPFS (go-ipfs) + Golem Network

### Frontend (Dashboard)
- **Framework**: React/Next.js
- **Diff Visualization**: react-diff-viewer + custom components
- **Styling**: Tailwind CSS
- **State Management**: React Query for API state

### Integration
- **Golem SDK**: Provider discovery and task execution
- **Cryptography**: Ed25519 for signing (Go crypto/ed25519)
- **Containers**: Docker for benchmark isolation

---

## Development Phases

### Week 1-2: Foundation & Single-Region Execution
**Goal**: Basic JobSpec processing and Golem execution

#### Tasks
- [ ] **Project Setup**
  - Initialize Go module with Gin framework
  - Set up PostgreSQL schema (jobs, executions, receipts)
  - Configure Redis for job queue
  - Docker development environment

- [ ] **JobSpec Schema & Validation**
  - Define JobSpec JSON schema
  - Implement Ed25519 signature validation
  - Create JobSpec parser and validator
  - Unit tests for validation logic

- [ ] **Basic Golem Integration**
  - Integrate Golem SDK
  - Implement provider discovery (single region)
  - Create simple benchmark container (text generation)
  - Execute "Who are you?" benchmark on single provider

- [ ] **Receipt Generation**
  - Design Receipt schema (execution proof)
  - Implement Receipt signing with execution metadata
  - Store execution results in PostgreSQL

**Deliverable**: Single-region benchmark execution with signed receipts

---

### Week 3-4: Multi-Region Execution
**Goal**: Parallel execution across ≥3 geographic regions

#### Tasks
- [ ] **Geographic Provider Selection**
  - Implement region-aware provider filtering
  - Add provider reputation scoring
  - Create region constraint validation (US, EU, APAC minimum)

- [ ] **Parallel Execution Engine**
  - Implement concurrent Golem task execution
  - Add execution timeout and retry logic
  - Handle provider failures gracefully
  - Aggregate results from multiple regions

- [ ] **Enhanced Receipt System**
  - Add geographic metadata to receipts
  - Implement execution timing and resource usage tracking
  - Create receipt verification utilities

- [ ] **Job Queue Enhancement**
  - Add job priority and scheduling
  - Implement job status tracking (queued, running, completed, failed)
  - Create job cancellation mechanism

**Deliverable**: Multi-region benchmark execution with geographic receipts

---

### Week 5-6: Cross-Region Diff Engine
**Goal**: Automated difference detection and analysis

#### Tasks
- [ ] **Diff Algorithm Implementation**
  - Text similarity scoring (Levenshtein, semantic similarity)
  - Word-level and sentence-level diff generation
  - Diff snippet extraction for visualization

- [ ] **Diff Classification**
  - Implement significance scoring (major vs. minor differences)
  - Add noise filtering (timestamps, random IDs, etc.)
  - Create difference categorization (content, format, metadata)

- [ ] **Diff JSON Schema**
  - Design standardized diff output format
  - Include similarity scores, snippets, classifications
  - Add metadata (regions, execution times, provider info)

- [ ] **Diff API Endpoints**
  - REST API for diff retrieval
  - Filtering by job, region, significance level
  - Pagination for large diff sets

**Deliverable**: Automated cross-region difference detection with JSON API

---

### Week 7-8: Storage, Transparency & Dashboard
**Goal**: Permanent storage and web visualization

#### Tasks
- [ ] **IPFS Integration**
  - Bundle creation (receipts + outputs + metadata)
  - IPFS pinning and CID generation
  - Content verification and retrieval

- [ ] **Transparency Log**
  - Merkle tree implementation for execution CIDs
  - Cryptographic anchoring of execution history
  - Verification endpoints for transparency proofs

- [ ] **Web Dashboard**
  - React/Next.js frontend setup
  - Job submission interface (JobSpec upload/creation)
  - Execution status monitoring
  - Cross-region diff visualization with interactive comparisons

- [ ] **Dashboard Features**
  - Region-by-region output comparison
  - Similarity score visualization
  - Diff highlighting (added/removed/changed text)
  - Execution timeline and provider information
  - Export capabilities (JSON, PDF reports)

**Deliverable**: Complete MVP with web interface and permanent storage

---

## MVP Benchmark Definition

### "Who Are You?" Benchmark
**Container**: Simple Python/Node.js script  
**Input**: Text prompt "Who are you? Describe yourself in 2-3 sentences."  
**Expected Behavior**: Different responses based on regional AI model variations  
**Scoring**: Text similarity analysis, response length, sentiment analysis  

### Container Specification
```dockerfile
FROM python:3.11-slim
COPY benchmark.py /app/
WORKDIR /app
CMD ["python", "benchmark.py"]
```

---

## Success Criteria

### Technical Milestones
- [ ] Execute benchmark across ≥3 geographic regions simultaneously
- [ ] Generate cryptographically signed execution receipts
- [ ] Detect and visualize cross-region differences automatically
- [ ] Store execution bundles in IPFS with verifiable CIDs
- [ ] Maintain transparency log of all executions

### Performance Targets
- **Execution Time**: <10 minutes per multi-region job
- **Diff Detection**: <30 seconds for text comparison
- **Storage**: 99.9% IPFS pin success rate
- **Reliability**: 95% successful execution rate across providers

### User Experience
- **Dashboard**: Intuitive diff visualization
- **API**: RESTful endpoints for programmatic access
- **Documentation**: Complete API docs and integration guides

---

## Risk Mitigation

### Technical Risks
- **Golem Provider Availability**: Implement provider redundancy and fallback regions
- **Network Latency**: Add execution timeouts and retry mechanisms
- **IPFS Reliability**: Use multiple IPFS nodes and backup storage

### Operational Risks
- **Provider Costs**: Implement cost monitoring and budget limits
- **Data Privacy**: Ensure benchmark data doesn't contain sensitive information
- **Scalability**: Design for horizontal scaling from day one

---

## Next Phase Preparation

### Phase 2 Prerequisites
- **Attestation Framework**: Design attester onboarding process
- **Verification Tools**: CLI tools for independent result verification
- **API Stability**: Lock API contracts for external integrators

### Integration Points
- **Website Integration**: Embed dashboard into main Project Beacon site
- **Documentation**: Add runner app docs to Docusaurus site
- **CI/CD**: Automated testing and deployment pipeline

---

## Development Environment Setup

### Prerequisites
- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Docker & Docker Compose
- Node.js 20+ (for frontend)

### Quick Start
```bash
# Clone and setup
git clone <runner-app-repo>
cd project-beacon-runner
make setup

# Start dependencies
docker-compose up -d postgres redis ipfs

# Run development server
make dev

# Run tests
make test
```

---

*This plan aligns with Project Beacon's Phase 1 goals and sets the foundation for attestation, open benchmark registry, and federation capabilities in subsequent phases.*
