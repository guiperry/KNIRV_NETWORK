# KNIRV Network Secure Deployment Keys Configuration
# This file defines secure storage and management of deployment credentials

# AWS Secrets Manager for storing sensitive deployment keys
resource "aws_secretsmanager_secret" "knirv_deployment_keys" {
  name                    = "${var.project_name}-${var.environment}-deployment-keys"
  description             = "Secure storage for KNIRV Network deployment API keys and credentials"
  recovery_window_in_days = 7

  tags = {
    Name        = "${var.project_name}-deployment-keys"
    Environment = var.environment
    Purpose     = "Deployment Credentials"
    Sensitive   = "true"
  }
}

# Secret version containing all deployment keys
resource "aws_secretsmanager_secret_version" "knirv_deployment_keys" {
  secret_id = aws_secretsmanager_secret.knirv_deployment_keys.id
  secret_string = jsonencode({
    # Cloudflare API credentials
    cloudflare_api_token = var.cloudflare_api_token
    cloudflare_zone_id   = var.cloudflare_zone_id
    
    # AWS deployment credentials
    aws_access_key_id     = var.aws_access_key_id
    aws_secret_access_key = var.aws_secret_access_key
    
    # GitHub deployment credentials
    github_token          = var.github_token
    github_deploy_key     = var.github_deploy_key
    
    # Container registry credentials
    container_registry_username = var.container_registry_username
    container_registry_password = var.container_registry_password
    
    # SSL/TLS certificates
    ssl_certificate = var.ssl_certificate
    ssl_private_key = var.ssl_private_key
    
    # Database encryption keys
    database_encryption_key = var.database_encryption_key
    
    # JWT secrets
    jwt_secret = var.jwt_secret
    
    # API keys for external services
    openai_api_key     = var.openai_api_key
    anthropic_api_key  = var.anthropic_api_key
    cerebras_api_key   = var.cerebras_api_key
    deepseek_api_key   = var.deepseek_api_key
    google_api_key     = var.google_api_key
    
    # Monitoring and alerting
    sentry_dsn         = var.sentry_dsn
    slack_webhook_url  = var.slack_webhook_url
    
    # Wallet integration
    wallet_connect_project_id = var.wallet_connect_project_id
    infura_id                = var.infura_id
    
    # Analytics
    analytics_id = var.analytics_id
    gtm_id      = var.gtm_id
  })
}

# IAM role for accessing deployment secrets
resource "aws_iam_role" "knirv_deployment_role" {
  name = "${var.project_name}-${var.environment}-deployment-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = [
            "ec2.amazonaws.com",
            "ecs-tasks.amazonaws.com"
          ]
        }
      },
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Condition = {
          StringEquals = {
            "sts:ExternalId" = var.deployment_external_id
          }
        }
      }
    ]
  })

  tags = {
    Name        = "${var.project_name}-deployment-role"
    Environment = var.environment
    Purpose     = "Deployment Access"
  }
}

# IAM policy for accessing deployment secrets
resource "aws_iam_role_policy" "knirv_deployment_secrets_policy" {
  name = "${var.project_name}-${var.environment}-deployment-secrets-policy"
  role = aws_iam_role.knirv_deployment_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret"
        ]
        Resource = [
          aws_secretsmanager_secret.knirv_deployment_keys.arn
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "kms:Decrypt",
          "kms:DescribeKey"
        ]
        Resource = [
          aws_kms_key.knirv_secrets_key.arn
        ]
      }
    ]
  })
}

# KMS key for encrypting secrets
resource "aws_kms_key" "knirv_secrets_key" {
  description             = "KMS key for encrypting KNIRV Network deployment secrets"
  deletion_window_in_days = 7
  enable_key_rotation     = true

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action   = "kms:*"
        Resource = "*"
      },
      {
        Sid    = "Allow use of the key for deployment"
        Effect = "Allow"
        Principal = {
          AWS = aws_iam_role.knirv_deployment_role.arn
        }
        Action = [
          "kms:Encrypt",
          "kms:Decrypt",
          "kms:ReEncrypt*",
          "kms:GenerateDataKey*",
          "kms:DescribeKey"
        ]
        Resource = "*"
      }
    ]
  })

  tags = {
    Name        = "${var.project_name}-secrets-key"
    Environment = var.environment
    Purpose     = "Secrets Encryption"
  }
}

# KMS key alias
resource "aws_kms_alias" "knirv_secrets_key_alias" {
  name          = "alias/${var.project_name}-${var.environment}-secrets"
  target_key_id = aws_kms_key.knirv_secrets_key.key_id
}

# Instance profile for EC2 instances
resource "aws_iam_instance_profile" "knirv_deployment_profile" {
  name = "${var.project_name}-${var.environment}-deployment-profile"
  role = aws_iam_role.knirv_deployment_role.name

  tags = {
    Name        = "${var.project_name}-deployment-profile"
    Environment = var.environment
    Purpose     = "EC2 Deployment Access"
  }
}

# Data source for current AWS account
data "aws_caller_identity" "current" {}

# Output the secret ARN for use in other configurations
output "deployment_secrets_arn" {
  description = "ARN of the deployment secrets in AWS Secrets Manager"
  value       = aws_secretsmanager_secret.knirv_deployment_keys.arn
  sensitive   = true
}

output "deployment_role_arn" {
  description = "ARN of the deployment IAM role"
  value       = aws_iam_role.knirv_deployment_role.arn
}

output "deployment_kms_key_id" {
  description = "ID of the KMS key used for secrets encryption"
  value       = aws_kms_key.knirv_secrets_key.key_id
  sensitive   = true
}

output "deployment_instance_profile_name" {
  description = "Name of the instance profile for EC2 deployment access"
  value       = aws_iam_instance_profile.knirv_deployment_profile.name
}
