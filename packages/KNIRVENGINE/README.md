# KNIRVENGINE

## Overview

The KNIRVENGINE is a comprehensive platform for designing, deploying, and managing autonomous AI agents through an intuitive no-code interface. Built with a powerful Go backend and a modern React/TypeScript frontend, it empowers users to define complex goals, equip agents with diverse capabilities, and orchestrate their operations to achieve sophisticated outcomes. The engine supports multiple AI language models (including Cerebras, Gemini, and DeepSeek) with intelligent fallback mechanisms to ensure reliable operation.

## Key Features

### Agent Management
* **NFT-Agents:** Create, configure, and manage AI agents with unique identities
* **Agent Builder:** Generate custom agents from templates with specific capabilities
* **Agent Inferencer:** Process inference requests through appropriate agent plugins
* **Plugin System:** Extensible architecture for adding custom agent capabilities
* **Terminal Integration:** Interact with agents through terminal sessions

### Built-in Sub-Agent Infrastructure
* **Autonomous Sub-Agent Spawning:** Main agents can independently spawn sub-agents based on task requirements without external API calls
* **SubagentManager:** Complete sub-agent lifecycle management (create, start, stop, delete) with resource limits and TEE isolation
* **8 Orchestration Patterns:** Support for Sequential Agents, Parallel Processing, Hierarchical Delegation, and Collaborative Problem Solving
* **Dedicated Terminal Sessions:** Each sub-agent gets its own terminal for independent operation and logging
* **Communication Hub:** Structured parent-sub-agent messaging with peer communication and error escalation
* **Multi-Language Support:** Python and JavaScript sub-agent templates with specialized prompt generation
* **Resource Management:** Memory, CPU, and timeout limits for sub-agents with monitoring and performance tracking
* **TEE Security:** Sub-agents run within the main agent's Trusted Execution Environment for secure isolation

### AI Model Integration
* **Multi-Provider Support:** Seamlessly integrate with Cerebras, Gemini, and DeepSeek models
* **Intelligent Fallback:** Automatically switch to backup providers if primary ones fail
* **Mixture of Agents (MOA):** Combine multiple AI models for enhanced capabilities
* **Context Management:** Process large inputs exceeding token limits with intelligent chunking strategies

### Workflow Orchestration
* **Target Systems:** Define and manage objectives for agents to pursue
* **Inference Orchestration:** Coordinate complex workflows involving multiple agents
* **Workflow Repository:** Track and manage workflow execution and results
* **Analytics Dashboard:** Monitor agent performance and system metrics

### Security & Authentication
* **User Management:** Secure authentication with role-based access control
* **JWT Authentication:** Token-based security for API access
* **Permission System:** Granular control over user capabilities
* **Trusted Execution Environment (TEE):** Secure environment for sensitive operations

### Web & System Integration
* **Web Connections:** Integrate with external web services and APIs
* **System Connections:** Interface with local system resources
* **Database Integration:** Persistent storage for agents, workflows, and user data
* **WebSocket Support:** Real-time communication for terminal sessions and monitoring

## 🚀 Key Innovation: AI Error Inference Engine

The KNIRVENGINE features a groundbreaking **AI Error Inference Engine** that transforms error handling from reactive debugging to proactive intelligent assistance:

### 🤖 **Intelligent Error Analysis**
* **LLM-Powered Diagnosis:** Automatically analyzes system errors using advanced language models
* **Smart Categorization:** Classifies errors by type, severity, and root cause
* **Confidence Scoring:** Provides confidence levels for suggested solutions
* **Context-Aware Analysis:** Collects comprehensive system information for accurate diagnosis

### 🔔 **Real-Time Error Notifications**
* **Smart Notification Bell:** Header-mounted indicator with error count badges and severity alerts
* **Automatic Triggering:** Auto-analyzes critical and high-severity errors immediately
* **Priority-Based Alerts:** Different notification styles based on error severity
* **Unread Indicators:** Visual cues for new errors requiring attention

### 💬 **Interactive Error Assistant**
* **Chat Modal Interface:** Conversational AI assistant for error troubleshooting
* **Follow-Up Questions:** Ask detailed questions about specific errors
* **Step-by-Step Guidance:** Detailed resolution instructions with estimated time
* **Recovery Strategies:** Automatic retry logic and intelligent recovery actions

### 🔄 **Self-Healing Capabilities**
* **Automated Recovery:** Intelligent retry strategies with exponential backoff
* **Fallback Analysis:** Rule-based analysis when LLM inference is unavailable
* **System Context Collection:** Captures user agent, URL, session info, and stack traces
* **Error History Tracking:** Maintains error patterns for improved future analysis

