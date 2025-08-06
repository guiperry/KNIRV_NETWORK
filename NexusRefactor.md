

I've moved the plugin server into the KNIRVNEXUS. I need you to ensure the KNIRVNEXUS utilizes the plugin server for serving agent WASM(or any other compiled binaries) files exclusively. This will be the main static repository for all agents on the NEXUS. Make the static storage available on the frontend when creating new agents manually. We need to rebrand the KNIRVNEXUS so that it actually says KNIRV-NEXUS and not Agentic Engine. Change the primary colors from the purple gradient to a dark-cobalt-blue to black gradient throughout the site for custom CSS with KNIRV brand colors (--knirv-primary: #00c0fa, --knirv-secondary: #2b56f5). Refactor the pages to match the following, keep all other functionality the same:


Here are the essential pages and their objectives for the KNIRV-NEXUS portal:

1. Dashboard
Objective: To provide a high-level, real-time overview of the developer's deployed tasks and the overall health of the KNIRV-NEXUS.

Usage: Upon login, a developer would see a summary of their active and completed tasks, resource consumption metrics (e.g., CPU, RAM, TEE usage), and a feed of security attestations and network alerts. It serves as a single pane of glass to quickly assess the status of their TEE-based workloads and the network's capacity.

2. Task & Workflow Management
Objective: To allow developers to create, deploy, and manage inference-enabled tasks on the network. This is where the core functionality of the CLEAN framework is exposed.

Usage: Developers can upload their AI models and execution code, specify the required TEE type (e.g., Intel SGX, AMD SEV), define resource constraints, and set execution strategies. This page would offer an intuitive interface for configuring "execution adaptability" parameters, allowing the system to dynamically adjust resource allocation based on real-time workload and threat analysis.

3. TEE Attestation & Security Logs
Objective: To provide a transparent and immutable record of security attestations and a comprehensive log of all security-related events for each task. This is paramount for building trust in the TEE-based network.

Usage: For each deployed task, a developer can access a detailed report confirming that the code executed within a genuine and uncompromised TEE. The page would display a timeline of cryptographic proofs, security audits, and any reported anomalies, directly demonstrating the "proactive security posture" of the CLEAN framework.

4. Performance & Observability
Objective: To give developers deep insight into the performance of their tasks, providing the data needed to optimize their AI models and execution parameters.

Usage: This page would feature detailed graphs and metrics on task latency, throughput, and resource utilization. Developers could filter logs to debug issues, analyze the efficiency of their inference workloads, and see how the CLEAN network's "cognitive AI/ML engines" are adapting the execution strategy to meet performance goals.

5. Network Status & Resource Explorer
Objective: To provide a public view of the KNIRV-NEXUS's decentralized nature, allowing developers to understand the network's topology and available resources.

Usage: Developers can view a map or list of available KNIRV-NEXUS nodes, their geographical locations, hardware specifications (e.g., TEE type, GPU capabilities), and current load. This page helps developers make informed decisions about where to deploy their tasks, ensuring they can meet their latency and computational requirements.

6. Billing & Usage Reports
Objective: To provide a clear breakdown of resource consumption and associated costs for a developer's tasks.

Usage: Developers can track their spending in real-time, view historical usage data, and generate reports on a per-task or per-project basis. This transparent accounting ensures that the cost of using the decentralized TEE network is predictable and justifiable.


Add the KAli Linux implementation as the foundation of the KNIRVNEXUS. The Go based application should run ontop of this implementation. PLease see the following guidelines on the Kali Linux Kernel version we want to use:

Here is the Kali LInux link: https://www.kali.org/docs/development/recompiling-the-kali-linux-kernel/ Recompiling the Kali Linux Kernel | Kali Linux Documentation Recompiling the Kali Linux Kernel | Kali Linux Documentation 

## 🎯 Recommendation: Which Kali Linux Base to Fork for the KNIRV-NEXUS DVE

