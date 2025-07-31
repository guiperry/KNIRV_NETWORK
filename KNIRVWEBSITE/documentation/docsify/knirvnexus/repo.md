

---

**Source**: KNIRVNEXUS/.zencoder/docs/repo.md

# Agentic Engine Information

## Summary
Agentic Engine is a comprehensive platform for designing, deploying, and managing autonomous AI agents through a no-code interface. It features a Go backend with a React/TypeScript frontend, supporting multiple AI language models (Cerebras, Gemini, DeepSeek) with intelligent fallback mechanisms. The platform enables users to create agents with unique identities, configure them with specific capabilities, and orchestrate complex workflows.

## Structure
- **agent/**: Agent management, builder, and templates
- **agentify/**: Agent inferencing, plugins, and TEE security
- **api/**: RESTful API endpoints and service implementations
- **database/**: Data persistence and models
- **electron/**: Desktop application packaging
- **gui/**: React/TypeScript frontend
- **inference/**: LLM provider integrations
- **knirvchain/**: Blockchain integration services
- **scripts/**: Build and utility scripts

## Language & Runtime
**Backend**:
- **Language**: Go
- **Version**: 1.23.10
- **Build System**: Go modules
- **Package Manager**: Go modules

**Frontend**:
- **Language**: TypeScript/JavaScript
- **Framework**: React 18
- **Build System**: Vite
- **Package Manager**: npm

## Dependencies

### Backend Dependencies
**Main Dependencies**:
- github.com/gorilla/mux v1.8.1 (HTTP routing)
- github.com/gorilla/websocket v1.5.3 (WebSocket support)
- github.com/joho/godotenv v1.5.1 (Environment configuration)
- github.com/tetratelabs/wazero v1.9.0 (WebAssembly runtime)
- github.com/google/generative-ai-go v0.19.0 (AI integration)
- github.com/gin-gonic/gin v1.10.1 (HTTP framework)

### Frontend Dependencies
**Main Dependencies**:
- react v18.3.1
- react-dom v18.3.1
- react-router-dom v6.22.3
- @xterm/xterm v5.5.0 (Terminal emulation)
- lucide-react v0.344.0 (Icons)

**Development Dependencies**:
- typescript v5.5.3
- vite v5.4.2
- tailwindcss v3.4.1
- jest v29.7.0 (Testing)

## Build & Installation

### Backend Build
```bash
# Build for current platform
go build -o agentic-engine

# Cross-platform build
make build/all
```

### Frontend Build
```bash
# Install dependencies
cd gui
npm install

# Development mode
npm run dev

# Production build
npm run build
```

### Desktop Application Build
```bash
# Build Electron app
cd electron
npm install
npm run build

# Platform-specific builds
npm run build:win
npm run build:mac
npm run build:linux
```

## Docker
No explicit Dockerfile was found in the repository, but the application appears to be containerizable using standard Go and Node.js container patterns.

## Testing
**Backend Testing**:
- **Framework**: Go testing package
- **Test Location**: `*_test.go` files throughout the codebase
- **Run Command**:
```bash
go test -v -race ./...
# With coverage
make test/cover
```

**Frontend Testing**:
- **Framework**: Jest with React Testing Library
- **Test Location**: `gui/src/tests`
- **Run Command**:
```bash
cd gui
npm run test
```

## Main Entry Points
- **Backend**: `main.go` (Go application entry point)
- **Frontend**: `gui/src/main.tsx` (React application entry)
- **Desktop**: `electron/main.js` (Electron application entry)

## Configuration
- **Environment**: `.env` file for API keys and server configuration
- **Port Configuration**: Configurable via `ports.config`
- **JWT Authentication**: Secure token-based authentication

## Additional Components
- **Agent Inferencer**: Processes inference requests through appropriate plugins
- **Terminal Integration**: Interactive terminal sessions for agent interaction
- **Sub-Agent Infrastructure**: Supports spawning and managing sub-agents
- **Trusted Execution Environment (TEE)**: Secure isolation for sensitive operations
- **Error Inference Engine**: AI-powered error analysis and troubleshooting

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
