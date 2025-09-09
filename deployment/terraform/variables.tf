# Core Configuration Variables
variable "project_name" {
  description = "Name of the project"
  type        = string
  default     = "knirv-network"
}

variable "environment" {
  description = "Environment name (dev, staging, production)"
  type        = string
  default     = "dev"
  
  validation {
    condition     = contains(["dev", "staging", "production"], var.environment)
    error_message = "Environment must be one of: dev, staging, production."
  }
}

variable "aws_region" {
  description = "AWS region for resources"
  type        = string
  default     = "us-east-1"
}

# Network Configuration
variable "vpc_cidr" {
  description = "CIDR block for VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "public_subnet_cidrs" {
  description = "CIDR blocks for public subnets"
  type        = list(string)
  default     = ["10.0.1.0/24", "10.0.2.0/24"]
}

variable "private_subnet_cidrs" {
  description = "CIDR blocks for private subnets"
  type        = list(string)
  default     = ["10.0.10.0/24", "10.0.20.0/24"]
}

# EC2 Configuration
variable "instance_type" {
  description = "EC2 instance type for KNIRV services"
  type        = string
  default     = "t3.medium"
}

variable "key_pair_name" {
  description = "Name of the AWS key pair for EC2 access"
  type        = string
  default     = "knirv-network-key"
}

variable "ssh_public_key_path" {
  description = "Path to SSH public key file"
  type        = string
  default     = "~/.ssh/id_rsa.pub"
}

# Security Configuration
variable "allowed_ssh_cidrs" {
  description = "CIDR blocks allowed for SSH access"
  type        = list(string)
  default     = ["0.0.0.0/0"] # WARNING: Restrict this in production
}

variable "allowed_http_cidrs" {
  description = "CIDR blocks allowed for HTTP/HTTPS access"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

# KNIRV Service Ports
variable "knirv_service_ports" {
  description = "Port ranges for KNIRV services"
  type = object({
    oracle_port     = number
    chain_port      = number
    graph_port      = number
    nexus_port      = number
    router_port     = number
    controller_port = number
    gateway_port    = number
    cortex_port     = number
    engine_port     = number
  })
  default = {
    oracle_port     = 1317
    chain_port      = 8090
    graph_port      = 8082
    nexus_port      = 8084
    router_port     = 8086
    controller_port = 3000
    gateway_port    = 8888
    cortex_port     = 8080
    engine_port     = 8081
  }
}

# Cloudflare Configuration
variable "cloudflare_api_token" {
  description = "Cloudflare API token for DNS management"
  type        = string
  sensitive   = true
  default     = ""
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for DNS management"
  type        = string
  sensitive   = true
  default     = ""
}

# AWS Deployment Credentials
variable "aws_access_key_id" {
  description = "AWS access key ID for deployment"
  type        = string
  sensitive   = true
  default     = ""
}

variable "aws_secret_access_key" {
  description = "AWS secret access key for deployment"
  type        = string
  sensitive   = true
  default     = ""
}

# GitHub Deployment Credentials
variable "github_token" {
  description = "GitHub personal access token for deployment"
  type        = string
  sensitive   = true
  default     = ""
}

variable "github_deploy_key" {
  description = "GitHub deploy key for repository access"
  type        = string
  sensitive   = true
  default     = ""
}

# Container Registry Credentials
variable "container_registry_username" {
  description = "Container registry username"
  type        = string
  sensitive   = true
  default     = ""
}

variable "container_registry_password" {
  description = "Container registry password"
  type        = string
  sensitive   = true
  default     = ""
}

# SSL/TLS Configuration
variable "ssl_certificate" {
  description = "SSL certificate for HTTPS"
  type        = string
  sensitive   = true
  default     = ""
}

variable "ssl_private_key" {
  description = "SSL private key for HTTPS"
  type        = string
  sensitive   = true
  default     = ""
}

# Security Keys
variable "database_encryption_key" {
  description = "Database encryption key"
  type        = string
  sensitive   = true
  default     = ""
}

variable "jwt_secret" {
  description = "JWT secret for authentication"
  type        = string
  sensitive   = true
  default     = ""
}

variable "deployment_external_id" {
  description = "External ID for deployment role assumption"
  type        = string
  sensitive   = true
  default     = ""
}

# API Keys for External Services
variable "openai_api_key" {
  description = "OpenAI API key for AI services"
  type        = string
  sensitive   = true
  default     = ""
}

variable "anthropic_api_key" {
  description = "Anthropic API key for Claude AI"
  type        = string
  sensitive   = true
  default     = ""
}

variable "cerebras_api_key" {
  description = "Cerebras API key for inference"
  type        = string
  sensitive   = true
  default     = ""
}

variable "deepseek_api_key" {
  description = "DeepSeek API key for AI services"
  type        = string
  sensitive   = true
  default     = ""
}

variable "google_api_key" {
  description = "Google API key for services"
  type        = string
  sensitive   = true
  default     = ""
}

# Monitoring and Alerting
variable "sentry_dsn" {
  description = "Sentry DSN for error monitoring"
  type        = string
  sensitive   = true
  default     = ""
}

variable "slack_webhook_url" {
  description = "Slack webhook URL for notifications"
  type        = string
  sensitive   = true
  default     = ""
}

# Wallet Integration
variable "wallet_connect_project_id" {
  description = "WalletConnect project ID"
  type        = string
  sensitive   = true
  default     = ""
}

variable "infura_id" {
  description = "Infura project ID for Ethereum integration"
  type        = string
  sensitive   = true
  default     = ""
}

# Analytics
variable "analytics_id" {
  description = "Analytics tracking ID"
  type        = string
  sensitive   = true
  default     = ""
}

variable "gtm_id" {
  description = "Google Tag Manager ID"
  type        = string
  sensitive   = true
  default     = ""
}

variable "cloudflare_zone_id" {
  description = "Cloudflare zone ID for DNS records"
  type        = string
  default     = ""
}

variable "domain_name" {
  description = "Domain name for KNIRV services"
  type        = string
  default     = "knirv.com"
}

# Container Configuration
variable "container_registry" {
  description = "Container registry URL"
  type        = string
  default     = "ghcr.io"
}

variable "container_image_tag" {
  description = "Container image tag to deploy"
  type        = string
  default     = "latest"
}

# Monitoring and Logging
variable "enable_monitoring" {
  description = "Enable CloudWatch monitoring"
  type        = bool
  default     = true
}

variable "log_retention_days" {
  description = "CloudWatch log retention in days"
  type        = number
  default     = 30
}

# Auto Scaling Configuration
variable "min_instances" {
  description = "Minimum number of instances in auto scaling group"
  type        = number
  default     = 1
}

variable "max_instances" {
  description = "Maximum number of instances in auto scaling group"
  type        = number
  default     = 3
}

variable "desired_instances" {
  description = "Desired number of instances in auto scaling group"
  type        = number
  default     = 1
}

# Database Configuration
variable "enable_rds" {
  description = "Enable RDS database for persistent storage"
  type        = bool
  default     = false
}

variable "db_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.t3.micro"
}

# Backup Configuration
variable "enable_backups" {
  description = "Enable automated backups"
  type        = bool
  default     = true
}

variable "backup_retention_days" {
  description = "Number of days to retain backups"
  type        = number
  default     = 7
}
