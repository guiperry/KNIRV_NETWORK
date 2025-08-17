#!/bin/bash

# CORTEX Multi-Agent Collaboration Demo
# Demonstrates multiple CORTEX agents collaborating on a complex task

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEMO_NAME="multi-agent-collaboration"
DEMO_DURATION="15m"
LOG_FILE="$SCRIPT_DIR/logs/collaboration-demo.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Demo configuration
CORTEX_ENDPOINT="http://localhost:3001"
KNIRV_ROUTER_ENDPOINT="http://localhost:5001"
KNIRVGRAPH_ENDPOINT="http://localhost:8082"
KNIRV_ROOT_ENDPOINT="http://localhost:1317"

# Demo state
DEMO_ID=""
COORDINATOR_AGENT="cortex-dev-001"
SPECIALIST_A="cortex-collab-001"
SPECIALIST_B="cortex-learner-001"
TASK_ID=""
SESSION_ID=""
COLLABORATION_RESULT=""

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
    print_step "Initializing CORTEX Multi-Agent Collaboration Demo..."
    
    # Create log directory
    mkdir -p "$(dirname "$LOG_FILE")"
    
    # Initialize log file
    echo "CORTEX Multi-Agent Collaboration Demo Log" > "$LOG_FILE"
    echo "Started: $(date)" >> "$LOG_FILE"
    echo "========================================" >> "$LOG_FILE"
    
    # Generate demo ID
    DEMO_ID="collab_$(date +%s)"
    
    print_success "Demo initialized with ID: $DEMO_ID"
    log "Demo initialized: $DEMO_ID"
}

# Step 1: Initialize All Agents
initialize_agents() {
    print_step "Step 1: Initializing all participating agents..."
    
    local agents=("$COORDINATOR_AGENT" "$SPECIALIST_A" "$SPECIALIST_B")
    local agent_types=("task_coordinator" "specialist_a" "specialist_b")
    local capabilities=(
        '["task-decomposition", "coordination", "integration"]'
        '["data-analysis", "pattern-recognition"]'
        '["optimization", "quality-assurance"]'
    )
    
    for i in "${!agents[@]}"; do
        local agent_id="${agents[$i]}"
        local agent_type="${agent_types[$i]}"
        local agent_caps="${capabilities[$i]}"
        
        print_info "Initializing agent: $agent_id ($agent_type)"
        
        local agent_data='{
            "agent_id": "'$agent_id'",
            "type": "'$agent_type'",
            "capabilities": '$agent_caps',
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
            local status=$(echo "$response" | jq -r '.status // "unknown"')
            print_success "Agent $agent_id initialized (Status: $status)"
            log "Agent $agent_id initialized with status: $status"
        else
            local error=$(echo "$response" | jq -r '.error // "Unknown error"')
            print_error "Failed to initialize agent $agent_id: $error"
            log "Agent $agent_id initialization failed: $error"
            return 1
        fi
    done
    
    print_success "✓ Validation: All agents are active"
    return 0
}

# Step 2: Receive Complex Task
receive_complex_task() {
    print_step "Step 2: Coordinator receives complex analysis task..."
    
    local task_data='{
        "agent_id": "'$COORDINATOR_AGENT'",
        "demo_id": "'$DEMO_ID'",
        "task_type": "market_analysis",
        "complexity": "high",
        "deadline": "10m",
        "requirements": ["data_processing", "pattern_analysis", "optimization"],
        "description": "Comprehensive market trend analysis with predictive modeling"
    }'
    
    # Make request to CORTEX
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$task_data" \
        "$CORTEX_ENDPOINT/api/tasks/receive" || echo '{"success": false, "error": "Connection failed"}')
    
    # Parse response
    local success=$(echo "$response" | jq -r '.success // false')
    
    if [[ "$success" == "true" ]]; then
        TASK_ID=$(echo "$response" | jq -r '.task_id')
        local status=$(echo "$response" | jq -r '.status // "unknown"')
        print_success "Complex task received (ID: $TASK_ID, Status: $status)"
        log "Task received: $TASK_ID - $status"
        
        # Validate task was received
        if [[ -n "$TASK_ID" && "$TASK_ID" != "null" ]]; then
            print_success "✓ Validation: Task ID generated"
            return 0
        else
            print_error "✗ Validation: No task ID returned"
            return 1
        fi
    else
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        print_error "Failed to receive task: $error"
        log "Task reception failed: $error"
        return 1
    fi
}

