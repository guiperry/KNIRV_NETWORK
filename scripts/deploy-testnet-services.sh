
#!/bin/bash

# Enhanced error handler function
handle_uncaught_error() {
    local exit_code=$?
    local last_command="${BASH_COMMAND}"
    
    print_error "Unexpected error occurred (exit code: $exit_code)"
    print_error "Last command: $last_command"
    
    # Log the failure
    log_checkpoint "UNCAUGHT_ERROR: $last_command (exit code: $exit_code)"
    
    # Invoke AI troubleshooter for uncaught errors
    invoke_ai_troubleshooter "UNCAUGHT_ERROR" "Unexpected error: $last_command (exit code: $exit_code)" ""
    
    exit $exit_code
}

# Set up error handling - moved to after function definitions
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Configuration
TESTNET_DIR="$PROJECT_ROOT/packages/KNIRVTESTNET"
ANSIBLE_DIR="$PROJECT_ROOT/deployment/ansible"
TESTNET_IP_FILE="$ANSIBLE_DIR/testnet_ip.txt"
TESTNET_INSTANCE_ID_FILE="$ANSIBLE_DIR/testnet_instance_id.txt"
SSH_KEY="~/.ssh/AEGONG.pem"

# Load environment variables from deployment/ansible/.env if present
# Deployment session log file
DEPLOYMENT_LOG="$ANSIBLE_DIR/deployment_session.log"



# Note: load_dotenv will be called after helper print_* functions are defined

# Will be set based on deployment type
INSTANCE_ID=""
DEPLOYMENT_TYPE=""

# Function to find existing running instances
find_existing_instances() {
    # Don't use print_step here as it will be captured by command substitution
    # Instead, the calling function should handle the step display
    
    # Query AWS for instances with KNIRV tags that are running
    local instances=$(aws ec2 describe-instances \
        --filters "Name=tag:Project,Values=KNIRV" "Name=tag:Environment,Values=testnet" "Name=instance-state-name,Values=running" \
        --query 'Reservations[*].Instances[*].[InstanceId,State.Name,LaunchTime,Tags[?Key==`DeploymentType`].Value | [0]]' \
        --output text \
        --region "$AWS_REGION_VAL" 2>/dev/null | sort -k3 -r || true)
    
    # Return only the raw instance data (without any formatted output)
    echo "$instances"
}

# Function to select the most appropriate instance
select_appropriate_instance() {
    local deployment_type="$1"
    local existing_instances="$2"
    
    # First, try to find an instance with matching deployment type
    local matching_instance=$(echo "$existing_instances" | grep "$deployment_type" | head -1 | awk '{print $1}')
    
    if [ -n "$matching_instance" ] && [ "$matching_instance" != "None" ]; then
        print_success "Found existing $deployment_type instance: $matching_instance"
        INSTANCE_ID="$matching_instance"
        DEPLOYMENT_TYPE="$deployment_type"
        return 0
    fi
    
    # If no matching deployment type, use the most recent running instance
    local most_recent_instance=$(echo "$existing_instances" | head -1 | awk '{print $1}')
    local most_recent_type=$(echo "$existing_instances" | head -1 | awk '{print $4}')
    
    # Handle cases where deployment type might be None or empty
    if [ -z "$most_recent_type" ] || [ "$most_recent_type" = "None" ]; then
        most_recent_type="unknown"
    fi
    
    if [ -n "$most_recent_instance" ] && [ "$most_recent_type" != "None" ]; then
        print_warning "No $deployment_type instance found. Using most recent $most_recent_type instance: $most_recent_instance"
        INSTANCE_ID="$most_recent_instance"
        DEPLOYMENT_TYPE="$most_recent_type"
        return 0
    fi
    
    # No existing instances found
    return 1
}

# Deployment options
INCREMENTAL_DEPLOYMENT=false
FORCE_FULL_DEPLOYMENT=false

# Container runtime selection (will be set by detect_container_runtime)
CONTAINER_RUNTIME=""
COMPOSE_COMMAND=""

# Functions
print_header() {
    echo -e "${BLUE}=================================${NC}"
    echo -e "${BLUE}🧪 KNIRVTESTNET Services Deploy${NC}"
    echo -e "${BLUE}=================================${NC}"
    echo ""
}

print_step() {
    echo -e "${PURPLE}[STEP]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Log deployment checkpoints
log_checkpoint() {
    local message="$1"
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $message" >> "$DEPLOYMENT_LOG"
}

# Read the last checkpoint from the log
read_last_checkpoint() {
    if [[ -f "$DEPLOYMENT_LOG" ]]; then
        tail -n 1 "$DEPLOYMENT_LOG" 2>/dev/null | cut -d'-' -f2- | xargs
    else
        echo "No previous deployment log found"
    fi
}

# Check deployment state
check_deployment_state() {
    if [[ -f "$DEPLOYMENT_LOG" ]]; then
        local last_entry=$(tail -n 1 "$DEPLOYMENT_LOG" 2>/dev/null)
        if [[ "$last_entry" == *"FAILED"* ]] || [[ "$last_entry" == *"EXIT"* ]]; then
            echo "failed"
        elif [[ "$last_entry" == *"COMPLETED"* ]]; then
            echo "completed"
        else
            echo "in-progress"
        fi
    else
        echo "new"
    fi
}

# Get last checkpoint message
get_last_checkpoint() {
    if [[ -f "$DEPLOYMENT_LOG" ]]; then
        tail -n 1 "$DEPLOYMENT_LOG" 2>/dev/null | cut -d'-' -f2- | xargs
    else
        echo "No previous deployment found"
    fi
}

# AI Troubleshooter integration
invoke_ai_troubleshooter() {
    local failure_type="$1"
    local error_message="$2"
    local unhealthy_services="$3"
    
    print_step "🔧 AI Troubleshooter: Analyzing $failure_type..."
    
    # Check if AI troubleshooter script exists
    if [ ! -f "$SCRIPT_DIR/ai_troubleshoot.js" ]; then
        print_warning "AI troubleshooter script not found at $SCRIPT_DIR/ai_troubleshoot.js"
        return 1
    fi
    
    # Check if testnet IP is available
    if [ -z "$TESTNET_IP" ]; then
        print_warning "Testnet IP not available - cannot invoke AI troubleshooter"
        return 1
    fi
    
    print_status "Failure type: $failure_type"
    print_status "Error: $error_message"
    
    if [ -n "$unhealthy_services" ]; then
        print_status "Unhealthy services: $unhealthy_services"
    fi
    
    # Ask user if they want to run AI troubleshooter
    echo ""
    echo -e "${YELLOW}Would you like to run the AI troubleshooter to diagnose the issue?${NC}"
    read -p "Run AI troubleshooter? (y/N): " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_status "AI troubleshooter skipped by user"
        return 0
    fi
    
    print_step "Starting AI troubleshooter analysis..."
    
    # Run the AI troubleshooter
    if node "$SCRIPT_DIR/ai_troubleshoot.js" --ip "$TESTNET_IP" --ssh-key "$SSH_KEY"; then
        print_success "AI troubleshooter completed successfully"
        return 0
    else
        print_error "AI troubleshooter failed or was interrupted"
        return 1
    fi
}

# Enhanced error handler with AI troubleshooter integration
handle_deployment_error() {
    local error_type="$1"
    local error_message="$2"
    local unhealthy_services="$3"
    
    print_error "$error_message"
    
    # Log the failure
    log_checkpoint "FAILED: $error_type - $error_message"
    
    # Invoke AI troubleshooter
    invoke_ai_troubleshooter "$error_type" "$error_message" "$unhealthy_services"
    
    # Exit with error code
    exit 1
}



# Determine AMI to use for the target AWS region. Prefers AMI_ID from .env, otherwise queries AWS for
# the latest Ubuntu 22.04 LTS (Jammy) AMI owned by Canonical (owner 099720109477) or via SSM parameter.
determine_ami() {
  if [ -n "${AMI_ID:-}" ]; then
    print_status "Using AMI_ID from environment: $AMI_ID"
    return 0
  fi

  print_status "Determining latest Ubuntu 22.04 (Jammy) AMI for region $AWS_REGION_VAL"

  # Try AWS Systems Manager parameter which is the most reliable
  AMI_ID=$(aws ssm get-parameter --name /aws/service/canonical/ubuntu/server/jammy/stable/current/amd64/hvm/ebs-gp2/ami-id \
    --query Parameter.Value --output text --region "$AWS_REGION_VAL" 2>/dev/null || true)

  if [ -n "$AMI_ID" ]; then
    print_success "Found AMI via SSM: $AMI_ID"
    return 0
  fi

  # Fallback: query images owned by Canonical and pick the newest matching name pattern
  AMI_ID=$(aws ec2 describe-images \
    --owners 099720109477 \
    --filters 'Name=name,Values=ubuntu-jammy-22.04-amd64-server*' 'Name=root-device-type,Values=ebs' 'Name=virtualization-type,Values=hvm' \
    --query 'Images | sort_by(@, &CreationDate)[-1].ImageId' --output text --region "$AWS_REGION_VAL" 2>/dev/null || true)

  if [ -n "$AMI_ID" ] && [ "$AMI_ID" != "None" ]; then
    print_success "Found AMI via describe-images: $AMI_ID"
    return 0
  fi

  print_warning "Failed to determine latest Ubuntu AMI for region $AWS_REGION_VAL. Falling back to a default ami if set."
  # If still unset, set AMI_ID to empty and let downstream steps fail with an informative message
  AMI_ID=""
}

# New function to detect and select container runtime
detect_container_runtime() {
    print_step "Detecting available container runtimes..."

    local docker_available=false
    local podman_available=false
    local docker_running_services=false
    local podman_running_services=false

    # Check if Docker is available and running
    if command -v docker &> /dev/null; then
        if docker info &> /dev/null; then
            docker_available=true
            print_success "Docker is available and running"

            # Check for running KNIRV services in Docker
            local docker_knirv_containers=$(docker ps --filter "name=knirv" --format "{{.Names}}" 2>/dev/null || true)
            if [ -n "$docker_knirv_containers" ]; then
                docker_running_services=true
                print_warning "Found running KNIRV services in Docker:"
                echo "$docker_knirv_containers" | sed 's/^/  - /'
            fi
        else
            print_warning "Docker is installed but not running"
        fi
    else
        print_warning "Docker is not available"
    fi

    # Check if Podman is available
    if command -v podman &> /dev/null; then
        podman_available=true
        print_success "Podman is available"

        # Check for running KNIRV services in Podman
        local podman_knirv_containers=$(podman ps --filter "name=knirv" --format "{{.Names}}" 2>/dev/null || true)
        if [ -n "$podman_knirv_containers" ]; then
            podman_running_services=true
            print_warning "Found running KNIRV services in Podman:"
            echo "$podman_knirv_containers" | sed 's/^/  - /'
        fi
    else
        print_warning "Podman is not available"
    fi

    # Handle cases where services are already running
    if [ "$docker_running_services" = true ] && [ "$podman_running_services" = true ]; then
        print_error "KNIRV services are running in both Docker and Podman!"
        print_error "Please stop services in one runtime before proceeding:"
        echo "  Stop Docker services: docker stop \$(docker ps --filter \"name=knirv\" -q)"
        echo "  Stop Podman services: podman stop \$(podman ps --filter \"name=knirv\" -q)"
        exit 1
    elif [ "$docker_running_services" = true ]; then
        print_warning "KNIRV services are already running in Docker"
        read -p "Do you want to stop existing Docker services and redeploy? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_status "Stopping existing Docker services..."
            docker stop $(docker ps --filter "name=knirv" -q) 2>/dev/null || true
            docker rm $(docker ps -a --filter "name=knirv" -q) 2>/dev/null || true
        else
            print_error "Deployment cancelled"
            exit 0
        fi
    elif [ "$podman_running_services" = true ]; then
        print_warning "KNIRV services are already running in Podman"
        read -p "Do you want to stop existing Podman services and redeploy? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_status "Stopping existing Podman services..."
            podman stop $(podman ps --filter "name=knirv" -q) 2>/dev/null || true
            podman rm $(podman ps -a --filter "name=knirv" -q) 2>/dev/null || true
        else
            print_error "Deployment cancelled"
            exit 0
        fi
    fi

    # Interactive selection - always show all options including Native
    echo ""
    print_step "Multiple deployment options available. Please choose:"
    echo "  1) Docker (traditional container runtime)"
    echo "  2) Podman (rootless, daemonless container runtime)"
    echo "  3) Native (upload KNIRVTESTNET directory and run npm scripts)"
    echo ""
    while true; do
        read -p "Enter your choice (1, 2, or 3): " choice
        case $choice in
            1)
                if [ "$docker_available" = false ]; then
                    print_error "Docker is not available. Please install Docker or choose another option."
                    continue
                fi
                CONTAINER_RUNTIME="docker"
                print_success "Selected Docker as container runtime"
                break
                ;;
            2)
                if [ "$podman_available" = false ]; then
                    print_error "Podman is not available. Please install Podman or choose another option."
                    continue
                fi
                CONTAINER_RUNTIME="podman"
                COMPOSE_COMMAND="podman-compose"
                print_success "Selected Podman as container runtime"
                break
                ;;
            3)
                CONTAINER_RUNTIME="native"
                print_success "Selected Native deployment with Go orchestrator"
                build_and_upload_orchestrator
                break
                ;;
            *)
                print_error "Invalid choice. Please enter 1, 2, or 3."
                ;;
        esac
    done

    # Set up Docker Compose command (handle both v1 and v2)
    if [ "$CONTAINER_RUNTIME" = "docker" ]; then
        if command -v docker-compose &> /dev/null; then
            COMPOSE_COMMAND="docker-compose"
            print_success "Using Docker Compose v1 (docker-compose)"
        elif docker compose version &> /dev/null; then
            COMPOSE_COMMAND="docker compose"
            print_success "Using Docker Compose v2 (docker compose)"
        else
            print_error "Docker Compose not found. Please install Docker Compose."
            exit 1
        fi
    fi

    # Verify Podman compose command is available
    if [ "$CONTAINER_RUNTIME" = "podman" ] && ! command -v "$COMPOSE_COMMAND" &> /dev/null; then
        print_warning "podman-compose not found. Installing via pip..."
        pip3 install podman-compose || {
            print_error "Failed to install podman-compose. Please install manually:"
            echo "  pip3 install podman-compose"
            exit 1
        }
    fi

    print_success "Container runtime configured: $CONTAINER_RUNTIME with $COMPOSE_COMMAND"
}

