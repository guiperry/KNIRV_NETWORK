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

// No more ISO variables needed!

source "virtualbox-ovf" "kali_guest_builder" {
  source_path      = "../packer-base-kali/output-kali-base-box/kali-base-box.ova" // Point to your OVA
  vm_name          = "KataKaliGuestBuilder"
  guest_os_type    = "Debian_64"

  // We can increase resources for the compile step
  memory           = 4096
  cpus             = 4
}

build {
  sources = ["source.virtualbox-ovf.kali_guest_builder"]

  // The Ansible provisioner is the same as before.
  // It will now run on a machine that already has all the build tools!
  provisioner "ansible" {
    playbook_file = "kata-guest-provisioner.yml"
    user          = "kaliadmin"
  }
}