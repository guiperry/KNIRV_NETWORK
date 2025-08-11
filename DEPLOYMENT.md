# KNIRV Network Deployment Guide

This guide covers all deployment options for the KNIRV Network, including the new KNIRVTESTNET AWS deployment with Netlify frontend integration.

## 🧪 KNIRVTESTNET Deployment (Recommended)

Deploy a complete testnet environment on AWS with Netlify frontend integration.

### Prerequisites

1. **AWS Account** with appropriate permissions
2. **Cloudflare Account** for DNS management
3. **Netlify Account** for frontend hosting
4. **SSH Key Pair** for EC2 access

### Quick Start

```bash
# 1. Configure environment
cp deployment/ansible/.env.example deployment/ansible/.env
# Edit .env with your credentials

# 2. Deploy complete testnet
make deploy-testnet

# 3. Access testnet
open https://knirv.com/testnet
```

### Step-by-Step Deployment

#### 1. Environment Setup

```bash
# Copy environment template
cp deployment/ansible/.env.example deployment/ansible/.env

# Edit configuration
vim deployment/ansible/.env
```

Required environment variables:
- `AWS_ACCESS_KEY_ID` - AWS access key
- `AWS_SECRET_ACCESS_KEY` - AWS secret key
- `CLOUDFLARE_API_TOKEN` - Cloudflare API token
- `CLOUDFLARE_ZONE_ID` - Cloudflare zone ID
- `DOMAIN_NAME` - Your domain (default: knirv.com)

#### 2. AWS Key Pair Setup

```bash
# Create AWS key pair
aws ec2 create-key-pair \
  --key-name knirv-testnet-key \
  --query 'KeyMaterial' \
  --output text > ~/.ssh/knirv-testnet-key.pem

# Set permissions
chmod 400 ~/.ssh/knirv-testnet-key.pem
```

#### 3. Deploy Infrastructure

```bash
# Deploy AWS infrastructure only
make deploy-testnet-infrastructure

# This creates:
# - EC2 instance (t3.large)
# - VPC and security groups
# - Elastic IP
# - Cloudflare DNS records
```

#### 4. Deploy Services

```bash
# Deploy KNIRV services to EC2
make deploy-testnet-services

# This deploys:
# - IPFS node
# - KNIRVROOT (Cosmos SDK)
# - KNIRVCHAIN (Smart contracts)
# - KNIRVGRAPH (Graph storage)
# - KNIRVNEXUS (TEE simulation)
# - KNIRVROUTER (Network routing)
# - KNIRVGATEWAY (API gateway)
```

#### 5. Update Frontend

```bash
# Update Netlify frontend integration
make update-testnet-frontend

# This updates:
# - KNIRVGATEWAY/knirvtestnet/ directory
# - Netlify configuration
# - API proxy endpoints
# - Real-time dashboard
```

### Service Endpoints

Once deployed, the following endpoints are available:

| Service | Port | Endpoint | Description |
|---------|------|----------|-------------|
| KNIRVROOT | 1317 | `http://testnet-ip:1317` | Core blockchain API |
| KNIRVCHAIN | 8080 | `http://testnet-ip:8080` | Smart contracts |
| KNIRVGRAPH | 8081 | `http://testnet-ip:8081` | Graph storage |
| KNIRVNEXUS-1 | 8082 | `http://testnet-ip:8082` | TEE simulation |
| KNIRVNEXUS-2 | 8083 | `http://testnet-ip:8083` | TEE simulation |
| KNIRVROUTER | 8086 | `http://testnet-ip:8086` | Network routing |
| KNIRVGATEWAY | 8087 | `http://testnet-ip:8087` | API gateway |
| IPFS API | 5001 | `http://testnet-ip:5001` | IPFS API |
| IPFS Gateway | 8080 | `http://testnet-ip:8080` | IPFS Gateway |

### Frontend Access

- **Main Dashboard**: https://knirv.com/testnet
- **Direct API Access**: `/api/testnet/{service}/*`
- **Health Monitoring**: Real-time service status
- **Documentation**: Integrated testing guides