run_infrastructure_deployment() {
    print_step "Running infrastructure deployment..."

    # Path to infrastructure script
    local infra_script="$PROJECT_ROOT/deployment/ansible/deploy-testnet-infrastructure.sh"

    if [ ! -f "$infra_script" ]; then
        print_error "Infrastructure script not found at $infra_script"
        exit 1
    fi

    # Run the infrastructure script
    if bash "$infra_script"; then
        print_success "Infrastructure deployment completed"

        # The infrastructure script should have created the instance ID file
        if [ -f "$TESTNET_INSTANCE_ID_FILE" ]; then
            INSTANCE_ID=$(cat "$TESTNET_INSTANCE_ID_FILE")
            print_success "Using instance created by infrastructure: $INSTANCE_ID"
            DEPLOYMENT_TYPE="$CONTAINER_RUNTIME"
            return 0
        fi

        # Fallback: try to find by IP if instance ID file wasn't created
        if [ -f "$TESTNET_IP_FILE" ]; then
            local testnet_ip=$(cat "$TESTNET_IP_FILE")
            print_status "Infrastructure created IP: $testnet_ip"

            # Try to find the instance by public IP
            local instance_from_ip=$(aws ec2 describe-instances \
                --filters "Name=ip-address,Values=$testnet_ip" \
                --query 'Reservations[*].Instances[*].InstanceId' \
                --output text \
                --region "$AWS_REGION_VAL" 2>/dev/null | head -1)

            if [ -n "$instance_from_ip" ] && [ "$instance_from_ip" != "None" ]; then
                print_success "Found newly created instance: $instance_from_ip"
                INSTANCE_ID="$instance_from_ip"
                DEPLOYMENT_TYPE="$CONTAINER_RUNTIME"
                return 0
            fi
        fi

        print_error "Could not find instance ID after infrastructure deployment"
        exit 1
    else
        print_error "Infrastructure deployment failed"
        exit 1
    fi
}

select_deployment_instance() {
    print_step "Selecting deployment EC2 instance..."

    # First, check if infrastructure has been deployed and we have an instance ID file
    if [ -f "$TESTNET_INSTANCE_ID_FILE" ]; then
        local instance_id=$(cat "$TESTNET_INSTANCE_ID_FILE")
        print_status "Found existing testnet instance ID: $instance_id"

        # Verify the instance exists and is running
        local instance_state=$(aws ec2 describe-instances \
            --instance-ids "$instance_id" \
            --query 'Reservations[0].Instances[0].State.Name' \
            --output text \
            --region "$AWS_REGION_VAL" 2>/dev/null || echo "not-found")

        if [ "$instance_state" = "running" ]; then
            print_success "Instance $instance_id is running"
            INSTANCE_ID="$instance_id"
            DEPLOYMENT_TYPE="$CONTAINER_RUNTIME"  # Assume it matches our container runtime

            # Ask user if they want to use this instance or create a new one
            echo ""
            print_warning "Found existing instance from infrastructure: $INSTANCE_ID"
            read -p "Do you want to use this existing instance or create a new one? (use/new): " -n 1 -r
            echo

            if [[ $REPLY =~ ^[Uu]$ ]]; then
                print_status "Using existing instance: $INSTANCE_ID"
                return 0
            else
                print_status "Creating new instance for $CONTAINER_RUNTIME deployment..."
                create_new_instance_with_confirmation
                return 0
            fi
        else
            print_warning "Instance $instance_id is not running (state: $instance_state)"
        fi
    fi

    # Second, check if we have an IP file but no instance ID file
    if [ -f "$TESTNET_IP_FILE" ]; then
        local testnet_ip=$(cat "$TESTNET_IP_FILE")
        print_status "Found existing testnet IP: $testnet_ip"

        # Try to find the instance by public IP
        local instance_from_ip=$(aws ec2 describe-instances \
            --filters "Name=ip-address,Values=$testnet_ip" \
            --query 'Reservations[*].Instances[*].InstanceId' \
            --output text \
            --region "$AWS_REGION_VAL" 2>/dev/null | head -1)

        if [ -n "$instance_from_ip" ] && [ "$instance_from_ip" != "None" ]; then
            print_success "Found instance from IP: $instance_from_ip"
            INSTANCE_ID="$instance_from_ip"
            DEPLOYMENT_TYPE="$CONTAINER_RUNTIME"  # Assume it matches our container runtime

            # Ask user if they want to use this instance or create a new one
            echo ""
            print_warning "Found existing instance from infrastructure: $INSTANCE_ID"
            read -p "Do you want to use this existing instance or create a new one? (use/new): " -n 1 -r
            echo

            if [[ $REPLY =~ ^[Uu]$ ]]; then
                print_status "Using existing instance: $INSTANCE_ID"
                return 0
            else
                print_status "Creating new instance for $CONTAINER_RUNTIME deployment..."
                create_new_instance_with_confirmation
                return 0
            fi
        else
            print_warning "Could not find instance for IP $testnet_ip, instance may not be running yet"
        fi
    fi

    # Fallback: Find existing running instances by tags
    print_step "Searching for existing KNIRV testnet instances..."
    local existing_instances=$(find_existing_instances)
    
    # Display the results to the user
    if [ -n "$existing_instances" ]; then
        print_success "Found existing KNIRV testnet instances:"
        echo "$existing_instances" | while read -r instance_id state launch_time deployment_type; do
            echo "  - $instance_id ($deployment_type) - Launched: $launch_time"
        done
    else
        print_warning "No existing running KNIRV testnet instances found"
    fi

    # Try to select an appropriate existing instance
    if select_appropriate_instance "$CONTAINER_RUNTIME" "$existing_instances"; then
        # Verify that INSTANCE_ID was actually set
        if [ -z "$INSTANCE_ID" ] || [ "$INSTANCE_ID" = "None" ]; then
            print_error "Instance selection failed - no valid instance ID found"
            print_status "Running infrastructure deployment to create a new instance..."
            run_infrastructure_deployment
        else
            print_success "Found existing instance: $INSTANCE_ID ($DEPLOYMENT_TYPE)"

            # Ask user if they want to use this instance or create a new one
            echo ""
            print_warning "Found existing instance: $INSTANCE_ID ($DEPLOYMENT_TYPE)"
            read -p "Do you want to use this existing instance or create a new one? (use/new): " -n 1 -r
            echo

            if [[ $REPLY =~ ^[Uu]$ ]]; then
                print_status "Using existing instance: $INSTANCE_ID"
            else
                print_status "Creating new instance for $CONTAINER_RUNTIME deployment..."
                create_new_instance_with_confirmation
            fi
        fi
    else
        # No existing instances found - call infrastructure script to create one
        print_warning "No existing KNIRV testnet instances found."
        print_status "Running infrastructure deployment to create a new instance..."
        run_infrastructure_deployment
    fi

    # Check deployment state and handle resume/restart
    local deployment_state=$(check_deployment_state)
    if [[ "$deployment_state" == "failed" || "$deployment_state" == "in-progress" ]]; then
        print_warning "Previous deployment detected: $(get_last_checkpoint)"
        
        if [[ "$RESUME_DEPLOYMENT" == "true" ]]; then
            print_status "Resuming deployment from last checkpoint..."
            return 0
        elif [[ "$CLEAN_DEPLOYMENT" == "true" ]]; then
            print_status "Starting fresh deployment..."
            RESUME_DEPLOYMENT=false
            echo "" > "$DEPLOYMENT_LOG"
        else
            while true; do
                read -p "Do you want to (R)esume from last checkpoint or (S)tart fresh? [R/s]: " choice
                case $choice in
                    [Ss]* )
                        print_status "Starting fresh deployment..."
                        RESUME_DEPLOYMENT=false
                        echo "" > "$DEPLOYMENT_LOG"
                        break
                        ;;
                    * )
                        print_status "Resuming deployment from last checkpoint..."
                        RESUME_DEPLOYMENT=true
                        break
                        ;;
                esac
            done
        fi
        
        while true; do
            read -p "Do you want to resume from last checkpoint or start fresh? (resume/fresh): " choice
            case $choice in
                resume|r)
                    print_status "Resuming $DEPLOYMENT_TYPE deployment from last checkpoint..."
                    # We'll implement resume logic later in the main function
                    RESUME_DEPLOYMENT=true
                    break
                    ;;
                fresh|f)
                    print_status "Starting fresh $DEPLOYMENT_TYPE deployment..."
                    RESUME_DEPLOYMENT=false
                    # Clean the deployment log for fresh start
                    echo "" > "$DEPLOYMENT_LOG"
                    break
                    ;;
                *)
                    print_error "Invalid choice. Please enter 'resume' or 'fresh'."
                    ;;
            esac
        done
    else
        print_status "No previous deployment failures detected. Starting fresh deployment."
        RESUME_DEPLOYMENT=false
    fi

    # Update the IP file path to be deployment-specific
    TESTNET_IP_FILE="$ANSIBLE_DIR/testnet_ip_${CONTAINER_RUNTIME}.txt"

    print_success "Instance selected: $INSTANCE_ID for $DEPLOYMENT_TYPE deployment"
}

create_new_instance_with_confirmation() {
    print_step "Creating new EC2 instance for $CONTAINER_RUNTIME deployment..."
    
    # Show instance creation details
    print_status "Instance type: t3.medium"
    print_status "AMI: Ubuntu 22.04 LTS"
    print_status "Region: $AWS_REGION_VAL"
    print_status "Key pair: AEGONG"
    
    # Final confirmation
    read -p "Are you sure you want to create a new instance? (y/N): " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_error "Instance creation cancelled"
        exit 0
    fi
    
    # Create the new instance
    create_deployment_instance
    DEPLOYMENT_TYPE="$CONTAINER_RUNTIME"
}

validate_instance_id() {
    # Validate that INSTANCE_ID is properly set and not empty
    if [ -z "$INSTANCE_ID" ]; then
        print_error "Instance ID is empty. Instance selection failed."
        return 1
    fi
    
    if [ "$INSTANCE_ID" = "None" ]; then
        print_error "Instance ID is 'None'. Instance selection failed."
        return 1
    fi
    
    # Check if it looks like a valid AWS instance ID (starts with i-)
    if [[ ! "$INSTANCE_ID" =~ ^i-[a-z0-9]+$ ]]; then
        print_error "Instance ID '$INSTANCE_ID' does not look like a valid AWS instance ID."
        return 1
    fi
    
    return 0
}

ensure_instance_exists() {
    print_step "Ensuring deployment instance exists and is properly tagged..."

    # First validate the instance ID format
    if ! validate_instance_id; then
        print_error "Invalid instance ID detected: $INSTANCE_ID"
        print_error "Please run the deployment script again to select a valid instance."
        exit 1
    fi

    # Check if instance exists with more flexible approach
    print_status "Checking instance $INSTANCE_ID with flexible filters..."

    # Try multiple approaches to find the instance
    INSTANCE_STATE=$(aws ec2 describe-instances \
        --instance-ids "$INSTANCE_ID" \
        --query 'Reservations[0].Instances[0].State.Name' \
        --output text \
        --region "$AWS_REGION_VAL" 2>/dev/null || echo "not-found")

    # If not found by exact instance ID, try to find by IP or tags
    if [ "$INSTANCE_STATE" = "not-found" ]; then
        print_warning "Instance $INSTANCE_ID not found by exact ID, trying alternative methods..."

        # Get the testnet IP if available
        if [ -f "$TESTNET_IP_FILE" ]; then
            local testnet_ip=$(cat "$TESTNET_IP_FILE")
            print_status "Searching for instance by IP: $testnet_ip"

            # Try to find instance by public IP
            local instance_from_ip=$(aws ec2 describe-instances \
                --filters "Name=ip-address,Values=$testnet_ip" \
                --query 'Reservations[*].Instances[*].InstanceId' \
                --output text \
                --region "$AWS_REGION_VAL" 2>/dev/null | head -1)

            if [ -n "$instance_from_ip" ] && [ "$instance_from_ip" != "None" ]; then
                print_success "Found instance by IP: $instance_from_ip"
                INSTANCE_ID="$instance_from_ip"
                INSTANCE_STATE="running"
            fi
        fi

        # If still not found, try broader tag search
        if [ "$INSTANCE_STATE" = "not-found" ]; then
            print_status "Searching for instance with broader tag filters..."

            # Try to find any running KNIRV instances (less restrictive)
            local instances=$(aws ec2 describe-instances \
                --filters "Name=tag:Project,Values=KNIRV" "Name=instance-state-name,Values=running" \
                --query 'Reservations[*].Instances[*].[InstanceId,State.Name,LaunchTime]' \
                --output text \
                --region "$AWS_REGION_VAL" 2>/dev/null | sort -k3 -r || true)

            if [ -n "$instances" ]; then
                local most_recent_instance=$(echo "$instances" | head -1 | awk '{print $1}')
                print_success "Found most recent KNIRV instance: $most_recent_instance"
                INSTANCE_ID="$most_recent_instance"
                INSTANCE_STATE="running"
            fi
        fi
    fi

    if [ "$INSTANCE_STATE" = "not-found" ]; then
        print_error "Instance $INSTANCE_ID not found with any method."
        print_error "This suggests the instance may have been terminated or there's a region mismatch."
        print_error "Please run the deployment script again to select a valid instance."
        exit 1
    else
        print_success "Instance $INSTANCE_ID exists (state: $INSTANCE_STATE)"
        update_instance_tags
    fi
}

