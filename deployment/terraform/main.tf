terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Configure the AWS Provider
provider "aws" {
  region = "us-east-1" # Or your preferred region
}

# 1. Create a VPC for network isolation
resource "aws_vpc" "knirv_vpc" {
  cidr_block = "10.0.0.0/16"
  tags = {
    Name = "knirv-testnet-vpc"
  }
}

# 2. Create an Internet Gateway to allow public access
resource "aws_internet_gateway" "knirv_igw" {
  vpc_id = aws_vpc.knirv_vpc.id
  tags = {
    Name = "knirv-testnet-igw"
  }
}

# 3. Create a public subnet
resource "aws_subnet" "knirv_public_subnet" {
  vpc_id     = aws_vpc.knirv_vpc.id
  cidr_block = "10.0.1.0/24"
  map_public_ip_on_launch = true
  availability_zone = "us-east-1a"
  tags = {
    Name = "knirv-testnet-public-subnet"
  }
}

# 4. Create a Route Table to route traffic to the Internet Gateway
resource "aws_route_table" "knirv_public_rt" {
  vpc_id = aws_vpc.knirv_vpc.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.knirv_igw.id
  }

  tags = {
    Name = "knirv-testnet-public-rt"
  }
}

# 5. Associate the Route Table with the Subnet
resource "aws_route_table_association" "a" {
  subnet_id      = aws_subnet.knirv_public_subnet.id
  route_table_id = aws_route_table.knirv_public_rt.id
}

# 6. Create a Security Group (Firewall)
resource "aws_security_group" "knirv_sg" {
  name        = "knirv-testnet-sg"
  description = "Allow traffic for KNIRV testnet"
  vpc_id      = aws_vpc.knirv_vpc.id

  # Allow SSH access
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] # WARNING: For development only. Restrict to your IP in production.
  }

  # Allow HTTP/HTTPS for web access
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Allow access to your service ports (example from your scripts)
  ingress {
    from_port   = 8000
    to_port     = 8093 # A range covering most of your services
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# 7. Add your SSH public key
resource "aws_key_pair" "knirv_key" {
  key_name   = "knirv-testnet-key"
  # Replace with your actual public key content or use a file
  public_key = file("~/.ssh/knirv-testnet-key.pem.pub")
}

# 8. Create the EC2 Instance
resource "aws_instance" "testnet_server" {
  ami           = "ami-0c55b159cbfafe1f0" # Ubuntu 22.04 LTS in us-east-1
  instance_type = "t3.medium"             # A good starting point
  subnet_id     = aws_subnet.knirv_public_subnet.id
  vpc_security_group_ids = [aws_security_group.knirv_sg.id]
  key_name      = aws_key_pair.knirv_key.key_name

  # This script runs once when the instance is first created.
  # It installs Podman and podman-compose.
  user_data = <<-EOF
              #!/bin/bash
              sudo apt-get update -y
              # Install Podman and Python pip for podman-compose
              sudo apt-get install -y podman python3-pip
              # Install podman-compose system-wide
              sudo pip3 install podman-compose
              # Enable the Podman socket for Docker API compatibility
              sudo systemctl enable --now podman.socket
              # Enable lingering for the ubuntu user to allow rootless containers to run in the background
              sudo loginctl enable-linger ubuntu
              EOF

  tags = {
    Name = "knirv-testnet-server"
  }
}

# 9. Output the public IP of the server
output "testnet_server_ip" {
  value = aws_instance.testnet_server.public_ip
}