# Step 3: Decompose Task
decompose_task() {
    print_step "Step 3: Coordinator decomposes task into subtasks..."
    
    local decompose_data='{
        "agent_id": "'$COORDINATOR_AGENT'",
        "task_id": "'$TASK_ID'",
        "demo_id": "'$DEMO_ID'",
        "decomposition_strategy": "capability_based"
    }'
    
    # Make request to CORTEX
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$decompose_data" \
        "$CORTEX_ENDPOINT/api/tasks/decompose" || echo '{"success": false, "error": "Connection failed"}')
    
    # Parse response
    local success=$(echo "$response" | jq -r '.success // false')
    
    if [[ "$success" == "true" ]]; then
        local subtask_count=$(echo "$response" | jq -r '.subtask_count // 0')
        local subtasks=$(echo "$response" | jq -r '.subtasks // []')
        print_success "Task decomposed into $subtask_count subtasks"
        log "Task decomposition: $subtask_count subtasks created"
        
        # Validate subtasks were created
        if [[ "$subtask_count" -ge 2 ]]; then
            print_success "✓ Validation: Multiple subtasks created ($subtask_count)"
            return 0
        else
            print_error "✗ Validation: Expected >= 2 subtasks, got $subtask_count"
            return 1
        fi
    else
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        print_error "Failed to decompose task: $error"
        log "Task decomposition failed: $error"
        return 1
    fi
}

# Step 4: Assign Subtasks
assign_subtasks() {
    print_step "Step 4: Assigning subtasks to specialist agents..."
    
    local assignment_data='{
        "coordinator": "'$COORDINATOR_AGENT'",
        "task_id": "'$TASK_ID'",
        "demo_id": "'$DEMO_ID'",
        "assignments": [
            {
                "agent": "'$SPECIALIST_A'",
                "subtask": "data_analysis",
                "description": "Analyze market data patterns and trends"
            },
            {
                "agent": "'$SPECIALIST_B'",
                "subtask": "optimization",
                "description": "Optimize prediction models and validate results"
            }
        ]
    }'
    
    # Make request to CORTEX
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$assignment_data" \
        "$CORTEX_ENDPOINT/api/tasks/assign" || echo '{"success": false, "error": "Connection failed"}')
    
    # Parse response
    local success=$(echo "$response" | jq -r '.success // false')
    
    if [[ "$success" == "true" ]]; then
        local assignments_confirmed=$(echo "$response" | jq -r '.assignments_confirmed // false')
        local assigned_count=$(echo "$response" | jq -r '.assigned_count // 0')
        print_success "Subtasks assigned to $assigned_count agents"
        log "Task assignment: $assigned_count assignments confirmed"
        
        # Validate assignments were confirmed
        if [[ "$assignments_confirmed" == "true" ]]; then
            print_success "✓ Validation: All assignments confirmed"
            return 0
        else
            print_error "✗ Validation: Assignments not confirmed"
            return 1
        fi
    else
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        print_error "Failed to assign subtasks: $error"
        log "Task assignment failed: $error"
        return 1
    fi
}

# Step 5: Execute Parallel Work
execute_parallel_work() {
    print_step "Step 5: Agents execute their assigned subtasks in parallel..."
    
    # Start parallel execution for both specialists
    local execution_data='{
        "task_id": "'$TASK_ID'",
        "demo_id": "'$DEMO_ID'",
        "execution_mode": "parallel",
        "agents": ["'$SPECIALIST_A'", "'$SPECIALIST_B'"],
        "timeout": "8m"
    }'
    
    # Make request to CORTEX
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$execution_data" \
        "$CORTEX_ENDPOINT/api/tasks/execute_parallel" || echo '{"success": false, "error": "Connection failed"}')
    
    # Parse response
    local success=$(echo "$response" | jq -r '.success // false')
    
    if [[ "$success" == "true" ]]; then
        local execution_id=$(echo "$response" | jq -r '.execution_id')
        print_success "Parallel execution started (ID: $execution_id)"
        log "Parallel execution started: $execution_id"
        
        # Wait for execution to complete
        print_info "Waiting for parallel execution to complete..."
        sleep 10
        
        # Check execution status
        local execution_status=$(check_execution_status "$execution_id")
        if [[ "$execution_status" == "completed" ]]; then
            print_success "✓ Validation: All subtasks completed successfully"
            return 0
        else
            print_error "✗ Validation: Parallel execution failed or incomplete"
            return 1
        fi
    else
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        print_error "Failed to start parallel execution: $error"
        log "Parallel execution failed: $error"
        return 1
    fi
}