create_deployment_instance() {
    print_step "Creating new EC2 instance for $DEPLOYMENT_TYPE deployment..."

  # Ensure we have a valid AMI for the target region
  determine_ami

  # Build run-instances arguments conditionally based on provided env vars
  run_args=(--image-id "$AMI_ID" --instance-type t3.medium --key-name AEGONG)

  # Allow override from .env: SECURITY_GROUP_ID (single) or SECURITY_GROUP_IDS (comma-separated)
  if [ -n "${SECURITY_GROUP_IDS:-}" ]; then
    # split comma-separated into multiple --security-group-ids entries
    IFS=',' read -ra SG_ARR <<< "$SECURITY_GROUP_IDS"
    for sg in "${SG_ARR[@]}"; do
      run_args+=(--security-group-ids "$sg")
    done
  elif [ -n "${SECURITY_GROUP_ID:-}" ]; then
    run_args+=(--security-group-ids "$SECURITY_GROUP_ID")
  fi

  # Subnet: if provided, include --subnet-id; otherwise let AWS pick (may use default subnet)
  if [ -n "${SUBNET_ID:-}" ]; then
    run_args+=(--subnet-id "$SUBNET_ID")
  fi

  # Tag specifications
  run_args+=(--tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=KNIRV-Testnet-${DEPLOYMENT_TYPE}},{Key=DeploymentType,Value=${CONTAINER_RUNTIME}},{Key=Project,Value=KNIRV},{Key=Environment,Value=testnet}]" --query 'Instances[0].InstanceId' --output text)

  # Run instance
  NEW_INSTANCE=$(aws ec2 run-instances "${run_args[@]}" --region "$AWS_REGION_VAL" 2>/dev/null || true)

  # If aws returned JSON, extract InstanceId; if empty, keep NEW_INSTANCE as-is
  if [ -n "$NEW_INSTANCE" ]; then
    # If response was InstanceId directly, keep it. If full JSON, try to parse
    if echo "$NEW_INSTANCE" | jq -e . >/dev/null 2>&1; then
      NEW_INSTANCE=$(echo "$NEW_INSTANCE" | jq -r '.Instances[0].InstanceId' 2>/dev/null || true)
    fi
  fi

    if [ $? -eq 0 ] && [ -n "$NEW_INSTANCE" ]; then
        print_success "Created new instance: $NEW_INSTANCE"

        # Update the instance ID variable based on deployment type
        case "$CONTAINER_RUNTIME" in
            "native") NATIVE_INSTANCE_ID="$NEW_INSTANCE" ;;
            "docker") DOCKER_INSTANCE_ID="$NEW_INSTANCE" ;;
            "podman") PODMAN_INSTANCE_ID="$NEW_INSTANCE" ;;
        esac

        INSTANCE_ID="$NEW_INSTANCE"

        # Save the instance ID to file
        echo "$NEW_INSTANCE" > "$TESTNET_INSTANCE_ID_FILE"
        print_status "Instance ID saved to $TESTNET_INSTANCE_ID_FILE"

        # Wait for instance to be running
        print_status "Waiting for instance to be running..."
  aws ec2 wait instance-running --instance-ids "$INSTANCE_ID" --region "$AWS_REGION_VAL"

        print_success "Instance $INSTANCE_ID is now running"
    else
        print_error "Failed to create new instance"
        exit 1
    fi
}

update_instance_tags() {
    print_step "Updating instance tags for $DEPLOYMENT_TYPE deployment..."

  aws ec2 create-tags \
    --resources "$INSTANCE_ID" \
    --tags "Key=Name,Value=KNIRV-Testnet-${DEPLOYMENT_TYPE}" \
         "Key=DeploymentType,Value=${CONTAINER_RUNTIME}" \
         "Key=Project,Value=KNIRV" \
         "Key=Environment,Value=testnet" \
         "Key=LastDeployment,Value=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --region "$AWS_REGION_VAL" > /dev/null 2>&1

    print_success "Instance tags updated for $DEPLOYMENT_TYPE deployment"
}

get_instance_ip() {
    # Use the utility script to get current IP for the selected instance
    if [ -n "$INSTANCE_ID" ]; then
        TESTNET_IP=$("$PROJECT_ROOT/scripts/get-testnet-ip.sh" get-ip "$INSTANCE_ID" 2>/dev/null | tail -n1)
    else
        TESTNET_IP=$("$PROJECT_ROOT/scripts/get-testnet-ip.sh" get-ip 2>/dev/null | tail -n1)
    fi
    
    if [ $? -ne 0 ] || [ -z "$TESTNET_IP" ]; then
        print_error "Failed to get testnet IP address for instance $INSTANCE_ID"
        # Try the verbose version to see what's wrong
        if [ -n "$INSTANCE_ID" ]; then
            "$PROJECT_ROOT/scripts/get-testnet-ip.sh" get-ip "$INSTANCE_ID"
        else
            "$PROJECT_ROOT/scripts/get-testnet-ip.sh" get-ip
        fi
        exit 1
    fi
    
    # Update the deployment-specific IP file
    mkdir -p "$(dirname "$TESTNET_IP_FILE")"
    echo "$TESTNET_IP" > "$TESTNET_IP_FILE"
    
    print_success "Testnet IP: $TESTNET_IP"
}

detect_subnet_environment() {
    print_step "Detecting network environment..."

    # Check if the server is behind a subnet/NAT
    SUBNET_DETECTED=$(ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
# Check if we're behind NAT by comparing internal and external IPs
INTERNAL_IP=$(hostname -I | awk '{print $1}')
EXTERNAL_IP=$(curl -s --max-time 10 ifconfig.me 2>/dev/null || curl -s --max-time 10 ipinfo.io/ip 2>/dev/null || echo "unknown")

echo "Internal IP: $INTERNAL_IP"
echo "External IP: $EXTERNAL_IP"

# Check if internal IP is in private ranges
if [[ "$INTERNAL_IP" =~ ^10\. ]] || [[ "$INTERNAL_IP" =~ ^172\.(1[6-9]|2[0-9]|3[0-1])\. ]] || [[ "$INTERNAL_IP" =~ ^192\.168\. ]]; then
    echo "SUBNET_DETECTED=true"
else
    echo "SUBNET_DETECTED=false"
fi
EOF
)

    if echo "$SUBNET_DETECTED" | grep -q "SUBNET_DETECTED=true"; then
        BEHIND_SUBNET=true
        print_warning "Server is behind a subnet/NAT - NGINX network manager will be activated"
    else
        BEHIND_SUBNET=false
        print_success "Server has direct public IP - NGINX network manager will remain inactive"
    fi

    echo "$SUBNET_DETECTED"
}

configure_ufw_firewall() {
    # Check if we should skip this step when resuming
    if [ "$RESUME_DEPLOYMENT" = "true" ] && [ "$(read_last_checkpoint)" = "Configuring UFW firewall..." ]; then
        print_status "Skipping configure_ufw_firewall (already completed)"
        return 0
    fi
    
    log_checkpoint "Configuring UFW firewall..."
    print_step "Configuring UFW firewall to match AWS EC2 security group..."

    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
# Install UFW if not present
if ! command -v ufw >/dev/null 2>&1; then
    echo "Installing UFW..."
    sudo apt-get update && sudo apt-get install -y ufw
fi

# Reset UFW to defaults
sudo ufw --force reset

# Set default policies
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH (port 22)
sudo ufw allow 22/tcp

# Allow HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Allow KNIRV service ports (matching AWS EC2 security group)
sudo ufw allow 1317/tcp   # KNIRVORACLE
sudo ufw allow 8090/tcp   # KNIRVCHAIN
sudo ufw allow 8082/tcp   # KNIRVGRAPH
    sudo ufw allow 8084/tcp   # KNIRVSERVER DVE
    sudo ufw allow 8085/tcp   # KNIRVSERVER Validation
sudo ufw allow 8086/tcp   # KNIRVROUTER
sudo ufw allow 10000/tcp  # TESTNET GATEWAY
sudo ufw allow 3000/tcp   # KNIRVANA
sudo ufw allow 8088/tcp   # XION Bridge
sudo ufw allow 5001/tcp   # IPFS API
sudo ufw allow 8081/tcp   # IPFS Gateway
sudo ufw allow 3478/tcp   # STUN/TURN
sudo ufw allow 5349/tcp   # TURN TLS

# Allow internal Docker network communication
sudo ufw allow from 172.16.0.0/12
sudo ufw allow from 10.0.0.0/8

# Enable UFW
sudo ufw --force enable

# Show status
sudo ufw status numbered

echo "UFW firewall configured successfully"
EOF

    print_success "UFW firewall configured to match AWS EC2 ports"
}

generate_docker_compose_file() {
    local temp_dir="$1"
    local include_nginx="$2"

    print_step "Generating Docker Compose file..."

    if [ "$include_nginx" = "true" ]; then
        print_status "Including NGINX Network Manager (subnet detected)"
        cat > "$temp_dir/docker-compose-prod.yml" << 'EOF'
version: '3.8'

services:
  # NGINX Network Manager - Routes traffic to containers (ACTIVE)
  nginx:
    image: nginx:alpine
    container_name: knirv-testnet-nginx
    ports:
      - "80:80"       # HTTP
      - "443:443"     # HTTPS
      - "1317:1317"   # KNIRVORACLE
      - "8090:8090"   # KNIRVCHAIN
      - "8082:8082"   # KNIRVGRAPH
      - "8084:8084"   # KNIRVSERVER-1
      - "8085:8085"   # KNIRVSERVER-2
      - "8086:8086"   # KNIRVROUTER
      - "10000:10000" # TESTNET-GATEWAY
      - "3000:3000"   # KNIRVANA
      - "8088:8088"   # XION-BRIDGE
      - "5001:5001"   # IPFS-API
      - "8081:8081"   # IPFS-GATEWAY
    volumes:
      - ./config/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./logs/nginx:/var/log/nginx
    depends_on:
      - ipfs
      - knirv-oracle
      - knirv-chain
      - knirv-graph
      - knirv-nexus
      - knirv-router
      - testnet-gateway
      - knirvana
      - xion-bridge
    networks:
      - knirv-testnet
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/nginx-health"]
      interval: 30s
      timeout: 10s
      retries: 3
EOF
    else
        print_status "NGINX Network Manager disabled (direct public IP detected)"
        cat > "$temp_dir/docker-compose-prod.yml" << 'EOF'
version: '3.8'

services:
  # NGINX Network Manager - DISABLED (direct public IP)
  # nginx:
  #   image: nginx:alpine
  #   container_name: knirv-testnet-nginx-disabled
  #   restart: "no"
  #   command: ["echo", "NGINX disabled - direct public IP detected"]
EOF
    fi

    # Continue with the rest of the services (common to both configurations)
    cat >> "$temp_dir/docker-compose-prod.yml" << 'EOF'

  # IPFS for distributed storage
  ipfs:
    image: ipfs/kubo:latest
    container_name: knirv-testnet-ipfs
    ports:
      - "5001:5001"   # API port
      - "8081:8080"   # Gateway port (changed from 8080 to 8081)
EOF

    if [ "$include_nginx" = "true" ]; then
        cat >> "$temp_dir/docker-compose-prod.yml" << 'EOF'
    expose:
      - "5001"        # API port (internal only)
      - "8080"        # Gateway port (internal only)
EOF
    else
        cat >> "$temp_dir/docker-compose-prod.yml" << 'EOF'
    ports:
      - "5001:5001"   # API port (direct)
      - "8081:8080"   # Gateway port (direct)
EOF
    fi

    cat >> "$temp_dir/docker-compose-prod.yml" << 'EOF'
    volumes:
      - ./data/ipfs:/data/ipfs
    environment:
      - IPFS_PROFILE=server
    networks:
      - knirv-testnet
    restart: unless-stopped

  # Testnet Gateway - Node.js application
  testnet-gateway:
    image: node:20-alpine
    container_name: knirv-testnet-gateway
    working_dir: /app
EOF

    if [ "$include_nginx" = "true" ]; then
        cat >> "$temp_dir/docker-compose-prod.yml" << 'EOF'
    expose:
      - "10000"       # Internal port only
EOF
    else
        cat >> "$temp_dir/docker-compose-prod.yml" << 'EOF'
    ports:
      - "10000:10000" # Direct port
EOF
    fi

    print_success "Docker Compose file generated with NGINX $([ "$include_nginx" = "true" ] && echo "enabled" || echo "disabled")"
}