### 📊 **Production-Ready Features**
* **Error Statistics Dashboard:** Real-time metrics and error trend analysis
* **Severity Thresholds:** Configurable auto-analysis triggers
* **React Hook Integration:** Seamless integration with frontend components
* **Performance Monitoring:** Tracks error resolution success rates and response times

This innovation creates a **self-healing system** where the inference engine detects, analyzes, and often suggests automated fixes for system errors, significantly reducing debugging time and improving system reliability.

## Architecture

The KNIRVENGINE employs a modular, service-oriented architecture:

### Backend Components
* **API Server:** RESTful API built with Go's HTTP package and Gorilla Mux router
* **Inference Services:** Manages interactions with LLM providers
* **Agent Services:** Handles agent lifecycle, capabilities, and inference
* **Database Services:** SQLite-based persistence for domain objects and authentication
* **Workflow Services:** Orchestrates complex agent workflows and tracks execution

### Frontend Components
* **React/TypeScript SPA:** Modern single-page application with TypeScript
* **Tailwind CSS:** Utility-first styling for responsive design
* **Component Architecture:** Modular UI components for each functional area
* **WebSocket Integration:** Real-time updates and terminal sessions
* **Authentication Flow:** Secure login and session management

### Deployment Options
* **Development Mode:** Integrated development server with hot reloading
* **Production Mode:** Static file serving with embedded assets
* **Desktop Application:** Electron-based desktop version available
* **Cross-Platform Support:** Runs on Linux, macOS, and Windows

## Setup and Installation

### Prerequisites

