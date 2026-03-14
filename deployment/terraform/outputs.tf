# Network Outputs
output "vpc_id" {
  description = "ID of the VPC"
  value       = aws_vpc.knirv_vpc.id
}

output "vpc_cidr_block" {
  description = "CIDR block of the VPC"
  value       = aws_vpc.knirv_vpc.cidr_block
}

output "public_subnet_ids" {
  description = "IDs of the public subnets"
  value       = aws_subnet.knirv_public_subnets[*].id
}

output "private_subnet_ids" {
  description = "IDs of the private subnets"
  value       = aws_subnet.knirv_private_subnets[*].id
}

output "internet_gateway_id" {
  description = "ID of the Internet Gateway"
  value       = aws_internet_gateway.knirv_igw.id
}

# Security Group Outputs
output "knirv_security_group_id" {
  description = "ID of the KNIRV security group"
  value       = aws_security_group.knirv_sg.id
}

# EC2 Instance Outputs
output "knirv_instance_ids" {
  description = "IDs of the KNIRV EC2 instances"
  value       = aws_instance.knirv_instances[*].id
}

output "knirv_instance_public_ips" {
  description = "Public IP addresses of the KNIRV instances"
  value       = aws_instance.knirv_instances[*].public_ip
}

output "knirv_instance_private_ips" {
  description = "Private IP addresses of the KNIRV instances"
  value       = aws_instance.knirv_instances[*].private_ip
}

output "knirv_instance_public_dns" {
  description = "Public DNS names of the KNIRV instances"
  value       = aws_instance.knirv_instances[*].public_dns
}

# Load Balancer Outputs (if enabled)
output "load_balancer_dns_name" {
  description = "DNS name of the load balancer"
  value       = var.environment == "production" ? aws_lb.knirv_alb[0].dns_name : null
}

output "load_balancer_zone_id" {
  description = "Zone ID of the load balancer"
  value       = var.environment == "production" ? aws_lb.knirv_alb[0].zone_id : null
}

# Cloudflare DNS Outputs
output "cloudflare_dns_records" {
  description = "Cloudflare DNS records created"
  value = var.cloudflare_zone_id != "" ? {
    for record in cloudflare_record.knirv_dns : record.name => record.value
  } : {}
}

# Service Endpoints
output "service_endpoints" {
  description = "Service endpoints for KNIRV components"
  value = {
    oracle     = "http://${aws_instance.knirv_instances[0].public_ip}:${var.knirv_service_ports.oracle_port}"
    chain      = "http://${aws_instance.knirv_instances[0].public_ip}:${var.knirv_service_ports.chain_port}"
    graph      = "http://${aws_instance.knirv_instances[0].public_ip}:${var.knirv_service_ports.graph_port}"
    nexus      = "http://${aws_instance.knirv_instances[0].public_ip}:${var.knirv_service_ports.nexus_port}"
    router     = "http://${aws_instance.knirv_instances[0].public_ip}:${var.knirv_service_ports.router_port}"
    controller = "http://${aws_instance.knirv_instances[0].public_ip}:${var.knirv_service_ports.controller_port}"
    gateway    = "http://${aws_instance.knirv_instances[0].public_ip}:${var.knirv_service_ports.gateway_port}"
    cortex     = "http://${aws_instance.knirv_instances[0].public_ip}:${var.knirv_service_ports.cortex_port}"
    engine     = "http://${aws_instance.knirv_instances[0].public_ip}:${var.knirv_service_ports.engine_port}"
  }
}

# SSH Connection Information
output "ssh_connection_commands" {
  description = "SSH commands to connect to instances"
  value = [
    for i, instance in aws_instance.knirv_instances :
    "ssh -i ~/.ssh/${var.key_pair_name}.pem ubuntu@${instance.public_ip}"
  ]
}

# Container Registry Information
output "container_registry_urls" {
  description = "Container registry URLs for KNIRV images"
  value = {
    oracle     = "${var.container_registry}/${var.project_name}/knirvoracle:${var.container_image_tag}"
    chain      = "${var.container_registry}/${var.project_name}/knirvchain:${var.container_image_tag}"
    graph      = "${var.container_registry}/${var.project_name}/knirvgraph:${var.container_image_tag}"
    nexus      = "${var.container_registry}/${var.project_name}/knirvserver:${var.container_image_tag}"
    router     = "${var.container_registry}/${var.project_name}/knirvrouter:${var.container_image_tag}"
    controller = "${var.container_registry}/${var.project_name}/knirvcontroller:${var.container_image_tag}"
    gateway    = "${var.container_registry}/${var.project_name}/knirvgateway:${var.container_image_tag}"
    cortex     = "${var.container_registry}/${var.project_name}/knirvcortex:${var.container_image_tag}"
    engine     = "${var.container_registry}/${var.project_name}/knirvengine:${var.container_image_tag}"
  }
}

# Deployment Information
output "deployment_info" {
  description = "Deployment information and next steps"
  value = {
    environment     = var.environment
    region         = var.aws_region
    instance_count = length(aws_instance.knirv_instances)
    deployment_time = timestamp()
    next_steps = [
      "1. SSH to instances using the provided commands",
      "2. Deploy containers using docker-compose or podman-compose",
      "3. Configure Cloudflare DNS if not automated",
      "4. Set up monitoring and alerting",
      "5. Configure SSL certificates"
    ]
  }
}
