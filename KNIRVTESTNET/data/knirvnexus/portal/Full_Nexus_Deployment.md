# Full NEXUS Deployment Analysis - AWS EC2 Native Approach

## Executive Summary

This document analyzes the implementation strategy for deploying the unified KNIRV-NEXUS application natively on AWS EC2 instances running hardened Kali Linux. Based on the CLEAN whitepaper requirements and the decision to eliminate containerization, we focus on native deployment automation using Ansible with CloudFlare DNS integration.

## Deployment Requirements Analysis

### Core Requirements from CLEAN Whitepaper
1. **Kali Linux Foundation**: Hardened, forked Kali Linux 2024.3 distribution
2. **TEE Support**: Intel SGX, AMD SEV-SNP, Intel TDX compatibility on AWS
3. **Single Binary Deployment**: Unified Go application with embedded frontend and BuntDB
4. **Native Process Management**: Direct process control without containerization
5. **P2P Networking**: libp2p-based decentralized communication across cloud infrastructure
6. **Security Toolchain**: Continuous self-auditing and threat monitoring

### Technical Constraints
- **AWS Instance Requirements**: TEE-capable EC2 instance types (M5, C5, R5 series)
- **Kernel Requirements**: Linux 6.6.x LTS with TEE modules
- **Database**: Embedded BuntDB for zero external dependencies
- **Network Requirements**: P2P mesh networking with AWS security groups
- **Storage Requirements**: EBS encryption for persistent data

## AWS EC2 Native Deployment Architecture

### Primary Approach: Native Kali Linux on EC2
**Description**: Direct deployment of unified NEXUS binary on hardened Kali Linux EC2 instances

**Advantages**:
- Zero containerization overhead
- Direct hardware access for TEE functionality
- Simplified security model
- Native process management
- Reduced attack surface
- Better performance for TEE operations

**Implementation Strategy**:
```yaml
# AWS Infrastructure Components
EC2 Instance:
  - Type: m5.xlarge (Intel SGX support) or c5.xlarge (general TEE)
  - AMI: Custom Kali Linux 2024.3 hardened image
  - Storage: Encrypted EBS volumes
  - Network: VPC with custom security groups

Security Groups:
  - SSH (22): Restricted to management IPs
  - NEXUS API (8080): Public access
  - Agent Server (8082): Internal network only
  - Data Engine (9000): Internal network only
  - P2P (4001): Public for mesh networking

Elastic IP:
  - Static IP allocation for consistent DNS
  - Automatic CloudFlare DNS updates
```

**Feasibility**: ✅ **HIGH** - Optimal for TEE and performance requirements

## Embedded BuntDB Integration


### Embedded Database Advantages
**Description**: BuntDB as the primary data storage solution

**Benefits**:
- **Zero External Dependencies**: No database server setup or management
- **High Performance**: In-memory operations with disk persistence
- **ACID Transactions**: Reliable data consistency
- **JSON Support**: Native JSON document storage and querying
- **Indexing**: Fast lookups with custom indexes
- **Small Footprint**: Minimal resource usage
- **Backup Simplicity**: Single file backup and restore

**Implementation**:
```go
// BuntDB configuration for NEXUS
type NexusDB struct {
    db *buntdb.DB
    path string
}

func NewNexusDB(path string) (*NexusDB, error) {
    db, err := buntdb.Open(path)
    if err != nil {
        return nil, err
    }

    // Create indexes for efficient querying
    db.CreateIndex("metrics_time", "metrics:*", buntdb.IndexJSON("timestamp"))
    db.CreateIndex("agents_status", "agents:*", buntdb.IndexJSON("status"))
    db.CreateIndex("validations_type", "validations:*", buntdb.IndexJSON("type"))

    return &NexusDB{db: db, path: path}, nil
}

// Data collections
const (
    MetricsPrefix     = "metrics:"
    AgentsPrefix      = "agents:"
    ValidationsPrefix = "validations:"
    ConfigPrefix      = "config:"
    AlertsPrefix      = "alerts:"
)
```

**Feasibility**: ✅ **HIGH** - Perfect fit for embedded deployment

## TEE Integration on AWS EC2

### AWS Instance Types with TEE Support
**Intel SGX Support**:
- **M5 Series**: m5.large, m5.xlarge, m5.2xlarge (Intel Xeon Platinum 8175M)
- **C5 Series**: c5.large, c5.xlarge, c5.2xlarge (Intel Xeon Platinum 8124M)
- **R5 Series**: r5.large, r5.xlarge, r5.2xlarge (Intel Xeon Platinum 8175M)

