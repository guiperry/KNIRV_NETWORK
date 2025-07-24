# GraphChain Explorer with BluntDB

A comprehensive graph-based chain application featuring a Go-based GraphChain with Tendermint consensus, BluntDB storage, and a beautiful React frontend explorer.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend (React)                        │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │  Dashboard  │ │   Nodes     │ │    Transactions     │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │  Accounts   │ │   Search    │ │   Node Details      │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ HTTP/REST API
                              │
┌─────────────────────────────────────────────────────────────┐
│                    Backend (Go)                            │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │ RPC Server  │ │ GraphChain  │ │    Consensus        │   │
│  │   (REST)    │ │    Core     │ │   (Tendermint)      │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │   Storage   │ │   Network   │ │      Types          │   │
│  │  (BluntDB)  │ │    (P2P)    │ │   (Node/Edge)       │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Features

### Backend (Go)
- **Graph Consensus**: Byzantine fault-tolerant consensus adapted for graph structures
- **State Management**: Account balances and transaction processing
- **P2P Networking**: Distributed node communication
- **REST API**: Complete graph data access
- **BluntDB Storage**: High-performance graph data persistence
- **CLI Tools**: Command-line interface for graph interaction
- **Graph Operations**: Node creation, edge management, path finding
- **Multi-dimensional Relationships**: Complex node interconnections

### Frontend (React)
- **Real-time Dashboard**: Live graph statistics
- **Node Explorer**: Browse and search graph nodes
- **Transaction Viewer**: Detailed transaction analysis
- **Account Lookup**: Balance and transaction history
- **Graph Visualization**: Interactive graph structure display
- **Path Finding**: Visual path discovery between nodes
- **Dark Theme**: Modern UI inspired by ai.google.dev
- **Responsive Design**: Mobile-friendly interface

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- npm or yarn

### Development Setup

1. **Clone and setup backend**
   ```bash
   # Install Go dependencies
   make deps
   
   # Build GraphChain node
   make build
   ```

2. **Setup frontend**
   ```bash
   # Install frontend dependencies
   npm install
   ```

3. **Start development servers**
   ```bash
   # Terminal 1: Start GraphChain node
   ./build/graphchain-node -home ./data -rpc-port 8080
   
   # Terminal 2: Start frontend dev server
   npm run dev
   ```

4. **Access the application**
   - Frontend: http://localhost:5173
   - Backend API: http://localhost:8080

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

### Example API Usage

```bash
# Get current height
curl http://localhost:8080/height

# Get node
curl http://localhost:8080/node/genesis

# Get edge
curl http://localhost:8080/edge/edge123

# Get graph heads
curl http://localhost:8080/graph/heads

# Get neighbors
curl http://localhost:8080/graph/neighbors/node1

# Find path
curl http://localhost:8080/graph/path/node1/node2

# Get account
curl http://localhost:8080/account/0x1234567890123456789012345678901234567890

# Create node
curl -X POST http://localhost:8080/node \
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

# Create edge
curl -X POST http://localhost:8080/edge \
  -H "Content-Type: application/json" \
  -d '{
    "from": "node1",
    "to": "node2",
    "weight": 0.5,
    "type": 0
  }'
```

## Project Structure

```
graphchain-app/
├── cmd/                    # Command-line applications
│   ├── node/              # GraphChain node
│   └── cli/               # CLI tool
├── internal/              # Private application code
│   ├── app/               # Application logic
│   ├── graphchain/        # GraphChain core
│   ├── consensus/         # Graph consensus mechanism
│   ├── network/           # Networking and RPC
│   ├── storage/           # Data persistence
│   └── types/             # Data structures
├── pkg/                   # Public libraries
│   ├── crypto/            # Cryptographic functions
│   └── utils/             # Utility functions
├── config/                # Configuration files
├── scripts/               # Setup and deployment scripts
├── src/                   # React frontend
│   ├── components/        # React components
│   ├── pages/             # Page components
│   ├── services/          # API services
│   └── context/           # React context
├── public/                # Static assets
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

- [ ] Graph smart contracts
- [ ] Enhanced graph consensus mechanisms
- [ ] Cross-graph bridges
- [ ] Graph visualization tools
- [ ] Advanced graph algorithms
- [ ] Graph machine learning integration
- [ ] Mobile applications
- [ ] Graph analytics dashboard
- [ ] Multi-language support