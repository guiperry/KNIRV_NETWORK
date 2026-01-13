#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

# Define docker-compose file path (relative to the repository root)
DOCKER_COMPOSE_FILE="deployment/docker-compose.knirv-production.yml"
DOCKER_REGISTRY="knirv" # Local registry tag prefix
VERSION="latest" # Or "v1.0.0" if you want to be specific, but for local testing "latest" is fine

# Function to run docker compose command, handling new syntax
run_docker_compose() {
    if command -v docker >/dev/null && docker compose version >/dev/null 2>&1; then
        docker compose "$@"
    elif command -v docker-compose >/dev/null; then
        docker-compose "$@"
    else
        echo "Error: docker compose or docker-compose not found." >&2
        exit 1
    fi
}

# Function to build Docker images
build_images() {
    echo "Building Docker images..."
    
    # Mapping of service name in docker-compose to local directory
    declare -A services_to_build
    services_to_build["chain"]="KNIRVCHAIN"
    services_to_build["graph"]="KNIRVGRAPH"
    services_to_build["nexus"]="KNIRVNEXUS"
    services_to_build["router"]="KNIRVROUTER"
    services_to_build["oracle"]="KNIRVGATEWAY" # For knirv-gateway image: knirv/oracle
    # services_to_build["oracled"] and services_to_build["gateway"] will be handled by KNIRVGATEWAY as well.

    for service_tag_name in "${!services_to_build[@]}"; do
        local_dir="${services_to_build[$service_tag_name]}"
        image_full_tag="$DOCKER_REGISTRY/$service_tag_name:$VERSION"

        echo "--- Processing $local_dir ---"
        # Temporarily copy KNIRVBASE into the service directory to make it available for Docker build context
        if [ -d "./KNIRVBASE" ] && [ -d "./$local_dir" ]; then
            echo "Copying KNIRVBASE into $local_dir..."
            cp -R ./KNIRVBASE "./$local_dir/KNIRVBASE"
        fi

        echo "Building $image_full_tag from $local_dir (build context: ./$local_dir/)..."
        docker build -t "$image_full_tag" "./$local_dir/"
        echo "Successfully built $image_full_tag."

        # Clean up copied KNIRVBASE
        if [ -d "./$local_dir/KNIRVBASE" ]; then
            echo "Cleaning up copied KNIRVBASE from $local_dir..."
            rm -rf "./$local_dir/KNIRVBASE"
        fi
        echo "--- Finished processing $local_dir ---"
    done

    # Add explicit build for knirv/oracled:latest and knirv/gateway:latest using KNIRVGATEWAY
    # This assumes KNIRVGATEWAY/Dockerfile can build these targets via build arguments or separate Dockerfiles
    # Given the single Dockerfile in KNIRVGATEWAY, it's more likely a single binary gets tagged multiple ways.
    # For now, we will explicitly build knirv/oracled and knirv/gateway using KNIRVGATEWAY context
    
    # For knirv/oracled:latest
    echo "Building knirv/oracled:$VERSION from KNIRVGATEWAY..."
    if [ -d "./KNIRVBASE" ] && [ -d "./KNIRVGATEWAY" ]; then
        echo "Copying KNIRVBASE into KNIRVGATEWAY..."
        cp -R ./KNIRVBASE "./KNIRVGATEWAY/KNIRVBASE"
    fi
    docker build -t "$DOCKER_REGISTRY/oracled:$VERSION" "./KNIRVGATEWAY/"
    echo "Successfully built knirv/oracled:$VERSION."
    if [ -d "./KNIRVGATEWAY/KNIRVBASE" ]; then
        echo "Cleaning up copied KNIRVBASE from KNIRVGATEWAY..."
        rm -rf "./KNIRVGATEWAY/KNIRVBASE"
    fi

    # For knirv/gateway:latest (api-gateway)
    echo "Building knirv/gateway:$VERSION from KNIRVGATEWAY..."
    if [ -d "./KNIRVBASE" ] && [ -d "./KNIRVGATEWAY" ]; then
        echo "Copying KNIRVBASE into KNIRVGATEWAY..."
        cp -R ./KNIRVBASE "./KNIRVGATEWAY/KNIRVBASE"
    fi
    docker build -t "$DOCKER_REGISTRY/gateway:$VERSION" "./KNIRVGATEWAY/"
    echo "Successfully built knirv/gateway:$VERSION."
    if [ -d "./KNIRVGATEWAY/KNIRVBASE" ]; then
        echo "Cleaning up copied KNIRVBASE from KNIRVGATEWAY..."
        rm -rf "./KNIRVGATEWAY/KNIRVBASE"
    fi

    echo "All Docker images built."
}

# Function to start Docker Compose services
start_services() {
    echo "Starting Docker Compose services..."
    run_docker_compose -f "$DOCKER_COMPOSE_FILE" up -d
    echo "Docker Compose services started."
}

# Function to stop Docker Compose services
stop_services() {
    echo "Stopping Docker Compose services..."
    run_docker_compose -f "$DOCKER_COMPOSE_FILE" down
    echo "Docker Compose services stopped."
}

# Trap to ensure services are stopped even if tests fail
trap stop_services EXIT

# Build images
build_images

# Start services
start_services

# Give services some time to initialize (initial bootstrapping, etc.)
echo "Giving services some time to initialize (30 seconds)..."
sleep 30

# Now, execute the Go tests.
# The Go tests themselves will use CheckServiceHealth for specific services.
echo "Running Go integration tests..."
go test ./...

echo "All integration tests completed successfully."