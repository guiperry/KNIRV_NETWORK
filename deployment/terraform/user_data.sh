#!/bin/bash

# KNIRV Network EC2 Instance Initialization Script
# This script sets up a fresh Ubuntu instance for KNIRV services

set -e

# Variables passed from Terraform
PROJECT_NAME="${project_name}"
ENVIRONMENT="${environment}"
CONTAINER_REGISTRY="${container_registry}"
CONTAINER_IMAGE_TAG="${container_image_tag}"

# Log all output
exec > >(tee /var/log/knirv-init.log)
exec 2>&1

echo "Starting KNIRV Network initialization for $PROJECT_NAME ($ENVIRONMENT)"
echo "Timestamp: $(date)"

# Update system packages
echo "Updating system packages..."
apt-get update -y
apt-get upgrade -y

# Install essential packages
echo "Installing essential packages..."
apt-get install -y \
    curl \
    wget \
    git \
    unzip \
    htop \
    jq \
    python3 \
    python3-pip \
    ca-certificates \
    gnupg \
    lsb-release

# Install Docker
echo "Installing Docker..."
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
apt-get update -y
apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Install Podman
echo "Installing Podman..."
apt-get install -y podman podman-compose

# Configure Docker
echo "Configuring Docker..."
systemctl enable docker
systemctl start docker
usermod -aG docker ubuntu

# Configure Podman
echo "Configuring Podman..."
systemctl enable --now podman.socket
loginctl enable-linger ubuntu

# Install Docker Compose (standalone)
echo "Installing Docker Compose..."
DOCKER_COMPOSE_VERSION="v2.20.0"
curl -L "https://github.com/docker/compose/releases/download/$DOCKER_COMPOSE_VERSION/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose

# Create KNIRV directories
echo "Creating KNIRV directories..."
mkdir -p /opt/knirv/{data,logs,config,scripts}
chown -R ubuntu:ubuntu /opt/knirv

# Create systemd service for KNIRV
echo "Creating KNIRV systemd service..."
cat > /etc/systemd/system/knirv-network.service << EOF
[Unit]
Description=KNIRV Network Services
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/knirv
ExecStart=/opt/knirv/scripts/start-services.sh
ExecStop=/opt/knirv/scripts/stop-services.sh
User=ubuntu
Group=ubuntu

[Install]
WantedBy=multi-user.target
EOF

# Create start services script
echo "Creating service management scripts..."
cat > /opt/knirv/scripts/start-services.sh << 'EOF'
#!/bin/bash
set -e

echo "Starting KNIRV Network services..."

# Pull latest images
docker-compose -f /opt/knirv/docker-compose.yml pull

# Start services
docker-compose -f /opt/knirv/docker-compose.yml up -d

echo "KNIRV Network services started successfully"
EOF

cat > /opt/knirv/scripts/stop-services.sh << 'EOF'
#!/bin/bash
set -e

echo "Stopping KNIRV Network services..."

# Stop services
docker-compose -f /opt/knirv/docker-compose.yml down

echo "KNIRV Network services stopped successfully"
EOF

chmod +x /opt/knirv/scripts/*.sh

# Create basic docker-compose.yml template
echo "Creating Docker Compose configuration..."
cat > /opt/knirv/docker-compose.yml << EOF
version: '3.8'

services:
  knirv-oracle:
    image: $CONTAINER_REGISTRY/$PROJECT_NAME/knirvoracle:$CONTAINER_IMAGE_TAG
    container_name: knirv-oracle
    ports:
      - "1317:1317"
    environment:
      - KNIRV_ENV=$ENVIRONMENT
    volumes:
      - ./data/oracle:/app/data
    restart: unless-stopped

  knirv-chain:
    image: $CONTAINER_REGISTRY/$PROJECT_NAME/knirvchain:$CONTAINER_IMAGE_TAG
    container_name: knirv-chain
    ports:
      - "8090:8090"
    environment:
      - KNIRV_ENV=$ENVIRONMENT
    volumes:
      - ./data/chain:/app/data
    restart: unless-stopped

  knirv-graph:
    image: $CONTAINER_REGISTRY/$PROJECT_NAME/knirvgraph:$CONTAINER_IMAGE_TAG
    container_name: knirv-graph
    ports:
      - "8082:8082"
    environment:
      - KNIRV_ENV=$ENVIRONMENT
    volumes:
      - ./data/graph:/app/data
    restart: unless-stopped

  knirv-nexus:
    image: $CONTAINER_REGISTRY/$PROJECT_NAME/knirvnexus:$CONTAINER_IMAGE_TAG
    container_name: knirv-nexus
    ports:
      - "8084:8084"
    environment:
      - KNIRV_ENV=$ENVIRONMENT
    volumes:
      - ./data/nexus:/app/data
    restart: unless-stopped

  knirv-router:
    image: $CONTAINER_REGISTRY/$PROJECT_NAME/knirvrouter:$CONTAINER_IMAGE_TAG
    container_name: knirv-router
    ports:
      - "8086:8086"
    environment:
      - KNIRV_ENV=$ENVIRONMENT
    volumes:
      - ./data/router:/app/data
    restart: unless-stopped

  knirv-controller:
    image: $CONTAINER_REGISTRY/$PROJECT_NAME/knirvcontroller:$CONTAINER_IMAGE_TAG
    container_name: knirv-controller
    ports:
      - "3000:3000"
    environment:
      - KNIRV_ENV=$ENVIRONMENT
    volumes:
      - ./data/controller:/app/data
    restart: unless-stopped

  knirv-gateway:
    image: $CONTAINER_REGISTRY/$PROJECT_NAME/knirvgateway:$CONTAINER_IMAGE_TAG
    container_name: knirv-gateway
    ports:
      - "8888:80"
    environment:
      - KNIRV_ENV=$ENVIRONMENT
    volumes:
      - ./data/gateway:/app/data
    restart: unless-stopped

networks:
  default:
    name: knirv-network
EOF

chown ubuntu:ubuntu /opt/knirv/docker-compose.yml

# Enable and start the service
systemctl daemon-reload
systemctl enable knirv-network.service

# Install CloudWatch agent (optional)
if [ "$ENVIRONMENT" = "production" ]; then
    echo "Installing CloudWatch agent..."
    wget https://s3.amazonaws.com/amazoncloudwatch-agent/ubuntu/amd64/latest/amazon-cloudwatch-agent.deb
    dpkg -i amazon-cloudwatch-agent.deb
fi

# Create health check script
cat > /opt/knirv/scripts/health-check.sh << 'EOF'
#!/bin/bash
# Health check script for KNIRV services

services=("oracle:1317" "chain:8090" "graph:8082" "nexus:8084" "router:8086" "controller:3000" "gateway:8888")

for service in "${services[@]}"; do
    name=$(echo $service | cut -d: -f1)
    port=$(echo $service | cut -d: -f2)
    
    if curl -f -s http://localhost:$port/health > /dev/null 2>&1; then
        echo "✓ $name service is healthy"
    else
        echo "✗ $name service is not responding"
    fi
done
EOF

chmod +x /opt/knirv/scripts/health-check.sh

echo "KNIRV Network initialization completed successfully!"
echo "Services will be available after container images are pulled and started."
echo "Use 'sudo systemctl start knirv-network' to start services manually."
echo "Use '/opt/knirv/scripts/health-check.sh' to check service health."
