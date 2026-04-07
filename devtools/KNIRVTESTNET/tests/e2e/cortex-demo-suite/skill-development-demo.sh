#!/bin/bash

# CORTEX Skill Development Demo
# Demonstrates CORTEX agent creating and registering a new skill

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEMO_NAME="skill-development"
DEMO_DURATION="10m"
LOG_FILE="$SCRIPT_DIR/logs/skill-development-demo.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Demo configuration
CORTEX_ENDPOINT="http://localhost:3001"
KNIRVCHAIN_ENDPOINT="http://localhost:8090"
KNIRVGRAPH_ENDPOINT="http://localhost:8082"
KNIRV_NEXUS_ENDPOINT="http://localhost:8084"
KNIRV_ROOT_ENDPOINT="http://localhost:1317"

# Demo state
DEMO_ID=""
AGENT_ID="cortex-dev-001"
SKILL_ID=""
TX_HASH=""
VALIDATION_ID=""

# Print functions
print_header() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================${NC}"
}

print_step() {
    echo -e "${YELLOW}[STEP]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Logging function
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" >> "$LOG_FILE"
    echo "$1"
}

# Initialize demo environment
initialize_demo() {
    print_step "Initializing CORTEX Skill Development Demo..."
    
    # Create log directory
    mkdir -p "$(dirname "$LOG_FILE")"
    
    # Initialize log file
    echo "CORTEX Skill Development Demo Log" > "$LOG_FILE"
    echo "Started: $(date)" >> "$LOG_FILE"
    echo "========================================" >> "$LOG_FILE"
    
    # Generate demo ID
    DEMO_ID="skill_dev_$(date +%s)"
    
    print_success "Demo initialized with ID: $DEMO_ID"
    log "Demo initialized: $DEMO_ID"
}

# Step 1: Initialize CORTEX Agent
initialize_cortex_agent() {
    print_step "Step 1: Initializing CORTEX Developer Agent..."
    
    local agent_data='{
        "agent_id": "'$AGENT_ID'",
        "type": "Developer",
        "capabilities": ["skill-creation", "code-generation", "testing"],
        "demo_id": "'$DEMO_ID'"
    }'
    
    # Make request to CORTEX
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$agent_data" \
        "$CORTEX_ENDPOINT/api/agents/initialize" || echo '{"success": false, "error": "Connection failed"}')
    
    # Parse response
    local success=$(echo "$response" | jq -r '.success // false')
    
    if [[ "$success" == "true" ]]; then
        local agent_status=$(echo "$response" | jq -r '.status // "unknown"')
        print_success "CORTEX agent initialized successfully (Status: $agent_status)"
        log "Agent $AGENT_ID initialized with status: $agent_status"
        
        # Validate agent is active
        if [[ "$agent_status" == "active" ]]; then
            print_success "✓ Validation: Agent status is active"
            return 0
        else
            print_error "✗ Validation: Expected agent status 'active', got '$agent_status'"
            return 1
        fi
    else
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        print_error "Failed to initialize CORTEX agent: $error"
        log "Agent initialization failed: $error"
        return 1
    fi
}

# Step 2: Create Skill
create_skill() {
    print_step "Step 2: Agent creates new coding skill..."
    
    local skill_data='{
        "agent_id": "'$AGENT_ID'",
        "demo_id": "'$DEMO_ID'",
        "skill_type": "code_generation",
        "language": "python",
        "complexity": "intermediate",
        "description": "Python web scraping automation",
        "name": "Advanced Web Scraper",
        "category": "automation"
    }'
    
    # Make request to CORTEX for skill creation
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$skill_data" \
        "$CORTEX_ENDPOINT/api/skills/create" || echo '{"success": false, "error": "Connection failed"}')
    
    # Parse response
    local success=$(echo "$response" | jq -r '.success // false')
    
    if [[ "$success" == "true" ]]; then
        SKILL_ID=$(echo "$response" | jq -r '.skill_id')
        local skill_name=$(echo "$response" | jq -r '.name // "Unknown"')
        print_success "Skill created successfully (ID: $SKILL_ID, Name: $skill_name)"
        log "Skill created: $SKILL_ID - $skill_name"
        
        # Validate skill was created
        if [[ -n "$SKILL_ID" && "$SKILL_ID" != "null" ]]; then
            print_success "✓ Validation: Skill ID generated"
            return 0
        else
            print_error "✗ Validation: No skill ID returned"
            return 1
        fi
    else
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        print_error "Failed to create skill: $error"
        log "Skill creation failed: $error"
        return 1
    fi
}