# Step 6: Share Knowledge
share_knowledge() {
    print_step "Step 6: Agents share findings via KNIRVGRAPH..."
    
    # Create collaboration session
    local session_data='{
        "demo_id": "'$DEMO_ID'",
        "task_id": "'$TASK_ID'",
        "participants": ["'$COORDINATOR_AGENT'", "'$SPECIALIST_A'", "'$SPECIALIST_B'"],
        "knowledge_type": "collaborative_findings"
    }'
    
    # Make request to KNIRVGRAPH
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$session_data" \
        "$KNIRVGRAPH_ENDPOINT/knowledge/share" || echo '{"success": false, "error": "Connection failed"}')
    
    # Parse response
    local success=$(echo "$response" | jq -r '.success // false')
    
    if [[ "$success" == "true" ]]; then
        SESSION_ID=$(echo "$response" | jq -r '.session_id')
        local shared_nodes=$(echo "$response" | jq -r '.shared_nodes // 0')
        print_success "Knowledge shared (Session: $SESSION_ID, Nodes: $shared_nodes)"
        log "Knowledge sharing: $SESSION_ID - $shared_nodes nodes"
        
        # Validate knowledge was shared
        if [[ -n "$SESSION_ID" && "$SESSION_ID" != "null" ]]; then
            print_success "✓ Validation: Knowledge sharing session created"
            return 0
        else
            print_error "✗ Validation: No session ID returned"
            return 1
        fi
    else
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        print_error "Failed to share knowledge: $error"
        log "Knowledge sharing failed: $error"
        return 1
    fi
}

# Step 7: Integrate Results
integrate_results() {
    print_step "Step 7: Coordinator integrates all results..."
    
    local integration_data='{
        "coordinator": "'$COORDINATOR_AGENT'",
        "task_id": "'$TASK_ID'",
        "session_id": "'$SESSION_ID'",
        "demo_id": "'$DEMO_ID'",
        "integration_strategy": "weighted_consensus"
    }'
    
    # Make request to CORTEX
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$integration_data" \
        "$CORTEX_ENDPOINT/api/tasks/integrate" || echo '{"success": false, "error": "Connection failed"}')
    
    # Parse response
    local success=$(echo "$response" | jq -r '.success // false')
    
    if [[ "$success" == "true" ]]; then
        COLLABORATION_RESULT=$(echo "$response" | jq -r '.result_id')
        local integration_score=$(echo "$response" | jq -r '.integration_score // 0')
        print_success "Results integrated (Result: $COLLABORATION_RESULT, Score: $integration_score)"
        log "Result integration: $COLLABORATION_RESULT - score $integration_score"
        
        # Validate integration was successful
        if [[ -n "$COLLABORATION_RESULT" && "$COLLABORATION_RESULT" != "null" ]]; then
            print_success "✓ Validation: Integration successful"
            return 0
        else
            print_error "✗ Validation: Integration failed"
            return 1
        fi
    else
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        print_error "Failed to integrate results: $error"
        log "Result integration failed: $error"
        return 1
    fi
}

# Step 8: Distribute Collaborative Rewards
distribute_collaborative_rewards() {
    print_step "Step 8: Distributing collaborative rewards..."
    
    local reward_data='{
        "demo_id": "'$DEMO_ID'",
        "task_id": "'$TASK_ID'",
        "session_id": "'$SESSION_ID'",
        "participants": ["'$COORDINATOR_AGENT'", "'$SPECIALIST_A'", "'$SPECIALIST_B'"],
        "total_reward": 300,
        "distribution": "equal",
        "bonus_criteria": {
            "collaboration_quality": 0.95,
            "task_completion": true
        }
    }'
    
    # Make request to KNIRV-ORACLE
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$reward_data" \
        "$KNIRV_ROOT_ENDPOINT/rewards/collaborative" || echo '{"success": false, "error": "Connection failed"}')
    
    # Parse response
    local success=$(echo "$response" | jq -r '.success // false')
    
    if [[ "$success" == "true" ]]; then
        local total_reward=$(echo "$response" | jq -r '.total_reward // 0')
        local distribution=$(echo "$response" | jq -r '.distribution')
        print_success "Collaborative rewards distributed (Total: $total_reward NRN)"
        log "Collaborative rewards: $total_reward NRN distributed"
        
        # Show individual distributions
        local coord_reward=$(echo "$distribution" | jq -r '.["'$COORDINATOR_AGENT'"] // 0')
        local spec_a_reward=$(echo "$distribution" | jq -r '.["'$SPECIALIST_A'"] // 0')
        local spec_b_reward=$(echo "$distribution" | jq -r '.["'$SPECIALIST_B'"] // 0')
        
        print_info "Reward distribution:"
        print_info "  $COORDINATOR_AGENT: $coord_reward NRN"
        print_info "  $SPECIALIST_A: $spec_a_reward NRN"
        print_info "  $SPECIALIST_B: $spec_b_reward NRN"
        
        # Validate rewards were distributed
        if [[ "$total_reward" -gt 0 ]]; then
            print_success "✓ Validation: Rewards distributed successfully"
            return 0
        else
            print_error "✗ Validation: No rewards distributed"
            return 1
        fi
    else
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        print_error "Failed to distribute rewards: $error"
        log "Reward distribution failed: $error"
        return 1
    fi
}

