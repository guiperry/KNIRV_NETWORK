# KNIRV Controller Backend

## Overview

The KNIRV Controller Backend is the **revolutionary TypeScript-written agent-core compiler** that creates complete `agent.wasm` files with embedded cognitive capabilities. This backend serves as the compilation engine for the triple-layer WASM architecture.

## Why Do We Need to Build the Backend?

### 1. **Revolutionary Architecture Separation**
The backend implements a **paradigm shift** from traditional AI systems:

- **Traditional AI**: Cognitive processing runs directly in browser/frontend
- **KNIRV Revolutionary**: Cognitive processing is **compiled into WASM** for embedded execution

### 2. **Template-Based Agent Compilation**
The backend contains the **TypeScript Agent-Core Compiler** that:
- Translates Go templates from `agent-builder` to TypeScript templates
- Converts cognitive-shell files to TypeScript templates
- Compiles complete `agent.wasm` files with embedded cognitive capabilities
- Enables **agent self-improvement** through template modification

### 3. **Performance & Security Benefits**
- **WASM Execution**: Near-native performance in sandboxed environment
- **Portability**: Runs consistently across all platforms
- **Security**: Isolated execution prevents malicious code execution
- **Modularity**: Swap cognitive and model components independently

### 4. **Separation of Concerns**
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Sensory Shell  │    │ Cognitive Shell  │    │   Model WASM    │
│   (Frontend)    │◄──►│     (WASM)       │◄──►│   (Inference)   │
│                 │    │                  │    │                 │
│ • Input/Output  │    │ • Reasoning      │    │ • Text Gen      │
│ • UI/UX         │    │ • Planning       │    │ • LLM Models    │
│ • Sensory Proc  │    │ • Skill Exec     │    │ • Phi-3, Gemma  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Architecture Components

### 1. **Agent-Core Compiler** (`agent-core-compiler/`)
The heart of the revolutionary system:

```typescript
// Compiles templates into complete agent.wasm files
const compiler = new AgentCoreCompiler();
const result = await compiler.compileAgentCore(config);
// Result: Complete agent.wasm with embedded cognitive capabilities
```

**Templates Included:**
- `main.ts.template` - Main agent-core module with WASM exports
- `CognitiveEngine.ts.template` - Core reasoning and decision making
- `AdaptiveLearningPipeline.ts.template` - Learning and adaptation
- `LoRAAdapter.ts.template` - Skill adaptation through LoRA weights
- `SEALFramework.ts.template` - Secure encrypted processing
- `tool.ts.template` - Tool implementation framework
- And more...

### 2. **LoRA Adapter Engine** (`lora/`)
Manages Low-Rank Adaptation for skill modification:
- **Skills ARE LoRA adapters** containing weights and biases
- Dynamic skill loading and execution
- Performance tracking and optimization
- Skill compilation from solutions and errors

### 3. **WASM Compiler** (`wasm/`)
Compiles TypeScript to WASM:
- Rust-based WASM compilation pipeline
- Optimization and size reduction
- Cross-platform compatibility
- Integration with `wasm-pack` toolchain

### 4. **Protobuf Handler** (`protobuf/`)
Efficient serialization for WASM communication:
- Schema-based serialization/deserialization
- Type-safe data exchange between layers
- Performance optimization for large data transfers

### 5. **Cortex API** (`api/`)
RESTful API for agent management:
- Agent compilation endpoints
- Skill management and invocation
- LoRA adapter operations
- Real-time WebSocket communication

## Build Process

### Why Building is Required

1. **TypeScript Compilation**: Convert TypeScript to JavaScript with proper module resolution
2. **Template Processing**: Ensure all templates are available for runtime compilation
3. **Dependency Resolution**: Bundle all required modules and dependencies
4. **WASM Integration**: Prepare WASM compilation toolchain
5. **Distribution Packaging**: Create deployable backend package

### Build Steps

```bash
# 1. TypeScript Compilation
npm run build:backend

# 2. What happens during build:
# - Compiles TypeScript to JavaScript with source maps
# - Copies agent-core-compiler templates to dist/
# - Copies protobuf schemas
# - Creates distribution package.json
# - Sets up WASM build scripts
```

