# LLM Reasoning Validation - Implementation Guide

## Overview

This implementation provides a **highly deterministic** validation framework for LLM outputs, with clear separation between deterministic validators (100% reliable) and LLM-based validators (require external API calls).

## Architecture

```
┌─────────────────────────────────────────────────┐
│         ValidationOrchestrator                  │
│  - Coordinates all validators                   │
│  - Manages execution flow                       │
│  - Aggregates results                           │
└────────────┬────────────────────────────────────┘
             │
             ├──► Deterministic Validators (Pure Go)
             │    ├─ Keyword Presence
             │    ├─ Forbidden Content
             │    ├─ Output Length
             │    ├─ Structural Patterns
             │    ├─ Contradiction Detection
             │    └─ JSON Format
             │
             └──► LLM-Based Validators (Require APIs)
                  ├─ Reasoning Quality
                  └─ Factual Accuracy
```

## Deterministic Validators (No External Calls)

### 1. KeywordPresenceValidator
**Purpose**: Ensures required terms appear in output  
**Use Case**: Technical documents, compliance checks  
**Deterministic**: ✓ Yes (100% reliable)

```go
validator := &KeywordPresenceValidator{
    RequiredKeywords: []string{"risk", "mitigation", "compliance"},
    CaseSensitive: false,
}
```

### 2. ForbiddenContentValidator
**Purpose**: Blocks unwanted terms/patterns  
**Use Case**: Safety, brand guidelines, content policy  
**Deterministic**: ✓ Yes

```go
validator := &ForbiddenContentValidator{
    ForbiddenPatterns: []string{
        "absolutely certain",  // Overconfident language
        "guaranteed to work",  // Unrealistic claims
        `\b\d{3}-\d{2}-\d{4}\b`, // SSN pattern (regex)
    },
    UseRegex: true,
}
```

### 3. OutputLengthValidator
**Purpose**: Enforces response length constraints  
**Use Case**: UX requirements, cost optimization  
**Deterministic**: ✓ Yes

```go
validator := &OutputLengthValidator{
    MinWords: 50,
    MaxWords: 500,
    MinChars: 200,
    MaxChars: 2000,
}
```

### 4. StructuralPatternValidator
**Purpose**: Checks for expected structural elements  
**Use Case**: Format compliance (e.g., "Step 1:", "Conclusion:")  
**Deterministic**: ✓ Yes

```go
validator := &StructuralPatternValidator{
    RequiredPatterns: []string{
        "Step 1:",
        "Step 2:",
        "Conclusion:",
    },
    MinOccurrences: 1,
}
```

### 5. ContradictionDetector
**Purpose**: Identifies contradictory statements  
**Use Case**: Logical consistency checks  
**Deterministic**: ✓ Yes (pattern-based)

```go
validator := &ContradictionDetector{
    ContradictionPairs: [][]string{
        {"increase", "decrease"},
        {"safe", "dangerous"},
        {"always", "never"},
    },
}
```

### 6. JSONFormatValidator
**Purpose**: Validates JSON structure and required fields  
**Use Case**: API responses, structured data  
**Deterministic**: ✓ Yes

```go
validator := &JSONFormatValidator{
    RequireValidJSON: true,
    RequiredKeys: []string{"status", "data", "timestamp"},
}
```

## LLM-Based Validators (Require Implementation)

### Setting Up External LLM Calls

You need to implement the `LLMEvaluatorClient` interface. Here are specific implementation examples:

#### Option 1: OpenAI API

```go
package validation

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
)

type OpenAIEvaluator struct {
    APIKey string
    Model  string // e.g., "gpt-4" or "gpt-3.5-turbo"
}

func (o *OpenAIEvaluator) EvaluateReasoning(ctx context.Context, prompt, output, criteria string) (float64, string, error) {
    evaluationPrompt := fmt.Sprintf(`
You are evaluating the reasoning quality of an LLM response.

Original Prompt: %s

LLM Response: %s

Evaluation Criteria: %s

Provide:
1. A score from 0.0 to 1.0 (0=poor, 1=excellent)
2. A brief explanation

Respond in JSON format:
{"score": 0.85, "explanation": "..."}
`, prompt, output, criteria)

    reqBody := map[string]interface{}{
        "model": o.Model,
        "messages": []map[string]string{
            {"role": "user", "content": evaluationPrompt},
        },
        "temperature": 0.1, // Low temperature for consistency
    }

    // Make HTTP request to OpenAI
    // ... (implement HTTP client logic)
    
    // Parse response
    var result struct {
        Score       float64 `json:"score"`
        Explanation string  `json:"explanation"`
    }
    
    return result.Score, result.Explanation, nil
}

func (o *OpenAIEvaluator) CheckFactualClaim(ctx context.Context, claim string) (bool, string, error) {
    checkPrompt := fmt.Sprintf(`
Is this claim factually accurate? Answer with high confidence only.

Claim: %s

Respond in JSON format:
{"is_accurate": true/false, "explanation": "..."}
`, claim)

    // Similar HTTP request logic
    // ...
    
    var result struct {
        IsAccurate  bool   `json:"is_accurate"`
        Explanation string `json:"explanation"`
    }
    
    return result.IsAccurate, result.Explanation, nil
}
```

