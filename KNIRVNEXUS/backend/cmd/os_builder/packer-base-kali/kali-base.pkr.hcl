packer {
  required_plugins {
    virtualbox = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/virtualbox"
    }
  }
}

variable "kali_iso_url" {
  type    = string
  default = "https://cdimage.kali.org/kali-2025.3/kali-linux-2025.3-installer-netinst-amd64.iso"
}
variable "kali_iso_checksum" {
  type    = string
  default = "sha256:f237246c2ec4391aa0d82f705736d70dd57476042fbfdaa9c9786904d770f745" // 
}

variable "use_local_iso" {
  type    = bool
  default = false
  description = "Set to true to use the local Kali ISO instead of downloading."
}

variable "output_directory" {
  type    = string
  default = "output-kali-base-box"
  description = "Directory where Packer outputs the VM files. Set to artifacts/output-kali-base-box for persistent storage."
}

locals {
  remote_iso = {
    url      = var.kali_iso_url
    checksum = var.kali_iso_checksum
  }
  local_iso = {
    url      = "../kali-linux-2025.3-installer-amd64.iso"
    checksum = "sha256:fcf1999799f6642b7d6c6bd79bc1e516be7340b7203fe86c6be3ac21f693f42a"
  }
  iso = var.use_local_iso ? local.local_iso : local.remote_iso
}

source "virtualbox-iso" "kali_base" {
  headless          = false
  vm_name           = "KaliBaseBox"
  guest_os_type     = "Debian_64"
  output_directory  = var.output_directory
  iso_url           = local.iso.url
  iso_checksum      = local.iso.checksum

  // Ensure VM is exported before cleanup (default: false, but being explicit)
  skip_export       = false
  // Keep the VM registered during post-processor stage
  keep_registered   = true

  vboxmanage        = [
    
    ["modifyvm", "{{.Name}}","--nic1", "nat"],
    ["modifyvm", "{{.Name}}", "--nic2", "hostonly", "--host-only-adapter2", "vboxnet0"]
  ]

// --- End Network Configuration ---

  disk_size         = 50000
  memory            = 5120
  boot_wait         = "15s"
  cpus              = 2

  http_directory    = "http"
  http_port_min     = 8082
  http_port_max     = 8082

  ssh_username      = "kaliadmin"
  ssh_password      = "kaliadmin"
  ssh_timeout       = "25m"
  ssh_port          = 2222
  ssh_skip_nat_mapping = false
  ssh_read_write_timeout = "5m"
  ssh_keep_alive_interval = "10s"
  ssh_handshake_attempts = 300
  ssh_pty            = true
  ssh_wait_timeout   = "30m"
  
  
// --- Boot and Automation Configuration ---

  boot_command      = [
    "<esc><wait>",
    "install ",
    "auto=true ",
    "priority=critical ",
    "locale=en_US.UTF-8 ",
    "keymap=us ",
    "hostname=kali-guest ",
    "domain=local ",
    "interface=auto ",
    "url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed-kali.cfg ",
    "passwd/user-fullname=\"Kali Admin\" ",
    "passwd/username=kaliadmin ",
    "passwd/user-password=kaliadmin ",
    "passwd/user-password-again=kaliadmin ",
    "netcfg/target_network_config=/etc/network/interfaces ",
    "-- quiet ",
    "<enter>"
  ]

shutdown_command  = "echo 'kaliadmin' | sudo -S shutdown -P now"
  
}

build {
  sources = ["source.virtualbox-iso.kali_base"]

  // FIRST: Wait for system to be fully ready and SSH responsive
  provisioner "shell" {
    inline = [
      "echo 'System is up and SSH is responding'",
      "sleep 5",
      "echo 'Giving additional time for services to stabilize'",
      "sleep 5"
    ]
  }

  // SECOND: Fix sudo access (uses password authentication)
  provisioner "shell" {
    inline = [
      "echo 'kaliadmin' | sudo -S bash -c 'echo \"kaliadmin ALL=(ALL) NOPASSWD:ALL\" > /etc/sudoers.d/99-packer'",
      "echo 'kaliadmin' | sudo -S chmod 0440 /etc/sudoers.d/99-packer",
      "echo 'Passwordless sudo configured'"
    ]
  }

  // THIRD: Verify everything (now uses passwordless sudo)
  provisioner "shell" {
    inline = [
      "echo '=== SSH Connection Successful ==='",
      "echo 'System Information:'",
      "uname -a",
      "echo ''",
      "echo 'Checking SSH Keys:'",
      "ls -la /etc/ssh/ssh_host_*",
      "echo ''",
      "echo 'SSH Service Status:'",
      "sudo systemctl status ssh.service --no-pager",
      "echo ''",
      "echo 'Checking regenerate-ssh-host-keys service:'",
      "sudo systemctl status regenerate-ssh-host-keys.service --no-pager || echo 'Service disabled (good!)'",
      "echo ''",
      "echo 'Preseed Log:'",
      "sudo cat /var/log/preseed-late.log 2>&1 || echo 'No preseed log found'",
      "echo ''",
      "echo 'Checking sudoers:'",
      "sudo cat /etc/sudoers.d/99-packer",
      "echo ''",
      "echo 'Testing passwordless sudo:'",
      "sudo whoami || echo 'Sudo still requires password!'"
    ]
  }

  // FOURTH: Install basic tools needed for the next stage
  provisioner "shell" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get install -y build-essential libncurses5-dev fakeroot xz-utils git debootstrap cpio linux-headers-$(uname -r)",
      "sudo apt-get clean"
    ]
  }

  // Export to an OVA file for use by packer-kata-guest
  // OVA is just a tar archive of the OVF and VMDK files
  post-processor "shell-local" {
    inline = [
      "echo 'Post-processor: Checking files in output directory: ${var.output_directory}'",
      "ls -lah ${var.output_directory}/",
      "echo 'Post-processor: Creating OVA file...'",
      "cd ${var.output_directory} && tar -cf kali-base-box.ova KaliBaseBox.ovf KaliBaseBox*.vmdk",
      "echo 'Post-processor: OVA creation complete'",
      "ls -lah kali-base-box.ova",
      "cd - > /dev/null"
    ]
  }
}