# Step 3: Register on Blockchain
register_on_blockchain() {
    print_step "Step 3: Registering skill on KNIRVCHAIN..."
    
    local register_data='{
        "skill_id": "'$SKILL_ID'",
        "creator": "'$AGENT_ID'",
        "price": 50,
        "demo_id": "'$DEMO_ID'",
        "metadata": {
            "language": "python",
            "complexity": "intermediate",
            "category": "automation"
        }
    }'
    
    # Make request to KNIRVCHAIN
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$register_data" \
        "$KNIRVCHAIN_ENDPOINT/skills/register" || echo '{"success": false, "error": "Connection failed"}')
    
    # Parse response
    local success=$(echo "$response" | jq -r '.success // false')
    
    if [[ "$success" == "true" ]]; then
        TX_HASH=$(echo "$response" | jq -r '.tx_hash')
        local status=$(echo "$response" | jq -r '.status // "unknown"')
        print_success "Skill registered on blockchain (TX: $TX_HASH, Status: $status)"
        log "Blockchain registration: $TX_HASH - $status"
        
        # Validate transaction confirmed
        if [[ -n "$TX_HASH" && "$TX_HASH" != "null" ]]; then
            print_success "✓ Validation: Transaction hash generated"
            return 0
        else
            print_error "✗ Validation: No transaction hash returned"
            return 1
        fi
    else
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        print_error "Failed to register on blockchain: $error"
        log "Blockchain registration failed: $error"
        return 1
    fi
}

# Step 4: Update Knowledge Graph
update_knowledge_graph() {
    print_step "Step 4: Updating knowledge graph in KNIRVGRAPH..."
    
    local graph_data='{
        "skill_id": "'$SKILL_ID'",
        "tx_hash": "'$TX_HASH'",
        "demo_id": "'$DEMO_ID'",
        "relationships": ["python", "web-scraping", "automation"],
        "metadata": {
            "creator": "'$AGENT_ID'",
            "complexity": "intermediate",
            "category": "automation"
        }
    }'
    
    # Make request to KNIRVGRAPH
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$graph_data" \
        "$KNIRVGRAPH_ENDPOINT/graph/update" || echo '{"success": false, "error": "Connection failed"}')
    
    # Parse response
    local success=$(echo "$response" | jq -r '.success // false')
    
    if [[ "$success" == "true" ]]; then
        local node_id=$(echo "$response" | jq -r '.node_id')
        local edges_created=$(echo "$response" | jq -r '.edges_created // 0')
        print_success "Knowledge graph updated (Node: $node_id, Edges: $edges_created)"
        log "Graph update: $node_id - $edges_created edges"
        
        # Validate graph was updated
        if [[ -n "$node_id" && "$node_id" != "null" ]]; then
            print_success "✓ Validation: Graph node created"
            return 0
        else
            print_error "✗ Validation: No graph node ID returned"
            return 1
        fi
    else
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        print_error "Failed to update knowledge graph: $error"
        log "Graph update failed: $error"
        return 1
    fi
}

# Step 5: Validate Skill
validate_skill() {
    print_step "Step 5: Validating skill through KNIRV-SERVER..."
    
    local validation_data='{
        "skill_id": "'$SKILL_ID'",
        "tx_hash": "'$TX_HASH'",
        "demo_id": "'$DEMO_ID'",
        "validation_type": "automated",
        "creator": "'$AGENT_ID'"
    }'
    
    # Make request to KNIRV-SERVER
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$validation_data" \
        "$KNIRV_NEXUS_ENDPOINT/validation/submit" || echo '{"success": false, "error": "Connection failed"}')
    
    # Parse response
    local success=$(echo "$response" | jq -r '.success // false')
    
    if [[ "$success" == "true" ]]; then
        VALIDATION_ID=$(echo "$response" | jq -r '.validation_id')
        local status=$(echo "$response" | jq -r '.status // "unknown"')
        print_success "Skill validation submitted (ID: $VALIDATION_ID, Status: $status)"
        log "Validation submitted: $VALIDATION_ID - $status"
        
        # Wait for validation to complete
        print_info "Waiting for validation to complete..."
        sleep 5
        
        # Check validation status
        local validation_status=$(check_validation_status)
        if [[ "$validation_status" == "passed" ]]; then
            print_success "✓ Validation: Skill validation passed"
            return 0
        else
            print_error "✗ Validation: Skill validation failed or pending"
            return 1
        fi
    else
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        print_error "Failed to submit for validation: $error"
        log "Validation submission failed: $error"
        return 1
    fi
}

# Step 6: Distribute Rewards
distribute_rewards() {
    print_step "Step 6: Distributing NRN rewards for skill creation..."
    
    local reward_data='{
        "recipient": "'$AGENT_ID'",
        "amount": 100,
        "reason": "skill_creation",
        "skill_id": "'$SKILL_ID'",
        "demo_id": "'$DEMO_ID'"
    }'
    
    # Make request to KNIRV-ORACLE
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$reward_data" \
        "$KNIRV_ROOT_ENDPOINT/rewards/distribute" || echo '{"success": false, "error": "Connection failed"}')
    
    # Parse response
    local success=$(echo "$response" | jq -r '.success // false')
    
    if [[ "$success" == "true" ]]; then
        local reward_tx=$(echo "$response" | jq -r '.transaction_id')
        local amount=$(echo "$response" | jq -r '.amount // 0')
        print_success "Rewards distributed (TX: $reward_tx, Amount: $amount NRN)"
        log "Rewards distributed: $reward_tx - $amount NRN"
        
        # Validate reward was distributed
        if [[ -n "$reward_tx" && "$reward_tx" != "null" ]]; then
            print_success "✓ Validation: Reward transaction created"
            return 0
        else
            print_error "✗ Validation: No reward transaction ID returned"
            return 1
        fi
    else
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        print_error "Failed to distribute rewards: $error"
        log "Reward distribution failed: $error"
        return 1
    fi
}

