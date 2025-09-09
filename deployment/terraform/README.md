# KNIRV Network Terraform Infrastructure and Secure Deployment Guide

[TOC]

## Overview

This document provides a comprehensive guide for deploying and securely managing the KNIRV Network infrastructure using Terraform and AWS Secrets Manager.  The Terraform configuration creates a robust and scalable infrastructure on AWS, including VPC, EC2 instances, security groups, load balancing, and optional Cloudflare DNS integration.  Sensitive credentials are managed securely using AWS Secrets Manager with KMS encryption and IAM role-based access.

## Features

### Infrastructure Deployment (Terraform)

- **VPC:** VPC with public and private subnets across multiple availability zones.
- **EC2 Instances:** Auto-scaling EC2 instances running Ubuntu 22.04 LTS.
- **Security Groups:**  Security groups with minimal required access (SSH, HTTP/HTTPS, KNIRV service ports: 1317, 3000, 8080-8090, etc.).
- **Load Balancing:** Application Load Balancer (production only).
- **DNS:** Optional Cloudflare DNS integration for automated record management.
- **Monitoring & Logging:** CloudWatch monitoring and logging.
- **Backup:** Optional S3 backup storage with lifecycle policies.
- **Database:** Optional RDS instance (enabled for staging and production environments).


### Secure Credential Management (AWS Secrets Manager)

- **Centralized Storage:** All deployment credentials stored securely in AWS Secrets Manager.
- **KMS Encryption:** Secrets encrypted with customer-managed KMS keys.
- **IAM Role-Based Access:** Least-privilege access through IAM roles and policies.
- **Multi-Environment Support:** Separate secret stores for dev/staging/production environments.
- **Supported Credentials:** Cloudflare API Token, AWS Access Keys, GitHub Tokens, Container Registry credentials, SSL/TLS Certificates, Database Encryption Keys, JWT Secrets, OpenAI, Anthropic, Cerebras, DeepSeek, Google API keys, Sentry DSN, Slack webhook URLs, WalletConnect/Infura project IDs, Google Analytics/Tag Manager IDs.


## Prerequisites

1. **AWS CLI:** Configured with appropriate credentials (admin access recommended for initial setup).
2. **Terraform:** Installed (version >= 1.0).
3. **SSH Key Pair:** For EC2 access.
4. **Cloudflare API Token:** (Optional, for DNS management).


## Configuration

### Terraform Configuration Files

- `main.tf`: Main infrastructure definitions.
- `variables.tf`: Input variable definitions.
- `outputs.tf`: Output value definitions.
- `user_data.sh`: EC2 initialization script.
- `terraform.tfvars.example`: Example configuration file.


### Environment-Specific Configurations

| Environment | `environment` | `instance_type` | `min_instances` | `max_instances` | `desired_instances` | `enable_rds` | `db_instance_class` |
|---|---|---|---|---|---|---|---|
| Development | "dev" | "t3.small" | 1 | 1 | 1 | false |  |
| Staging | "staging" | "t3.medium" | 1 | 1 | 1 | true | "db.t3.micro" |
| Production | "production" | "t3.large" | 2 | 5 | 2 | true | "db.t3.small" |


### `terraform.tfvars` Configuration

```hcl
# Core Configuration
project_name = "knirv-network"
environment  = "production"  # or "dev", "staging"

# Cloudflare (Required for DNS)
cloudflare_api_token = "your-cloudflare-api-token"
cloudflare_zone_id   = "your-zone-id"

# AWS Deployment (Optional - uses current AWS CLI credentials if not provided)
aws_access_key_id     = "AKIA..."
aws_secret_access_key = "your-secret-key"

# GitHub (Required for CI/CD)
github_token      = "ghp_your-github-token"
github_deploy_key = "ssh-rsa AAAAB3..."

# Security Keys (Generate secure random values)
database_encryption_key = "base64-encoded-32-byte-key"
jwt_secret             = "secure-random-jwt-secret"
deployment_external_id = "unique-external-id-for-role-assumption"

# API Keys (Optional - only include if using these services)
openai_api_key    = "sk-..."
anthropic_api_key = "sk-ant-..."
cerebras_api_key  = "csk-..."
```