**AMD SEV-SNP Support**:
- **M5a Series**: m5a.large, m5a.xlarge, m5a.2xlarge (AMD EPYC 7571)
- **C5a Series**: c5a.large, c5a.xlarge, c5a.2xlarge (AMD EPYC 7R32)
- **R5a Series**: r5a.large, r5a.xlarge, r5a.2xlarge (AMD EPYC 7571)

### Native TEE Setup on EC2
**Advantages of Native Deployment**:
1. **Direct Hardware Access**: No virtualization layer interference
2. **Full Privilege Access**: Complete control over TEE devices
3. **Optimal Performance**: No container overhead
4. **Simplified Attestation**: Direct access to TPM and TEE hardware

**Implementation**:
```bash
#!/bin/bash
# setup-tee.sh - Native TEE setup on Kali Linux EC2

# Detect available TEE technologies
detect_tee_support() {
    echo "Detecting TEE support..."

    # Check for Intel SGX
    if lscpu | grep -q "sgx"; then
        echo "Intel SGX detected"
        setup_sgx
    fi

    # Check for AMD SEV
    if lscpu | grep -q "sev"; then
        echo "AMD SEV detected"
        setup_sev
    fi

    # Check for Intel TDX
    if lscpu | grep -q "tdx"; then
        echo "Intel TDX detected"
        setup_tdx
    fi
}

setup_sgx() {
    # Install SGX SDK and PSW
    apt-get update
    apt-get install -y intel-sgx-sdk intel-sgx-psw intel-sgx-dcap-ql

    # Load SGX driver
    modprobe intel_sgx
    echo 'intel_sgx' >> /etc/modules

    # Set permissions
    chmod 666 /dev/sgx_enclave /dev/sgx_provision
}

setup_sev() {
    # Install SEV tools
    apt-get install -y amd-sev-tools

    # Load SEV modules
    modprobe ccp
    modprobe sev-guest
    echo 'ccp' >> /etc/modules
    echo 'sev-guest' >> /etc/modules
}
```

## AWS Network Architecture

### P2P Networking on EC2
**Advantages of Native Deployment**:
- Direct access to network interfaces
- No NAT traversal issues within VPC
- Simplified port management
- Better performance for P2P protocols

**Network Configuration**:
```yaml
# AWS VPC Configuration
VPC:
  CIDR: 10.0.0.0/16

Subnets:
  Public:
    CIDR: 10.0.1.0/24
    Purpose: NEXUS instances with public access
  Private:
    CIDR: 10.0.2.0/24
    Purpose: Internal services and databases

Security Groups:
  nexus-public:
    Ingress:
      - Port: 22 (SSH) - Source: Management IPs only
      - Port: 8080 (NEXUS API) - Source: 0.0.0.0/0
      - Port: 4001 (P2P) - Source: 0.0.0.0/0
    Egress:
      - All traffic - Destination: 0.0.0.0/0

  nexus-internal:
    Ingress:
      - Port: 8082 (Agent Server) - Source: nexus-public
      - Port: 9000 (Data Engine) - Source: nexus-public
    Egress:
      - All traffic - Destination: 0.0.0.0/0
```

### CloudFlare DNS Integration
**Automatic DNS Updates**:
```yaml
# CloudFlare DNS automation
- name: Update DNS record with instance IP
  cloudflare_dns:
    zone: "{{ domain_name }}"
    record: "{{ nexus_subdomain }}"
    type: A
    value: "{{ ansible_default_ipv4.address }}"
    account_email: "{{ cloudflare_email }}"
    account_api_token: "{{ cloudflare_api_token }}"
    ttl: 300
  delegate_to: localhost
```

### Security Considerations
1. **VPC Isolation**: Private subnets for sensitive services
2. **Security Groups**: Restrictive ingress rules
3. **NACLs**: Additional network-level security
4. **TLS Encryption**: All inter-service communication encrypted
5. **SSH Hardening**: Key-based authentication only

## Storage and Data Management

### EBS Storage Strategy
```yaml
# EBS Volume Configuration
EBS Volumes:
  Root Volume:
    Size: 20GB
    Type: gp3
    Encrypted: true
    Purpose: OS and application binaries

  Data Volume:
    Size: 100GB
    Type: gp3
    Encrypted: true
    Purpose: BuntDB files and logs
    Mount: /var/lib/knirv-nexus

  Backup Volume:
    Size: 50GB
    Type: gp3
    Encrypted: true
    Purpose: Automated backups
    Mount: /backup
```