### Build Output Structure
```
dist/
├── index.js                    # Main backend entry point
├── agent-core-compiler/
│   ├── src/
│   │   └── AgentCoreCompiler.js
│   └── templates/              # TypeScript templates for compilation
│       ├── main.ts.template
│       ├── CognitiveEngine.ts.template
│       └── ...
├── lora/
│   └── LoRAAdapterEngine.js
├── wasm/
│   └── WASMCompiler.js
├── protobuf/
│   └── ProtobufHandler.js
└── api/
    └── CortexAPI.js
```

## Revolutionary Features

### 1. **Template-Based Agent Improvement**
```typescript
// Primary agent can improve future iterations
const improvedTemplate = await primaryAgent.improveTemplate(
  'CognitiveEngine.ts.template',
  learningData
);

// Compile improved agent
const betterAgent = await compiler.compileAgentCore({
  ...config,
  customTemplates: { 'CognitiveEngine.ts.template': improvedTemplate }
});
```

### 2. **Cross-WASM Communication**
```typescript
// Sensory Shell ↔ Cognitive WASM ↔ Model WASM
const response = await sensoryShell.processSensoryInput(input);
// → Cognitive WASM processes reasoning
// → Model WASM handles text generation  
// → Cognitive WASM post-processes result
// → Sensory Shell displays output
```

### 3. **Dynamic Skill Loading**
```typescript
// Load LoRA adapter as skill
const skillLoaded = await agentCore.loadLoRAAdapter({
  skillId: 'advanced-reasoning',
  weightsA: Float32Array,
  weightsB: Float32Array,
  rank: 16,
  alpha: 32
});
```

## API Endpoints

### Agent Compilation
- `POST /api/compile-agent` - Compile agent from templates
- `GET /api/templates` - List available templates
- `PUT /api/templates/:name` - Update template (for self-improvement)

### Skill Management  
- `POST /api/skills/compile` - Compile skill from solutions/errors
- `POST /api/skills/invoke` - Execute skill through LoRA adapter
- `GET /api/skills` - List available skills

### WASM Operations
- `POST /wasm/compile` - Compile TypeScript to WASM
- `GET /wasm/status` - Check WASM compiler status

### Health & Monitoring
- `GET /health` - System health check
- `GET /metrics` - Performance metrics
- `WS /` - Real-time WebSocket API

## Development

### Prerequisites
```bash
# Required tools
node >= 20.0.0
rust >= 1.70.0
wasm-pack
```

### Local Development
```bash
# Install dependencies
npm install

# Build backend
npm run build:backend

# Start development server
npm run dev:backend

# Run tests
npm run test:backend
```

### Environment Variables
```bash
PORT=3004                    # Backend server port
NODE_ENV=development         # Environment mode
WASM_OPTIMIZATION=true       # Enable WASM optimization
LORA_CACHE_SIZE=1000        # LoRA adapter cache size
```

## Integration with Frontend

The backend integrates with the **Sensory Shell** (frontend) through:

1. **Agent Upload**: Upload compiled `agent.wasm` files
2. **Skill Management**: Load/unload LoRA adapters
3. **Model Selection**: Switch between different LLM models
4. **Real-time Communication**: WebSocket for live updates

## Future Enhancements

1. **Distributed Compilation**: Multi-node agent compilation
2. **Advanced Optimization**: ML-based WASM optimization
3. **Template Marketplace**: Share and discover agent templates
4. **Federated Learning**: Cross-agent knowledge sharing
5. **Quantum Integration**: Quantum-enhanced cognitive processing

## Troubleshooting

### Common Issues

1. **WASM Compilation Fails**
   ```bash
   # Ensure Rust toolchain is installed
   curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
   cargo install wasm-pack
   ```

2. **Template Not Found**
   ```bash
   # Rebuild to ensure templates are copied
   npm run build:backend
   ```

3. **Port Already in Use**
   ```bash
   # Change port in environment
   PORT=3005 npm start
   ```

## Contributing

1. Follow TypeScript strict mode guidelines
2. Add tests for new template features
3. Update this README for new capabilities
4. Ensure WASM compatibility for all changes

---

**The KNIRV Controller Backend represents a revolutionary approach to AI agent architecture, where cognitive processing is compiled rather than interpreted, enabling unprecedented performance, security, and modularity.**
