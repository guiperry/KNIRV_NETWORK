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
  description = "Path to the knirv-server binary on the host."
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

  provisioner "shell" {
    # We use environment variables to pass your credentials if you still need private repos,
    # though for CUDA, we will use the public network repo method below.
    environment_vars = [
      "DEBIAN_FRONTEND=noninteractive",
      "NVIDIA_DEV_USER=${env("NVIDIA_DEV_USER")}",
      "NVIDIA_DEV_PASS=${env("NVIDIA_DEV_PASS")}"
    ]

    inline = [
      "apt-get update",
      
      # 1. Install Base Build Dependencies for MLC-LLM
      "apt-get install -y cmake build-essential pkg-config git python3-dev python3-pip libvulkan-dev",

      # 2. Add NVIDIA Network Repository (No login required)
      "curl -fSsL https://developer.download.nvidia.com/compute/cuda/repos/debian12/x86_64/cuda-keyring_1.1-1_all.deb -o /tmp/cuda-keyring.deb",
      "dpkg -i /tmp/cuda-keyring.deb && apt-get update",

      # 3. Intelligent Driver-to-Toolkit Mapping Script
      "echo 'Detecting NVIDIA Driver and installing corresponding Toolkit...'",
      <<-EOF
      # Check if nvidia-smi is available to detect host driver
      if command -v nvidia-smi &> /dev/null; then
          DRIVER_VER=$(nvidia-smi --query-gpu=driver_version --format=csv,noheader,nounits | head -n 1 | cut -d'.' -f1)
          echo "Detected Driver Version: $DRIVER_VER"
          
          # Logic to select Toolkit based on Driver (e.g., Driver 535 supports up to CUDA 12.2)
          if [ "$DRIVER_VER" -ge 550 ]; then TOOLKIT_VER="12-4";
          elif [ "$DRIVER_VER" -ge 525 ]; then TOOLKIT_VER="12-0";
          else TOOLKIT_VER="11-8"; fi
      else
          echo "No active driver detected in build environment. Defaulting to CUDA 12-4 for EC2 G5 compatibility."
          TOOLKIT_VER="12-4"
      fi

      # Install the specific toolkit version
      apt-get install -y "cuda-toolkit-$TOOLKIT_VER"
      EOF
      ,
    
      # 4. Set persistent environment variables for the AI Engine
      "echo 'export PATH=/usr/local/cuda/bin:\$PATH' >> /etc/profile.d/cuda.sh",
      "echo 'export LD_LIBRARY_PATH=/usr/local/cuda/lib64:\$LD_LIBRARY_PATH' >> /etc/profile.d/cuda.sh",

      # 5. Install wazero CLI for testing and debugging Wasm modules
      "curl -fSsL https://wazero.io/install.sh | sh -s -- -b /usr/local/bin",

      # 6. Existing Kali Tools and Cleanup
      "apt-get install -y strace ltrace gdb tcpdump tshark",
      "rm -rf /var/lib/apt/lists/* /tmp/cuda-keyring.deb"
    ]
  }


  post-processor "docker-tag" {
    repository = var.image_name
    tags       = ["latest"]
  }
}
