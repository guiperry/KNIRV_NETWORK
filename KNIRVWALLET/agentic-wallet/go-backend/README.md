# Crypto Wallet Backend with AI Agents

A comprehensive Go backend for a cryptocurrency wallet application with AI agent plugin support.

## Features

- **Multi-chain Wallet Support**: Ethereum, Bitcoin, Solana
- **AI Agent System**: WASM-based plugin execution with sandboxing
- **Security**: Hardware security module integration, encrypted storage
- **Real-time Data**: WebSocket support for live updates
- **Scalable Architecture**: Microservices-ready design
- **Android APK Support**: Can be compiled to Android using gomobile

## Architecture

```
├── cmd/server/          # Application entry point
├── internal/
│   ├── api/            # HTTP handlers and routing
│   ├── config/         # Configuration management
│   ├── database/       # Database connection and migrations
│   ├── models/         # Data models
│   └── services/       # Business logic
├── pkg/
│   ├── logger/         # Logging utilities
│   ├── wasm/          # WASM runtime for AI agents
│   └── crypto/        # Cryptographic utilities
└── migrations/         # Database migrations
```

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 13+
- Redis 6+

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd go-backend
```

2. Install dependencies:
```bash
make deps
```

3. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Run database migrations:
```bash
make migrate-up
```

5. Start the server:
```bash
make run
```

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/refresh` - Refresh JWT token

### Wallets
- `GET /api/v1/wallets` - Get user wallets
- `POST /api/v1/wallets` - Create new wallet
- `GET /api/v1/wallets/:id/transactions` - Get wallet transactions

### AI Agents
- `GET /api/v1/agents` - Get user's AI agents
- `POST /api/v1/agents` - Create new AI agent
- `POST /api/v1/agents/:id/execute` - Execute AI agent
- `GET /api/v1/agents/marketplace` - Browse marketplace agents

### Market Data
- `GET /api/v1/market/prices` - Get cryptocurrency prices
- `GET /api/v1/market/portfolio` - Get portfolio value

## AI Agent Development

AI agents are compiled to WebAssembly (WASM) and executed in a sandboxed environment. Each agent has specific permissions and resource limits.

### Example Agent (Rust)

```rust
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub fn main() -> String {
    // AI agent logic here
    "Agent executed successfully".to_string()
}
```

### Permissions System

- `read_portfolio` - Access portfolio data
- `execute_trades` - Execute buy/sell orders
- `read_market_data` - Access market data
- `network_access` - Make external API calls
- `storage_access` - Store persistent data

## Android APK Compilation

To compile the backend into an Android APK:

1. Install gomobile:
```bash
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
```

2. Build Android library:
```bash
make android-build
```

3. Integrate with Android project using the generated `.aar` file

## Security Features

- **Hardware Security Module (HSM)** integration
- **Encrypted private key storage**
- **WASM sandboxing** for AI agents
- **JWT authentication** with refresh tokens
- **Rate limiting** and DDoS protection
- **Input validation** and sanitization

## Deployment

### Docker

```bash
make docker-build
make docker-run
```

### Kubernetes

```bash
kubectl apply -f k8s/
```

## Development

### Running Tests

```bash
make test
```

### Hot Reload

```bash
make dev
```

### Code Quality

```bash
make lint
make security-scan
make fmt
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details