packer {
  required_plugins {
    docker = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/docker"
    }
  }
}

variable "image_name" {
  type    = string
  default = "knirvnexus-kali-base"
}

variable "knirv_nexus_binary_path" {
  type    = string
  description = "Path to the knirv-nexus binary on the host."
}

source "docker" "kali_base" {
  image  = "debian:bookworm-slim" # Changed to match containerfile-kali.j2
  commit = true
  changes = [
    "USER root",
    "WORKDIR /app" # Changed to match containerfile-kali.j2
  ]
}

build {
  sources = ["source.docker.kali_base"]

  provisioner "shell" {
    inline = [
      "apt-get update",
      "DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates wget curl gnupg2 software-properties-common python3 python3-pip python3-dev python3-setuptools python3-wheel git build-essential pkg-config strace ltrace gdb linux-perf tcpdump tshark wireshark-common mitmproxy sleuthkit python3-bandit docker.io podman iptables iproute2 util-linux mount procps apparmor apparmor-utils selinux-utils selinux-basics libseccomp2 libseccomp-dev seccomp iputils-ping",
      "rm -rf /var/lib/apt/lists/*",
      "", # Newline for readability
      "# Install Go 1.24.11",
      "wget -q https://go.dev/dl/go1.24.11.linux-amd64.tar.gz",
      "tar -C /usr/local -xzf go1.24.11.linux-amd64.tar.gz",
      "rm go1.24.11.linux-amd64.tar.gz",
      "echo 'export PATH=\"/usr/local/go/bin:$PATH\"' > /etc/profile.d/go_path.sh", # Make PATH permanent
      "chmod +x /etc/profile.d/go_path.sh",
      "export PATH=\"/usr/local/go/bin:$${PATH}\"", # Also set for current build session
      "", # Newline for readability
      "# Install radare2 from source",
      "git clone --depth 1 https://github.com/radareorg/radare2 /tmp/radare2",
      "cd /tmp/radare2",
      "./sys/install.sh",
      "cd /",
      "rm -rf /tmp/radare2",
      "", # Newline for readability
      "# Install Python-based security tools via pip",
      "pip3 install --no-cache-dir --break-system-packages semgrep volatility3",
      "", # Newline for readability
      "# Install Ghidra",
      "apt-get update",
      "DEBIAN_FRONTEND=noninteractive apt-get install -y openjdk-17-jdk unzip",
      "rm -rf /var/lib/apt/lists/*",
      "GHIDRA_VERSION=\"11.2.1\"",
      "GHIDRA_DATE=\"20241105\"",
      "wget -q \"https://github.com/NationalSecurityAgency/ghidra/releases/download/Ghidra_$${GHIDRA_VERSION}_build/ghidra_$${GHIDRA_VERSION}_PUBLIC_$${GHIDRA_DATE}.zip\" -O /tmp/ghidra.zip",
      "unzip -q /tmp/ghidra.zip -d /opt",
      "mv /opt/ghidra_$${GHIDRA_VERSION}_PUBLIC /opt/ghidra",
      "rm /tmp/ghidra.zip",
      "ln -s /opt/ghidra/ghidraRun /usr/local/bin/ghidra",
      "", # Newline for readability
      "# Install Autopsy (if available, otherwise skip)",
      "apt-get update",
      "DEBIAN_FRONTEND=noninteractive apt-get install -y autopsy || true",
      "rm -rf /var/lib/apt/lists/*",
      "", # Newline for readability
      "# Ensure /tmp is writable and create knirv-containers directory",
      "chmod 1777 /tmp 2>/dev/null || true",
      "mkdir -p /tmp/knirv-containers",
      "chmod 755 /tmp/knirv-containers 2>/dev/null || true",
      "", # Newline for readability
      "# Create writable test directories",
      "mkdir -p /var/tmp && chmod 1777 /var/tmp 2>/dev/null || true",
      "", # Newline for readability
      "# Ensure /sys/fs/cgroup structure exists",
      "mkdir -p /sys/fs/cgroup 2>/dev/null || true"
    ]
  }

  provisioner "shell" { # NEW PROVISIONER
    inline = [
      "mkdir -p /app", # Explicitly create /app
      "chmod 755 /app" # Ensure permissions
    ]
  }

  post-processor "docker-tag" {
    repository = var.image_name
    tags       = ["latest"]
  }
}
