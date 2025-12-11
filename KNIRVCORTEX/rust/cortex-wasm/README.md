# KNIRV-CORTEX Rust WASM

WebAssembly implementation of the KNIRV cognitive shell with external AI integration support.

## 🆕 Recent Updates

### ✅ External AI Integration (Beta Phase)
- **Multi-Provider Support**: Added support for Google Gemini, Anthropic Claude, OpenAI ChatGPT-5, and Deepseek
- **External Inference Engine**: New `external_inference.rs` module for routing inference through external APIs
- **WASM-Compatible**: Async inference processing compatible with WebAssembly runtime
- **Provider Configuration**: Dynamic provider configuration and switching
- **Statistics Tracking**: Request metrics and performance monitoring

### ✅ HEART Integration (Error Analysis System)
- **HEART Client**: Integrated heuristic error analysis transformer for intelligent error handling
- **Error Tracking**: Automatic error inquiry tracking with 100-error history buffer
- **Pattern Recognition**: Identifies error patterns and provides recommended actions
- **Local Fallback**: Rule-based heuristics when HEART service unavailable
- **Rate Limiting**: Max 1 HEART query per second with response caching
- **wasm-bindgen Exports**: Full TypeScript/JavaScript support via async methods

## Features

### Core Cognitive Processing
- **Deterministic WASM execution** with ProtoBuf ABI
- **LoRA adapter integration** for skill learning
- **Memory policy management** with configurable limits
- **Context and tool management** for cognitive tasks

### External Inference (Beta)
- **Provider Management**: Configure and switch between external AI providers
- **Async Processing**: Non-blocking inference requests with proper error handling
- **Caching Support**: Response caching for improved performance
- **Fallback Mechanisms**: Graceful degradation when external APIs are unavailable

## API Reference

### Core Methods
```rust
// Initialize KNIRV-CORTEX
let mut cortex = KnirvCortex::new();

// Configure external inference (beta)
cortex.init_external_inference()?;
cortex.configure_external_provider(provider_json)?;
cortex.set_external_provider("gemini")?;

// Process cognitive tasks
let result = cortex.run_cognitive_task(prompt_ptr, prompt_len);

// External inference (beta)
let response = cortex.process_external_inference(request_json).await?;
```

### External Inference Configuration
```json
{
  "provider": "gemini",
  "api_key": "your-api-key",
  "endpoint": "https://api.gemini.com/v1",
  "model": "gemini-pro",
  "max_tokens": 1024,
  "temperature": 0.7,
  "enabled": true
}
```

## Building

### Prerequisites
- Rust 1.70+
- wasm-pack
- Node.js 18+

### Build Commands

#### Option 1: Standard Cargo Build (Raw WASM)
```bash
# Build raw WASM binary
cargo build --target wasm32-unknown-unknown --release

# Output: target/wasm32-unknown-unknown/release/knirv_cortex_wasm.wasm
```

#### Option 2: wasm-pack Build (TypeScript-friendly)
```bash
# Build for web browsers
wasm-pack build --target web --out-dir pkg-web

# Build for Node.js
wasm-pack build --target nodejs --out-dir pkg-nodejs

# Build for bundlers (Webpack, Vite, etc.)
wasm-pack build --target bundler --out-dir pkg-bundler
```

**Output includes:**
- `.wasm` file - WebAssembly binary
- `.js` file - JavaScript glue code
- `.d.ts` file - TypeScript type definitions
- `package.json` - npm package manifest

## Integration

### TypeScript/JavaScript
```typescript
import init, { KnirvCortex } from './pkg/knirv_cortex_wasm.js';

async function initializeCortex() {
  await init();
  const cortex = new KnirvCortex();
  
  // Initialize external inference
  await cortex.init_external_inference();
  
  // Configure provider
  const config = {
    provider: "gemini",
    api_key: "your-api-key",
    enabled: true
  };
  await cortex.configure_external_provider(JSON.stringify(config));
  
  return cortex;
}
```