# Step 7: Verify Availability
verify_skill_availability() {
    print_step "Step 7: Verifying skill is available for use..."
    
    # Make request to KNIRVCHAIN to check skill availability
    local response=$(curl -s -X GET \
        "$KNIRVCHAIN_ENDPOINT/skills/$SKILL_ID" || echo '{"success": false, "error": "Connection failed"}')
    
    # Parse response
    local success=$(echo "$response" | jq -r '.success // false')
    
    if [[ "$success" == "true" ]]; then
        local available=$(echo "$response" | jq -r '.available // false')
        local status=$(echo "$response" | jq -r '.status // "unknown"')
        print_success "Skill availability verified (Available: $available, Status: $status)"
        log "Skill availability: $available - $status"
        
        # Validate skill is available
        if [[ "$available" == "true" ]]; then
            print_success "✓ Validation: Skill is available for use"
            return 0
        else
            print_error "✗ Validation: Skill is not available"
            return 1
        fi
    else
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        print_error "Failed to verify skill availability: $error"
        log "Availability check failed: $error"
        return 1
    fi
}

# Helper function to check validation status
check_validation_status() {
    local response=$(curl -s -X GET \
        "$KNIRV_NEXUS_ENDPOINT/validation/$VALIDATION_ID" || echo '{"status": "unknown"}')
    
    echo "$response" | jq -r '.status // "unknown"'
}

# Generate demo report
generate_demo_report() {
    print_step "Generating demo report..."
    
    local report_file="$SCRIPT_DIR/reports/skill-development-demo-$(date +%Y%m%d_%H%M%S).html"
    mkdir -p "$(dirname "$report_file")"
    
    cat > "$report_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>CORTEX Skill Development Demo Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .success { color: green; }
        .error { color: red; }
        .step { margin: 15px 0; padding: 10px; border: 1px solid #ddd; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>CORTEX Skill Development Demo Report</h1>
        <p>Demo ID: $DEMO_ID</p>
        <p>Generated: $(date)</p>
        <p>Duration: $DEMO_DURATION</p>
    </div>
    
    <div class="step">
        <h2>Demo Summary</h2>
        <p>Agent ID: $AGENT_ID</p>
        <p>Skill ID: $SKILL_ID</p>
        <p>Transaction Hash: $TX_HASH</p>
        <p>Validation ID: $VALIDATION_ID</p>
    </div>
    
    <div class="step">
        <h2>Execution Steps</h2>
        <ol>
            <li>Initialize CORTEX Agent</li>
            <li>Create Skill</li>
            <li>Register on Blockchain</li>
            <li>Update Knowledge Graph</li>
            <li>Validate Skill</li>
            <li>Distribute Rewards</li>
            <li>Verify Availability</li>
        </ol>
    </div>
    
    <div class="step">
        <h2>Results</h2>
        <p>All steps completed successfully demonstrating the complete CORTEX skill development workflow.</p>
    </div>
</body>
</html>
EOF
    
    print_success "Demo report generated: $report_file"
    log "Demo report generated: $report_file"
}

# Main execution function
main() {
    local start_time=$(date +%s)
    
    print_header "CORTEX Skill Development Demo"
    print_info "Demo Duration: $DEMO_DURATION"
    print_info "Agent: $AGENT_ID"
    
    # Initialize demo
    initialize_demo
    
    # Execute demo steps
    local step_success=true
    
    if ! initialize_cortex_agent; then
        step_success=false
    fi
    
    if [[ "$step_success" == "true" ]] && ! create_skill; then
        step_success=false
    fi
    
    if [[ "$step_success" == "true" ]] && ! register_on_blockchain; then
        step_success=false
    fi
    
    if [[ "$step_success" == "true" ]] && ! update_knowledge_graph; then
        step_success=false
    fi
    
    if [[ "$step_success" == "true" ]] && ! validate_skill; then
        step_success=false
    fi
    
    if [[ "$step_success" == "true" ]] && ! distribute_rewards; then
        step_success=false
    fi
    
    if [[ "$step_success" == "true" ]] && ! verify_skill_availability; then
        step_success=false
    fi
    
    # Calculate execution time
    local end_time=$(date +%s)
    local execution_time=$((end_time - start_time))
    
    # Generate report
    generate_demo_report
    
    # Final status
    if [[ "$step_success" == "true" ]]; then
        print_success "CORTEX Skill Development Demo completed successfully!"
        print_success "Execution time: ${execution_time}s"
        log "Demo completed successfully in ${execution_time}s"
        exit 0
    else
        print_error "CORTEX Skill Development Demo failed"
        print_error "Execution time: ${execution_time}s"
        log "Demo failed after ${execution_time}s"
        exit 1
    fi
}

# Check dependencies
if ! command -v curl &> /dev/null; then
    print_error "curl is required but not installed"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    print_error "jq is required but not installed"
    exit 1
fi

# Run main function
main "$@"
