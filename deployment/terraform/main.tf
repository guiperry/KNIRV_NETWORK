terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }

  backend "s3" {
    # Configure S3 backend for state storage
    # bucket = "knirv-terraform-state"
    # key    = "knirv-network/terraform.tfstate"
    # region = "us-east-1"
    # encrypt = true
    # dynamodb_table = "knirv-terraform-locks"
  }
}

# Configure the AWS Provider
provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "KNIRV-Network"
      Environment = var.environment
      ManagedBy   = "Terraform"
    }
  }
}

# Configure Cloudflare Provider
provider "cloudflare" {
  api_token = var.cloudflare_api_token
}

# Data sources
data "aws_availability_zones" "available" {
  state = "available"
}

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-22.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

# 1. Create a VPC for network isolation
resource "aws_vpc" "knirv_vpc" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "${var.project_name}-${var.environment}-vpc"
  }
}

# 2. Create an Internet Gateway to allow public access
resource "aws_internet_gateway" "knirv_igw" {
  vpc_id = aws_vpc.knirv_vpc.id

  tags = {
    Name = "${var.project_name}-${var.environment}-igw"
  }
}

# 3. Create public subnets in multiple AZs for high availability
resource "aws_subnet" "knirv_public_subnets" {
  count = length(var.public_subnet_cidrs)

  vpc_id                  = aws_vpc.knirv_vpc.id
  cidr_block              = var.public_subnet_cidrs[count.index]
  availability_zone       = data.aws_availability_zones.available.names[count.index]
  map_public_ip_on_launch = true

  tags = {
    Name = "${var.project_name}-${var.environment}-public-subnet-${count.index + 1}"
    Type = "Public"
  }
}

# 4. Create private subnets for internal services
resource "aws_subnet" "knirv_private_subnets" {
  count = length(var.private_subnet_cidrs)

  vpc_id            = aws_vpc.knirv_vpc.id
  cidr_block        = var.private_subnet_cidrs[count.index]
  availability_zone = data.aws_availability_zones.available.names[count.index]

  tags = {
    Name = "${var.project_name}-${var.environment}-private-subnet-${count.index + 1}"
    Type = "Private"
  }
}

# 5. Create NAT Gateway for private subnets
resource "aws_eip" "knirv_nat_eips" {
  count  = length(var.public_subnet_cidrs)
  domain = "vpc"

  tags = {
    Name = "${var.project_name}-${var.environment}-nat-eip-${count.index + 1}"
  }

  depends_on = [aws_internet_gateway.knirv_igw]
}

resource "aws_nat_gateway" "knirv_nat_gws" {
  count = length(var.public_subnet_cidrs)

  allocation_id = aws_eip.knirv_nat_eips[count.index].id
  subnet_id     = aws_subnet.knirv_public_subnets[count.index].id

  tags = {
    Name = "${var.project_name}-${var.environment}-nat-gw-${count.index + 1}"
  }

  depends_on = [aws_internet_gateway.knirv_igw]
}

# 6. Create Route Tables
resource "aws_route_table" "knirv_public_rt" {
  vpc_id = aws_vpc.knirv_vpc.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.knirv_igw.id
  }

  tags = {
    Name = "${var.project_name}-${var.environment}-public-rt"
  }
}

resource "aws_route_table" "knirv_private_rts" {
  count  = length(var.private_subnet_cidrs)
  vpc_id = aws_vpc.knirv_vpc.id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.knirv_nat_gws[count.index].id
  }

  tags = {
    Name = "${var.project_name}-${var.environment}-private-rt-${count.index + 1}"
  }
}

# 7. Associate Route Tables with Subnets
resource "aws_route_table_association" "public" {
  count = length(var.public_subnet_cidrs)

  subnet_id      = aws_subnet.knirv_public_subnets[count.index].id
  route_table_id = aws_route_table.knirv_public_rt.id
}

resource "aws_route_table_association" "private" {
  count = length(var.private_subnet_cidrs)

  subnet_id      = aws_subnet.knirv_private_subnets[count.index].id
  route_table_id = aws_route_table.knirv_private_rts[count.index].id
}

# 8. Create Security Groups
resource "aws_security_group" "knirv_sg" {
  name_prefix = "${var.project_name}-${var.environment}-sg"
  description = "Security group for KNIRV network services"
  vpc_id      = aws_vpc.knirv_vpc.id

  # SSH access
  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.allowed_ssh_cidrs
  }

  # HTTP/HTTPS access
  ingress {
    description = "HTTP"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = var.allowed_http_cidrs
  }

  ingress {
    description = "HTTPS"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = var.allowed_http_cidrs
  }

  # KNIRV service ports
  dynamic "ingress" {
    for_each = var.knirv_service_ports
    content {
      description = "KNIRV ${ingress.key}"
      from_port   = ingress.value
      to_port     = ingress.value
      protocol    = "tcp"
      cidr_blocks = var.allowed_http_cidrs
    }
  }

  # Allow all outbound traffic
  egress {
    description = "All outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project_name}-${var.environment}-sg"
  }
}

# 9. Create SSH Key Pair
resource "aws_key_pair" "knirv_key" {
  key_name   = var.key_pair_name
  public_key = file(var.ssh_public_key_path)

  tags = {
    Name = "${var.project_name}-${var.environment}-key"
  }
}

