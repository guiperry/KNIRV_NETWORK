# KNIRV Revolutionary Architecture
## Triple-Layer WASM Cognitive System

### Overview
The KNIRV Network implements a revolutionary triple-layer architecture where cognitive processing is compiled into WASM modules, creating self-contained, improvable AI agents.

## Architecture Layers

### 1. Sensory Shell (Frontend)
**Location**: `KNIRVCONTROLLER/receiver/src/sensory-shell/`
**Purpose**: Input/output, user interface, sensory processing
**Components**:
- Voice processing and recognition
- Visual processing and computer vision
- Text input and gesture recognition
- User interface and interaction management
- Communication bridge to WASM layers

**Key Files**:
- `AgentCoreInterface.ts` - Communication bridge to cognitive WASM
- `WASMOrchestrator.ts` - Manages dual-WASM architecture
- `ModelManager.ts` - Handles different SLM models
- `VoiceProcessor.ts` - Voice input processing
- `VisualProcessor.ts` - Visual input processing

### 2. Cognitive Shell WASM (Compiled Templates)
**Location**: Compiled from `KNIRVCORTEX/agent-core/backend/agent-core-compiler/`
**Purpose**: Cognitive processing, reasoning, planning, skill execution
**Components**:
- Adaptive Learning Pipeline
- Cognitive Engine (reasoning and decision making)
- LoRA Adapter integration
- SEAL Framework
- Event system and communication

**Template Sources**:
- Translated Go templates from `KNIRVCORTEX/agent-builder/`
- Cognitive shell TypeScript templates:
  - `AdaptiveLearningPipeline.ts.template`
  - `CognitiveEngine.ts.template`
  - `EcosystemCommunicationLayer.ts.template`
  - `EnhancedLoRAAdapter.ts.template`
  - `EventEmitter.ts.template`
  - `LoRAAdapter.ts.template`
  - `SEALFramework.ts.template`

### 3. Model WASM (LLM Inference)
**Purpose**: Language model inference and text generation
**Supported Models**:

#### Default Models (Built-in)
- **HRM Cognitive** (27M parameters)
  - Hierarchical Reasoning Model
  - Default cognitive processing engine
  - Path: `/models/hrm_cognitive.wasm`

- **KNIRV Cortex** (40M parameters)
  - Advanced cognitive processing with LoRA adaptation
  - Path: `/models/knirv_cortex_wasm.wasm`

#### Alternative Models (from ALT_MODELS.md)
- **Phi-3 Mini** (3.8B parameters)
  - Microsoft's high-performance small model
  - Excellent reasoning capabilities
  - Path: `/models/phi-3-mini.wasm`

- **RecurrentGemma 2B** (2.7B parameters)
  - Google's novel recurrent architecture (GrGrU)
  - Efficient long sequence processing
  - Path: `/models/recurrentgemma-2b.wasm`

- **TinyLlama** (1.1B parameters)
  - Lightweight model for constrained environments
  - Path: `/models/tinyllama.wasm`

## Revolutionary Features

### 1. Template-Based Agent Compilation
- **Cognitive capabilities compiled into WASM** rather than running directly in browser
- **Templates allow primary agent to improve future iterations**
- **Wholistic integration** of Go templates + cognitive shell capabilities
- **Self-contained agents** that can be uploaded and executed

### 2. Dual-WASM Orchestration
- **Cognitive Shell WASM** handles reasoning, planning, skill execution
- **Model WASM** handles language inference and text generation
- **Cross-WASM communication** for coordinated processing
- **Dynamic model switching** without restart

### 3. Separation of Concerns
- **Sensory Shell**: Input/output, user interface (client-side)
- **Cognitive Shell**: Reasoning, planning (WASM-embedded)
- **Model Layer**: Language inference (WASM-embedded)
- **Clear communication channels** between all layers

### 4. LoRA Adapter Integration
- **Skills ARE LoRA adapters** containing weights and biases
- **Dynamic skill loading** into cognitive WASM
- **Adaptive learning** through weight modification
- **Skill compilation** from solutions and errors

## Communication Flow

```
User Input → Sensory Shell → Cognitive Shell WASM → Model WASM
                ↓                    ↓                  ↓
            UI Processing    →   Reasoning/Planning  →  Text Generation
                ↓                    ↓                  ↓
            Response UI    ←   Post-processing     ←   Model Output
```

## Key Interfaces

### AgentCoreInterface
- **Purpose**: Communication bridge between sensory-shell and cognitive WASM
- **Methods**: 
  - `processSensoryInput()` - Send input to cognitive WASM
  - `executeTool()` - Execute specific cognitive tools
  - `loadLoRAAdapter()` - Load skills into cognitive WASM
  - `getAgentCoreStatus()` - Get cognitive WASM status

### WASMOrchestrator
- **Purpose**: Manages both cognitive and model WASM modules
- **Methods**:
  - `initialize()` - Load both WASM modules
  - `processSensoryInput()` - Coordinate processing across both WASM layers
  - `switchModel()` - Dynamically switch language models
  - `getModuleInfo()` - Get status of all WASM modules

### ModelManager
- **Purpose**: Manages different SLM model options
- **Methods**:
  - `getAvailableModels()` - List all available models
  - `switchModel()` - Switch to different model
  - `getModelInstructions()` - Get setup instructions for models
  - `isModelAvailable()` - Check if model files are available

## File Structure

```
KNIRVCONTROLLER/receiver/src/
├── sensory-shell/                    # Frontend layer
│   ├── AgentCoreInterface.ts         # Cognitive WASM communication
│   ├── WASMOrchestrator.ts          # Dual-WASM management
│   ├── ModelManager.ts              # Model management
│   ├── VoiceProcessor.ts            # Voice input processing
│   ├── VisualProcessor.ts           # Visual input processing
│   └── EventEmitter.ts              # Event system
├── react-app/components/
│   ├── WASMAgentUploader.tsx        # Upload custom agents
│   ├── ModelSelector.tsx           # Select and switch models
│   └── SkillCompiler.tsx           # Compile skills from templates
└── ...

KNIRVCORTEX/agent-core/
├── backend/agent-core-compiler/     # TypeScript agent compiler
│   ├── src/AgentCoreCompiler.ts    # Main compiler
│   └── templates/                   # TypeScript templates
│       ├── main.ts.template         # Main agent template
│       ├── tool.ts.template         # Tool template
│       ├── CognitiveEngine.ts.template
│       ├── AdaptiveLearningPipeline.ts.template
│       └── ...
└── rust-wasm/                      # WASM build artifacts
    └── target/wasm32-unknown-unknown/release/
        └── knirv_cortex_wasm.wasm   # Default cognitive model
```

## Benefits

1. **Performance**: WASM execution is near-native speed
2. **Security**: Sandboxed execution environment
3. **Portability**: Runs consistently across platforms
4. **Modularity**: Swap cognitive and model components independently
5. **Improvability**: Templates allow agents to improve future iterations
6. **Scalability**: Efficient resource usage with WASM
7. **Flexibility**: Support for multiple model architectures

## Future Enhancements

1. **Agent Self-Improvement**: Primary agent modifying templates for better iterations
2. **Cross-WASM Optimization**: Enhanced communication protocols
3. **Model Quantization**: Smaller, faster model variants
4. **Distributed Processing**: Multi-WASM coordination
5. **Advanced LoRA**: More sophisticated adapter architectures

This revolutionary architecture represents a paradigm shift from traditional AI systems to compiled, self-contained, improvable cognitive agents that can run efficiently in any environment while maintaining clear separation of concerns and elegant orchestration between layers.