# Helper function to check execution status
check_execution_status() {
    local execution_id="$1"
    local response=$(curl -s -X GET \
        "$CORTEX_ENDPOINT/api/tasks/execution/$execution_id" || echo '{"status": "unknown"}')
    
    echo "$response" | jq -r '.status // "unknown"'
}

# Generate demo report
generate_demo_report() {
    print_step "Generating collaboration demo report..."
    
    local report_file="$SCRIPT_DIR/reports/collaboration-demo-$(date +%Y%m%d_%H%M%S).html"
    mkdir -p "$(dirname "$report_file")"
    
    cat > "$report_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>CORTEX Multi-Agent Collaboration Demo Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .success { color: green; }
        .error { color: red; }
        .step { margin: 15px 0; padding: 10px; border: 1px solid #ddd; border-radius: 5px; }
        .agent { background-color: #f9f9f9; padding: 10px; margin: 5px 0; border-radius: 3px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>CORTEX Multi-Agent Collaboration Demo Report</h1>
        <p>Demo ID: $DEMO_ID</p>
        <p>Generated: $(date)</p>
        <p>Duration: $DEMO_DURATION</p>
    </div>
    
    <div class="step">
        <h2>Participating Agents</h2>
        <div class="agent">
            <strong>Coordinator:</strong> $COORDINATOR_AGENT<br>
            <em>Role:</em> Task coordination and result integration
        </div>
        <div class="agent">
            <strong>Specialist A:</strong> $SPECIALIST_A<br>
            <em>Role:</em> Data analysis and pattern recognition
        </div>
        <div class="agent">
            <strong>Specialist B:</strong> $SPECIALIST_B<br>
            <em>Role:</em> Optimization and quality assurance
        </div>
    </div>
    
    <div class="step">
        <h2>Collaboration Results</h2>
        <p>Task ID: $TASK_ID</p>
        <p>Session ID: $SESSION_ID</p>
        <p>Result ID: $COLLABORATION_RESULT</p>
    </div>
    
    <div class="step">
        <h2>Execution Summary</h2>
        <p>Successfully demonstrated multi-agent collaboration with task decomposition, parallel execution, knowledge sharing, and result integration.</p>
    </div>
</body>
</html>
EOF
    
    print_success "Collaboration demo report generated: $report_file"
    log "Demo report generated: $report_file"
}

# Main execution function
main() {
    local start_time=$(date +%s)
    
    print_header "CORTEX Multi-Agent Collaboration Demo"
    print_info "Demo Duration: $DEMO_DURATION"
    print_info "Participants: $COORDINATOR_AGENT, $SPECIALIST_A, $SPECIALIST_B"
    
    # Initialize demo
    initialize_demo
    
    # Execute demo steps
    local step_success=true
    
    if ! initialize_agents; then
        step_success=false
    fi
    
    if [[ "$step_success" == "true" ]] && ! receive_complex_task; then
        step_success=false
    fi
    
    if [[ "$step_success" == "true" ]] && ! decompose_task; then
        step_success=false
    fi
    
    if [[ "$step_success" == "true" ]] && ! assign_subtasks; then
        step_success=false
    fi
    
    if [[ "$step_success" == "true" ]] && ! execute_parallel_work; then
        step_success=false
    fi
    
    if [[ "$step_success" == "true" ]] && ! share_knowledge; then
        step_success=false
    fi
    
    if [[ "$step_success" == "true" ]] && ! integrate_results; then
        step_success=false
    fi
    
    if [[ "$step_success" == "true" ]] && ! distribute_collaborative_rewards; then
        step_success=false
    fi
    
    # Calculate execution time
    local end_time=$(date +%s)
    local execution_time=$((end_time - start_time))
    
    # Generate report
    generate_demo_report
    
    # Final status
    if [[ "$step_success" == "true" ]]; then
        print_success "CORTEX Multi-Agent Collaboration Demo completed successfully!"
        print_success "Execution time: ${execution_time}s"
        log "Demo completed successfully in ${execution_time}s"
        exit 0
    else
        print_error "CORTEX Multi-Agent Collaboration Demo failed"
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