### React Integration
```tsx
import { useEffect, useState } from 'react';
import { KnirvCortex } from './pkg/knirv_cortex_wasm.js';

export function CognitiveEngine() {
  const [cortex, setCortex] = useState<KnirvCortex | null>(null);
  const [isReady, setIsReady] = useState(false);
  
  useEffect(() => {
    initializeCortex().then(cortexInstance => {
      setCortex(cortexInstance);
      setIsReady(true);
    });
  }, []);
  
  const processInference = async (prompt: string) => {
    if (!cortex || !isReady) return;
    
    const request = {
      prompt,
      max_tokens: 1024,
      temperature: 0.7
    };
    
    const response = await cortex.process_external_inference(
      JSON.stringify(request)
    );
    
    return JSON.parse(response);
  };
  
  return (
    <div>
      {isReady ? (
        <button onClick={() => processInference("Hello, world!")}>
          Process with External AI
        </button>
      ) : (
        <div>Loading cognitive engine...</div>
      )}
    </div>
  );
}
```

## HEART Integration (Error Analysis)

KNIRV-CORTEX now includes full integration with the HEART (Heuristic Error Analysis and Recognition Transformer) system for intelligent error analysis and recovery recommendations.

### Initializing HEART Client

```typescript
import init, { KnirvCortex } from './pkg/knirv_cortex_wasm.js';

async function setupCortexWithHEART() {
  await init();
  const cortex = new KnirvCortex();

  // Initialize HEART client
  const heartConfig = {
    endpoint: "http://localhost:8080/heart/analyze",
    timeout_ms: 5000,
    min_confidence_threshold: 0.7,
    enable_pattern_recognition: true,
    enable_similarity_search: true,
    fallback_to_local: true
  };

  await cortex.init_heart_client(JSON.stringify(heartConfig));
  console.log("HEART client initialized");

  return cortex;
}
```

### Querying HEART for Error Analysis

```typescript
async function analyzeError(cortex: KnirvCortex, error: Error) {
  // Construct error inquiry
  const inquiry = {
    error_id: `err-${Date.now()}`,
    error_type: error.name,
    error_message: error.message,
    error_context: "User inference request",
    stack_trace: error.stack || "",
    metadata: {
      user_agent: navigator.userAgent,
      timestamp: new Date().toISOString()
    },
    prompt: "User's original prompt",
    model_response: null,
    confidence_score: null,
    timestamp: Date.now()
  };

  try {
    // Query HEART for heuristic analysis
    const responseJson = await cortex.query_heart_for_error(
      JSON.stringify(inquiry)
    );

    const response = JSON.parse(responseJson);

    console.log(`HEART Analysis (confidence: ${response.confidence_score}):`);
    console.log(`Alert Level: ${response.alert_level}`);
    console.log(`Summary: ${response.analysis_summary}`);
    console.log(`Recommended Actions:`);
    response.recommended_actions.forEach((action: string) => {
      console.log(`  - ${action}`);
    });

    // Apply heuristics to recover
    if (response.identified_patterns.length > 0) {
      console.log(`Identified patterns:`, response.identified_patterns);
    }

    return response;
  } catch (e) {
    console.error("HEART query failed:", e);
    throw e;
  }
}
```

### React Integration with HEART

```tsx
import { useEffect, useState } from 'react';
import { KnirvCortex } from './pkg/knirv_cortex_wasm.js';

export function CognitiveEngineWithHEART() {
  const [cortex, setCortex] = useState<KnirvCortex | null>(null);
  const [heartStats, setHeartStats] = useState<any>(null);

  useEffect(() => {
    setupCortexWithHEART().then(async (instance) => {
      setCortex(instance);

      // Check HEART health
      const isHealthy = await instance.heart_health_check();
      console.log("HEART service healthy:", isHealthy);
    });
  }, []);

  const processWithErrorHandling = async (prompt: string) => {
    if (!cortex) return;

    try {
      // Attempt inference
      const result = await cortex.process_external_inference(
        JSON.stringify({ prompt, max_tokens: 1024 })
      );

      return JSON.parse(result);
    } catch (error) {
      // Query HEART for error analysis
      const heartResponse = await analyzeError(cortex, error as Error);

      // Update UI with HEART recommendations
      alert(`Error detected: ${heartResponse.analysis_summary}\n\nRecommended: ${heartResponse.recommended_actions[0]}`);

      throw error;
    }
  };

  const checkHEARTStats = async () => {
    if (!cortex) return;
    const stats = cortex.get_heart_stats();
    setHeartStats(JSON.parse(stats));
  };

  return (
    <div>
      <h2>Cognitive Engine with HEART Error Analysis</h2>
      {cortex && cortex.is_heart_available() ? (
        <>
          <p>HEART Status: ✅ Active</p>
          <button onClick={() => processWithErrorHandling("Test prompt")}>
            Run Inference
          </button>
          <button onClick={checkHEARTStats}>
            View HEART Stats
          </button>
          {heartStats && (
            <pre>{JSON.stringify(heartStats, null, 2)}</pre>
          )}
        </>
      ) : (
        <p>HEART Status: ❌ Not Available</p>
      )}
    </div>
  );
}
```