## Usage

### Quick Start (Terraform Deployment)

1. Copy the example configuration: `cp terraform.tfvars.example terraform.tfvars`
2. Edit `terraform.tfvars` with your specific values (including AWS region and SSH key path).
3. Initialize Terraform: `terraform init`
4. Plan the deployment: `terraform plan`
5. Apply the configuration: `terraform apply`


### Using Secrets in Applications

#### EC2 Instance Access

```bash
aws secretsmanager get-secret-value \
  --secret-id $(terraform output -raw deployment_secrets_arn) \
  --query SecretString --output text | jq .
```

#### Application Configuration Examples

**Go:**

```go
import "github.com/aws/aws-sdk-go/service/secretsmanager"
// ... (rest of the Go code)
```

**Node.js:**

```javascript
const AWS = require('aws-sdk');
// ... (rest of the Node.js code)
```

#### GitHub Actions Integration

```yaml
# .github/workflows/deploy.yml
env:
  AWS_REGION: us-east-1
  DEPLOYMENT_ROLE_ARN: ${{ secrets.DEPLOYMENT_ROLE_ARN }}
// ... (rest of the GitHub Actions workflow)
```


## Post-Deployment Steps

1. SSH to instances: `ssh -i ~/.ssh/your-key.pem ubuntu@<instance-ip>`
2. Check service status: `sudo systemctl status knirv-network`, `/opt/knirv/scripts/health-check.sh`
3. Start services manually (if needed): `sudo systemctl start knirv-network`
4. View logs: `sudo journalctl -u knirv-network -f`, `tail -f /var/log/knirv-init.log`


## Customization

### Adding New Services

1. Add the port to `knirv_service_ports` in `variables.tf`.
2. Update the security group ingress rules.
3. Add the service to the docker-compose template in `user_data.sh`.

### Scaling

- **Vertical Scaling:** Adjust `instance_type`.
- **Horizontal Scaling:** Adjust `min_instances`, `max_instances`, `desired_instances`.

### Security Hardening

- Restrict `allowed_ssh_cidrs`.
- Enable VPC Flow Logs.
- Configure AWS WAF for the load balancer.
- Enable GuardDuty for threat detection.


## Troubleshooting

### Common Issues

1. **SSH key not found:** Ensure `ssh_public_key_path` is correct and has readable permissions.
2. **Services not starting:** Check `/var/log/knirv-init.log` for errors, container image accessibility, and Docker/Podman service status.
3. **DNS not resolving:** Verify Cloudflare API token permissions, zone ID, and domain name.


### Useful Commands

```bash
terraform show
terraform refresh
terraform destroy
terraform fmt
terraform validate
```


## Security Considerations

- **Never commit `terraform.tfvars`** with sensitive data.
- Use AWS Secrets Manager or Parameter Store for sensitive values.
- Enable MFA for AWS accounts.
- Regularly rotate access keys.
- Monitor CloudTrail logs.


## Key Rotation (AWS Secrets Manager)

```bash
aws secretsmanager rotate-secret \
  --secret-id knirv-network-production-deployment-keys \
  --rotation-lambda-arn arn:aws:lambda:region:account:function:rotation-function
```

## Access Monitoring (AWS Secrets Manager)

```bash
aws logs filter-log-events \
  --log-group-name /aws/secretsmanager/knirv-network-production-deployment-keys
```

## Emergency Procedures

### Credential Compromise

1. Rotate compromised credentials using `aws secretsmanager update-secret`.
2. Revoke old credentials at the provider (Cloudflare, GitHub, etc.).

### Access Revocation

```bash
aws iam detach-role-policy \
  --role-name knirv-network-production-deployment-role \
  --policy-arn arn:aws:iam::account:policy/compromised-policy
```

## Monitoring & Compliance

- CloudTrail integration for audit logging.
- Monitor costs in AWS Cost Explorer.
- SOC 2, GDPR, and HIPAA compliance considerations.


## Support

- **Infrastructure:** This README and Terraform documentation.
- **KNIRV Services:** Individual component documentation.
- **AWS Resources:** AWS documentation.


## License

This infrastructure code is part of the KNIRV Network project.