check_prerequisites() {
    # Check if we should skip this step when resuming
    if [ "$RESUME_DEPLOYMENT" = "true" ] && [ "$(read_last_checkpoint)" = "Checking prerequisites..." ]; then
        print_status "Skipping check_prerequisites (already completed)"
        return 0
    fi
    
    log_checkpoint "Checking prerequisites..."
    print_step "Checking prerequisites..."
    # Detect and select container runtime first
    detect_container_runtime

    # Select deployment-specific instance
    select_deployment_instance

    # Ensure instance exists and is properly tagged
    ensure_instance_exists

    # Get current IP address dynamically from AWS
    get_instance_ip

    # Check if SSH key exists
    if [ ! -f "${SSH_KEY/#\~/$HOME}" ]; then
        handle_deployment_error "SSH_KEY_MISSING" "SSH key not found at $SSH_KEY. Please ensure your AWS key pair is available."
    fi

    # Check if KNIRVTESTNET directory exists
    if [ ! -d "$TESTNET_DIR" ]; then
        handle_deployment_error "TESTNET_DIR_MISSING" "KNIRVTESTNET directory not found at $TESTNET_DIR"
    fi

    # Check if docker-compose.yml exists
    if [ ! -f "$TESTNET_DIR/docker-compose.yml" ]; then
        handle_deployment_error "DOCKER_COMPOSE_MISSING" "docker-compose.yml not found in KNIRVTESTNET directory"
    fi

    print_success "Prerequisites check passed"
    print_success "Testnet IP: $TESTNET_IP"
    print_success "Container Runtime: $CONTAINER_RUNTIME"
}

test_ssh_connection() {
    # Check if we should skip this step when resuming
    if [ "$RESUME_DEPLOYMENT" = "true" ] && [ "$(read_last_checkpoint)" = "Testing SSH connection..." ]; then
        print_status "Skipping test_ssh_connection (already completed)"
        return 0
    fi
    
    log_checkpoint "Testing SSH connection..."
    print_step "Testing SSH connection to testnet..."

    if ssh -i "${SSH_KEY/#\~/$HOME}" -o ConnectTimeout=10 -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "echo 'SSH connection successful'" > /dev/null 2>&1; then
        print_success "SSH connection established"
    else
        handle_deployment_error "SSH_CONNECTION_FAILED" "Cannot connect to testnet via SSH. Please check your SSH key and security group settings."
    fi
}

check_disk_space() {
    log_checkpoint "Checking disk space..."
    print_step "Checking available disk space on testnet server..."

    # Get disk usage information
    DISK_INFO=$(ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "df -h / | tail -n1")
    DISK_USAGE=$(echo "$DISK_INFO" | awk '{print $5}' | sed 's/%//')
    AVAILABLE_SPACE=$(echo "$DISK_INFO" | awk '{print $4}')

    print_status "Current disk usage: $DISK_USAGE% (Available: $AVAILABLE_SPACE)"

    # Check if disk usage is too high (>85%)
    if [ "$DISK_USAGE" -gt 85 ]; then
        print_error "Disk usage is too high ($DISK_USAGE%). Available space: $AVAILABLE_SPACE"
        print_warning "Attempting to clean server automatically..."
        print_warning "To clean up manually, you can use: ssh -i ${SSH_KEY/#\~/$HOME} -o StrictHostKeyChecking=no ubuntu@$TESTNET_IP 'sudo rm -rf /opt/knirv-testnet/*'"

        # Clean server deployment
        clean_server_deployment

        # Recheck disk space after cleaning
        DISK_INFO=$(ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "df -h / | tail -n1")
        DISK_USAGE=$(echo "$DISK_INFO" | awk '{print $5}' | sed 's/%//')
        AVAILABLE_SPACE=$(echo "$DISK_INFO" | awk '{print $4}')

        print_status "Disk usage after cleaning: $DISK_USAGE% (Available: $AVAILABLE_SPACE)"

        # If still high, optionally attempt to expand root EBS volume
        if [ "$DISK_USAGE" -gt 85 ]; then
            if [ "${AUTO_INCREASE_ROOT_VOLUME:-false}" = "true" ]; then
                print_status "AUTO_INCREASE_ROOT_VOLUME=true - attempting to expand root EBS volume"

        # Determine root device name and volume id
        ROOT_DEV=$(aws ec2 describe-instances --instance-ids "$INSTANCE_ID" --query 'Reservations[0].Instances[0].RootDeviceName' --output text --region "$AWS_REGION_VAL" 2>/dev/null || true)
        if [ -z "$ROOT_DEV" ] || [ "$ROOT_DEV" = "None" ]; then
          handle_deployment_error "ROOT_DEVICE_UNKNOWN" "Unable to determine root device for instance $INSTANCE_ID"
        fi

        VOLUME_ID=$(aws ec2 describe-instances --instance-ids "$INSTANCE_ID" --query "Reservations[0].Instances[0].BlockDeviceMappings[?DeviceName=='$ROOT_DEV'].Ebs.VolumeId | [0]" --output text --region "$AWS_REGION_VAL" 2>/dev/null || true)
        if [ -z "$VOLUME_ID" ] || [ "$VOLUME_ID" = "None" ]; then
          handle_deployment_error "VOLUME_ID_UNKNOWN" "Unable to determine volume ID for root device $ROOT_DEV on instance $INSTANCE_ID"
        fi

        print_status "Root device: $ROOT_DEV (volume: $VOLUME_ID)"

        # Get current size (GiB) and choose new size (+20 GiB)
        CUR_SIZE=$(aws ec2 describe-volumes --volume-ids "$VOLUME_ID" --query 'Volumes[0].Size' --output text --region "$AWS_REGION_VAL" 2>/dev/null || true)
        if [ -z "$CUR_SIZE" ] || [ "$CUR_SIZE" = "None" ]; then
          handle_deployment_error "VOLUME_SIZE_UNKNOWN" "Unable to determine current volume size for $VOLUME_ID"
        fi

        # Allow override via env var ROOT_VOLUME_GROW_GB, default to +20 GiB
        GROW_GB=${ROOT_VOLUME_GROW_GB:-20}
        NEW_SIZE=$((CUR_SIZE + GROW_GB))
        print_status "Modifying volume $VOLUME_ID: $CUR_SIZE -> ${NEW_SIZE} GiB"

        aws ec2 modify-volume --volume-id "$VOLUME_ID" --size "$NEW_SIZE" --region "$AWS_REGION_VAL" >/dev/null 2>&1 || {
          handle_deployment_error "VOLUME_MODIFY_FAILED" "Failed to submit modify-volume request for $VOLUME_ID"
        }

        # Wait for modification state
        print_status "Waiting for volume modification to reach 'completed' or 'optimizing' state (this may take a minute)"
        for i in {1..60}; do
          MOD_STATE=$(aws ec2 describe-volumes-modifications --volume-id "$VOLUME_ID" --query 'VolumesModifications[0].ModificationState' --output text --region "$AWS_REGION_VAL" 2>/dev/null || true)
          if [ "$MOD_STATE" = "completed" ] || [ "$MOD_STATE" = "optimizing" ]; then
            print_status "Volume modification state: $MOD_STATE"
            break
          fi
          sleep 5
        done

        # Resize partition & filesystem on the instance
        print_status "Expanding partition and filesystem on instance $INSTANCE_ID"
        ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" bash -s <<'EOF'
set -e
ROOT_PART=$(df -P / | tail -1 | awk '{print $1}')
ROOT_DISK=$(echo "$ROOT_PART" | sed -E 's/([0-9]+)$//')
PART_NUM=$(echo "$ROOT_PART" | sed -E 's/^.*[^0-9]([0-9]+)$/\1/')

# Install growpart if needed
if ! command -v growpart >/dev/null 2>&1; then
  sudo apt-get update -y && sudo apt-get install -y cloud-guest-utils
fi

sudo growpart "$ROOT_DISK" "$PART_NUM" || true

FSTYPE=$(df -T / | tail -1 | awk '{print $2}')
if [ "$FSTYPE" = "xfs" ]; then
  sudo xfs_growfs /
else
  sudo resize2fs "$ROOT_PART"
fi
EOF

        # Re-check disk usage
        DISK_INFO=$(ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "df -h / | tail -n1")
        DISK_USAGE=$(echo "$DISK_INFO" | awk '{print $5}' | sed 's/%//')
        AVAILABLE_SPACE=$(echo "$DISK_INFO" | awk '{print $4}')
        print_status "Post-resize disk usage: $DISK_USAGE% (Available: $AVAILABLE_SPACE)"

        if [ "$DISK_USAGE" -gt 85 ]; then
          handle_deployment_error "DISK_SPACE_CRITICAL" "Disk usage still high after resize: $DISK_USAGE%. To clean up manually, use: ssh -i ${SSH_KEY/#\~/$HOME} -o StrictHostKeyChecking=no ubuntu@$TESTNET_IP 'sudo rm -rf /opt/knirv-testnet/*'"
        fi

        print_success "Disk resized and filesystem expanded successfully"
      else
        handle_deployment_error "DISK_SPACE_INSUFFICIENT" "Disk usage is still too high ($DISK_USAGE%) after cleaning. Available space: $AVAILABLE_SPACE. Please clean up the server or increase disk size before deployment. To auto-expand root EBS volume, set AUTO_INCREASE_ROOT_VOLUME=true in $ANSIBLE_DIR/.env and re-run"
      fi
    fi
  fi

    print_success "Sufficient disk space available ($AVAILABLE_SPACE free)"
}

build_and_upload_orchestrator() {
    log_checkpoint "Building and uploading Go orchestrator..."
    print_step "Building and uploading Go orchestrator..."

    local testnet_binary="$TESTNET_DIR/knirvtestnet"

    # Build the orchestrator if it doesn't exist
    if [ ! -f "$testnet_binary" ]; then
        print_status "Building the Go orchestrator..."
        cd "$TESTNET_DIR"
        if [ ! -f "build-orchestrator.sh" ]; then
            print_error "build-orchestrator.sh not found!"
            exit 1
        fi
        ./build-orchestrator.sh
        cd -
    else
        print_status "Using existing orchestrator binary: $testnet_binary"
        local bin_size=$(du -h "$testnet_binary" | cut -f1)
        print_status "Binary size: $bin_size"
    fi

    # Verify binary exists after build
    if [ ! -f "$testnet_binary" ]; then
        print_error "Orchestrator binary not found after build: $testnet_binary"
        exit 1
    fi

    # Upload the orchestrator
    print_status "Uploading orchestrator to $TESTNET_IP..."
    scp -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no "$testnet_binary" ubuntu@"$TESTNET_IP":/tmp/knirvtestnet

    print_success "Go orchestrator built and uploaded successfully"
}

