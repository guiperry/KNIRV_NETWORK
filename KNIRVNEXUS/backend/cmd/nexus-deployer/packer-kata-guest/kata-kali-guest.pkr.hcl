packer {
  required_plugins {
    virtualbox = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/virtualbox"
    }
    ansible = {
      version = ">= 1.1.0"
      source  = "github.com/hashicorp/ansible"
    }
  }
}

// No ISO variables needed!

source "virtualbox-ovf" "kali_guest_builder" {
  source_path      = "../../artifacts/output-kali-base-box/kali-base-box.ova" // Point to your OVA
  vm_name          = "KataKaliGuestBuilder"

  // Increase resources for the compile step using vboxmanage commands
  vboxmanage = [
    ["modifyvm", "{{ .Name }}", "--memory", "4096"],
    ["modifyvm", "{{ .Name }}", "--cpus", "4"]
  ]

  // SSH configuration with extended timeouts for first-time builds
  ssh_username      = "kaliadmin"
  ssh_password      = "kaliadmin"
  ssh_timeout       = "60m"
  ssh_port          = 22
  ssh_skip_nat_mapping = false
  ssh_read_write_timeout = "30m"
  ssh_keep_alive_interval = "10s"
  ssh_handshake_attempts = 1000
  ssh_pty            = true
  ssh_wait_timeout   = "60m"

  shutdown_command  = "echo 'kaliadmin' | sudo -S shutdown -P now"
}

build {
  sources = ["source.virtualbox-ovf.kali_guest_builder"]

  // The Ansible provisioner is the same as before.
  // It will now run on a machine that already has all the build tools!
  provisioner "ansible" {
    playbook_file = "kata-guest-provisioner.yml"
    user          = "kaliadmin"
    extra_arguments = [
      "--timeout=3600",  // 60 minutes timeout for kernel compilation
      "-e", "ansible_ssh_timeout=3600",
      "-e", "ansible_ssh_retries=20",
      "-e", "ansible_connect_timeout=120",
      "-e", "ansible_ssh_common_args='-o ConnectTimeout=120 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ServerAliveInterval=60 -o ServerAliveCountMax=10'",
      "-vvv"  // Enable very verbose output for debugging
    ]
  }
}