# KNIRV-CORTEX Rust WASM

WebAssembly implementation of the KNIRV cognitive shell with external AI integration support.

## 🆕 Recent Updates

### ✅ External AI Integration (Beta Phase)
- **Multi-Provider Support**: Added support for Google Gemini, Anthropic Claude, OpenAI ChatGPT-5, and Deepseek
- **External Inference Engine**: New `external_inference.rs` module for routing inference through external APIs
- **WASM-Compatible**: Async inference processing compatible with WebAssembly runtime
- **Provider Configuration**: Dynamic provider configuration and switching
- **Statistics Tracking**: Request metrics and performance monitoring

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
```bash
# Build for web
wasm-pack build --target web --out-dir pkg-web

# Build for Node.js
wasm-pack build --target nodejs --out-dir pkg-nodejs

# Build for bundlers
wasm-pack build --target bundler --out-dir pkg-bundler
```

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
