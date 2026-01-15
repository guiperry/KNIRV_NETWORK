# KNIRVGRAPH - Decentralized Knowledge Graph with Economics

A comprehensive graph-based blockchain application featuring Network Resolution Vectors (NRV), Proof-of-Solution consensus, NRN token economics, and seamless integration with the KNIRV ecosystem.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    KNIRVGRAPH Layer                        │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │ NRV System  │ │ ErrorNodes  │ │   SkillNodes        │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │ Economics   │ │ Proof-of-   │ │   Graph Explorer    │   │
│  │ Integration │ │ Solution    │ │   (React)           │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ HTTP/REST API + Economics
                              │
┌─────────────────────────────────────────────────────────────┐
│                    Backend (Go)                            │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │ RPC Server  │ │ GraphChain  │ │    NRV System       │   │
│  │ +Economics  │ │    Core     │ │   (Vectors)         │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │   Storage   │ │ NRN Token   │ │   KNIRVORACLE         │   │
│  │  (BluntDB)  │ │ Integration │ │   Integration       │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Features

### 🧠 Network Resolution Vector (NRV) System
- **Vector Management**: Create, resolve, and maintain NetworkResolutionVectors
- **ErrorNodes**: Specialized nodes for tracking and resolving network errors
- **SkillNodes**: Capability-based nodes for AI agent skills and solutions
- **Confidence Scoring**: Cryptographic confidence verification for resolutions
- **Signature Verification**: Secure validation of vector authenticity

### 💰 Economics Integration
- **NRN Token Integration**: Seamless integration with KNIRVORACLE for NRN operations
- **Skill Confirmation**: Skill validation for KNIRVCHAIN commitment
- **Proof-of-Solution**: Reward distribution based on solution quality and efficiency
- **Economic Metrics**: Real-time tracking of network economic activity
- **Reward Distribution**: Automated NRN rewards for network participation

## 🧪 Phase 7 Testing Infrastructure

### Testing Architecture
KNIRVGRAPH implements comprehensive testing for both backend (Go) and frontend (React) components:

#### Frontend Testing (React + TypeScript)
```bash
# Run frontend tests
npm test

# Run tests with coverage
npm run test:coverage

# Run tests in watch mode
npm run test:watch

# Lint code
npm run lint
```

#### Backend Testing (Go)
```bash
# Run Go backend tests
go test -v ./...

# Run specific test packages
go test -v ./internal/...
go test -v ./pkg/...
go test -v ./cmd/...
```

#### Integration with Network Tests
```bash
# From project root - run KNIRVGRAPH network integration tests
cd integration-tests && go test -v -run TestKNIRVGRAPH

# Run comprehensive KNIRVGRAPH test suite
make test-graph

# Run KNIRVGRAPH comprehensive tests
cd KNIRVGRAPH && ./scripts/run-comprehensive-tests.sh
```

#### Test Structure
```
src/
├── __tests__/          # Frontend unit tests
│   └── App.test.tsx
├── test/              # Test utilities and setup
│   └── setup.ts
└── components/        # Component tests alongside source

backend/
├── internal/          # Go backend tests
├── pkg/              # Package tests
└── cmd/              # Command tests
```

### 🔗 KNIRV Ecosystem Integration
- **KNIRVORACLE Connectivity**: Direct integration with KNIRV blockchain
- **Cross-Component Communication**: Seamless interaction with other KNIRV services
- **Unified Authentication**: Shared authentication across KNIRV ecosystem
- **Real-time Synchronization**: Live updates and state synchronization

### 🏗️ Backend (Go)
- **Graph Consensus**: Byzantine fault-tolerant consensus adapted for graph structures
- **BluntDB Storage**: High-performance graph data persistence with NRV support
- **REST API**: Complete graph data access with economics endpoints
- **P2P Networking**: Distributed node communication
- **CLI Tools**: Command-line interface for graph and NRV interaction
- **Graph Operations**: Node creation, edge management, path finding
- **Multi-dimensional Relationships**: Complex node interconnections with NRV vectors

### 🎨 Frontend (React)
- **Real-time Dashboard**: Live graph and economics statistics
- **NRV Explorer**: Browse and manage Network Resolution Vectors
- **ErrorNode Management**: Track and resolve network errors
- **SkillNode Registry**: Manage AI capabilities and skills
- **Economics Dashboard**: Monitor NRN token flows and rewards
- **Graph Visualization**: Interactive graph structure display with NRV overlays
- **Dark Theme**: Modern UI inspired by ai.google.dev
- **Responsive Design**: Mobile-friendly interface

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- npm or yarn
- KNIRVORACLE running (for economics integration)

### Development Setup

