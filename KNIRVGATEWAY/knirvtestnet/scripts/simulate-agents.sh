#!/bin/bash
set -e

echo "=========================================="
echo "KNIRV-TESTNET: Agent Simulation"
echo "=========================================="

# Already in KNIRVTESTNET directory

echo "🤖 Starting agent simulation..."

# Simulate agent discovering an error
echo "🔍 Step 1: Agent discovering an error..."
curl -X POST http://localhost:8087/api/graph/error-nodes \
  -H "Content-Type: application/json" \
  -d '{
    "id": "error_sim_001",
    "description": "Simulated parsing error in JSON handler",
    "category": "parsing_error",
    "severity": "medium",
    "context": {
      "module": "json_parser",
      "input": "{invalid_json}",
      "error": "unexpected token"
    }
  }' > /dev/null 2>&1 && echo "    ✅ Error node created" || echo "    ⚠️  Error creation failed (using direct API)"

# Try direct KNIRVGRAPH API
curl -X POST http://localhost:8081/api/v1/error-nodes \
  -H "Content-Type: application/json" \
  -d '{
    "id": "error_sim_001",
    "description": "Simulated parsing error in JSON handler",
    "category": "parsing_error",
    "severity": "medium",
    "context": {
      "module": "json_parser",
      "input": "{invalid_json}",
      "error": "unexpected token"
    }
  }' > /dev/null 2>&1 && echo "    ✅ Error node created via KNIRVGRAPH" || echo "    ⚠️  Error creation via KNIRVGRAPH failed"

# Simulate agent proposing a solution
echo "💡 Step 2: Agent proposing a solution..."
curl -X POST http://localhost:8087/api/nexus/validate/skill \
  -H "Content-Type: application/json" \
  -d '{
    "skill_code": "function safeJsonParse(input) { try { return JSON.parse(input); } catch(e) { return null; } }",
    "test_cases": [
      {"input": "{\"valid\": true}", "expected": {"valid": true}},
      {"input": "{invalid}", "expected": null}
    ],
    "resolves": ["error_sim_001"]
  }' > /dev/null 2>&1 && echo "    ✅ Solution proposed via gateway" || echo "    ⚠️  Solution proposal failed (using direct API)"

# Try direct KNIRV-NEXUS API
curl -X POST http://localhost:8082/validate/skill \
  -H "Content-Type: application/json" \
  -d '{
    "skill_code": "function safeJsonParse(input) { try { return JSON.parse(input); } catch(e) { return null; } }",
    "test_cases": [
      {"input": "{\"valid\": true}", "expected": {"valid": true}},
      {"input": "{invalid}", "expected": null}
    ],
    "resolves": ["error_sim_001"]
  }' > /dev/null 2>&1 && echo "    ✅ Solution validated via KNIRV-NEXUS" || echo "    ⚠️  Solution validation failed"

# Simulate SEAL loop iteration
echo "🔄 Step 3: SEAL loop iteration..."
echo "    🧠 Agent analyzing performance..."
echo "    📊 Collecting failure metrics..."
echo "    🎯 Identifying improvement opportunities..."
echo "    🔧 Generating LoRA adapter updates..."
echo "    ✅ SEAL loop iteration completed"

# Simulate skill registration
echo "📝 Step 4: Skill registration..."
curl -X POST http://localhost:8080/v2/skill/register \
  -H "Content-Type: application/json" \
  -d '{
    "skill_id": "skill_sim_001",
    "name": "Safe JSON Parser",
    "description": "JSON parser with comprehensive error handling",
    "code_hash": "QmSimulatedSkillHash123...",
    "validation_proof": "proof_sim_001",
    "resolves": ["error_sim_001"]
  }' > /dev/null 2>&1 && echo "    ✅ Skill registered" || echo "    ⚠️  Skill registration failed"

echo ""
echo "=========================================="
echo "🎉 Agent simulation completed!"
echo "=========================================="
echo ""
echo "📊 Simulation Summary:"
echo "  🔍 Error discovery: Simulated"
echo "  💡 Solution proposal: Simulated"
echo "  🔄 SEAL loop: Simulated"
echo "  📝 Skill registration: Attempted"
echo ""
echo "📝 Note: In testnet mode, agent operations are simulated."
echo "🔍 Check ./logs/ for detailed service interactions."
