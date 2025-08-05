# KNIRV Network Infrastructure Ansible Playbook

This directory contains a **minimal Ansible playbook** for KNIRV Network infrastructure provisioning and DNS management. Following the recommendations in `docs/Ansible_Deployment_Analysis.md`, this playbook is designed for **infrastructure provisioning only** and hands off to the existing Docker/Kubernetes deployment system.

## 🎯 Purpose

**Scenario A: Infrastructure Provisioning Only**
```yaml
# Use Ansible ONLY for:
- Cloud infrastructure setup (VMs, networks, storage)
- Initial server configuration (users, SSH keys, firewall)
- Docker/Kubernetes installation
- Cloudflare DNS updates for all KNIRV services
# Then hand off to existing Docker deployment
```

## 📁 Directory Structure

```
deployment/ansible/
├── infrastructure-playbook.yml    # Main playbook for infrastructure setup
├── deploy-infrastructure.sh       # Deployment script
├── ansible.cfg                    # Ansible configuration
├── inventory/
│   └── hosts.ini                  # Inventory file (dynamic hosts)
├── group_vars/
│   └── all.yml                    # Global variables
└── environments/
    ├── production.yml             # Production environment config
    ├── development.yml            # Development environment config
    └── staging.yml                # Staging environment config (to be created)
```

## 🚀 Quick Start

### 1. Prerequisites

```bash
# Install Ansible
pip install ansible

# Install required collections
ansible-galaxy collection install amazon.aws
ansible-galaxy collection install community.general
ansible-galaxy collection install community.crypto
```

### 2. Set Environment Variables

```bash
# Cloud Provider Credentials (AWS example)
export AWS_ACCESS_KEY_ID="your-aws-access-key"
export AWS_SECRET_ACCESS_KEY="your-aws-secret-key"

# Cloudflare DNS Management (optional)
export CLOUDFLARE_API_TOKEN="your-cloudflare-api-token"
export CLOUDFLARE_ZONE_ID="your-cloudflare-zone-id"
```

### 3. Deploy Infrastructure

```bash
# Production deployment
./deploy-infrastructure.sh production

# Development deployment
./deploy-infrastructure.sh development --cloud-provider aws

# Dry run (preview changes)
./deploy-infrastructure.sh production --dry-run

# Skip DNS updates
./deploy-infrastructure.sh production --skip-dns
```

## 🔧 Configuration

### Environment Files

Each environment has its own configuration file in `environments/`:

- **`production.yml`**: Production settings with enhanced security and monitoring
- **`development.yml`**: Development settings with cost optimization and relaxed security
- **`staging.yml`**: Staging settings (to be created based on needs)

### Key Configuration Options

```yaml
# Infrastructure
cloud_provider: aws              # aws, gcp, azure, digitalocean
instance_type: t3.medium        # Instance size
region: us-east-1               # Cloud region

# Domain Configuration
domain_name: knirv.com          # Your domain
knirv_subdomains:               # Subdomains for KNIRV services
  - chain                       # chain.knirv.com
  - graph                       # graph.knirv.com
  - nexus                       # nexus.knirv.com
  - root                        # root.knirv.com
  - wallet                      # wallet.knirv.com
  - shell                       # shell.knirv.com
  - router                      # router.knirv.com
  - gateway                     # gateway.knirv.com

# Security
firewall_enabled: true
ssl_config:
  enable_letsencrypt: true
  force_https: true
```

## 📋 What This Playbook Does

### 1. Infrastructure Provisioning
- ✅ Creates VPC, subnets, security groups
- ✅ Launches EC2 instances with proper sizing
- ✅ Configures networking and firewall rules
- ✅ Allocates static IP addresses

### 2. Server Configuration
- ✅ Updates system packages
- ✅ Creates KNIRV system user
- ✅ Configures SSH hardening
- ✅ Sets up UFW firewall rules
- ✅ Installs fail2ban for security

### 3. Container Runtime Setup
- ✅ Installs Docker CE
- ✅ Installs Docker Compose
- ✅ Installs kubectl
- ✅ Sets up k3s (lightweight Kubernetes)
- ✅ Configures container optimization

### 4. DNS Management
- ✅ Updates Cloudflare DNS records
- ✅ Points all KNIRV subdomains to server IP
- ✅ Supports environment-specific subdomains
- ✅ Creates wildcard records for development