1. **Clone and setup backend**
   ```bash
   # Install Go dependencies
   cd KNIRVGRAPH
   go mod tidy

   # Build KNIRVGRAPH node
   go build -o bin/knirvgraph ./cmd/node/main.go
   ```

2. **Setup frontend**
   ```bash
   # Install frontend dependencies
   npm install
   ```

3. **Start KNIRVGRAPH with Economics Integration**
   ```bash
   # From project root - starts KNIRVGRAPH with full economics integration
   ./scripts/start-knirvgraph-with-economics.sh
   ```

4. **Alternative: Manual startup**
   ```bash
   # Terminal 1: Start KNIRVGRAPH node
   cd KNIRVGRAPH
   ./bin/knirvgraph --home ~/.knirvgraph --port 8081

   # Terminal 2: Start frontend dev server
   npm run dev
   ```

5. **Access the application**
   - Frontend: http://localhost:5173
   - Backend API: http://localhost:8081
   - Economics Endpoints: http://localhost:8081/economics/
   - NRV System: http://localhost:8081/nrv/

### Using the CLI

```bash
# Build CLI tool
make build

# Get GraphChain height
./build/graphchain-cli height

# Get node information
./build/graphchain-cli node genesis

# Get edge information
./build/graphchain-cli edge edge123

# Get graph heads
./build/graphchain-cli heads

# Get node neighbors
./build/graphchain-cli neighbors node1

# Find path between nodes
./build/graphchain-cli path node1 node2

# Create a new node
./build/graphchain-cli create-node --id node1 --parents genesis --weight 0.8

# Create a new edge
./build/graphchain-cli create-edge --from node1 --to node2 --weight 0.5

# Get account balance
./build/graphchain-cli account 0x1234567890123456789012345678901234567890

# Send graph transaction
./build/graphchain-cli send-tx \
  --from 0x1234567890123456789012345678901234567890 \
  --to 0x0987654321098765432109876543210987654321 \
  --amount 100 \
  --fee 1 \
  --type 5
```

## API Endpoints

### Graph REST API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/height` | Get current GraphChain height |
| GET | `/node/{nodeID}` | Get node by ID |
| GET | `/edge/{edgeID}` | Get edge by ID |
| GET | `/graph/heads` | Get current graph heads |
| GET | `/graph/neighbors/{nodeID}` | Get node neighbors |
| GET | `/graph/path/{from}/{to}` | Find path between nodes |
| POST | `/graph/traverse` | Execute graph traversal query |
| GET | `/account/{address}` | Get account information |
| POST | `/transaction` | Submit new graph transaction |
| POST | `/node` | Create new graph node |
| POST | `/edge` | Create new graph edge |

### NRV System API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/nrv/vectors` | Get all NRV vectors |
| POST | `/nrv/vectors` | Create new NRV vector |
| GET | `/nrv/vectors/resolve/{targetHash}` | Resolve target hash to vectors |
| GET | `/nrv/errors` | Get all error nodes |
| POST | `/nrv/errors` | Create new error node |
| GET | `/nrv/skills` | Get all skill nodes |
| POST | `/nrv/skills` | Create new skill node |
| GET | `/nrv/skills/for-error/{errorType}` | Get skills for specific error type |

### Economics API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/economics/metrics` | Get economic metrics and statistics |
| POST | `/economics/skill/confirm` | Confirm skill for KNIRVCHAIN commitment |
| POST | `/economics/rewards/distribute` | Distribute NRN rewards |
| POST | `/economics/proof/solution` | Submit proof of solution for rewards |

### System API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | System health check with service status |

### Example API Usage

#### Graph Operations
```bash
# Get current height
curl http://localhost:8081/height

# Get node
curl http://localhost:8081/node/genesis

# Get graph heads
curl http://localhost:8081/graph/heads

# Find path between nodes
curl http://localhost:8081/graph/path/node1/node2

# Create node
curl -X POST http://localhost:8081/node \
  -H "Content-Type: application/json" \
  -d '{
    "id": "node1",
    "parents": ["genesis"],
    "weight": 0.8,
    "data": {
      "transactions": [],
      "state_changes": [],
      "edges": []
    }
  }'
```