* **Go:** Version 1.21 or later for the backend. [https://go.dev/doc/install](https://go.dev/doc/install)
* **Node.js and npm:** Version 16+ for frontend development. [https://nodejs.org/](https://nodejs.org/)
* **AI Provider Accounts:** API keys from Cerebras, Google (for Gemini), or DeepSeek

### Installation

1. **Clone the Repository:**
   ```bash
   git clone https://github.com/guiperry/KNIRV-Engine.git
   cd KNIRV-Engine
   ```

2. **Configure Environment Variables:**
   * Create a `.env` file in the project root:
     ```dotenv
     # API Keys
     CEREBRAS_API_KEY=your_cerebras_api_key_here
     GEMINI_API_KEY=your_gemini_api_key_here
     DEEPSEEK_API_KEY=your_deepseek_api_key_here
     
     # Security
     JWT_SECRET=your_jwt_secret_key_here
     
     # Server Configuration
     API_PORT=8081
     GUI_PORT=8080
     ```

3. **Build and Run:**
   * **Quick Start:**
     ```bash
     # Build the backend
     go build -o knirv-engine
     
     # Sync environment variables
     ./sync-env.sh
     
     # Start the application
     ./knirv-engine
     ```
   
   * **Development Mode:**
     ```bash
     # Start backend
     go run main.go
     
     # In another terminal, start frontend
     cd gui
     npm install
     npm run dev
     ```
   
   * **Production Mode:**
     ```bash
     # Build backend with production flag
     go build -o knirv-engine
     
     # Build frontend
     cd gui
     npm install
     npm run build
     
     # Run in production mode
     cd ..
     ./knirv-engine --production
     ```

### Port Configuration

The application uses configurable ports via the `ports.config` file:
- **API_PORT:** Backend server port (default: 8081)
- **GUI_PORT:** Frontend server port (default: 8080)

To change ports:
1. Edit `ports.config` file with your desired ports
2. Run `./sync-env.sh` to update frontend configuration
3. Restart the application

See [docs/PORT_CONFIGURATION.md](docs/PORT_CONFIGURATION.md) for detailed port configuration guide.

## Usage Guide

### Authentication
1. Access the application through your web browser at `http://localhost:8080` (or your configured port)
2. Log in with your credentials (default admin user is created on first run)
3. For first-time setup, navigate to Settings to configure API keys

### Creating and Managing Agents
1. Navigate to the "NFT-Agents" section
2. Click "Create New Agent" to define a new agent
3. Configure the agent's profile, base instructions, and capabilities
4. Select the AI model(s) the agent should use
5. Save and activate the agent

### Defining Capabilities and Targets
1. Visit the "Capabilities" section to enable tools for your agents
2. In "Target Systems," define objectives for agents to pursue
3. Assign agents to specific targets and configure parameters

### Orchestrating Workflows
1. Use the "Inference" section to design complex workflows
2. Configure sequential or parallel execution of agent tasks
3. Monitor workflow execution in real-time
4. View results and analytics in the Dashboard

### Advanced Features
1. Terminal Integration: Interact with agents through command-line interfaces
2. Context Management: Configure chunking strategies for large inputs
3. LLM Provider Settings: Set primary and fallback models
4. Workflow Analytics: Track performance metrics and success rates

## API Reference

The KNIRVENGINE provides a comprehensive RESTful API:

### Authentication Endpoints
- `POST /api/auth/login`: Authenticate and receive JWT token
- `POST /api/auth/refresh`: Refresh an existing JWT token
- `POST /api/auth/logout`: Invalidate current token

### Agent Endpoints
- `GET /api/v1/agents`: List all agents
- `POST /api/v1/agents`: Create a new agent
- `GET /api/v1/agents/{id}`: Get agent details
- `PUT /api/v1/agents/{id}`: Update agent configuration
- `POST /api/v1/agents/{id}/build`: Build agent from template

### Inference Endpoints
- `POST /api/v1/adk/agents/inference`: Process inference request
- `GET /api/v1/inference/models`: List available LLM models

### Workflow Endpoints
- `POST /api/v1/workflows`: Create a new workflow
- `GET /api/v1/workflows`: List all workflows
- `GET /api/v1/workflows/{id}`: Get workflow details
- `PUT /api/v1/workflows/{id}`: Update workflow status

### Terminal Endpoints
- `POST /api/v1/terminal/create`: Create a terminal session
- `GET /api/v1/terminal/ws`: WebSocket for terminal I/O

## System Requirements

### Minimum Requirements
- **CPU:** 2+ cores
- **RAM:** 4GB+
- **Disk:** 1GB free space
- **OS:** Linux, macOS, or Windows 10+
- **Browser:** Chrome, Firefox, Safari, or Edge (latest versions)

### Recommended Requirements
- **CPU:** 4+ cores
- **RAM:** 8GB+
- **Disk:** 5GB+ free space
- **Network:** Stable internet connection for API access

## Development and Extension

### Plugin Development
The KNIRVENGINE supports custom plugins for extending agent capabilities:
1. Use the plugin development templates in `agentify/examples/`
2. Implement the `AgentPluginInterface` for custom functionality
3. Build and import plugins through the UI or API

### Frontend Customization
1. The React frontend can be customized by modifying components in `gui/src/components/`
2. Styling uses Tailwind CSS for easy customization
3. New views can be added by extending the routing in `App.tsx`

### Backend Extension
1. Add new services by implementing appropriate interfaces
2. Register new API endpoints in `api/simple_server.go`
3. Extend database models in `database/models/`

## License

*MIT, Apache 2.0*

## Contributing

We welcome contributions! Please read our [contributing guidelines](CONTRIBUTING.md) before submitting pull requests.

## Testing

The KNIRVENGINE includes a comprehensive test suite covering all components:

```bash
# Run all tests
make test

# Run specific test categories
make test-unit          # Unit tests
make test-integration   # Integration tests
make test-frontend      # Frontend tests
make test-api          # API tests
make test-mcp          # MCP integration tests
make test-security     # Security tests
make test-performance  # Performance tests
make test-connectivity # End-to-end connectivity tests
make test-chat         # Agent chat functionality tests

# Run API endpoint tests
make test-api-endpoints # API endpoint integration tests
make test-api-simple   # Simple API tests

# Run comprehensive test suite
./scripts/run_comprehensive_tests.sh [mode]
# Available modes: unit, integration, frontend, api, mcp, cloud, desktop,
#                  security, performance, wasm, connectivity, chat, ci, all, full
```

## Current Implementation Status

The KNIRVENGINE is currently **production-ready** with comprehensive functionality across all major components. For detailed implementation status, see [current_status_implementation.md](current_status_implementation.md).

### Phase 2: Data Consistency and Validation (Weeks 5-8)

We are now ready to implement **Phase 2** of the [full_continuity_implementation_plan.md](full_continuity_implementation_plan.md), which focuses on:

- **Data Storage Consolidation**: Unifying storage systems for improved consistency
- **Enhanced Validation**: Implementing comprehensive data validation across all components
- **Performance Optimization**: Optimizing database queries and API response times
- **Advanced Error Handling**: Enhanced error reporting and recovery mechanisms
- **System Monitoring**: Comprehensive monitoring and alerting capabilities

The foundation is solid with excellent test coverage, and Phase 2 will focus on optimization and advanced features.

## Documentation

Additional documentation is available in the `docs/` directory:
- [PORT_CONFIGURATION.md](docs/PORT_CONFIGURATION.md): Detailed port configuration guide
- [AGENT_BUILDER_IMPLEMENTATION_SUMMARY.md](docs/AGENT_BUILDER_IMPLEMENTATION_SUMMARY.md): Agent builder details
- [MCP_INTEGRATION.md](docs/MCP_INTEGRATION.md): MCP integration documentation
- [Testing.md](docs/Testing.md): Testing strategies and procedures
- [current_status_implementation.md](current_status_implementation.md): Current implementation status and roadmap