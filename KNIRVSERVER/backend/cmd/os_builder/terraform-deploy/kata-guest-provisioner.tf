terraform {
  required_version = ">= 1.0"
  required_providers {
    local = {
      source  = "hashicorp/local"
      version = "~> 2.0"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.0"
    }
  }
}

provider "null" {
  # Default provider configuration
}

variable "ova_source_path" {
  description = "Path to the pre-built Kali OVA file"
  type        = string
}

variable "vm_name" {
  description = "Name of the VirtualBox VM"
  type        = string
  default     = "KataKaliGuestBuilder"
}

variable "ssh_username" {
  description = "SSH username for VM access"
  type        = string
  default     = "kaliadmin"
}

variable "ssh_password" {
  description = "SSH password for VM access"
  type        = string
  default     = "kaliadmin"
  sensitive   = true
}

variable "ssh_host" {
  description = "SSH host address for VM"
  type        = string
  default     = "127.0.0.1"
}

variable "ssh_port" {
  description = "SSH port for VM"
  type        = number
  default     = 22
}

variable "output_directory" {
  description = "Output directory for built artifacts (kernel and rootfs)"
  type        = string
}

variable "artifact_directory" {
  description = "Base artifact directory for storing builds"
  type        = string
}

variable "vm_memory_mb" {
  description = "Memory allocated to VM in MB"
  type        = number
  default     = 4096
}

variable "vm_cpus" {
  description = "Number of CPU cores for VM"
  type        = number
  default     = 4
}

variable "vm_disk_size_mb" {
  description = "Disk size for VM in MB (optional, only if OVA needs expansion)"
  type        = number
  default     = 0
}

variable "host_ssh_port" {
  description = "Host port to forward to VM SSH port 22"
  type        = number
  default     = 2222
}

variable "cleanup_vm_on_destroy" {
  description = "Whether to cleanup/delete VM when Terraform destroy is run"
  type        = bool
  default     = true
}

# VirtualBox VM import script - imports OVA and configures the VM
resource "local_file" "vm_import_script" {
  filename = "${var.artifact_directory}/import-kata-vm.sh"
  content  = <<-EOF
#!/bin/bash
set -euo pipefail

VM_NAME="${var.vm_name}"
OVA_PATH="${var.ova_source_path}"
VM_MEMORY="${var.vm_memory_mb}"
VM_CPUS="${var.vm_cpus}"
HOST_SSH_PORT="${var.host_ssh_port}"

echo "[VM-IMPORT] Starting VirtualBox VM import process..."
echo "[VM-IMPORT] VM Name: $VM_NAME"
echo "[VM-IMPORT] OVA Path: $OVA_PATH"

# Check if OVA file exists
if [ ! -f "$OVA_PATH" ]; then
  echo "[ERROR] OVA file not found: $OVA_PATH"
  exit 1
fi

# Check if VM already exists
if VBoxManage list vms | grep -q "\"$VM_NAME\""; then
  echo "[VM-IMPORT] VM already exists: $VM_NAME"
  echo "[VM-IMPORT] Skipping import (VM already registered)"
  exit 0
fi

# Import OVA
echo "[VM-IMPORT] Importing OVA file (this may take a few minutes)..."
VBoxManage import "$OVA_PATH" \
  --vsys 0 \
  --vmname "$VM_NAME" \
  --memory "$VM_MEMORY" \
  --cpus "$VM_CPUS" || {
  echo "[ERROR] Failed to import OVA"
  exit 1
}

echo "[VM-IMPORT] OVA imported successfully"

# Configure network port forwarding for SSH
echo "[VM-IMPORT] Configuring SSH port forwarding (host:$HOST_SSH_PORT -> VM:22)..."
VBoxManage modifyvm "$VM_NAME" \
  --nic1 nat \
  --nictype1 82540EM || true

VBoxManage controlvm "$VM_NAME" natpf1 "SSH,tcp,127.0.0.1,$HOST_SSH_PORT,,22" || {
  # Try without the VM running
  VBoxManage modifyvm "$VM_NAME" \
    --natpf1 "SSH,tcp,127.0.0.1,$HOST_SSH_PORT,,22" || true
}

# Disable sleep/suspend to keep VM running
echo "[VM-IMPORT] Disabling power-saving features..."
VBoxManage modifyvm "$VM_NAME" --usbxhci off || true

echo "[VM-IMPORT] VM import and configuration completed successfully!"
EOF
  file_permission = "0755"
}

# VirtualBox VM startup script - starts VM and waits for SSH to be available
resource "local_file" "vm_startup_script" {
  filename = "${var.artifact_directory}/start-kata-vm.sh"
  content  = <<-EOF
#!/bin/bash
set -euo pipefail

VM_NAME="${var.vm_name}"
SSH_HOST="${var.ssh_host}"
SSH_PORT="${var.host_ssh_port}"
MAX_RETRIES=60
RETRY_DELAY=5

echo "[VM-STARTUP] Starting VirtualBox VM: $VM_NAME"

# Check if VM is already running
VM_STATE=$(VBoxManage showvminfo "$VM_NAME" --machinereadable | grep "^VMState=" | cut -d'=' -f2 | tr -d '"')
echo "[VM-STARTUP] Current VM state: $VM_STATE"

if [ "$VM_STATE" = "running" ]; then
  echo "[VM-STARTUP] VM is already running"
else
  echo "[VM-STARTUP] Starting VM..."
  VBoxManage startvm "$VM_NAME" --type headless || {
    echo "[ERROR] Failed to start VM"
    exit 1
  }
  sleep 5
fi

# Wait for SSH to be available
echo "[VM-STARTUP] Waiting for SSH to be available at $SSH_HOST:$SSH_PORT..."
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
  if timeout 5 bash -c "echo > /dev/tcp/$SSH_HOST/$SSH_PORT" 2>/dev/null; then
    echo "[VM-STARTUP] SSH is now available!"
    sleep 2  # Give SSH a moment to fully initialize
    exit 0
  fi
  
  RETRY_COUNT=$((RETRY_COUNT + 1))
  echo "[VM-STARTUP] Retry $RETRY_COUNT/$MAX_RETRIES - SSH not yet available, waiting $${RETRY_DELAY}s..."
  sleep $RETRY_DELAY
done

echo "[ERROR] SSH failed to become available after $((MAX_RETRIES * RETRY_DELAY)) seconds"
exit 1
EOF
  file_permission = "0755"
}