### BuntDB File Management
```bash
# BuntDB file structure
/var/lib/knirv-nexus/
├── db/
│   ├── nexus.db          # Main BuntDB file
│   ├── nexus.db.backup   # Automatic backup
│   └── nexus.db.wal      # Write-ahead log
├── logs/
│   ├── nexus.log         # Application logs
│   ├── audit.log         # Security audit logs
│   └── performance.log   # Performance metrics
└── config/
    ├── nexus.yaml        # Main configuration
    └── tee.conf          # TEE configuration
```

### Encryption and Security
- **EBS Encryption**: AWS KMS encryption for all volumes
- **Data in Transit**: TLS 1.3 for all network communication
- **TEE Data**: Hardware-encrypted memory regions
- **BuntDB**: Built-in encryption for sensitive data
- **Backup Encryption**: GPG encryption for backup files

## Ansible Deployment Automation

### Complete Deployment Pipeline
```bash
#!/bin/bash
# deploy.sh - AWS EC2 deployment automation

set -e

# Check prerequisites
check_prerequisites() {
    echo "Checking deployment requirements..."

    # Check Ansible
    if ! command -v ansible-playbook &> /dev/null; then
        echo "Error: Ansible not found"
        exit 1
    fi

    # Check AWS CLI
    if ! command -v aws &> /dev/null; then
        echo "Error: AWS CLI not found"
        exit 1
    fi

    # Check CloudFlare credentials
    if [[ -z "$CLOUDFLARE_API_TOKEN" ]]; then
        echo "Error: CLOUDFLARE_API_TOKEN not set"
        exit 1
    fi

    # Verify binary exists
    if [[ ! -f "backend/bin/knirv-nexus" ]]; then
        echo "Error: NEXUS binary not found. Run 'make build' first."
        exit 1
    fi
}

# Deploy to AWS
deploy_aws() {
    echo "Deploying KNIRV-NEXUS to AWS EC2..."

    # Setup AWS infrastructure
    echo "Setting up AWS infrastructure..."
    cd deployment
    ansible-playbook -i inventory/aws.yml playbooks/setup-infrastructure.yml

    # Deploy NEXUS application
    echo "Deploying NEXUS application..."
    ansible-playbook -i inventory/aws.yml playbooks/deploy-nexus.yml \
        --extra-vars "nexus_binary_path=../backend/bin/knirv-nexus"

    # Update DNS records
    echo "Updating CloudFlare DNS..."
    ansible-playbook -i inventory/aws.yml playbooks/update-dns.yml

    echo "KNIRV-NEXUS deployed successfully to AWS EC2"

    # Display deployment info
    echo "Deployment Information:"
    echo "======================"
    ansible-inventory -i inventory/aws.yml --list | jq '.nexus_servers.hosts[]'
}

# Health check
health_check() {
    echo "Performing health check..."

    # Wait for service to start
    sleep 30

    # Check NEXUS API
    NEXUS_URL=$(ansible-inventory -i deployment/inventory/aws.yml --list | jq -r '.nexus_servers.hosts[0]')
    if curl -f "http://${NEXUS_URL}:8080/health" > /dev/null 2>&1; then
        echo "✅ NEXUS API is healthy"
    else
        echo "❌ NEXUS API health check failed"
        exit 1
    fi
}

# Main execution
main() {
    check_prerequisites
    deploy_aws
    health_check
}

main "$@"
```

## Monitoring and Observability

### Native Monitoring Stack
```yaml
# Ansible playbook for monitoring setup
- name: Setup monitoring stack
  hosts: nexus_servers
  become: yes
  tasks:
    - name: Install monitoring tools
      apt:
        name:
          - prometheus
          - grafana
          - node-exporter
          - htop
          - iotop
          - nethogs
        state: present

    - name: Configure Prometheus
      template:
        src: prometheus.yml.j2
        dest: /etc/prometheus/prometheus.yml
      notify: restart prometheus

    - name: Configure Grafana
      template:
        src: grafana.ini.j2
        dest: /etc/grafana/grafana.ini
      notify: restart grafana

    - name: Start monitoring services
      systemd:
        name: "{{ item }}"
        state: started
        enabled: yes
      loop:
        - prometheus
        - grafana
        - node-exporter
```