### Executive Summary
> **Use the latest 2024.3 **rolling** Kali Linux **net-installer** image as the canonical upstream for the CLEAN fork.**  
> Strip it down to a **server-only, CLI-only, hardened profile** and **lock the kernel to the 6.6.x LTS tree** that Kali 2024.3 currently ships.  
> Re-compile that kernel once with the minimal set of hardening patches shown in the attached documentation.

---

### 🔍 Rationale (mapped to white-paper requirements)

| Requirement in White-paper | Why 2024.3 Rolling + 6.6.x LTS is the best fit |
|----------------------------|--------------------------------------------------|
| **Proactive security posture** (§4.2) | Rolling branch receives security patches daily; 6.6.x is a **Long-Term Support** kernel, minimizing future re-base effort. |
| **Minimalist, server-only profile** (§4.2) | Net-installer lets us cherry-pick **exactly** openssh-server, ca-certificates, containerd, nftables, and nothing else—no GUI, no kali-linux-* metapackages. |
| **Continuous self-auditing** (§4.2) | All offensive/defensive tools are still in the repos (`apt install kali-linux-headless`) but can be **added on demand** inside the TEE without bloating the base image. |
| **Deterministic builds & remote attestation** (§5) | The 6.6.x kernel config (`/boot/config-6.6.*-kali1-amd64`) is already reproducible; we copy it verbatim, add our 4 extra hardening flags, and freeze the `.deb` in our internal APT repo. |
| **Golang static binary inside TEE** (§4.2) | Kernel 6.6.x offers **fully working SGX/SEV-SNP** patches out-of-the-box, required for Intel TDX & AMD SEV-SNP remote attestation of our Go CLEAN enclave. |

---

### 🛠️ Step-by-Step Fork Recipe

1. **Download**
   ```bash
   wget https://cdimage.kali.org/kali-2024.3/kali-linux-2024.3-installer-netinst-amd64.iso
   ```

2. **Install (unattended)**
   ```bash
   # preseed.cfg -> only openssh-server, standard, no desktop
   virt-install --name kali-clean-base --memory 4096 --disk size=20 \
     --cdrom kali-linux-2024.3-installer-netinst-amd64.iso \
     --initrd-inject=preseed.cfg
   ```

3. **Lock kernel & harden**
   ```bash
   sudo apt-mark hold linux-image-amd64 linux-headers-amd64
   sudo apt install build-essential libncurses5-dev fakeroot xz-utils
   cp /boot/config-$(uname -r) ~/kernel/.config
   make menuconfig
   # enable: CONFIG_SECURITY_LOCKDOWN_LSM, CONFIG_INTEL_TXT, CONFIG_AMD_MEM_ENCRYPT
   make -j$(nproc) bindeb-pkg LOCALVERSION=-clean
   sudo dpkg -i ../linux-image-*-clean*.deb
   ```

4. **Shrink image**
   ```bash
   apt autoremove --purge -y
   apt clean
   rm -rf /usr/share/doc /usr/share/man
   ```

5. **Freeze & sign**
   - Push the resulting `.deb` kernel + rootfs tarball into your **internal APT repo**.  
   - Cosign the hash so every CLEAN node can verify the exact same image.

---

### ⚠️ What **NOT** to do
- ❌ Do **not** base on the 2023.x quarterly ISOs—those kernels (5.x) lack modern TEE patches.  
- ❌ Do **not** enable the full `kali-linux-everything` metapackage; it pulls 20 GB of offensive tools that break reproducibility.  
- ❌ Do **not** chase bleeding-edge 6.9-rc kernels; 6.6.x LTS gives us a **stable CVE feed** for the next 6 years.

---

### ✅ TL;DR

> **Base Image:** `kali-linux-2024.3-netinst-amd64`  
> **Kernel:** `6.6.x LTS` (recompiled once with hardening)  
> **Profile:** CLI-only, 2 GB compressed, reproducible, SGX/SEV-SNP ready.