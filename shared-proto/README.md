# KNIRV Network ProtoBuf Synchronization System

This directory contains the unified ProtoBuf definitions for the KNIRV Network ecosystem. These definitions are automatically synchronized across all platform-specific compilers and components.

## 📁 Directory Structure

```
shared-proto/
├── cortex/v1/
│   └── cortex.proto          # Core cortex inference and compilation messages
├── lora/v1/
│   └── lora.proto            # LoRA Adapter Engine messages
├── agent/v1/
│   └── agent.proto           # Agent compilation and deployment messages
├── memory/v1/
│   └── memory.proto          # Memory protocol messages
└── README.md                 # This file
```

## 🔄 Synchronization Targets

The ProtoBuf definitions are automatically synchronized to:

| Platform | Location | Purpose |
|----------|----------|---------|
| **KNIRVCORTEX** | `KNIRVCORTEX/shared-types/proto/` | Rust cortex.wasm compiler |
| **KNIRVCONTROLLER** | `KNIRVCONTROLLER/src/core/protobuf/schemas/` | TypeScript agent.wasm compiler |
| **KNIRVENGINE** | `KNIRVENGINE/desktop-client/proto/` | Go agent.wasm compiler |
| **KNIRVANA Gaming** | `KNIRVANA/gaming/cortex-compiler/proto/` | Gaming-specific Rust compiler |

## 🚀 Usage

### Synchronize All ProtoBuf Definitions

```bash
# Preview changes (safe)
make sync-protobuf-dry-run

# Synchronize all definitions
make sync-protobuf

# Sync and generate platform-specific code
make sync-protobuf-generate

# Validate definitions only
make sync-protobuf-validate
```

### Manual Synchronization

```bash
# Run the sync script directly
./scripts/sync-protobuf.sh

# Available options:
./scripts/sync-protobuf.sh --dry-run          # Preview only
./scripts/sync-protobuf.sh --validate-only    # Validate only
./scripts/sync-protobuf.sh --generate-code    # Sync + generate code
./scripts/sync-protobuf.sh --no-backup        # Skip backups
```

## 📋 ProtoBuf Definitions

### 1. Cortex Protocol (`cortex/v1/cortex.proto`)

Core messages for cortex inference and compilation:

- **InferenceInput/Output**: Main inference interface
- **CortexError**: Error handling
- **Envelope**: Response wrapper with status
- **Config**: Runtime configuration
- **Context**: Memory and tool context
- **MemoryPolicy**: Memory management policies
- **Tool/ToolList**: Tool system integration
- **ForgeManifest**: Model metadata
- **RuntimeConfig**: Runtime binding configuration

### 2. LoRA Adapter Engine (`lora/v1/lora.proto`)

LoRA (Low-Rank Adaptation) skill system:

- **LoRAAdapter**: Skill weights and metadata
- **SkillCompilation**: Skill creation from solutions/errors
- **SkillInvocation**: Skill execution requests
- **SkillChain**: Skill composition and merging
- **LoRATrainingConfig**: Training parameters
- **LoRAWeights**: Low-rank matrix weights
- **LoRACompilationResult**: Compilation output

### 3. Agent Protocol (`agent/v1/agent.proto`)

Agent compilation and deployment:

- **AgentCompilationRequest/Result**: Agent building
- **AgentConfig**: Platform-specific configuration
- **AgentManifest**: Agent metadata and capabilities
- **AgentExecution**: Runtime execution interface
- **AgentDeployment**: Multi-platform deployment
- **AgentValidation**: Testing and validation

### 4. Memory Protocol (`memory/v1/memory.proto`)

Comprehensive memory management:

- **MemoryRequest/Response**: Memory operations
- **EpisodicMemory**: Event-based memory
- **SemanticTriple**: Knowledge graph storage
- **ProceduralMemory**: Skill and procedure storage
- **WorkingMemory**: Short-term active memory
- **MemoryConsolidation**: Memory optimization
- **MemoryStatistics**: Usage and performance metrics

## 🔧 Platform-Specific Code Generation

### Rust (KNIRVCORTEX, KNIRVANA)

```rust
// Generated using prost
use cortex_proto::*;

let input = InferenceInput {
    prompt: "Hello, world!".to_string(),
    context: "".to_string(),
    config: Some(Config::default()),
    memory_policy: Some(MemoryPolicy::default()),
};
```

### TypeScript (KNIRVCONTROLLER)

```typescript
// Generated using protobufjs or buf
import { InferenceInput, Config, MemoryPolicy } from './generated/cortex';

const input: InferenceInput = {
  prompt: "Hello, world!",
  context: "",
  config: Config.create(),
  memoryPolicy: MemoryPolicy.create()
};
```

### Go (KNIRVENGINE)

```go
// Generated using protoc-gen-go
import cortexv1 "github.com/guiperry/KNIRV_NETWORK/pkg/gen/knirv/cortex/v1"

input := &cortexv1.InferenceInput{
    Prompt:  "Hello, world!",
    Context: "",
    Config:  &cortexv1.Config{},
    MemoryPolicy: &cortexv1.MemoryPolicy{},
}
```

## 🔒 Versioning and Compatibility

- **Semantic Versioning**: Each protocol uses semantic versioning (v1, v2, etc.)
- **Backward Compatibility**: New fields are optional to maintain compatibility
- **Breaking Changes**: Major version bumps for breaking changes
- **Migration**: Automatic migration scripts for version updates

## 🛠 Development Workflow

1. **Modify ProtoBuf**: Edit files in `shared-proto/`
2. **Validate**: Run `make sync-protobuf-validate`
3. **Preview**: Run `make sync-protobuf-dry-run`
4. **Synchronize**: Run `make sync-protobuf-generate`
5. **Test**: Run platform-specific tests
6. **Commit**: Commit both shared-proto and generated code

## 🚨 Important Notes

- **Single Source of Truth**: Only edit files in `shared-proto/`
- **Automatic Backups**: Backups are created before synchronization
- **Platform Testing**: Test all platforms after synchronization
- **Breaking Changes**: Coordinate breaking changes across teams
- **Code Generation**: Generated code should not be manually edited

## 🔍 Troubleshooting

### Validation Errors

```bash
# Check specific proto file
protoc --proto_path=shared-proto --descriptor_set_out=/dev/null cortex/v1/cortex.proto

# Validate all files
make sync-protobuf-validate
```

### Sync Failures

```bash
# Check backup directory
ls -la .proto-sync-backups/

# Restore from backup if needed
cp -r .proto-sync-backups/TIMESTAMP/TARGET/* TARGET_DIR/
```

### Missing Dependencies

```bash
# Install protoc
# Ubuntu/Debian: apt-get install protobuf-compiler
# macOS: brew install protobuf
# Windows: Download from https://protobuf.dev/

# Install language-specific generators
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
npm install -g protoc-gen-ts
cargo install protobuf-codegen
```

## 📚 References

- [Protocol Buffers Documentation](https://protobuf.dev/)
- [KNIRV Architecture Documentation](../KNIRVCORTEX/docs/)
- [Platform-Specific Integration Guides](../docs/)