### HEART API Methods

All HEART methods are exported via wasm-bindgen and available in TypeScript:

```typescript
// Initialization
cortex.init_heart_client(config_json: string): void

// Query HEART (async)
cortex.query_heart_for_error(inquiry_json: string): Promise<string>

// Health and stats
cortex.heart_health_check(): Promise<boolean>
cortex.get_heart_stats(): string
cortex.get_error_history_count(): number

// Cache management
cortex.clear_heart_cache(): void
cortex.is_heart_available(): boolean
```

### HEART Configuration Options

```typescript
interface HEARTConfig {
  endpoint: string;                    // HEART service endpoint
  timeout_ms: number;                  // Request timeout (default: 5000)
  min_confidence_threshold: number;    // Min confidence (0-1, default: 0.7)
  enable_pattern_recognition: boolean; // Enable pattern matching
  enable_similarity_search: boolean;   // Enable similar error search
  fallback_to_local: boolean;          // Use local heuristics if unavailable
}
```

### HEART Response Format

```typescript
interface HEARTHeuristicResponse {
  inquiry_id: string;
  command_vector: number[];           // 8-element heuristic vector
  alert_level: number;                // 1-5 severity
  heuristic_id: number;               // Analysis method used
  target_node_id: number;
  confidence_score: number;           // 0-1 confidence
  action_parameters: number[];
  analysis_summary: string;           // Human-readable analysis
  recommended_actions: string[];      // Concrete action steps
  debug_insights: string[];
  identified_patterns: ErrorPattern[];
  similar_errors: SimilarError[];
  processing_time_ms: number;
  timestamp: number;
}
```

## Architecture

### External Inference Flow
1. **Configuration**: Provider credentials and settings are configured
2. **Request Processing**: Inference requests are validated and prepared
3. **API Routing**: Requests are routed to the active external provider
4. **Response Handling**: Responses are processed and cached
5. **Fallback**: Local processing if external APIs fail

### Provider Support
- **Google Gemini**: Full support with streaming capabilities
- **Anthropic Claude**: Complete integration with conversation history
- **OpenAI ChatGPT-5**: Advanced model support with function calling
- **Deepseek**: Optimized for code generation and technical tasks

## Performance

### Benchmarks
- **WASM Module Size**: ~2.5MB (optimized)
- **Initialization Time**: <100ms
- **External API Latency**: 200-2000ms (provider dependent)
- **Memory Usage**: 64-256MB (configurable)

### Optimization
- **Lazy Loading**: External inference engine loads on demand
- **Request Batching**: Multiple requests can be batched for efficiency
- **Response Caching**: Frequently used responses are cached
- **Connection Pooling**: Reuse HTTP connections for better performance

## Security

### API Key Management
- Keys are stored securely in WASM memory
- No key logging or persistence in browser storage
- Automatic key rotation support

### Request Validation
- Input sanitization for all external requests
- Rate limiting to prevent API abuse
- Timeout handling for hung requests

## Troubleshooting

### Common Issues
1. **WASM Module Not Loading**: Ensure proper CORS headers and MIME types
2. **External API Errors**: Check API keys and rate limits
3. **Memory Issues**: Adjust memory policy configuration
4. **Performance Problems**: Enable caching and connection pooling

### Debug Mode
```rust
// Enable debug logging
console_log!("Debug: {}", debug_info);

// Get inference statistics
let stats = cortex.get_external_inference_stats();
```

## License

MIT License - see LICENSE file for details.