### NEXUS-Specific Metrics
```go
// Built-in metrics collection in NEXUS
type MetricsCollector struct {
    db *buntdb.DB
    registry *prometheus.Registry
}

func (m *MetricsCollector) CollectSystemMetrics() {
    // TEE metrics
    teeStatus := m.getTEEStatus()
    m.recordMetric("tee_status", teeStatus)

    // BuntDB metrics
    dbStats := m.getBuntDBStats()
    m.recordMetric("db_operations", dbStats.Operations)
    m.recordMetric("db_size", dbStats.Size)

    // P2P network metrics
    p2pStats := m.getP2PStats()
    m.recordMetric("p2p_peers", p2pStats.PeerCount)
    m.recordMetric("p2p_bandwidth", p2pStats.Bandwidth)
}
```



#### KNIRVTESTNET on NEXUS Server Deployment

**Architecture Proposal**:
```
NEXUS Server (Single EC2 Instance)
├── DVE-1: KNIRV-ORACLE (Port 1317)
├── DVE-2: KNIRVCHAIN (Port 8080)
├── DVE-3: KNIRVGRAPH (Port 8081)
├── DVE-4: KNIRV-ROUTER (Port 8086)
├── DVE-5: KNIRVANA (Port 3000)
└── DVE-Gateway: NEXUS (Port 8082) - Orchestration
```

**Benefits**:
- Simplified deployment and management
- Reduced network latency between services
- Centralized monitoring and logging
- Easier debugging and development

**Implementation Strategy**:
1. Each KNIRVTESTNET application runs in separate DVE environment
2. NEXUS server acts as orchestrator and gateway
3. Internal gRPC communication between DVEs
4. External HTTP API exposure through NEXUS gateway
5. Shared BuntDB for cross-DVE state management

**DVE Communication Flow**:
```
Client Request → NEXUS Gateway → DVE Router → Target DVE → Response
```


### Recommended Approach: Native AWS EC2 Deployment

**Primary Strategy**: Single EC2 instance with unified NEXUS binary
**Reasoning**: Optimal for TEE access, simplified security, and cost-effectiveness

**Benefits**:
- Zero containerization overhead
- Direct TEE hardware access
- Simplified security model
- Optimal performance
- Easy backup and disaster recovery

**Implementation Priority**:
1. **Phase 1**: Single instance deployment (MVP)
2. **Phase 2**: Add monitoring and alerting
3. **Phase 3**: Implement auto-scaling groups
4. **Phase 4**: Multi-region deployment (if needed)

### AWS Instance Recommendations
- **Development**: m5.large (2 vCPU, 8GB RAM) - $0.096/hour
- **Production**: m5.xlarge (4 vCPU, 16GB RAM) - $0.192/hour
- **High-Performance**: c5.2xlarge (8 vCPU, 16GB RAM) - $0.34/hour
- **Storage**: gp3 EBS volumes with encryption
- **Network**: Enhanced networking enabled

### Security Hardening
1. **EC2 Security**: Minimal AMI, security groups, NACLs
2. **Network Security**: VPC isolation, encrypted communication
3. **Host Security**: Kali Linux hardening, regular updates
4. **Data Security**: EBS encryption, BuntDB encryption
5. **Access Control**: IAM roles, SSH key management

### Cost Optimization
- **Reserved Instances**: 1-year term for 40% savings
- **Spot Instances**: For development and testing (up to 90% savings)
- **Auto-scaling**: Scale down during low usage periods
- **Storage Optimization**: Use gp3 instead of gp2 for better price/performance

## Conclusion

The native AWS EC2 deployment of KNIRV-NEXUS provides optimal performance, security, and scalability while eliminating containerization complexity. The embedded BuntDB approach significantly simplifies deployment and reduces operational overhead.

**Key Advantages**:
- **Zero Dependencies**: No external database or container runtime required
- **High Performance**: Direct hardware access for TEE operations
- **Simplified Operations**: Single binary deployment with embedded database
- **Cost Effective**: Reduced infrastructure complexity and management overhead
- **Secure by Design**: Hardened Kali Linux with native TEE support

**Next Steps**:
1. Implement Ansible playbooks for automated deployment
2. Develop comprehensive testing and validation suite
3. Create production-ready security configurations
4. Establish monitoring, alerting, and backup systems
5. Plan disaster recovery and business continuity procedures
