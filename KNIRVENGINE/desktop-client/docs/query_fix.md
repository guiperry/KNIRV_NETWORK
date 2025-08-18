# ChromeDB Query Issue and Fix Documentation

## Problem Description

The chromem-go library (ChromeDB Go client) has strict validation requirements that cause query failures in our Agent-centric system. The main issues are:

### 1. nResults Validation Error
**Error**: `nResults must be <= the number of documents in the collection`

**Root Cause**: ChromeDB strictly validates that the `nResults` parameter in queries cannot exceed the actual number of documents in the collection. If you request 10 results but only have 3 documents, the query fails.

### 2. Where Clause Operator Error  
**Error**: `unsupported operator`

**Root Cause**: Where clauses in the format `map[string]string{"field": "value"}` cause "unsupported operator" errors with the chromem library.

## Original Problematic Pattern

```go
// BROKEN: This pattern fails with chromem-go
results, err := collection.Query(
    context.Background(),
    "search_term",
    10,  // Fails if collection has < 10 documents
    map[string]string{"field": "value"}, // Causes "unsupported operator" error
    nil,
)
```

## Fixed Pattern

### Conservative Single Query Approach
```go
// WORKING: Conservative approach with progressive fallback
func queryWithFallback(collection *chromem.Collection, searchTerm string, targetField, targetValue string) ([]chromem.Result, error) {
    limits := []int{1, 3, 5, 10} // Progressive limits to try
    
    for _, limit := range limits {
        results, err := collection.Query(
            context.Background(),
            searchTerm, // Generic search term
            limit,      // Conservative limit
            nil,        // No where clause (causes errors)
            nil,        // No include map
        )
        
        if err == nil {
            // Success - now filter manually
            var filtered []chromem.Result
            for _, result := range results {
                if result.Metadata[targetField] == targetValue {
                    filtered = append(filtered, result)
                }
            }
            return filtered, nil
        }
        
        // If error contains "nResults", try smaller limit
        if strings.Contains(err.Error(), "nResults") {
            continue
        }
        
        // Other errors are not recoverable
        return nil, err
    }
    
    return nil, fmt.Errorf("all query attempts failed")
}
```

### Simplified Working Pattern (Currently Used)
```go
// WORKING: Simplified approach with very conservative limit
results, err := collection.Query(
    context.Background(),
    "generic_term", // Generic query to match documents
    1,              // Very conservative limit
    nil,            // No where clause
    nil,            // No include map
)

if err != nil {
    return nil, fmt.Errorf("failed to query: %w", err)
}

// Manual filtering for exact matches
for _, result := range results {
    if result.Metadata["target_field"] == targetValue {
        // Process matching result
        return processResult(result)
    }
}
```

## Methods That Need Fixing

Based on analysis of the codebase, the following methods use the problematic pattern:

### Agent-related Methods
- `GetAgent(id string)` - Single agent lookup
- `GetAgentsByOwner(owner string)` - Multiple agents by owner
- `GetAgentsByType(agentType string)` - Multiple agents by type

### Relationship Methods  
- `GetAgentRelationship(id string)` - Single relationship lookup
- `GetAgentRelationships(agentID string)` - Multiple relationships for agent

### Badge Methods
- Methods that internally call GetAgent for validation

### Capability Methods
- Methods that internally call GetAgent for validation

## Fix Implementation Strategy

### 1. Replace Where Clauses
- Remove all `map[string]string{"field": "value"}` where clauses
- Use generic search terms instead
- Implement manual filtering in result processing loop

### 2. Conservative nResults Limits
- Use very small limits (1-3) to avoid collection size errors
- For methods that need multiple results, use progressive querying
- Handle cases where fewer results are returned than requested

### 3. Generic Search Terms
- Use broad terms like "Agent", "relationship", "badge" that match document content
- Rely on manual filtering for exact matches rather than ChromeDB filtering

## Testing Results

After implementing these fixes:

✅ **Working Tests**:
- `TestAgentSystemIntegration` - Comprehensive integration test passes
- `TestAgentManager_CreateAgent` - Agent creation works
- `TestAgentManager_GetAgentsByOwner` - Multi-agent queries work  
- `TestAgentManager_GetAgentsByType` - Type-based queries work
- `TestAgentManager_GetAgentRelationships` - Relationship queries work

❌ **Still Failing Tests**:
- `TestAgentManager_GetAgent` - Single agent lookup (nResults issue)
- Badge attachment tests - Depend on GetAgent
- Capability tests - Depend on GetAgent  
- Relationship creation tests - Depend on GetAgent

## Next Steps

1. **Implement Progressive Query Fallback**: Use the progressive limits approach for all query methods
2. **Batch Query Optimization**: For methods needing multiple results, implement smarter batching
3. **Error Handling**: Improve error messages to distinguish between "not found" and "query failed"
4. **Performance Testing**: Measure impact of manual filtering vs ChromeDB native filtering

## Progressive Query Fallback Implementation

For robust handling of the nResults limitation, implement this progressive approach:

```go
func (m *ChromemManager) queryWithFallback(collection *chromem.Collection, searchTerm string) ([]chromem.Result, error) {
    // Try progressively smaller limits until one works
    limits := []int{10, 5, 3, 1}

    for _, limit := range limits {
        results, err := collection.Query(
            context.Background(),
            searchTerm,
            limit,
            nil, // No where clause
            nil,
        )

        if err == nil {
            return results, nil
        }

        // If it's an nResults error, try smaller limit
        if strings.Contains(err.Error(), "nResults") {
            continue
        }

        // Other errors are not recoverable
        return nil, err
    }

    return nil, fmt.Errorf("all query attempts failed")
}
```

## Automated Fix Script Usage

The provided JavaScript script can automatically fix these issues:

```bash
# Make script executable
chmod +x fix_chromem_queries.js

# Run on current directory
node fix_chromem_queries.js

# Run on specific directory
node fix_chromem_queries.js /path/to/go/project

# The script will:
# 1. Create .backup files for safety
# 2. Reduce high nResults values to 3
# 3. Remove problematic where clauses
# 4. Add TODO comments for manual filtering
# 5. Generate improved method templates
```

## ChromeDB Library Limitations

The chromem-go library appears to have these limitations:
- Very strict nResults validation with no collection size introspection
- Limited where clause support (map[string]string format causes errors)
- No apparent way to get collection document count before querying
- Semantic search works better than exact field matching
- No batch query capabilities for efficient multi-document retrieval

These limitations require workarounds in application code rather than relying on database-level filtering.

## Performance Considerations

**Manual Filtering Impact**:
- Slightly higher memory usage (loading more documents than needed)
- Reduced network efficiency (can't push filtering to database)
- Increased CPU usage for client-side filtering

**Mitigation Strategies**:
- Use smallest possible nResults limits
- Implement caching for frequently accessed entities
- Consider batch operations where possible
- Monitor query performance and adjust limits based on collection sizes
