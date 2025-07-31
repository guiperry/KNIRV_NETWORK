

---

**Source**: KNIRVNEXUS/docs/completedImplementations/CEREBRAS_INTEGRATION.md

# Cerebras Integration for Agent Embeddings

This document explains how to configure your Cerebras API key for agent embeddings in the Agentic-Engine.

## Overview

The Agentic-Engine now includes built-in support for Cerebras embeddings using your tool calling functionality from the `chroma-go_cerebras` repository. The integration uses Cerebras's chat completions API with tool calling to generate embeddings.

## Configuration Options

### Option 1: Environment Variable (Recommended)

Set your Cerebras API key as an environment variable:

```bash
export CEREBRAS_API_KEY="your_actual_cerebras_api_key_here"
```

### Option 2: Update Template Files

Replace the placeholder `YOUR_CEREBRAS_API_KEY_HERE` in the following files with your actual API key:

1. `agent/templates/main.go.template` (line ~293)
2. `agent/templates/agent_service.py.template` (line ~101)
3. `database/simple_domain_db.go` (line ~52)

### Option 3: Configuration File

Each generated agent will include a `config.env` file where you can set:

```env
CEREBRAS_API_KEY=your_actual_cerebras_api_key_here
```

## How It Works

### Cerebras Embeddings Implementation

The integration includes a complete Cerebras embeddings client (`cerebras_embeddings.go.template`) that:

1. **Uses Tool Calling**: Leverages Cerebras's tool calling functionality to generate embeddings
2. **Supports Batch Processing**: Can generate embeddings for multiple texts at once
3. **Configurable Parameters**: Supports different task types, instructions, and normalization options
4. **Chromem-go Compatible**: Provides the correct function signature for chromem-go integration

### Key Features

- **Model**: Uses `llama3.1-8b` by default (configurable)
- **Task Type**: Optimized for `retrieval_document` tasks
- **Normalization**: Supports L2 normalization of embedding vectors
- **Dimension**: Returns 384-dimensional embeddings (standard size)

### Integration Points

1. **Database Layer**: `database/simple_domain_db.go` uses Cerebras for agent storage embeddings
2. **Agent Templates**: Generated agents include Cerebras embedding functionality
3. **Python Service**: Python-based agents can also use Cerebras API

## File Structure

```
agent/templates/
├── cerebras_embeddings.go.template    # Cerebras embedding client implementation
├── main.go.template                   # Main agent with Cerebras integration
├── agent_service.py.template          # Python service with Cerebras support
├── config.env.template               # Configuration template
└── go.mod.template                   # Go module with dependencies
```

## Usage Example

Once configured, agents will automatically use Cerebras embeddings:

```go
// This happens automatically in generated agents
embeddingFunc := CreateCerebrasEmbeddingFunction(cerebrasAPIKey)
memory := chromem.NewDB()

// Embeddings are generated using Cerebras tool calling
collection, err := memory.GetOrCreateCollection("agent_memory", nil, embeddingFunc)
```

## Security Considerations

### Environment Variables (Most Secure)
- API keys are not embedded in code
- Can be different per environment
- Easy to rotate

### Template Replacement (Less Secure)
- API keys are embedded in generated plugins
- Visible in compiled .so files
- Harder to rotate

### Recommendation
Use environment variables for production deployments and only embed API keys in templates for development/testing.

## Monitoring Usage and Costs

### Usage Monitoring Script

A monitoring script is provided to help track your Cerebras API usage:

```bash
# Basic usage monitoring
go run scripts/monitor_cerebras_usage.go

# Health check
go run scripts/monitor_cerebras_usage.go --health-check

# Help
go run scripts/monitor_cerebras_usage.go --help
```

### Rate Limiting

The Cerebras API has rate limits. Our implementation includes:

- Graceful fallback to zero vectors when rate limited
- Debug messages indicating when rate limits are hit
- Recommendations for handling rate limits

### Cost Optimization

1. **Cache embeddings locally** to avoid repeated API calls
2. **Batch requests** when possible
3. **Use shorter texts** to reduce token usage
4. **Monitor usage regularly** through the Cerebras dashboard
5. **Implement exponential backoff** for rate limit handling

### Dashboard Monitoring

Check your usage and costs at: https://cerebras.ai/dashboard

## Troubleshooting

### Common Issues

1. **401 Unauthorized**: Check that your Cerebras API key is correct and has sufficient credits
2. **429 Rate Limited**: Implement delays between requests or use exponential backoff
3. **Import Errors**: Ensure all template files are included in the build
4. **Zero Vectors**: If embeddings return zero vectors, check the API key configuration

### Debug Mode

Enable debug output by setting:
```env
DEBUG_MODE=true
LOG_LEVEL=debug
```

### Testing

You can test the integration with:
```bash
# Test agent building (includes embedding functionality)
go test -v ./agent/ -run TestAgentBuilder

# Test database with embeddings
go test -v ./database/ -run TestSimpleDomainDB

# Test real Cerebras API integration (uses actual API calls)
go test -v ./database/ -run TestCerebrasEmbeddingGeneration
```

Note: Real API tests will use your Cerebras credits and may hit rate limits.

## Advanced Configuration

### Custom Models

To use a different Cerebras model, update the `Model` field in the embedding client:

```go
client := NewCerebrasEmbeddingClient(apiKey)
client.Model = "llama-3.3-70b"  // Use a different model
```

### Custom Instructions

Modify the embedding instructions for specific use cases:

```go
embeddingFunc, err := NewEmbeddingFunction(
    apiKey,
    nil,
    WithTaskType("similarity"),
    WithInstruction("Generate embeddings optimized for similarity search"),
    WithNormalizeOutput(true),
)
```

## Integration with Your Repository

To use your full `chroma-go_cerebras` implementation:

1. Add the module dependency to `go.mod.template`:
```go
require (
    github.com/guiperry/chroma-go_cerebras v0.1.0
)
```

2. Update the import in `database/simple_domain_db.go`:
```go
import cerebras_embeddings "github.com/guiperry/chroma-go_cerebras/pkg/embeddings/cerebras"
```

3. Replace the placeholder implementation with your full implementation.

This provides a complete integration path for your Cerebras tool calling embeddings functionality.


---

<div class="footer-links">


© 2025 KNIRV Network
</div>