### 5. System Optimization
- ✅ Configures kernel parameters
- ✅ Sets up log rotation
- ✅ Creates health check scripts
- ✅ Installs monitoring tools

## 🔗 Integration with Existing Infrastructure

This playbook is designed to work seamlessly with the existing KNIRV deployment system:

### After Infrastructure Deployment

1. **Deploy KNIRV Services** using existing Docker/Kubernetes infrastructure:
   ```bash
   cd /path/to/knirv
   ./deployment/deploy.sh deploy
   ```

2. **Monitor Services** using the installed health check script:
   ```bash
   /opt/knirv/scripts/health-check.sh
   ```

3. **Access Services** at the configured domains:
   - `https://chain.knirv.com:8080`
   - `https://graph.knirv.com:8081`
   - `https://nexus.knirv.com:8082`
   - etc.

## 🌐 DNS Management

The playbook automatically updates Cloudflare DNS records for all KNIRV services:

### Production Subdomains
```
chain.knirv.com     -> SERVER_IP:8080
graph.knirv.com     -> SERVER_IP:8081
nexus.knirv.com     -> SERVER_IP:8082
root.knirv.com      -> SERVER_IP:8083
wallet.knirv.com    -> SERVER_IP:8084
shell.knirv.com     -> SERVER_IP:8085
router.knirv.com    -> SERVER_IP:8086
gateway.knirv.com   -> SERVER_IP:8087
api.knirv.com       -> SERVER_IP:8087
```

### Development Subdomains
```
dev-chain.knirv.com   -> SERVER_IP:8080
dev-graph.knirv.com   -> SERVER_IP:8081
# ... etc
*.knirv.com           -> SERVER_IP (wildcard)
```

## 🔒 Security Features

### Network Security
- UFW firewall with minimal required ports
- Security groups with least privilege access
- fail2ban for SSH protection
- SSH key-only authentication

### SSL/TLS
- Let's Encrypt certificates (production)
- HTTPS enforcement
- HSTS headers
- OCSP stapling

### System Hardening
- Non-root service user
- Disabled root login
- Kernel parameter optimization
- Log rotation and monitoring

## 📊 Monitoring and Health Checks

### Built-in Health Monitoring
```bash
# Check all KNIRV services
/opt/knirv/scripts/health-check.sh

# Example output:
✅ KNIRVchain (8080): HEALTHY
✅ KNIRVgraph (8081): HEALTHY
✅ KNIRVnexus (8082): HEALTHY
❌ KNIRVroot (8083): UNHEALTHY
```

### System Monitoring
- Resource usage monitoring
- Log aggregation
- Service health checks
- Automated alerting (production)

## 🚨 Important Notes

### What This Playbook Does NOT Do
- ❌ Deploy KNIRV applications (use existing Docker/K8s)
- ❌ Manage application configuration
- ❌ Handle application updates
- ❌ Manage application secrets

### Handoff to Existing Infrastructure
After running this playbook, use the existing KNIRV deployment system:
- `deployment/deploy.sh` for application deployment
- `deployment/docker-compose.monitoring.yml` for monitoring
- `deployment/testing/final-test-suite.sh` for testing

## 🔧 Troubleshooting

### Common Issues

1. **AWS Credentials Error**
   ```bash
   export AWS_ACCESS_KEY_ID="your-key"
   export AWS_SECRET_ACCESS_KEY="your-secret"
   ```

2. **Ansible Collection Missing**
   ```bash
   ansible-galaxy collection install amazon.aws
   ```

3. **SSH Connection Issues**
   ```bash
   # Ensure your SSH key is available
   ssh-add ~/.ssh/knirv-network-key.pem
   ```

4. **DNS Updates Failing**
   ```bash
   # Check Cloudflare credentials
   export CLOUDFLARE_API_TOKEN="your-token"
   export CLOUDFLARE_ZONE_ID="your-zone-id"
   ```

### Logs and Debugging
```bash
# Check Ansible logs
tail -f deployment/ansible/ansible.log

# Verbose output
./deploy-infrastructure.sh production --verbose

# Dry run to preview changes
./deploy-infrastructure.sh production --dry-run
```

## 📞 Support

For issues with this infrastructure playbook:
1. Check the logs: `deployment/ansible/ansible.log`
2. Review the Ansible documentation
3. Verify cloud provider credentials
4. Ensure Cloudflare API access

For KNIRV application deployment issues, refer to the main deployment documentation in `deployment/README.md`.
