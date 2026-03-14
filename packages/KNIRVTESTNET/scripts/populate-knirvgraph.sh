#!/bin/bash
set -e

echo "Populating KNIRVGRAPH with sample data..."

# Wait for KNIRVGRAPH to be ready
echo "Waiting for KNIRVGRAPH to be ready..."
for i in {1..30}; do
    if curl -s -f http://localhost:8081/health > /dev/null 2>&1; then
        echo "KNIRVGRAPH is ready!"
        break
    fi
    echo "Attempt $i/30: KNIRVGRAPH not ready yet..."
    sleep 2
done

# Sample ErrorNodes
echo "Creating sample ErrorNodes..."

curl -X POST http://localhost:8081/api/v1/error-nodes \
  -H "Content-Type: application/json" \
  -d '{
    "id": "error_001",
    "description": "Division by zero in calculation module",
    "category": "arithmetic_error",
    "severity": "high",
    "context": {
      "module": "calculator",
      "function": "divide",
      "input": {"a": 10, "b": 0}
    }
  }' || echo "Failed to create error_001"

curl -X POST http://localhost:8081/api/v1/error-nodes \
  -H "Content-Type: application/json" \
  -d '{
    "id": "error_002", 
    "description": "Memory allocation failure",
    "category": "memory_error",
    "severity": "critical",
    "context": {
      "module": "allocator",
      "requested_size": "1GB",
      "available_memory": "512MB"
    }
  }' || echo "Failed to create error_002"

curl -X POST http://localhost:8081/api/v1/error-nodes \
  -H "Content-Type: application/json" \
  -d '{
    "id": "error_003",
    "description": "JSON parsing error with malformed input",
    "category": "parsing_error",
    "severity": "medium",
    "context": {
      "module": "json_parser",
      "input": "{invalid_json}",
      "error": "unexpected token"
    }
  }' || echo "Failed to create error_003"

# Sample SkillNodes
echo "Creating sample SkillNodes..."

curl -X POST http://localhost:8081/api/v1/skill-nodes \
  -H "Content-Type: application/json" \
  -d '{
    "id": "skill_001",
    "name": "Safe Division",
    "description": "Performs division with zero-check",
    "resolves": ["error_001"],
    "code_hash": "QmSafeDivisionSkill...",
    "validation_proof": "proof_001"
  }' || echo "Failed to create skill_001"

curl -X POST http://localhost:8081/api/v1/skill-nodes \
  -H "Content-Type: application/json" \
  -d '{
    "id": "skill_002",
    "name": "Memory Manager",
    "description": "Safe memory allocation with fallback",
    "resolves": ["error_002"],
    "code_hash": "QmMemoryManagerSkill...",
    "validation_proof": "proof_002"
  }' || echo "Failed to create skill_002"

curl -X POST http://localhost:8081/api/v1/skill-nodes \
  -H "Content-Type: application/json" \
  -d '{
    "id": "skill_003",
    "name": "Safe JSON Parser",
    "description": "JSON parser with error handling",
    "resolves": ["error_003"],
    "code_hash": "QmSafeJSONParserSkill...",
    "validation_proof": "proof_003"
  }' || echo "Failed to create skill_003"

echo "Sample data populated successfully!"
