# KNIRV Network Infrastructure and KNIRVORACLE Deployment

[TOC]

## Overview

This document provides comprehensive instructions for deploying the KNIRV network infrastructure and integrating KNIRVORACLE.  The Ansible playbook automates infrastructure provisioning, server configuration, container runtime setup, and DNS management.  KNIRVORACLE is seamlessly integrated, providing crucial oracle services within the KNIRV ecosystem.

## Features

### KNIRV Network Infrastructure

* **Infrastructure Provisioning:** Creates VPCs, subnets, security groups, launches EC2 instances, configures networking and firewall rules, and allocates static IP addresses.
* **Server Configuration:** Updates system packages, creates KNIRV system users, configures SSH hardening, sets up UFW firewall rules, and installs fail2ban.
* **Container Runtime Setup:** Installs Docker CE, Docker Compose, kubectl, and k3s (lightweight Kubernetes), and configures container optimization.
* **DNS Management:** Updates Cloudflare DNS records, points all KNIRV subdomains to server IPs, supports environment-specific subdomains, and creates wildcard records for development.
* **System Optimization:** Configures kernel parameters, sets up log rotation, creates health check scripts, and installs monitoring tools.
* **Security Features:** UFW firewall, security groups, fail2ban, SSH key-only authentication, Let's Encrypt certificates (production), HTTPS enforcement, HSTS headers, OCSP stapling, non-root service user, disabled root login, kernel parameter optimization, and log rotation and monitoring.
* **Monitoring and Health Checks:** Built-in health monitoring script (`/opt/knirv/scripts/health-check.sh`), resource usage monitoring, log aggregation, service health checks, and automated alerting (production).


### KNIRVORACLE Integration

* **Embedded Services:** Main Oracle Service (port 1317), Bootnode Registry (port 3006), Tunnel Registry (port 3003), Notary System (port 3007), Network Monitor (port 3008), and NANDA-ANS (port 3009).
* **Deployment Playbook:** `knirvoracle-deployment.yml`
* **Standalone Deployment Script:** `deploy-knirvoracle.sh`
* **CloudFlare DNS Integration:** Automatic DNS updates for all KNIRVORACLE services.
* **Infrastructure Integration:** Seamless integration with the main `infrastructure-playbook.yml`.
* **Templates:** Configuration templates for systemd, monitoring, and environment setup.
* **Health Monitoring:** Automated health monitoring script (`/opt/knirv/knirvoracle/scripts/monitor.sh`), cron job, log rotation, and systemd integration.
* **Management Commands:** `deploy-knirvoracle.sh` provides commands for status, logs, restart, and verification.
* **Security:** UFW firewall rules, user isolation, file permissions, optional API key authentication, and configurable rate limits.


## Directory Structure

```
deployment/ansible/
├── infrastructure-playbook.yml
├── deploy-infrastructure.sh
├── ansible.cfg
├���─ inventory/
│   └── hosts.ini
├── group_vars/
│   └── all.yml
├── environments/
│   ├── production.yml
│   ├── development.yml
│   └── staging.yml
├── knirvoracle-deployment.yml
├── deploy-knirvoracle.sh
├── templates/
│   ├── knirvoracle-config.toml.j2
│   ├── knirvoracle.service.j2
│   ├── knirvoracle.env.j2
│   └── knirvoracle-monitor.sh.j2
└── KNIRVORACLE-DEPLOYMENT.md
```

## Usage

### Prerequisites

```bash
pip install ansible
ansible-galaxy collection install amazon.aws community.general community.crypto
```

### Environment Variables

```bash
export AWS_ACCESS_KEY_ID="your-aws-access-key"
export AWS_SECRET_ACCESS_KEY="your-aws-secret-key"
export CLOUDFLARE_API_TOKEN="your-cloudflare-api-token"
export CLOUDFLARE_ZONE_ID="your-cloudflare-zone-id"
export KNIRVORACLE_API_KEY="your-api-key" # Optional
export ANSIBLE_VAULT_PASSWORD="vault-password" # Optional
```

### Deploying Infrastructure (with KNIRVORACLE)

```bash
cd deployment/ansible
./deploy-infrastructure.sh production # or development
```

### Deploying KNIRVORACLE Standalone

```bash
cd deployment/ansible
./deploy-knirvoracle.sh deploy --env production # or other environments
```

### Dry Run and Skipping DNS

```bash
./deploy-infrastructure.sh production --dry-run
./deploy-infrastructure.sh production --skip-dns
./deploy-knirvoracle.sh deploy --env production --dry-run
./deploy-knirvoracle.sh deploy --env production --skip-dns
```

### Updating DNS Only

```bash
./deploy-knirvoracle.sh dns-only --env production
```

### Post-Deployment Steps

1. **Deploy KNIRV Services:** `cd /path/to/knirv; ./deployment/deploy.sh deploy`
2. **Monitor Services:** `/opt/knirv/scripts/health-check.sh`
3. **Access Services:**  `https://chain.knirv.com:8080`, `https://graph.knirv.com:8081`, etc.


## Configuration

### Environment Files

* `environments/production.yml`
* `environments/development.yml`
* `environments/staging.yml`


### Key Configuration Options

```yaml
cloud_provider: aws
instance_type: t3.medium
region: us-east-1
domain_name: knirv.com
knirv_subdomains:
  - chain
  - graph
  - nexus
  - root
  - wallet
  - shell
  - router
  - gateway
firewall_enabled: true
ssl_config:
  enable_letsencrypt: true
  force_https: true
```

## KNIRVORACLE Services and DNS

| Service                 | Port | DNS Name                               |
|--------------------------|------|----------------------------------------|
| Main Oracle Service      | 1317 | oracle.knirv.com                      |
| Bootnode Registry        | 3006 | bootnode-registry.knirv.com           |
| Tunnel Registry          | 3003 | tunnel-registry.knirv.com             |
| Notary System            | 3007 | notary-system.knirv.com               |
| Network Monitor          | 3008 | network-monitor.knirv.com             |
| NANDA-ANS               | 3009 | nanda-ans.knirv.com                   |


## Troubleshooting

### Common Issues

1. **AWS Credentials Error:** Ensure `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` are correctly set.
2. **Ansible Collection Missing:** Run `ansible-galaxy collection install <collection_name>`.
3. **SSH Connection Issues:** Ensure your SSH key is available (`ssh-add ~/.ssh/knirv-network-key.pem`).
4. **DNS Updates Failing:** Check `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ZONE_ID`.
5. **KNIRVORACLE Build Failures:** Run `./deploy-knirvoracle.sh build`.
6. **KNIRVORACLE Service Won't Start:** Check logs (`sudo journalctl -u knirvoracle -n 50`) and configuration.
7. **Port Conflicts:** Check port usage (`sudo netstat -tlnp`).


### Logs and Debugging

* Ansible logs: `deployment/ansible/ansible.log`
* KNIRVORACLE logs: `/opt/knirv/knirvoracle/logs/knirvoracle.log`, `/opt/knirv/knirvoracle/logs/health-check.log`, `/opt/knirv/knirvoracle/logs/alerts.log`
* Verbose output: Add `--verbose` flag to deployment scripts.
* Dry run: Add `--dry-run` flag to deployment scripts.


## Support

For issues, check logs, review Ansible and Cloudflare documentation, and verify credentials.  For KNIRV application deployment issues, refer to `deployment/README.md`.