### Monitoring and Management

```bash
# SSH into testnet
ssh knirv-testnet

# Check service status
docker ps

# View logs
docker-compose -f /opt/knirv-testnet/docker-compose-prod.yml logs

# Restart services
docker-compose -f /opt/knirv-testnet/docker-compose-prod.yml restart

# Update services
make deploy-testnet-services
```

### Troubleshooting

#### Common Issues

1. **SSH Connection Failed**
   ```bash
   # Check security group settings
   # Ensure port 22 is open
   # Verify key pair permissions
   chmod 400 ~/.ssh/knirv-testnet-key.pem
   ```

2. **Services Not Starting**
   ```bash
   # Check Docker status
   ssh knirv-testnet 'sudo systemctl status docker'
   
   # Check logs
   ssh knirv-testnet 'docker-compose logs'
   ```

3. **DNS Not Resolving**
   ```bash
   # Check Cloudflare DNS records
   # Verify API token permissions
   # Wait for DNS propagation (up to 24 hours)
   ```

#### Cleanup

```bash
# Remove testnet infrastructure
cd deployment/ansible
ansible-playbook testnet-cleanup-playbook.yml
```

## 🏠 Local Development

For local development and testing:

```bash
# Start local testnet
cd KNIRVTESTNET
./start-testnet.sh

# Access local services
open http://localhost:8087
```

## 🐳 Docker Compose Deployment

For containerized local deployment:

```bash
# Deploy with monitoring
./scripts/deploy-and-test.sh --mode docker-compose --comprehensive

# Check status
./scripts/deploy-and-test.sh --status
```

## ☸️ Kubernetes Deployment

For production Kubernetes deployment:

```bash
# Deploy to Kubernetes
./scripts/deploy-and-test.sh --mode kubernetes --env production

# Monitor deployment
kubectl get pods -n knirv-production
```

## 📊 Cost Estimation

### AWS Costs (Monthly)

- **EC2 t3.large**: ~$60/month
- **Elastic IP**: ~$3.65/month
- **EBS Storage (100GB)**: ~$10/month
- **Data Transfer**: Variable
- **Total**: ~$75-100/month

### Optimization Tips

1. Use **Spot Instances** for development (50-90% savings)
2. **Schedule shutdown** during non-development hours
3. Use **Reserved Instances** for production (up to 75% savings)
4. Monitor usage with **AWS Cost Explorer**

## 🔒 Security Considerations

1. **Firewall Configuration**: Only necessary ports open
2. **SSH Key Management**: Secure key storage and rotation
3. **API Authentication**: Secure API access tokens
4. **Regular Updates**: Keep systems and dependencies updated
5. **Monitoring**: Set up alerts for suspicious activity

## 📚 Additional Resources

- [KNIRVTESTNET README](KNIRVTESTNET/README.md)
- [KNIRVGATEWAY README](KNIRVGATEWAY/README.md)
- [Ansible Playbooks](deployment/ansible/)
- [Environment Configuration](deployment/ansible/.env.example)

## 🆘 Support

For deployment issues:

1. Check the [troubleshooting section](#troubleshooting)
2. Review service logs
3. Verify environment configuration
4. Check AWS and Cloudflare status pages

## 🔄 Updates and Maintenance

### Regular Maintenance

```bash
# Update testnet services
make deploy-testnet-services

# Update frontend
make update-testnet-frontend

# Full redeployment
make deploy-testnet
```

### Backup and Recovery

```bash
# Backup testnet data
ssh knirv-testnet 'sudo tar -czf /tmp/knirv-backup.tar.gz /opt/knirv-testnet/data'

# Download backup
scp knirv-testnet:/tmp/knirv-backup.tar.gz ./backups/
```

This deployment guide provides comprehensive coverage of all KNIRV Network deployment options, with special focus on the new KNIRVTESTNET AWS deployment with Netlify frontend integration.