# VM cleanup script - stops and optionally deletes the VM (used on destroy)
resource "local_file" "vm_cleanup_script" {
  filename = "${var.artifact_directory}/cleanup-kata-vm.sh"
  content  = <<-EOF
#!/bin/bash
set -euo pipefail

VM_NAME="${var.vm_name}"
CLEANUP_ON_DESTROY="${var.cleanup_vm_on_destroy}"

echo "[VM-CLEANUP] Cleaning up VM: $VM_NAME"

# Check if VM exists
if ! VBoxManage list vms | grep -q "\"$VM_NAME\""; then
  echo "[VM-CLEANUP] VM does not exist: $VM_NAME"
  exit 0
fi

# Check if VM is running
VM_STATE=$$(VBoxManage showvminfo "$VM_NAME" --machinereadable | grep "^VMState=" | cut -d'=' -f2 | tr -d '"')

if [ "$VM_STATE" = "running" ]; then
  echo "[VM-CLEANUP] Stopping VM..."
  VBoxManage controlvm "$VM_NAME" poweroff || VBoxManage controlvm "$VM_NAME" acpipowerbutton || true
  sleep 3
fi

if [ "$CLEANUP_ON_DESTROY" = "true" ]; then
  echo "[VM-CLEANUP] Unregistering and deleting VM..."
  VBoxManage unregistervm "$VM_NAME" --delete || true
  echo "[VM-CLEANUP] VM deleted"
else
  echo "[VM-CLEANUP] Keeping VM registered (cleanup_vm_on_destroy = false)"
fi

echo "[VM-CLEANUP] Cleanup completed"
EOF
  file_permission = "0755"
}

# Provisioning script template - written as a file for easier execution
resource "local_file" "provision_script" {
  filename = "${var.output_directory}/provision-kata-guest.sh"
  content  = templatefile("${path.module}/provision-kata-guest.sh", {
    ssh_username      = var.ssh_username
    ssh_password      = var.ssh_password
    ssh_host          = var.ssh_host
    ssh_port          = var.host_ssh_port
    output_directory  = var.output_directory
  })
  file_permission = "0755"
}

# Import the OVA into VirtualBox
resource "null_resource" "import_vm" {
  provisioner "local-exec" {
    command = "bash -x ${local_file.vm_import_script.filename}"
  }

  depends_on = [local_file.vm_import_script]
}

# Start the VM and wait for SSH to be available
resource "null_resource" "start_vm" {
  provisioner "local-exec" {
    command = "bash -x ${local_file.vm_startup_script.filename}"
  }

  depends_on = [
    null_resource.import_vm,
    local_file.vm_startup_script
  ]
}

# Execute the Kata guest provisioning script
resource "null_resource" "provision_kata_guest" {
  provisioner "local-exec" {
    command     = "bash -x ${local_file.provision_script.filename}"
    environment = {
      SSHPASS = var.ssh_password
    }
  }

  depends_on = [
    null_resource.start_vm,
    local_file.provision_script
  ]
}

# Cleanup VM on terraform destroy (optional, controlled by cleanup_vm_on_destroy variable)
resource "null_resource" "cleanup_vm" {
  provisioner "local-exec" {
    when       = destroy
    command    = "bash -x ${self.triggers.cleanup_script}"
    on_failure = continue  # Don't fail if cleanup has issues
  }

  triggers = {
    cleanup_script = local_file.vm_cleanup_script.filename
  }

  depends_on = [
    null_resource.provision_kata_guest,
    local_file.vm_cleanup_script
  ]
}

output "vm_name" {
  description = "Name of the VirtualBox VM"
  value       = var.vm_name
}

output "vm_memory" {
  description = "Memory allocated to VM"
  value       = "${var.vm_memory_mb}MB"
}

output "vm_cpus" {
  description = "Number of CPU cores for VM"
  value       = var.vm_cpus
}

output "ssh_connection_string" {
  description = "SSH connection command for accessing the VM"
  value       = "ssh -p ${var.ssh_port} ${var.ssh_username}@${var.ssh_host}"
}

output "ssh_password" {
  description = "SSH password for VM access"
  value       = var.ssh_password
  sensitive   = true
}

output "kernel_path" {
  description = "Path to the built kernel"
  value       = "${var.output_directory}/vmlinuz-kali-clean-tee"
}

output "rootfs_path" {
  description = "Path to the built rootfs"
  value       = "${var.output_directory}/kata-rootfs-kali-clean-tee.img"
}

output "provisioning_status" {
  description = "Status of the provisioning"
  value       = "Provisioning completed. Check kernel_path and rootfs_path outputs above."
}

output "cleanup_note" {
  description = "Information about cleanup behavior"
  value       = var.cleanup_vm_on_destroy ? "VM will be deleted on terraform destroy" : "VM will be kept after terraform destroy"
}