#### NRV System Operations
```bash
# Create an error node
curl -X POST http://localhost:8081/nrv/errors \
  -H "Content-Type: application/json" \
  -d '{
    "error_type": "network_timeout",
    "description": "Connection timeout to external service",
    "context": {
      "service": "api.example.com",
      "timeout": 30000,
      "retry_count": 3
    },
    "severity": 2
  }'

# Create a skill node
curl -X POST http://localhost:8081/nrv/skills \
  -H "Content-Type: application/json" \
  -d '{
    "skill_type": "network_diagnostics",
    "capabilities": ["ping_test", "traceroute", "dns_lookup"],
    "requirements": {
      "min_confidence": 0.8,
      "network_access": true
    }
  }'

# Create NRV vector
curl -X POST http://localhost:8081/nrv/vectors \
  -H "Content-Type: application/json" \
  -d '{
    "target_hash": "abc123def456",
    "coordinates": [0.1, 0.2, 0.3, 0.4, 0.5],
    "metadata": {
      "error_type": "network_timeout",
      "skill_type": "network_diagnostics"
    }
  }'

# Get all error nodes
curl http://localhost:8081/nrv/errors

# Get skills for specific error type
curl http://localhost:8081/nrv/skills/for-error/network_timeout
```

#### Economics Operations
```bash
# Get economic metrics
curl http://localhost:8081/economics/metrics

# Confirm a skill for KNIRVCHAIN commitment
curl -X POST http://localhost:8081/economics/skill/confirm \
  -H "Content-Type: application/json" \
  -d '{
    "skill_id": "skill456",
    "nrv_id": "nrv789",
    "creator_id": "creator123"
  }'

# Distribute rewards
curl -X POST http://localhost:8081/economics/rewards/distribute \
  -H "Content-Type: application/json" \
  -d '{
    "recipient_id": "solver123",
    "amount": "500000",
    "reason": "successful_error_resolution"
  }'

# Submit solution proof
curl -X POST http://localhost:8081/economics/proof/solution \
  -H "Content-Type: application/json" \
  -d '{
    "error_node_id": "error123",
    "skill_node_id": "skill456",
    "solver_id": "solver789",
    "efficiency_score": 0.95,
    "quality_score": 0.88
  }'
```

#### System Health
```bash
# Check system health
curl http://localhost:8081/health
```

## Testing Economics Integration

### Automated Testing
```bash
# Run comprehensive economics integration tests
./scripts/test-knirvgraph-economics.sh

# This will test:
# - KNIRVGRAPH connectivity
# - KNIRVORACLE integration
# - NRV system operations
# - Economics endpoints
# - Skill confirmation for KNIRVCHAIN commitment
# - Reward distribution
# - Proof-of-Solution mechanism
```

### Manual Testing Workflow
```bash
# 1. Start KNIRVGRAPH with economics
./scripts/start-knirvgraph-with-economics.sh

# 2. In another terminal, test the integration
./scripts/test-knirvgraph-economics.sh

# 3. Monitor logs
tail -f KNIRVGRAPH/logs/knirvgraph.log
```

### Environment Configuration
```bash
# Set KNIRVORACLE URL (default: http://localhost:1317)
export KNIRVORACLE_URL=http://localhost:1317

# Set KNIRVGRAPH port (default: 8081)
export KNIRVGRAPH_PORT=8081

# Enable/disable economics (default: true)
export ECONOMICS_ENABLED=true
```

## Project Structure

```
KNIRVGRAPH/
├── cmd/                    # Command-line applications
│   ├── node/              # KNIRVGRAPH node with NRV and economics
│   └── cli/               # CLI tool for graph operations
├── internal/              # Private application code
│   ├── app/               # Application logic with economics integration
│   ├── graphchain/        # GraphChain core
│   ├── nrv/               # Network Resolution Vector system
│   ├── economics/         # NRN token integration and Proof-of-Solution
│   │   ├── nrn_integration.go    # KNIRVORACLE integration
│   │   └── proof_of_solution.go  # Proof-of-Solution consensus
│   ├── network/           # Networking and RPC with economics endpoints
│   ├── storage/           # Data persistence with NRV support
│   └── types/             # Data structures for graph and NRV
├── pkg/                   # Public libraries
│   ├── crypto/            # Cryptographic functions
│   └── utils/             # Utility functions
├── config/                # Configuration files
├── docs/                  # Documentation
│   └── NRV_SYSTEM.md     # NRV system documentation
├── src/                   # React frontend
│   ├── components/        # React components with NRV and economics
│   ├── pages/             # Page components
│   ├── services/          # API services for graph, NRV, and economics
│   └── context/           # React context
├── public/                # Static assets
├── bin/                   # Built binaries
├── logs/                  # Application logs
├── go.mod                 # Go module definition
├── package.json           # Node.js dependencies
├── Makefile              # Build automation
└── README.md             # This file
```

## Development

### Backend Development

```bash
# Run tests
make test

# Format code
make fmt

# Lint code
make lint

# Build binaries
make build

# Clean build artifacts
make clean
```

### Frontend Development

```bash
# Start development server
npm run dev

# Build for production
npm run build

# Run linting
npm run lint
```

### Full Stack Development

```bash
# Start both backend and frontend
make dev
```

### Graph Testing

