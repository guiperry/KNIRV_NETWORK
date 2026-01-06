packer {
  required_plugins {
    amazon = {
      version = ">= 1.2.0"
      source  = "github.com/hashicorp/amazon"
    }
  }
}

variable "aws_region" {
  type    = string
  default = "us-east-1"
  description = "AWS region to build AMI in"
}

variable "aws_ami_name" {
  type    = string
  default = "knirvnexus-kali-{{timestamp}}"
  description = "Name for the AMI"
}

variable "aws_instance_type" {
  type    = string
  default = "t3.medium"
  description = "EC2 instance type for the build"
}

variable "aws_subnet_id" {
  type    = string
  default = ""
  description = "Optional subnet ID for the build (leave empty for default)"
}

variable "aws_security_group_id" {
  type    = string
  default = ""
  description = "Optional security group ID for the build (leave empty to create one)"
}

variable "aws_ami_description" {
  type    = string
  default = "KNIRVNEXUS Kali Linux - Native deployment ready"
  description = "Description for the AMI"
}

locals {
  timestamp = formatdate("YYYY-MM-DD-hhmm", timestamp())
}

source "amazon-ebs" "kali-nexus" {
  ami_name              = replace(var.aws_ami_name, "{{timestamp}}", local.timestamp)
  ami_description       = var.aws_ami_description
  instance_type         = var.aws_instance_type
  region                = var.aws_region
  subnet_id             = var.aws_subnet_id
  security_group_id     = var.aws_security_group_id

  source_ami_filter {
    filters = {
      name                = "kali-linux-*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
    }
    most_recent = true
    owners      = ["679593333241"]
  }

  ssh_username = "kali"
  ssh_timeout  = "30m"
  ssh_port     = 22

  tags = {
    Name        = "KNIRVNEXUS Kali Linux"
    Environment = "production"
    Purpose     = "native-deployment"
    BuiltBy     = "packer"
    CreatedDate = local.timestamp
  }

  // Encrypt EBS volume for security
  encrypt_boot = true

  // Volume configuration
  volume_size = 50
  volume_type = "gp3"

  // Shutdown gracefully
  shutdown_command = "echo 'kali' | sudo -S shutdown -P now"
  shutdown_timeout = "5m"
}

build {
  sources = ["source.amazon-ebs.kali-nexus"]

  // Update system and install dependencies
  provisioner "shell" {
    inline = [
      "echo '=== Updating Kali Linux ==='",
      "sudo apt-get update",
      "sudo DEBIAN_FRONTEND=noninteractive apt-get upgrade -y",
      "echo '=== System updated ==='"
    ]
  }

  // Install container runtime dependencies
  provisioner "shell" {
    inline = [
      "echo '=== Installing container runtime dependencies ==='",
      "sudo apt-get install -y containerd runc cgroup-tools",
      "echo '=== Container runtime installed ==='"
    ]
  }

  // Install Kali security tools
  provisioner "shell" {
    inline = [
      "echo '=== Installing Kali security tools ==='",
      "sudo apt-get install -y kali-linux-default",
      "sudo apt-get install -y strace ltrace gdb",
      "sudo apt-get install -y tcpdump tshark wireshark-common",
      "sudo apt-get install -y radare2 ghidra sleuthkit",
      "sudo apt-get install -y semgrep bandit",
      "sudo apt-get install -y volatility3",
      "echo '=== Kali security tools installed ==='"
    ]
  }

  // Install security and TEE dependencies
  provisioner "shell" {
    inline = [
      "echo '=== Installing security and TEE dependencies ==='",
      "sudo apt-get install -y apparmor apparmor-utils",
      "sudo apt-get install -y libseccomp2 libseccomp-dev",
      "echo '=== Security dependencies installed ==='"
    ]
  }

  // Configure kernel for container support
  provisioner "shell" {
    inline = [
      "echo '=== Configuring kernel for container support ==='",
      "echo 'cgroup_enable=memory swapaccount=1' | sudo tee -a /etc/default/grub",
      "sudo update-grub",
      "echo '=== Kernel configured ==='"
    ]
  }

  // Enable cgroup v2
  provisioner "shell" {
    inline = [
      "echo '=== Enabling cgroup v2 ==='",
      "sudo mkdir -p /etc/systemd/system.conf.d",
      "echo '[Manager]' | sudo tee /etc/systemd/system.conf.d/cgroup.conf",
      "echo 'DefaultCPUAccounting=yes' | sudo tee -a /etc/systemd/system.conf.d/cgroup.conf",
      "echo 'DefaultMemoryAccounting=yes' | sudo tee -a /etc/systemd/system.conf.d/cgroup.conf",
      "echo '=== Cgroup v2 enabled ==='"
    ]
  }

  // Create application directories
  provisioner "shell" {
    inline = [
      "echo '=== Creating application directories ==='",
      "sudo mkdir -p /etc/knirv-nexus",
      "sudo mkdir -p /var/lib/knirv-nexus",
      "sudo mkdir -p /var/log/knirv-nexus",
      "sudo chmod 755 /etc/knirv-nexus",
      "sudo chmod 755 /var/lib/knirv-nexus",
      "sudo chmod 755 /var/log/knirv-nexus",
      "echo '=== Application directories created ==='"
    ]
  }

  // Install Go for binary building
  provisioner "shell" {
    inline = [
      "echo '=== Installing Go ==='",
      "wget -q https://go.dev/dl/go1.24.11.linux-amd64.tar.gz",
      "sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.11.linux-amd64.tar.gz",
      "rm go1.24.11.linux-amd64.tar.gz",
      "echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile.d/go.sh",
      "echo '=== Go installed ==='"
    ]
  }

  // Cleanup to reduce AMI size
  provisioner "shell" {
    inline = [
      "echo '=== Cleaning up ==='",
      "sudo apt-get clean",
      "sudo rm -rf /var/lib/apt/lists/*",
      "sudo rm -rf /tmp/*",
      "sudo rm -rf /var/tmp/*",
      "echo '=== Cleanup complete ==='"
    ]
  }

  // Print build completion message
  provisioner "shell" {
    inline = [
      "echo '=== AMI Build Complete ==='",
      "echo 'AMI is ready for KNIRVNEXUS native deployment'",
      "uname -a",
      "echo 'Go version:'",
      "/usr/local/go/bin/go version"
    ]
  }
}
