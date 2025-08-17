#!/bin/bash

# KNIRVANA Test Data Setup Script
# Creates mock data and test fixtures for both Rust and TypeScript implementations

set -e

echo "Setting up KNIRVANA test data..."

# Create test data directories
mkdir -p test-data/{rust,typescript,shared}
mkdir -p test-data/fixtures/{players,games,skills,errors}
mkdir -p test-data/mocks/{api,blockchain,network}

# Create shared test data
cat > test-data/shared/test-players.json << 'EOF'
[
  {
    "id": "player_001",
    "name": "TestPlayer1",
    "nrnBalance": 1000.0,
    "reputation": 85,
    "color": "#FF6B6B",
    "agents": [
      {
        "id": "agent_001",
        "type": "resolver",
        "specialization": "debugging",
        "energy": 100,
        "maxEnergy": 100,
        "skills": ["javascript", "rust", "debugging"]
      }
    ]
  },
  {
    "id": "player_002",
    "name": "TestPlayer2",
    "nrnBalance": 750.0,
    "reputation": 92,
    "color": "#4ECDC4",
    "agents": [
      {
        "id": "agent_002",
        "type": "observer",
        "specialization": "analysis",
        "energy": 80,
        "maxEnergy": 100,
        "skills": ["python", "data-analysis", "monitoring"]
      }
    ]
  }
]
EOF

# Create test error nodes
cat > test-data/shared/test-errors.json << 'EOF'
[
  {
    "id": "error_001",
    "type": "error",
    "difficulty": 3,
    "bounty": 50.0,
    "description": "Memory leak in game loop",
    "progress": 0,
    "isBeingResolved": false,
    "position": {"x": 10, "y": 5, "z": 0}
  },
  {
    "id": "error_002",
    "type": "error",
    "difficulty": 5,
    "bounty": 100.0,
    "description": "Race condition in networking module",
    "progress": 25,
    "isBeingResolved": true,
    "resolvedBy": "player_001",
    "position": {"x": -5, "y": 8, "z": 3}
  }
]
EOF

# Create test skill nodes
cat > test-data/shared/test-skills.json << 'EOF'
[
  {
    "id": "skill_001",
    "type": "skill",
    "name": "Advanced Debugging",
    "category": "debugging",
    "createdBy": "player_001",
    "usageCount": 15,
    "value": 25.0,
    "position": {"x": 0, "y": 0, "z": 0}
  },
  {
    "id": "skill_002",
    "type": "skill",
    "name": "Performance Optimization",
    "category": "optimization",
    "createdBy": "player_002",
    "usageCount": 8,
    "value": 40.0,
    "position": {"x": 15, "y": -3, "z": 7}
  }
]
EOF

# Create test game states
cat > test-data/shared/test-game-states.json << 'EOF'
[
  {
    "id": "game_001",
    "phase": "active",
    "timeRemaining": 1800,
    "networkActivity": 75,
    "currentPlayerId": "player_001"
  },
  {
    "id": "game_002",
    "phase": "lobby",
    "timeRemaining": 0,
    "networkActivity": 0,
    "currentPlayerId": null
  }
]
EOF

# Create Rust-specific test data
cat > test-data/rust/test-config.toml << 'EOF'
[game]
debug_mode = true
test_mode = true
max_players = 4
world_size = 1000.0

[graphics]
quality = "low"
vsync = false
fullscreen = false

[network]
server_url = "ws://localhost:8080/test"
timeout_ms = 5000
retry_attempts = 3

[mobile]
touch_sensitivity = 1.0
battery_optimization = true
reduced_effects = true
EOF

# Create TypeScript-specific test data
cat > test-data/typescript/test-config.json << 'EOF'
{
  "game": {
    "debugMode": true,
    "testMode": true,
    "maxPlayers": 4,
    "worldSize": 1000.0
  },
  "graphics": {
    "quality": "low",
    "vsync": false,
    "fullscreen": false
  },
  "network": {
    "serverUrl": "ws://localhost:8080/test",
    "timeoutMs": 5000,
    "retryAttempts": 3
  },
  "ui": {
    "theme": "dark",
    "animations": false,
    "soundEnabled": false
  }
}
EOF

# Create mock API responses
cat > test-data/mocks/api/player-stats.json << 'EOF'
{
  "success": true,
  "data": {
    "playerId": "player_001",
    "stats": {
      "gamesPlayed": 42,
      "errorsResolved": 156,
      "skillsCreated": 23,
      "totalEarnings": 2450.75,
      "averageRating": 4.7
    },
    "achievements": [
      "first_error_resolved",
      "skill_creator",
      "top_performer"
    ]
  }
}
EOF

# Create mock blockchain responses
cat > test-data/mocks/blockchain/nrn-balance.json << 'EOF'
{
  "success": true,
  "data": {
    "address": "xion1test...",
    "balance": "1000.500000",
    "denom": "nrn",
    "lastUpdated": "2024-01-15T10:30:00Z"
  }
}
EOF

# Create network simulation data
cat > test-data/mocks/network/latency-simulation.json << 'EOF'
{
  "scenarios": [
    {
      "name": "low_latency",
      "latency_ms": 50,
      "packet_loss": 0.1,
      "jitter_ms": 5
    },
    {
      "name": "high_latency",
      "latency_ms": 300,
      "packet_loss": 2.0,
      "jitter_ms": 50
    },
    {
      "name": "unstable_connection",
      "latency_ms": 150,
      "packet_loss": 5.0,
      "jitter_ms": 100
    }
  ]
}
EOF

# Create test environment variables
cat > test-data/.env.test << 'EOF'
# Test Environment Configuration
NODE_ENV=test
RUST_LOG=debug
RUST_BACKTRACE=1

# Test Database
DATABASE_URL=sqlite://test.db

# Test Network
NETWORK_MODE=test
SERVER_PORT=8080
WS_PORT=8081

# Test Blockchain
BLOCKCHAIN_NETWORK=testnet
NRN_CONTRACT_ADDRESS=test_contract_123

# Test Features
ENABLE_PERFORMANCE_MONITORING=true
ENABLE_DEBUG_LOGGING=true
MOCK_BLOCKCHAIN=true
MOCK_NETWORK=true
EOF

echo "Test data setup complete!"
echo "Created test fixtures in test-data/ directory"
echo "Use 'source test-data/.env.test' to load test environment variables"