```bash
# Create genesis graph
make create-genesis-graph

# Test graph operations
make test-graph
```

## Configuration

### Backend Configuration

Edit `config/default.toml`:

```toml
[network]
listen_address = "tcp://0.0.0.0:26656"
seeds = []
max_peers = 50

[consensus]
timeout_propose = 3000
timeout_propose_delta = 500
timeout_prevote = 1000
timeout_precommit = 1000

[storage]
db_path = "./data/graphchain.db"

[rpc]
listen_address = "tcp://0.0.0.0:26657"
port = 8080

[graph]
max_nodes_per_level = 1000
max_edges_per_node = 100
allow_cycles = false
consensus_threshold = 0.67
traversal_depth_limit = 50
max_heads = 10
weight_decay_factor = 0.95

[bluntdb]
db_path = "./data/graphchain.db"
sync_writes = true
value_log_max_entries = 1000000
gc_interval = 300
backup_interval = 3600
```

### Frontend Configuration

The frontend automatically connects to the backend API at `http://localhost:8080`. To change this, update `src/services/api.ts`.

## Testing

### Backend Tests
```bash
make test
```

### Frontend Tests
```bash
npm test
```

### Graph Operations Tests
```bash
# Test node creation
./build/graphchain-cli create-node --id test1 --weight 1.0

# Test edge creation
./build/graphchain-cli create-edge --from test1 --to genesis --weight 0.5

# Test path finding
./build/graphchain-cli path test1 genesis
```

## Production Deployment

1. **Build production assets**
   ```bash
   make prod
   ```

2. **Deploy backend**
   ```bash
   # Copy binary to server
   scp build/graphchain-node user@server:/usr/local/bin/
   
   # Start service
   graphchain-node -home /var/lib/graphchain -rpc-port 8080
   ```

3. **Deploy frontend**
   ```bash
   # Build and deploy static files
   npm run build
   # Deploy dist/ folder to web server
   ```

## Graph Features

### Node Operations
- **Create Nodes**: Add new nodes with parent relationships
- **Update Nodes**: Modify node properties and relationships
- **Delete Nodes**: Remove nodes and update graph structure
- **Query Nodes**: Retrieve node information and metadata

### Edge Operations
- **Create Edges**: Establish weighted connections between nodes
- **Update Edges**: Modify edge weights and properties
- **Delete Edges**: Remove connections while maintaining graph integrity
- **Query Edges**: Retrieve edge information and traversal data

### Graph Traversal
- **Path Finding**: Discover routes between any two nodes
- **Neighbor Discovery**: Find all connected nodes
- **Subgraph Extraction**: Extract portions of the graph
- **Cycle Detection**: Identify and prevent circular dependencies

### Consensus Features
- **Graph Validation**: Ensure graph structure integrity
- **Multi-dimensional Consensus**: Consensus across graph topology
- **Weight-based Validation**: Consensus based on node and edge weights
- **Byzantine Fault Tolerance**: Maintain consistency despite failures

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Support

- Open issues on GitHub
- Check the documentation
- Contact the development team

## Roadmap

### ✅ Completed (August 2025)
- [x] **NRV System Implementation** - Complete Network Resolution Vector system
- [x] **Economics Integration** - Full NRN token integration with KNIRVORACLE
- [x] **Proof-of-Solution** - Reward mechanism for error resolution
- [x] **ErrorNodes & SkillNodes** - Specialized node types for AI capabilities
- [x] **KNIRV Ecosystem Integration** - Seamless integration with other KNIRV components
- [x] **Economic Metrics Tracking** - Real-time economics monitoring
- [x] **Automated Testing** - Comprehensive test suite for economics integration

### 🔄 In Progress
- [ ] **Enhanced Graph Visualization** - NRV-aware graph visualization tools
- [ ] **Advanced NRV Algorithms** - Improved vector resolution and confidence scoring
- [ ] **Cross-Component DHT** - Enhanced DHT integration with other KNIRV services

### 📋 Planned
- [ ] **Graph Smart Contracts** - Smart contract layer for graph operations
- [ ] **Enhanced Graph Consensus** - Multi-dimensional consensus mechanisms
- [ ] **Cross-Graph Bridges** - Inter-graph communication protocols
- [ ] **Advanced Graph Algorithms** - Machine learning-enhanced graph operations
- [ ] **Mobile Applications** - Mobile interface for KNIRVGRAPH
- [ ] **Graph Analytics Dashboard** - Advanced analytics and insights
- [ ] **Multi-Language Support** - SDK support for multiple programming languages
- [ ] **Real-time Collaboration** - Multi-user graph editing and collaboration
- [ ] **Graph Machine Learning** - AI-powered graph analysis and prediction