# 10. Create Launch Template for EC2 instances
resource "aws_launch_template" "knirv_template" {
  name_prefix   = "${var.project_name}-${var.environment}-template"
  image_id      = data.aws_ami.ubuntu.id
  instance_type = var.instance_type
  key_name      = aws_key_pair.knirv_key.key_name

  vpc_security_group_ids = [aws_security_group.knirv_sg.id]

  user_data = base64encode(templatefile("${path.module}/user_data.sh", {
    project_name        = var.project_name
    environment         = var.environment
    container_registry  = var.container_registry
    container_image_tag = var.container_image_tag
  }))

  tag_specifications {
    resource_type = "instance"
    tags = {
      Name        = "${var.project_name}-${var.environment}-instance"
      Environment = var.environment
    }
  }

  lifecycle {
    create_before_destroy = true
  }
}

# 11. Create EC2 Instances
resource "aws_instance" "knirv_instances" {
  count = var.desired_instances

  launch_template {
    id      = aws_launch_template.knirv_template.id
    version = "$Latest"
  }

  subnet_id = aws_subnet.knirv_public_subnets[count.index % length(aws_subnet.knirv_public_subnets)].id

  tags = {
    Name = "${var.project_name}-${var.environment}-instance-${count.index + 1}"
  }
}

# 12. Application Load Balancer (for production)
resource "aws_lb" "knirv_alb" {
  count = var.environment == "production" ? 1 : 0

  name               = "${var.project_name}-${var.environment}-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.knirv_sg.id]
  subnets            = aws_subnet.knirv_public_subnets[*].id

  enable_deletion_protection = false

  tags = {
    Name = "${var.project_name}-${var.environment}-alb"
  }
}

# 13. Target Group for Load Balancer
resource "aws_lb_target_group" "knirv_tg" {
  count = var.environment == "production" ? 1 : 0

  name     = "${var.project_name}-${var.environment}-tg"
  port     = var.knirv_service_ports.gateway_port
  protocol = "HTTP"
  vpc_id   = aws_vpc.knirv_vpc.id

  health_check {
    enabled             = true
    healthy_threshold   = 2
    interval            = 30
    matcher             = "200"
    path                = "/health"
    port                = "traffic-port"
    protocol            = "HTTP"
    timeout             = 5
    unhealthy_threshold = 2
  }

  tags = {
    Name = "${var.project_name}-${var.environment}-tg"
  }
}

# 14. Load Balancer Listener
resource "aws_lb_listener" "knirv_listener" {
  count = var.environment == "production" ? 1 : 0

  load_balancer_arn = aws_lb.knirv_alb[0].arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.knirv_tg[0].arn
  }
}

# 15. Target Group Attachment
resource "aws_lb_target_group_attachment" "knirv_tg_attachment" {
  count = var.environment == "production" ? length(aws_instance.knirv_instances) : 0

  target_group_arn = aws_lb_target_group.knirv_tg[0].arn
  target_id        = aws_instance.knirv_instances[count.index].id
  port             = var.knirv_service_ports.gateway_port
}

# 16. Cloudflare DNS Records (optional)
resource "cloudflare_record" "knirv_dns" {
  for_each = var.cloudflare_zone_id != "" ? {
    "api"        = aws_instance.knirv_instances[0].public_ip
    "testnet"    = aws_instance.knirv_instances[0].public_ip
    "gateway"    = aws_instance.knirv_instances[0].public_ip
    "controller" = aws_instance.knirv_instances[0].public_ip
  } : {}

  zone_id = var.cloudflare_zone_id
  name    = "${each.key}.${var.domain_name}"
  value   = each.value
  type    = "A"
  ttl     = 300

  comment = "KNIRV Network ${var.environment} - Managed by Terraform"
}

# 17. CloudWatch Log Groups (for monitoring)
resource "aws_cloudwatch_log_group" "knirv_logs" {
  count = var.enable_monitoring ? 1 : 0

  name              = "/aws/ec2/${var.project_name}-${var.environment}"
  retention_in_days = var.log_retention_days

  tags = {
    Name = "${var.project_name}-${var.environment}-logs"
  }
}

# 18. S3 Bucket for backups (optional)
resource "aws_s3_bucket" "knirv_backups" {
  count = var.enable_backups ? 1 : 0

  bucket = "${var.project_name}-${var.environment}-backups-${random_id.bucket_suffix[0].hex}"

  tags = {
    Name = "${var.project_name}-${var.environment}-backups"
  }
}

resource "random_id" "bucket_suffix" {
  count = var.enable_backups ? 1 : 0

  byte_length = 4
}

resource "aws_s3_bucket_versioning" "knirv_backups_versioning" {
  count = var.enable_backups ? 1 : 0

  bucket = aws_s3_bucket.knirv_backups[0].id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "knirv_backups_lifecycle" {
  count = var.enable_backups ? 1 : 0

  bucket = aws_s3_bucket.knirv_backups[0].id

  rule {
    id     = "backup_lifecycle"
    status = "Enabled"

    expiration {
      days = var.backup_retention_days
    }

    noncurrent_version_expiration {
      noncurrent_days = 7
    }
  }
}