# Function to clean server deployment
clean_server_deployment() {
    print_step "Cleaning server deployment..."
    
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
# Stop and remove containers
if command -v docker &> /dev/null; then
    docker stop $(docker ps -q) 2>/dev/null || true
    docker rm $(docker ps -aq) 2>/dev/null || true
    docker system prune -f 2>/dev/null || true
fi

if command -v podman &> /dev/null; then
    podman stop --all 2>/dev/null || true
    podman rm --all 2>/dev/null || true
    podman system prune -f 2>/dev/null || true
fi

# Clean deployment directory
sudo rm -rf /opt/knirv-testnet/* 2>/dev/null || true

# Clean temporary files
rm -rf /tmp/knirv* 2>/dev/null || true

# Clean logs
sudo rm -rf /var/log/knirv* 2>/dev/null || true

echo "Server cleaned successfully"
EOF

    print_success "Server deployment cleaned"
}

# Function to load environment variables
load_dotenv() {
    local env_file="$ANSIBLE_DIR/.env"
    if [ -f "$env_file" ]; then
        print_status "Loading environment variables from $env_file"
        set -a  # automatically export all variables
        source "$env_file"
        set +a  # stop automatically exporting
        
        # Set AWS_REGION_VAL from AWS_DEFAULT_REGION or AWS_REGION, fallback to us-east-1
        export AWS_REGION_VAL="${AWS_DEFAULT_REGION:-${AWS_REGION:-us-east-1}}"
        print_success "Environment variables loaded (AWS Region: $AWS_REGION_VAL)"
    else
        print_warning "No .env file found at $env_file"
        print_warning "Using default AWS region: us-east-1"
        export AWS_REGION_VAL="us-east-1"
    fi
}



prepare_testnet_files() {
    # Check if we should skip this step when resuming
    if [ "$RESUME_DEPLOYMENT" = "true" ] && [ "$(read_last_checkpoint)" = "Preparing testnet files..." ]; then
        print_status "Skipping prepare_testnet_files (already completed)"
        return 0
    fi
    
    log_checkpoint "Preparing testnet files..."
    print_step "Preparing testnet files for deployment..."

    # Create temporary directory for deployment files
    TEMP_DIR=$(mktemp -d)

    # Only copy essential files for production deployment (not the entire KNIRVTESTNET directory)
    print_status "Copying essential files only to avoid disk space issues..."

    # Create directory structure
    mkdir -p "$TEMP_DIR/bin"
    mkdir -p "$TEMP_DIR/config"
    mkdir -p "$TEMP_DIR/data"

    # Copy only the essential binaries (not duplicates or development tools)
    cp "$TESTNET_DIR/bin/knirvoracle" "$TEMP_DIR/bin/" 2>/dev/null || true
    cp "$TESTNET_DIR/bin/knirvchain" "$TEMP_DIR/bin/" 2>/dev/null || true
    cp "$TESTNET_DIR/bin/knirvgraph" "$TEMP_DIR/bin/" 2>/dev/null || true
    cp "$TESTNET_DIR/bin/knirvnexus" "$TEMP_DIR/bin/" 2>/dev/null || true
    cp "$TESTNET_DIR/bin/knirvrouter" "$TEMP_DIR/bin/" 2>/dev/null || true
    cp "$TESTNET_DIR/bin/knirv-server.wasm" "$TEMP_DIR/bin/" 2>/dev/null || true
    cp "$TESTNET_DIR/bin/wasmtime" "$TEMP_DIR/bin/" 2>/dev/null || true

    # Copy configuration files
    cp -r "$TESTNET_DIR/config"/* "$TEMP_DIR/config/" 2>/dev/null || true

    # Copy essential data directories (excluding large development files)
    print_status "Copying data directories (excluding node_modules, dist, build, etc.)..."

    # Copy knirvchain data (small config files only)
    if [ -d "$TESTNET_DIR/data/knirvchain" ]; then
        mkdir -p "$TEMP_DIR/data/knirvchain"
        cp "$TESTNET_DIR/data/knirvchain"/*.toml "$TEMP_DIR/data/knirvchain/" 2>/dev/null || true
        cp "$TESTNET_DIR/data/knirvchain"/*.db "$TEMP_DIR/data/knirvchain/" 2>/dev/null || true
    fi

    # Copy knirvgraph data (config only, not large data files)
    if [ -d "$TESTNET_DIR/data/knirvgraph" ]; then
        mkdir -p "$TEMP_DIR/data/knirvgraph"
        cp "$TESTNET_DIR/data/knirvgraph"/*.yaml "$TEMP_DIR/data/knirvgraph/" 2>/dev/null || true
        cp "$TESTNET_DIR/data/knirvgraph"/*.yml "$TEMP_DIR/data/knirvgraph/" 2>/dev/null || true
    fi

    # Copy knirvnexus data (config files only, NOT the portal directory)
    if [ -d "$TESTNET_DIR/data/knirvnexus" ]; then
        mkdir -p "$TEMP_DIR/data/knirvnexus"
        cp "$TESTNET_DIR/data/knirvnexus"/*.yaml "$TEMP_DIR/data/knirvnexus/" 2>/dev/null || true
        cp "$TESTNET_DIR/data/knirvnexus"/*.yml "$TEMP_DIR/data/knirvnexus/" 2>/dev/null || true
        cp "$TESTNET_DIR/data/knirvnexus"/*.db "$TEMP_DIR/data/knirvnexus/" 2>/dev/null || true
        cp "$TESTNET_DIR/data/knirvnexus"/*.config "$TEMP_DIR/data/knirvnexus/" 2>/dev/null || true
        # Copy small directories but exclude portal
        [ -d "$TESTNET_DIR/data/knirvnexus/reports" ] && cp -r "$TESTNET_DIR/data/knirvnexus/reports" "$TEMP_DIR/data/knirvnexus/" 2>/dev/null || true
    fi

    # Copy knirvoracle data (small config files only)
    if [ -d "$TESTNET_DIR/data/knirvoracle" ]; then
        mkdir -p "$TEMP_DIR/data/knirvoracle"
        cp -r "$TESTNET_DIR/data/knirvoracle/genesis" "$TEMP_DIR/data/knirvoracle/" 2>/dev/null || true
    fi

    # Copy knirvrouter data (small config files only)
    if [ -d "$TESTNET_DIR/data/knirvrouter" ]; then
        mkdir -p "$TEMP_DIR/data/knirvrouter"
        cp "$TESTNET_DIR/data/knirvrouter"/*.env "$TEMP_DIR/data/knirvrouter/" 2>/dev/null || true
        cp "$TESTNET_DIR/data/knirvrouter"/*.yaml "$TEMP_DIR/data/knirvrouter/" 2>/dev/null || true
        cp "$TESTNET_DIR/data/knirvrouter"/*.yml "$TEMP_DIR/data/knirvrouter/" 2>/dev/null || true
    fi

    # Copy testnet-gateway (source files only, NOT node_modules, .netlify, etc.)
    if [ -d "$TESTNET_DIR/data/testnet-gateway" ]; then
        mkdir -p "$TEMP_DIR/data/testnet-gateway"
        # Copy essential files but exclude large development directories
        rsync -av --exclude='node_modules' --exclude='dist' --exclude='build' --exclude='.next' \
              --exclude='.netlify' --exclude='*.log' --exclude='.git' --exclude='.cache' \
              "$TESTNET_DIR/data/testnet-gateway/" "$TEMP_DIR/data/testnet-gateway/" 2>/dev/null || {
            # Fallback if rsync is not available
            find "$TESTNET_DIR/data/testnet-gateway" -type f \
                ! -path "*/node_modules/*" ! -path "*/dist/*" ! -path "*/build/*" ! -path "*/.next/*" \
                ! -path "*/.netlify/*" ! -name "*.log" ! -path "*/.git/*" ! -path "*/.cache/*" \
                -exec cp --parents {} "$TEMP_DIR/data/" \; 2>/dev/null || true
        }
    fi

    # Copy docker-compose file
    cp "$TESTNET_DIR/docker-compose.yml" "$TEMP_DIR/" 2>/dev/null || true

    # Detect subnet environment and generate appropriate Docker compose file
    detect_subnet_environment > /dev/null
    generate_docker_compose_file "$TEMP_DIR" "$BEHIND_SUBNET"

    # Continue with the rest of the services (this will be appended to the generated file)
    cat >> "$TEMP_DIR/docker-compose-prod.yml" << 'EOF'
    environment:
      - DEPLOYMENT_ENV=production
      - KNIRVORACLE_API=http://knirv-oracle:1317
      - KNIRVCHAIN_API=http://knirv-chain:8090
      - KNIRVGRAPH_API=http://knirv-graph:8082
      - KNIRVSERVER_API=http://knirv-nexus:8084
      - KNIRVROUTER_API=http://knirv-router:8086
    volumes:
      - ./data/testnet-gateway:/app
    command: sh -c "npm install --production && npm start"
    depends_on:
      - knirv-oracle
      - knirv-chain
      - knirv-graph
      - knirv-nexus
      - knirv-router
    networks:
      - knirv-testnet
    restart: unless-stopped

  # KNIRV Oracle - Using pre-built binary
  knirv-oracle:
    image: ubuntu:22.04
    container_name: knirv-oracle
    expose:
      - "1317"        # Internal port only
      - "26657"       # Tendermint RPC
      - "26656"       # Tendermint P2P
    volumes:
      - ./bin/knirvoracle:/usr/local/bin/knirvoracle:ro
      - ./data/knirvoracle:/root/.knirvoracle
      - ./config:/root/config:ro
    command: /usr/local/bin/knirvoracle start
    depends_on:
      - ipfs
    networks:
      - knirv-testnet
    restart: unless-stopped

  # KNIRV Chain - Using pre-built binary
  knirv-chain:
    image: ubuntu:22.04
    container_name: knirv-chain
    expose:
      - "8090"        # Internal port only
    volumes:
      - ./bin/knirvchain:/usr/local/bin/knirvchain:ro
      - ./data/knirvchain:/app/data
      - ./config:/app/config:ro
    command: /usr/local/bin/knirvchain --port 8090
    depends_on:
      - knirv-oracle
      - ipfs
    networks:
      - knirv-testnet
    restart: unless-stopped

  # KNIRV Graph - Using pre-built binary
  knirv-graph:
    image: ubuntu:22.04
    container_name: knirv-graph
    expose:
      - "8082"        # Internal port only
    volumes:
      - ./bin/knirvgraph:/usr/local/bin/knirvgraph:ro
      - ./data/knirvgraph:/app/data
      - ./config:/app/config:ro
    command: /usr/local/bin/knirvgraph --port 8082
    depends_on:
      - knirv-oracle
      - ipfs
    networks:
      - knirv-testnet
    restart: unless-stopped

  # KNIRV Nexus - Using pre-built binary
  knirv-nexus:
    image: ubuntu:22.04
    container_name: knirv-nexus
    expose:
      - "8084"        # Internal port only
    volumes:
      - ./bin/knirvnexus:/usr/local/bin/knirvnexus:ro
      - ./data/knirvnexus:/app/data
      - ./config:/app/config:ro
    command: /usr/local/bin/knirvnexus --port 8084
    depends_on:
      - knirv-oracle
    networks:
      - knirv-testnet
    restart: unless-stopped

  # KNIRV Router - Using pre-built binary
  knirv-router:
    image: ubuntu:22.04
    container_name: knirv-router
    expose:
      - "8086"        # Internal port only
      - "3478"        # STUN/TURN port
    volumes:
      - ./bin/knirvrouter:/usr/local/bin/knirvrouter:ro
      - ./data/knirvrouter:/app/data
      - ./config:/app/config:ro
    command: /usr/local/bin/knirvrouter --port 8086
    depends_on:
      - knirv-oracle
    networks:
      - knirv-testnet
    restart: unless-stopped

  # KNIRVANA Game - TypeScript client
  knirvana:
    image: node:20-alpine
    container_name: knirv-testnet-knirvana
    working_dir: /app
    expose:
      - "3000"        # Internal port only
    volumes:
      - ./data/knirvana:/app
    environment:
      - NODE_ENV=production
      - KNIRVGRAPH_API=http://knirv-graph:8082
      - KNIRVCHAIN_API=http://knirv-chain:8090
      - KNIRVORACLE_API=http://knirv-oracle:1317
    command: sh -c "npm install && npm start"
    depends_on:
      - knirv-graph
      - knirv-chain
    networks:
      - knirv-testnet
    restart: unless-stopped

  # XION Bridge - Cross-chain bridge
  xion-bridge:
    image: node:20-alpine
    container_name: knirv-testnet-xion-bridge
    working_dir: /app
    expose:
      - "8088"        # Internal port only
    volumes:
      - ./data/xion-bridge:/app
    environment:
      - NODE_ENV=production
      - KNIRVORACLE_API=http://knirv-oracle:1317
      - KNIRVCHAIN_API=http://knirv-chain:8090
      - BRIDGE_PORT=8088
    command: sh -c "npm install && npm start"
    depends_on:
      - knirv-oracle
      - knirv-chain
    networks:
      - knirv-testnet
    restart: unless-stopped

networks:
  knirv-testnet:
    driver: bridge

volumes:
  ipfs_data:
EOF

    # Create appropriate compose file based on selected runtime
    if [ "$CONTAINER_RUNTIME" = "podman" ]; then
        print_status "Creating Podman-specific compose file..."
        # Update docker-compose.yml for production deployment
        sed -i 's|build:|#build:|g' "$TEMP_DIR/docker-compose.yml"
        sed -i 's|context: \.\./|#context: ../|g' "$TEMP_DIR/docker-compose.yml"
        sed -i 's|dockerfile: Dockerfile|#dockerfile: Dockerfile|g' "$TEMP_DIR/docker-compose.yml"

        # Create Podman-specific compose file
        cat > "$TEMP_DIR/podman-compose-prod.yml" << 'EOF'
version: '3.8'

services:
  ipfs:
    image: docker.io/ipfs/kubo:latest # Use explicit registry for Podman
    container_name: knirv-testnet-ipfs
    ports:
      - "5001:5001"
      - "8081:8081"
    volumes:
      - ./data/ipfs:/data/ipfs:Z # Add :Z for SELinux
    environment:
      - IPFS_PROFILE=server
    networks:
      - knirv-testnet
    restart: unless-stopped

  knirvoracle:
    image: knirvoracle:latest
    container_name: knirv-testnet-root
    ports:
      - "1317:1317"
      - "26657:26657"
      - "26656:26656"
    volumes:
      - ./data/knirvoracle:/root/.knirvoracle:Z
      - ./config/knirvoracle-config.yaml:/root/config.yaml:Z
    depends_on:
      - ipfs
    networks:
      - knirv-testnet
    environment:
      - KNIRV_NETWORK=testnet
      - KNIRV_CHAIN_ID=knirv-testnet-1
    restart: unless-stopped

  knirvchain:
    image: knirvchain:latest
    container_name: knirv-testnet-chain
    ports:
      - "8090:8090"
    volumes:
      - ./data/knirvchain:/app/data:Z
      - ./config/knirvchain-config.toml:/app/config.toml:Z
    depends_on:
      - knirvoracle
      - ipfs
    networks:
      - knirv-testnet
    environment:
      - RUST_LOG=info
    restart: unless-stopped

  knirvgraph:
    image: knirvgraph:latest
    container_name: knirv-testnet-graph
    ports:
      - "8081:8081"
      - "4001:4001"
    volumes:
      - ./data/knirvgraph:/app/data:Z
      - ./config/knirvgraph-config.yaml:/app/config.yaml:Z
    depends_on:
      - knirvoracle
      - ipfs
    networks:
      - knirv-testnet
    environment:
      - GO_ENV=testnet
    restart: unless-stopped

  knirvnexus-1:
    image: knirvnexus:latest
    container_name: knirv-testnet-nexus-1
    ports:
      - "8082:8082"
    volumes:
      - ./data/knirvnexus/node-1:/app/data:Z
      - ./config/knirvnexus-config.yaml:/app/config.yaml:Z
    environment:
      - NODE_ID=1
      - MOCK_TEE=true
    depends_on:
      - knirvoracle
    networks:
      - knirv-testnet
    restart: unless-stopped

  knirvnexus-2:
    image: knirvnexus:latest
    container_name: knirv-testnet-nexus-2
    ports:
      - "8083:8083"
    volumes:
      - ./data/knirvnexus/node-2:/app/data:Z
      - ./config/knirvnexus-config.yaml:/app/config.yaml:Z
    environment:
      - NODE_ID=2
      - MOCK_TEE=true
    depends_on:
      - knirvoracle
    networks:
      - knirv-testnet
    restart: unless-stopped

  knirvrouter:
    image: knirvrouter:latest
    container_name: knirv-testnet-router
    ports:
      - "8086:8086"
      - "3478:3478"
    volumes:
      - ./data/knirvrouter:/app/data:Z
      - ./config/knirvrouter-config.yaml:/app/config.yaml:Z
    depends_on:
      - knirvoracle
    networks:
      - knirv-testnet
    restart: unless-stopped

  knirvgateway:
    image: knirvgateway:latest
    container_name: knirv-testnet-gateway
    ports:
      - "8087:8087"
    volumes:
      - ./config/knirvgateway-config.json:/app/config.json:Z
    depends_on:
      - knirvoracle
      - knirvchain
      - knirvgraph
      - knirvnexus-1
      - knirvnexus-2
      - knirvrouter
    networks:
      - knirv-testnet
    environment:
      - NODE_ENV=testnet
    restart: unless-stopped

networks:
  knirv-testnet:
    name: knirv-testnet
    driver: bridge
EOF
    else
        print_status "Creating Docker-specific compose file..."
        # For Docker, create a production version that uses pre-built images
        cat > "$TEMP_DIR/docker-compose-prod.yml" << 'EOF'
version: '3.8'

services:
  ipfs:
    image: ipfs/kubo:latest
    container_name: knirv-testnet-ipfs
    ports:
      - "5001:5001"
      - "8080:8080"
    volumes:
      - ./data/ipfs:/data/ipfs
    environment:
      - IPFS_PROFILE=server
    networks:
      - knirv-testnet
    restart: unless-stopped

  knirvoracle:
    image: knirvoracle:latest
    container_name: knirv-testnet-root
    ports:
      - "1317:1317"
      - "26657:26657"
      - "26656:26656"
    volumes:
      - ./data/knirvoracle:/root/.knirvoracle
      - ./config/knirvoracle-config.yaml:/root/config.yaml
    depends_on:
      - ipfs
    networks:
      - knirv-testnet
    environment:
      - KNIRV_NETWORK=testnet
      - KNIRV_CHAIN_ID=knirv-testnet-1
    restart: unless-stopped

  knirvchain:
    image: knirvchain:latest
    container_name: knirv-testnet-chain
    ports:
      - "8080:8080"
    volumes:
      - ./data/knirvchain:/app/data
      - ./config/knirvchain-config.toml:/app/config.toml
    depends_on:
      - knirvoracle
      - ipfs
    networks:
      - knirv-testnet
    environment:
      - RUST_LOG=info
    restart: unless-stopped

  knirvgraph:
    image: knirvgraph:latest
    container_name: knirv-testnet-graph
    ports:
      - "8081:8081"
      - "4001:4001"
    volumes:
      - ./data/knirvgraph:/app/data
      - ./config/knirvgraph-config.yaml:/app/config.yaml
    depends_on:
      - knirvoracle
      - ipfs
    networks:
      - knirv-testnet
    environment:
      - GO_ENV=testnet
    restart: unless-stopped

  knirvnexus-1:
    image: knirvnexus:latest
    container_name: knirv-testnet-nexus-1
    ports:
      - "8082:8082"
    volumes:
      - ./data/knirvnexus/node-1:/app/data
      - ./config/knirvnexus-config.yaml:/app/config.yaml
    environment:
      - NODE_ID=1
      - MOCK_TEE=true
    depends_on:
      - knirvoracle
    networks:
      - knirv-testnet
    restart: unless-stopped

  knirvnexus-2:
    image: knirvnexus:latest
    container_name: knirv-testnet-nexus-2
    ports:
      - "8083:8083"
    volumes:
      - ./data/knirvnexus/node-2:/app/data
      - ./config/knirvnexus-config.yaml:/app/config.yaml
    environment:
      - NODE_ID=2
      - MOCK_TEE=true
    depends_on:
      - knirvoracle
    networks:
      - knirv-testnet
    restart: unless-stopped

  knirvrouter:
    image: knirvrouter:latest
    container_name: knirv-testnet-router
    ports:
      - "8086:8086"
      - "3478:3478"
    volumes:
      - ./data/knirvrouter:/app/data
      - ./config/knirvrouter-config.yaml:/app/config.yaml
    depends_on:
      - knirvoracle
    networks:
      - knirv-testnet
    restart: unless-stopped

  knirvgateway:
    image: knirvgateway:latest
    container_name: knirv-testnet-gateway
    ports:
      - "8087:8087"
    volumes:
      - ./config/knirvgateway-config.json:/app/config.json
    depends_on:
      - knirvoracle
      - knirvchain
      - knirvgraph
      - knirvnexus-1
      - knirvnexus-2
      - knirvrouter
    networks:
      - knirv-testnet
    environment:
      - NODE_ENV=testnet
    restart: unless-stopped

networks:
  knirv-testnet:
    name: knirv-testnet
    driver: bridge
EOF
    fi

    print_success "Testnet files prepared in $TEMP_DIR"
    echo "$TEMP_DIR" > /tmp/testnet_temp_dir
}

setup_container_permissions() {
    # Check if we should skip this step when resuming
    if [ "$RESUME_DEPLOYMENT" = "true" ] && [ "$(read_last_checkpoint)" = "Setting up container permissions..." ]; then
        print_status "Skipping setup_container_permissions (already completed)"
        return 0
    fi
    
    log_checkpoint "Setting up container permissions..."
    print_step "Setting up container runtime permissions on server..."

    if [ "$CONTAINER_RUNTIME" = "podman" ]; then
        print_status "Configuring Podman permissions..."
        ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
# Enable lingering for the ubuntu user (allows rootless containers to persist)
sudo loginctl enable-linger ubuntu

# Ensure proper permissions for rootless Podman
mkdir -p ~/.config/containers
mkdir -p ~/.local/share/containers

# Set up proper SELinux contexts if available
if command -v setsebool &> /dev/null; then
    sudo setsebool -P container_manage_cgroup true
fi

# Ensure proper ownership of container directories
sudo chown -R ubuntu:ubuntu /opt/knirv-testnet
chmod -R 755 /opt/knirv-testnet/data
chmod -R 755 /opt/knirv-testnet/config

echo "Podman permissions configured"
EOF
    else
        print_status "Configuring Docker permissions..."
        ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
# Add ubuntu user to docker group if not already added
if ! groups ubuntu | grep -q docker; then
    sudo usermod -aG docker ubuntu
    echo "Added ubuntu user to docker group"
fi

# Ensure Docker daemon is running
sudo systemctl enable docker
sudo systemctl start docker

# Ensure proper ownership of container directories
sudo chown -R ubuntu:ubuntu /opt/knirv-testnet
chmod -R 755 /opt/knirv-testnet/data
chmod -R 755 /opt/knirv-testnet/config

echo "Docker permissions configured"
EOF
    fi

    print_success "Container runtime permissions configured"
}

upload_testnet_files() {
    # Check if we should skip this step when resuming
    if [ "$RESUME_DEPLOYMENT" = "true" ] && [ "$(read_last_checkpoint)" = "Uploading testnet files..." ]; then
        print_status "Skipping upload_testnet_files (already completed)"
        return 0
    fi
    
    log_checkpoint "Uploading testnet files..."
    print_step "Uploading testnet files to server..."

  TEMP_DIR=$(cat /tmp/testnet_temp_dir)

  # Ensure remote /opt dir exists (will be used after move)
  ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" \
    "sudo mkdir -p /opt/knirv-testnet && sudo chown ubuntu:ubuntu /opt/knirv-testnet"

  # Rsync to a remote temp dir then move with sudo to avoid permission denied errors
  REMOTE_TMP_DIR="/tmp/knirv_deploy_$(date +%s)"
  ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "mkdir -p $REMOTE_TMP_DIR"

  rsync -avz --delete -e "ssh -i ${SSH_KEY/#\~/$HOME} -o StrictHostKeyChecking=no" \
      "$TEMP_DIR/" ubuntu@"$TESTNET_IP":$REMOTE_TMP_DIR/ || {
    print_error "Rsync upload failed"
    return 1
  }

  ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << EOF
sudo rm -rf /opt/knirv-testnet/* || true
sudo mv $REMOTE_TMP_DIR/* /opt/knirv-testnet/
sudo chown -R ubuntu:ubuntu /opt/knirv-testnet
sudo chmod -R 755 /opt/knirv-testnet
rm -rf $REMOTE_TMP_DIR
EOF

    # Set up container runtime permissions
    setup_container_permissions

    print_success "Files uploaded to testnet server"
}

install_node_dependencies() {
    # Check if we should skip this step when resuming
    if [ "$RESUME_DEPLOYMENT" = "true" ] && [ "$(read_last_checkpoint)" = "Installing Node.js dependencies..." ]; then
        print_status "Skipping install_node_dependencies (already completed)"
        return 0
    fi
    
    log_checkpoint "Installing Node.js dependencies..."
    print_step "Installing Node.js dependencies on testnet server..."

    # First, ensure Node.js is installed
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" \
        "if ! command -v node >/dev/null 2>&1; then \
            echo 'Installing Node.js...' && \
            curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash - && \
            sudo apt-get install -y nodejs; \
        else \
            echo 'Node.js is already installed'; \
        fi"

    # Check if testnet-gateway exists and needs npm install
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" \
        "cd /opt/knirv-testnet && if [ -d data/testnet-gateway ] && [ -f data/testnet-gateway/package.json ]; then \
            echo 'Installing testnet-gateway dependencies...' && \
            cd data/testnet-gateway && \
            npm install --production --no-optional --no-audit --no-fund; \
        else \
            echo 'No testnet-gateway package.json found, skipping npm install'; \
        fi"

    print_success "Node.js dependencies installed"
}

build_container_images() {
    # Check if we should skip this step when resuming
    if [ "$RESUME_DEPLOYMENT" = "true" ] && [ "$(read_last_checkpoint)" = "Building container images..." ]; then
        print_status "Skipping build_container_images (already completed)"
        return 0
    fi
    
    log_checkpoint "Building container images..."
    print_step "Building container images on testnet server..."
    print_warning "Using real KNIRV binaries for production testnet deployment."

    # Build images on the server using the selected container runtime
    if [ "$CONTAINER_RUNTIME" = "podman" ]; then
        print_status "Building images with Podman..."
        ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
cd /opt/knirv-testnet

# Build KNIRV services using real binaries
echo "Building KNIRVORACLE..."
podman build -t knirvoracle:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvoracle /app/knirvoracle
WORKDIR /app
EXPOSE 1317 26656 26657
CMD ["./knirvoracle"]
CONTAINERFILE

echo "Building KNIRVCHAIN..."
podman build -t knirvchain:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvchain /app/knirvchain
WORKDIR /app
EXPOSE 8080
CMD ["./knirvchain"]
CONTAINERFILE

echo "Building KNIRVGRAPH..."
podman build -t knirvgraph:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvgraph /app/knirvgraph
WORKDIR /app
EXPOSE 8081
CMD ["./knirvgraph"]
CONTAINERFILE

echo "Building KNIRVNEXUS..."
podman build -t knirvnexus:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvnexus /app/knirvnexus
WORKDIR /app
EXPOSE 8082 8083
CMD ["./knirvnexus"]
CONTAINERFILE

echo "Building KNIRVROUTER..."
podman build -t knirvrouter:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvrouter /app/knirvrouter
WORKDIR /app
EXPOSE 8086 3478
CMD ["./knirvrouter"]
CONTAINERFILE

echo "Building KNIRVGATEWAY (using WASM runtime)..."
podman build -t knirvgateway:latest -f - . << 'CONTAINERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/wasmtime /app/wasmtime
COPY bin/knirv-server.wasm /app/knirv-server.wasm
WORKDIR /app
EXPOSE 8087
CMD ["./wasmtime", "knirv-server.wasm"]
CONTAINERFILE

echo "Podman images built successfully"
EOF
    else
        print_status "Building images with Docker..."
        ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
cd /opt/knirv-testnet

# Build KNIRV services using real binaries with Docker
echo "Building KNIRVORACLE..."
docker build -t knirvoracle:latest -f - . << 'DOCKERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvoracle /app/knirvoracle
WORKDIR /app
EXPOSE 1317 26656 26657
CMD ["./knirvoracle"]
DOCKERFILE

echo "Building KNIRVCHAIN..."
docker build -t knirvchain:latest -f - . << 'DOCKERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvchain /app/knirvchain
WORKDIR /app
EXPOSE 8080
CMD ["./knirvchain"]
DOCKERFILE

echo "Building KNIRVGRAPH..."
docker build -t knirvgraph:latest -f - . << 'DOCKERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvgraph /app/knirvgraph
WORKDIR /app
EXPOSE 8081
CMD ["./knirvgraph"]
DOCKERFILE

echo "Building KNIRVNEXUS..."
docker build -t knirvnexus:latest -f - . << 'DOCKERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvnexus /app/knirvnexus
WORKDIR /app
EXPOSE 8082 8083
CMD ["./knirvnexus"]
DOCKERFILE

echo "Building KNIRVROUTER..."
docker build -t knirvrouter:latest -f - . << 'DOCKERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/knirvrouter /app/knirvrouter
WORKDIR /app
EXPOSE 8086 3478
CMD ["./knirvrouter"]
DOCKERFILE

echo "Building KNIRVGATEWAY (using WASM runtime)..."
docker build -t knirvgateway:latest -f - . << 'DOCKERFILE'
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl ca-certificates
COPY bin/wasmtime /app/wasmtime
COPY bin/knirv-server.wasm /app/knirv-server.wasm
WORKDIR /app
EXPOSE 8087
CMD ["./wasmtime", "knirv-server.wasm"]
DOCKERFILE

echo "Docker images built successfully"
EOF
    fi

    print_success "Container images built on testnet server"
}

deploy_native_testnet() {
    # Check if we should skip this step when resuming
    if [ "$RESUME_DEPLOYMENT" = "true" ] && [ "$(read_last_checkpoint)" = "Deploying native testnet..." ]; then
        print_status "Skipping deploy_native_testnet (already completed)"
        return 0
    fi

    log_checkpoint "Deploying native testnet..."
    print_step "Deploying KNIRVTESTNET via unified Go orchestrator..."

    # Build and upload the orchestrator if not already done
    if ! ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "[ -f /usr/local/bin/knirvtestnet ]"; then
        print_status "Orchestrator not found on remote. Building and uploading..."
        build_and_upload_orchestrator
    else
        print_status "Orchestrator already exists on remote server"
    fi

    # Deploy and start the orchestrator
    print_status "Starting orchestrator on remote..."
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
# Move orchestrator to /usr/local/bin if it's in /tmp
if [ -f /tmp/knirvtestnet ]; then
    echo "Installing orchestrator to /usr/local/bin..."
    sudo mv /tmp/knirvtestnet /usr/local/bin/knirvtestnet
    sudo chmod +x /usr/local/bin/knirvtestnet
fi

# Stop existing orchestrator instance if running
echo "Stopping any existing orchestrator instances..."
sudo pkill knirvtestnet || true
sleep 2

# Start new orchestrator instance in background
echo "Starting KNIRV orchestrator..."
sudo nohup /usr/local/bin/knirvtestnet /opt/knirv-testnet > /var/log/knirvtestnet.log 2>&1 &

# Store PID
ORCH_PID=$!
echo $ORCH_PID > /tmp/knirvtestnet.pid
echo "✅ KNIRV Orchestrator started (PID: $ORCH_PID)"

# Wait for services to initialize
echo "⏳ Waiting for services to initialize (15 seconds)..."
sleep 15

# Verify services are running
echo ""
echo "🔍 Verifying service health..."
curl -s http://localhost:1317/health && echo "  ✅ ORACLE healthy" || echo "  ⚠️  ORACLE not ready yet"
curl -s http://localhost:8090/health && echo "  ✅ CHAIN healthy" || echo "  ⚠️  CHAIN not ready yet"
curl -s http://localhost:8082/height && echo "  ✅ GRAPH healthy" || echo "  ⚠️  GRAPH not ready yet"
curl -s http://localhost:8086/status && echo "  ✅ ROUTER healthy" || echo "  ⚠️  ROUTER not ready yet"

# Only check NEXUS if the binary exists (private/corporate testing)
if [ -f /usr/local/bin/knirvnexus ] || [ -f /opt/knirv-testnet/bin/knirvnexus ]; then
    curl -s http://localhost:8084/ && echo "  ✅ NEXUS healthy (private mode)" || echo "  ⚠️  NEXUS not ready yet"
fi

echo ""
echo "📄 Orchestrator log: /var/log/knirvtestnet.log"
echo "🎉 Native deployment completed!"
EOF

    if [ $? -eq 0 ]; then
        print_success "Native KNIRVTESTNET deployment completed successfully!"
        print_status "View logs: ssh -i ${SSH_KEY/#\~/$HOME} ubuntu@$TESTNET_IP 'tail -f /var/log/knirvtestnet.log'"
        print_status "Stop orchestrator: ssh -i ${SSH_KEY/#\~/$HOME} ubuntu@$TESTNET_IP 'sudo pkill knirvtestnet'"
    else
        handle_deployment_error "NATIVE_DEPLOYMENT" "Failed to deploy native testnet" ""
    fi
}

deploy_testnet_services() {
    # Check if we should skip this step when resuming
    if [ "$RESUME_DEPLOYMENT" = "true" ] && [ "$(read_last_checkpoint)" = "Deploying testnet services..." ]; then
        print_status "Skipping deploy_testnet_services (already completed)"
        return 0
    fi
    
    log_checkpoint "Deploying testnet services..."
    print_step "Deploying KNIRVTESTNET services..."

    # Deploy services using the selected container runtime
    if [ "$CONTAINER_RUNTIME" = "native" ]; then
        deploy_native_testnet
    elif [ "$CONTAINER_RUNTIME" = "podman" ]; then
        print_status "Deploying with Podman..."
        ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
cd /opt/knirv-testnet

# Stop any existing services
podman-compose -f podman-compose-prod.yml down || true

# Start services
podman-compose -f podman-compose-prod.yml up -d

# Wait for services to start
sleep 30

# Check service status
podman-compose -f podman-compose-prod.yml ps
EOF
    else
        print_status "Deploying with Docker..."
        # Pass the compose command to the remote server
        ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << EOF
cd /opt/knirv-testnet

# Determine Docker Compose command on remote server
if command -v docker-compose &> /dev/null; then
    COMPOSE_CMD="docker-compose"
elif docker compose version &> /dev/null; then
    COMPOSE_CMD="docker compose"
else
    handle_deployment_error "REMOTE_DOCKER_COMPOSE_MISSING" "Docker Compose not found on remote server"
fi

echo "Using Docker Compose command: \$COMPOSE_CMD"

# Stop any existing services
\$COMPOSE_CMD -f docker-compose-prod.yml down || true

# Start services
\$COMPOSE_CMD -f docker-compose-prod.yml up -d

# Wait for services to start
sleep 30

# Check service status
\$COMPOSE_CMD -f docker-compose-prod.yml ps
EOF
    fi

    print_success "KNIRVTESTNET services deployed"
}

verify_deployment() {
    # Check if we should skip this step when resuming
    if [ "$RESUME_DEPLOYMENT" = "true" ] && [ "$(read_last_checkpoint)" = "Verifying deployment..." ]; then
        print_status "Skipping verify_deployment (already completed)"
        return 0
    fi
    
    log_checkpoint "Verifying deployment..."
    print_step "Verifying testnet deployment with dynamic service discovery..."

    # Deploy the dynamic health check script and AI troubleshooter
    print_status "Deploying diagnostic scripts..."
    if scp -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no scripts/remote-health-check.sh ubuntu@"$TESTNET_IP":/tmp/remote-health-check.sh >/dev/null 2>&1; then
        ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "chmod +x /tmp/remote-health-check.sh" >/dev/null 2>&1
        print_success "Dynamic health check script deployed"
        
        # Also deploy AI troubleshooter script
        if scp -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no scripts/ai_troubleshoot.js ubuntu@"$TESTNET_IP":/tmp/ai_troubleshoot.js >/dev/null 2>&1; then
            print_success "AI troubleshooter script deployed"
            
            # Install Node.js dependencies for AI troubleshooter on remote server if needed
            print_status "Ensuring AI troubleshooter dependencies are available..."
            ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
            if command -v node >/dev/null 2>&1; then
                echo "Node.js is available on remote server"
                # Check if dependencies are installed
                if [ ! -d "/opt/knirv-testnet/node_modules" ]; then
                    echo "Installing AI troubleshooter dependencies..."
                    cd /opt/knirv-testnet
                    npm install dotenv yargs --no-save --no-package-lock >/dev/null 2>&1 || echo "Dependency installation completed"
                fi
            else
                echo "Node.js not available on remote server - AI troubleshooter will run locally"
            fi
EOF
        else
            print_warning "Could not deploy AI troubleshooter script"
        fi
    else
        print_warning "Could not deploy dynamic health check script, using basic verification"

        # Fallback to basic health check
        echo ""
        echo -e "${YELLOW}Basic Service Health Check:${NC}"

        basic_services=(
            "ipfs:5001:/api/v0/version"
            "testnet-gateway:10000:/"
            "knirv-oracle:1317:/health"
        )

        local unhealthy_services=()
        for service in "${basic_services[@]}"; do
            name=$(echo "$service" | cut -d: -f1)
            port=$(echo "$service" | cut -d: -f2)
            endpoint=$(echo "$service" | cut -d: -f3)

            if curl -s -f "http://$TESTNET_IP:$port$endpoint" > /dev/null 2>&1; then
                echo -e "  ${GREEN}✅ $name ($port): HEALTHY${NC}"
            else
                echo -e "  ${RED}❌ $name ($port): UNHEALTHY${NC}"
                unhealthy_services+=("$name:$port")
            fi
        done

        # Check if any services are unhealthy and offer AI troubleshooting
        if [ ${#unhealthy_services[@]} -gt 0 ]; then
            print_warning "${#unhealthy_services[@]} service(s) are unhealthy: ${unhealthy_services[*]}"
            
            # Ask user if they want to run AI troubleshooter
            echo ""
            echo -e "${YELLOW}Some services are unhealthy. Would you like to run the AI troubleshooter?${NC}"
            read -p "Run AI troubleshooter? (y/N): " -n 1 -r
            echo
            
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                print_step "Starting AI troubleshooter for unhealthy services..."
                
                # Try to run AI troubleshooter on remote server first, fallback to local
                if ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "cd /opt/knirv-testnet && node /tmp/ai_troubleshoot.js --ip localhost --ssh-key $SSH_KEY --health-endpoint /health" 2>/dev/null; then
                    print_success "AI troubleshooter completed (remote execution)"
                elif node "$SCRIPT_DIR/ai_troubleshoot.js" --ip "$TESTNET_IP" --ssh-key "$SSH_KEY"; then
                    print_success "AI troubleshooter completed (local execution)"
                else
                    print_warning "AI troubleshooter failed or was interrupted"
                fi
            else
                print_status "AI troubleshooter skipped by user"
            fi
        fi

        print_success "Basic deployment verification completed"
        return
    fi

    # Run dynamic health check
    print_status "Running dynamic service discovery..."
    echo ""
    echo -e "${YELLOW}Dynamic Service Discovery Results:${NC}"

    # Run dynamic health check and capture output
    local health_output=$(ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "/tmp/remote-health-check.sh --json" 2>/dev/null || echo "{}")
    
    if echo "$health_output" | grep -q "HEALTHY" || echo "$health_output" | jq -e '.services | length > 0' >/dev/null 2>&1; then
        print_success "Dynamic deployment verification completed"
        
        # Parse JSON output to check for unhealthy services
        if command -v jq >/dev/null 2>&1 && echo "$health_output" | jq -e '.services' >/dev/null 2>&1; then
            local unhealthy_count=$(echo "$health_output" | jq '[.services[] | select(.status != "HEALTHY")] | length')
            if [ "$unhealthy_count" -gt 0 ]; then
                print_warning "$unhealthy_count service(s) are unhealthy"
                
                # Extract unhealthy service names
                local unhealthy_services=$(echo "$health_output" | jq -r '[.services[] | select(.status != "HEALTHY") | "\(.service_name):\(.port)"] | join(",")')
                
                # Ask user if they want to run AI troubleshooter
                echo ""
                echo -e "${YELLOW}Some services are unhealthy. Would you like to run the AI troubleshooter?${NC}"
                read -p "Run AI troubleshooter? (y/N): " -n 1 -r
                echo
                
                if [[ $REPLY =~ ^[Yy]$ ]]; then
                    print_step "Starting AI troubleshooter for unhealthy services..."
                    
                    # Try to run AI troubleshooter on remote server first, fallback to local
                    if ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "cd /opt/knirv-testnet && node /tmp/ai_troubleshoot.js --ip localhost --ssh-key $SSH_KEY --health-endpoint /health" 2>/dev/null; then
                        print_success "AI troubleshooter completed (remote execution)"
                    elif node "$SCRIPT_DIR/ai_troubleshoot.js" --ip "$TESTNET_IP" --ssh-key "$SSH_KEY"; then
                        print_success "AI troubleshooter completed (local execution)"
                    else
                        print_warning "AI troubleshooter failed or was interrupted"
                    fi
                else
                    print_status "AI troubleshooter skipped by user"
                fi
            fi
        fi
    else
        print_warning "Dynamic health check failed or no healthy services detected"
        print_status "Services may still be starting up - check again in a few minutes"
        
        # Offer AI troubleshooting even if health check fails completely
        echo ""
        echo -e "${YELLOW}Health check failed. Would you like to run the AI troubleshooter to diagnose the issue?${NC}"
        read -p "Run AI troubleshooter? (y/N): " -n 1 -r
        echo
        
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_step "Starting AI troubleshooter for deployment diagnosis..."
            
            # Try to run AI troubleshooter on remote server first, fallback to local
            if ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" "cd /opt/knirv-testnet && node /tmp/ai_troubleshoot.js --ip localhost --ssh-key $SSH_KEY --health-endpoint /health" 2>/dev/null; then
                print_success "AI troubleshooter completed (remote execution)"
            elif node "$SCRIPT_DIR/ai_troubleshoot.js" --ip "$TESTNET_IP" --ssh-key "$SSH_KEY"; then
                print_success "AI troubleshooter completed (local execution)"
            else
                print_warning "AI troubleshooter failed or was interrupted"
            fi
        else
            print_status "AI troubleshooter skipped by user"
        fi
    fi
}

verify_service_ports() {
    # Check if we should skip this step when resuming
    if [ "$RESUME_DEPLOYMENT" = "true" ] && [ "$(read_last_checkpoint)" = "Verifying service ports..." ]; then
        print_status "Skipping verify_service_ports (already completed)"
        return 0
    fi
    
    log_checkpoint "Verifying service ports..."
    print_step "Verifying service ports and UFW configuration..."

    # Define expected services and ports
    declare -A expected_services=(
        ["knirvoracle"]="1317"
        ["knirvchain"]="8090"
        ["knirvgraph"]="8082"
        ["knirvnexus-dve"]="8084"
        ["knirvnexus-val"]="8085"
        ["knirvrouter"]="8086"
        ["testnet-gateway"]="10000"
        ["knirvana"]="3000"
        ["xion-bridge"]="8088"
        ["ipfs-api"]="5001"
        ["ipfs-gateway"]="8081"
    )

    local verified_services=()
    local failed_services=()

    for service in "${!expected_services[@]}"; do
        local port="${expected_services[$service]}"
        print_status "Verifying $service on port $port..."

        # Test port connectivity
        if timeout 10 bash -c "echo >/dev/tcp/$TESTNET_IP/$port" 2>/dev/null; then
            print_success "✅ $service ($port): Port accessible"
            verified_services+=("$service:$port")
        else
            print_warning "❌ $service ($port): Port not accessible"
            failed_services+=("$service:$port")
        fi
    done

    echo ""
    print_status "Port verification summary:"
    print_success "Verified services: ${#verified_services[@]}"
    print_warning "Failed services: ${#failed_services[@]}"

    if [ ${#failed_services[@]} -gt 0 ]; then
        print_warning "Failed services will not have DNS records updated:"
        for failed in "${failed_services[@]}"; do
            print_warning "  - $failed"
        done
    fi

    # Return the list of verified services
    printf '%s\n' "${verified_services[@]}"
}

update_cloudflare_dns() {
    # Check if we should skip this step when resuming
    if [ "$RESUME_DEPLOYMENT" = "true" ] && [ "$(read_last_checkpoint)" = "Updating CloudFlare DNS..." ]; then
        print_status "Skipping update_cloudflare_dns (already completed)"
        return 0
    fi
    
    log_checkpoint "Updating CloudFlare DNS..."
    print_step "Updating CloudFlare DNS records for healthy services..."

    # Check if CloudFlare credentials are available
    if [ -z "$CLOUDFLARE_API_TOKEN" ] && [ -z "$CLOUDFLARE_EMAIL" ]; then
        print_warning "CloudFlare credentials not found - skipping DNS updates"
        print_warning "To enable DNS updates, set either:"
        print_warning "  export CLOUDFLARE_API_TOKEN='your-token'"
        print_warning "  or export CLOUDFLARE_EMAIL='email' and CLOUDFLARE_API_KEY='key'"
        return 0
    fi

    # Wait for services to fully start before testing
    print_status "Waiting 45 seconds for services to fully initialize..."
    sleep 45

    # Verify UFW and port accessibility first
    print_status "Checking UFW status and port accessibility..."
    ssh -i "${SSH_KEY/#\~/$HOME}" -o StrictHostKeyChecking=no ubuntu@"$TESTNET_IP" << 'EOF'
echo "UFW Status:"
sudo ufw status numbered | head -20
echo ""
echo "Active listening ports:"
sudo netstat -tlnp | grep -E ':(1317|8090|8082|8084|8085|8086|10000|3000|8088|5001|8081)\s' | head -10
EOF

    # Verify service ports before DNS updates
    local verified_services_output=$(verify_service_ports)
    local verified_count=$(echo "$verified_services_output" | wc -l)

    if [ "$verified_count" -eq 0 ]; then
        print_error "No services are accessible - skipping DNS updates"
        print_warning "Please check service status and firewall configuration"
        return 1
    fi

    # Run CloudFlare DNS update script with enhanced error handling
    if [ -f "./scripts/update-cloudflare-dns.sh" ]; then
        chmod +x "./scripts/update-cloudflare-dns.sh"

        print_status "Testing service health before DNS updates..."
        if ./scripts/update-cloudflare-dns.sh test "$TESTNET_IP" 2>&1 | tee /tmp/dns_test.log; then
            print_success "Service health test completed"
        else
            print_warning "Service health test had issues - check /tmp/dns_test.log"
        fi

        echo ""
        print_status "Updating DNS records for verified healthy services only..."
        if ./scripts/update-cloudflare-dns.sh update "$TESTNET_IP" 2>&1 | tee /tmp/dns_update.log; then
            print_success "CloudFlare DNS records updated successfully"

            # Show updated records
            print_status "Updated DNS records:"
            grep -E "(Updated DNS record|Created DNS record)" /tmp/dns_update.log | head -10 || true
        else
            print_error "DNS update failed - check /tmp/dns_update.log for details"
            print_warning "Common issues:"
            print_warning "  - Invalid CloudFlare credentials"
            print_warning "  - Zone not found or access denied"
            print_warning "  - Rate limiting"
            print_warning "  - Network connectivity issues"
            return 1
        fi
    else
        print_warning "CloudFlare DNS update script not found at ./scripts/update-cloudflare-dns.sh"
        print_warning "DNS updates skipped - services are accessible but DNS won't be updated"
    fi

    # Cleanup temporary log files
    rm -f /tmp/dns_test.log /tmp/dns_update.log
}

cleanup() {
    # Check if we should skip this step when resuming
    if [ "$RESUME_DEPLOYMENT" = "true" ] && [ "$(read_last_checkpoint)" = "Cleaning up temporary files..." ]; then
        print_status "Skipping cleanup (already completed)"
        return 0
    fi
    
    log_checkpoint "Cleaning up temporary files..."
    print_step "Cleaning up temporary files..."

    if [ -f /tmp/testnet_temp_dir ]; then
        TEMP_DIR=$(cat /tmp/testnet_temp_dir)
        rm -rf "$TEMP_DIR"
        rm -f /tmp/testnet_temp_dir
    fi

    print_success "Cleanup completed"
}

display_summary() {
    # Check if we should skip this step when resuming
    if [ "$RESUME_DEPLOYMENT" = "true" ] && [ "$(read_last_checkpoint)" = "Displaying deployment summary..." ]; then
        print_status "Skipping display_summary (already completed)"
        return 0
    fi
    
    log_checkpoint "Displaying deployment summary..."
  print_step "Displaying deployment summary..."
    echo ""
    echo -e "${GREEN}🎉 KNIRVTESTNET Services Deployment Complete!${NC}"
    echo -e "${BLUE}=============================================${NC}"
    echo ""
    echo -e "${YELLOW}Testnet Information:${NC}"
    echo -e "  🌐 Public IP: $TESTNET_IP"
    echo -e "  🔗 Testnet URL: https://testnet.knirv.com"
    echo ""
    echo -e "${YELLOW}Service Endpoints (via NGINX Network Manager):${NC}"
    echo -e "  🌐 NGINX Manager: http://$TESTNET_IP:80"
    echo -e "  📊 IPFS API: http://$TESTNET_IP:5001"
    echo -e "  📊 IPFS Gateway: http://$TESTNET_IP:8081"
    echo -e "  🌐 Testnet Gateway: http://$TESTNET_IP:10000"
    echo -e "  🏛️ KNIRVORACLE: http://$TESTNET_IP:1317"
    echo -e "  ⛓️ KNIRVCHAIN: http://$TESTNET_IP:8090"
    echo -e "  📈 KNIRVGRAPH: http://$TESTNET_IP:8082"
    echo -e "  🤖 KNIRVSERVER DVE: http://$TESTNET_IP:8084"
    echo -e "  🤖 KNIRVSERVER VAL: http://$TESTNET_IP:8085"
    echo -e "  🌐 KNIRVROUTER: http://$TESTNET_IP:8086"
    echo -e "  🎮 KNIRVANA Game: http://$TESTNET_IP:3000"
    echo -e "  🌉 XION Bridge: http://$TESTNET_IP:8088"
    echo ""
    echo -e "${YELLOW}KNIRVCONTROLLER PWA:${NC}"
    echo -e "  📱 PWA Testnet: https://controller-testnet.knirv.network"
    echo -e "  📱 Android Download: https://controller-testnet.knirv.network/android"
    echo -e "  📱 iOS Download: https://controller-testnet.knirv.network/ios"
    echo ""
    echo -e "${YELLOW}Next Steps:${NC}"
    echo -e "  1. Check service health: ./check-testnet-health.sh"
    echo -e "  2. Deploy KNIRVCONTROLLER PWA: make deploy-controller-pwa ENVIRONMENT=testnet"
    echo -e "  3. Update frontend: make update-testnet-frontend"
    if [ "$CONTAINER_RUNTIME" = "native" ]; then
        echo -e "  3. Monitor services: ssh knirv-testnet 'ps aux | grep knirv'"
        echo -e "  4. View logs: ssh knirv-testnet 'tail -f /opt/knirv-testnet/logs/*.log'"
        echo -e "  5. Restart services: ssh knirv-testnet 'cd /opt/knirv-testnet && npm restart'"
    elif [ "$CONTAINER_RUNTIME" = "podman" ]; then
        echo -e "  3. Monitor services: ssh knirv-testnet 'podman ps'"
        echo -e "  4. View logs: ssh knirv-testnet 'podman-compose -f /opt/knirv-testnet/podman-compose-prod.yml logs'"
    else
        echo -e "  3. Monitor services: ssh knirv-testnet 'docker ps'"
        echo -e "  4. View logs: ssh knirv-testnet 'cd /opt/knirv-testnet && (\$COMPOSE_CMD -f docker-compose-prod.yml logs || docker-compose -f docker-compose-prod.yml logs || docker compose -f docker-compose-prod.yml logs)'"
        echo -e "     (Use whichever Docker Compose command is available on the server)"
    fi
    echo ""
    echo -e "${YELLOW}Health Check Options:${NC}"
    echo -e "  • Dynamic discovery: ./check-testnet-health.sh"
    echo -e "  • JSON output: ./check-testnet-health.sh --json"
    echo -e "  • Skip CloudFlare: ./check-testnet-health.sh --no-cloudflare"
    echo ""
}

# Main execution
main() {
    print_header

    # Load environment variables first
    load_dotenv

    # Check prerequisites
    check_prerequisites

    # Test SSH connection
    test_ssh_connection

    # Check disk space
    check_disk_space

    # Deploy based on selected container runtime
    if [ "$CONTAINER_RUNTIME" = "native" ]; then
        build_and_upload_orchestrator
    else
        # For container runtimes, prepare and upload files
        prepare_testnet_files
        upload_testnet_files

        # Install Node.js dependencies on the server
        install_node_dependencies

        # Build container images on the server
        build_container_images

        # Start services
        deploy_testnet_services
    fi

    # Final health check
    verify_deployment

    print_success "KNIRV Testnet deployment completed successfully!"
}

# Handle script arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --help, -h         Show this help message"
            echo "  --force            Skip confirmation prompts and force full deployment"
            echo "  --incremental      Enable incremental deployment (only upload changed files)"
            echo "  --full             Force full deployment (default)"
        echo ""
        echo "Features:"
        echo "  - Automatic detection of Docker and Podman"
        echo "  - Interactive selection between container runtimes and native deployment"
        echo "  - Native deployment option using npm scripts (like local testnet)"
        echo "  - Includes XION bridge and KNIRVANA components"
        echo "  - Checks for already running services"
        echo "  - Proper permission setup for both Docker and Podman"
        echo "  - Graceful handling of existing containers"
        echo ""
        echo "Deployment Options:"
        echo "  1. Docker - Traditional containerized deployment"
        echo "  2. Podman - Rootless containerized deployment"
        echo "  3. Native - Upload KNIRVTESTNET directory and run npm scripts"
        echo ""
        echo "Prerequisites:"
        echo "  - Testnet infrastructure must be deployed first"
        echo "  - SSH key must be available at ~/.ssh/AEGONG.pem"
        echo "  - KNIRVTESTNET directory must exist"
        echo "  - For container deployment: Docker or Podman must be installed"
            echo "  - For native deployment: Node.js will be installed automatically"
            echo ""
            echo "Deployment Modes:"
            echo "  --incremental      Only upload files that have changed since last deployment"
            echo "  --full             Upload all files (default, slower but more reliable)"
            echo "  --force            Force full deployment even if no changes detected"
            echo ""
            exit 0
            ;;
        --force)
            print_warning "Force mode - skipping confirmations and forcing full deployment"
            FORCE_FULL_DEPLOYMENT=true
            ;;
        --incremental)
            print_status "Incremental deployment mode enabled"
            INCREMENTAL_DEPLOYMENT=true
            ;;
        --full)
            print_status "Full deployment mode enabled"
            INCREMENTAL_DEPLOYMENT=false
            ;;
        *)
            if [[ $1 == --* ]]; then
                print_error "Unknown option: $1"
                handle_deployment_error "INVALID_ARGUMENT" "Unknown option: $1. Use --help for usage information"
            fi
            break
            ;;
    esac
    shift
done

# Confirm deployment if not in force mode
if [ "$FORCE_FULL_DEPLOYMENT" != "true" ]; then
    echo -e "${YELLOW}This will deploy KNIRVTESTNET services to AWS EC2.${NC}"
    echo -e "${YELLOW}The script will:${NC}"
    echo -e "${YELLOW}  - Detect available deployment options (Docker/Podman/Native)${NC}"
    echo -e "${YELLOW}  - Include XION bridge and KNIRVANA components${NC}"
    echo -e "${YELLOW}  - Check for existing services and handle them gracefully${NC}"
    echo -e "${YELLOW}  - Set up proper permissions for the selected deployment method${NC}"
    echo -e "${YELLOW}  - Deploy services using the chosen deployment method${NC}"
    if [ "$INCREMENTAL_DEPLOYMENT" = "true" ]; then
        echo -e "${YELLOW}  - Use incremental deployment (only upload changed files)${NC}"
    fi
    echo ""
    read -p "Continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Deployment cancelled."
        exit 0
    fi
fi

# Set up error handling AFTER all functions are defined
trap 'handle_uncaught_error' ERR

# Run main function
main
