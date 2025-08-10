# KNIRVNEXUS User and Deployment Guide

## Introduction

KNIRVNEXUS is a platform for validating SkillNodes and Base LLMs using Trusted Execution Environments (TEEs) and peer-to-peer (P2P) networking. This guide will walk you through the deployment and usage of KNIRVNEXUS, providing you with the necessary information to successfully use the platform.

## Prerequisites

Before you begin, ensure you have the following:

* A Kubernetes cluster (v1.25+)
* `kubectl` configured to access your cluster
* Docker or Podman
* A basic understanding of Kubernetes

## Installation and Deployment

### Step 1: Clone the Repository

Clone the KNIRV_NETWORK repository using the following command:

```bash
git clone https://github.com/knirv/KNIRV_NETWORK.git
cd KNIRV_NETWORK/KNIRVNEXUS
```

### Step 2: Build the Components

Build the KNIRVNEXUS components using the following command:

```bash
chmod +x scripts/build.sh
./scripts/build.sh
```

### Step 3: Deploy to Kubernetes

Deploy KNIRVNEXUS to your Kubernetes cluster using the following command:

```bash
chmod +x scripts/deploy.sh
./scripts/deploy.sh
```

### Step 4: Verify Deployment

Verify that KNIRVNEXUS has been successfully deployed by running the following command:

```bash
kubectl get pods -n knirv-nexus
```

You should see the KNIRVNEXUS pods running.

## Using KNIRVNEXUS

KNIRVNEXUS offers two operational modes: Headless (production) and GUI (local administration).

### Headless Mode (Production)

In headless mode, KNIRVNEXUS runs without a graphical user interface. Access is via the API (see API section below). This is the recommended mode for production deployments.

### GUI Mode (Local Development/Admin)

GUI mode provides a local web interface for administration and debugging. To start in GUI mode, run the following commands:

```bash
./dve-manager -gui
./validation-core -gui
```

Access the GUI at:

* DVE Manager: `http://localhost:9080`
* Validation Core: `http://localhost:9081`

## API Access

For production use, access KNIRVNEXUS APIs through the KNIRVGATEWAY at `/api/nexus/*`. For development or internal use, direct service access is available (see below). Authentication is required for production access (JWT).

**Example using KNIRVGATEWAY (Production):**

```bash
curl -X GET https://gateway.knirv.network/api/nexus/nodes
```

**Direct Service APIs (Development/Internal):**

* **DVE Manager (Port 8080):**  `/health`, `/api/v1/nodes`, `/api/v1/system/health`
* **Validation Core (Port 8081):** `/health`, `/api/v1/tasks`, `/api/v1/results`

(See the original README for detailed API examples.)

## Configuration

KNIRVNEXUS uses Viper for configuration management. Configuration can be set via CLI flags, environment variables, a YAML configuration file (`config/knirv-nexus.yaml`), or default values. The YAML file allows you to customize settings such as ports, database paths, and security options.

## Troubleshooting

### Common Issues and Solutions

* **Pod startup failures:** Check Kubernetes resource limits and node capacity.
* **P2P connectivity issues:** Verify firewall rules and port accessibility.
* **Database errors:** Check persistent volume availability.
* **Authentication failures:** Verify JWT secret configuration.

### Debugging Commands

Use these `kubectl` commands to troubleshoot:

* `kubectl get pods -n knirv-nexus -o wide` (Check pod status)
* `kubectl logs -f pod/<pod-name> -n knirv-nexus` (View pod logs)
* `kubectl get endpoints -n knirv-nexus` (Check service endpoints)
* `kubectl get events -n knirv-nexus --sort-by='.lastTimestamp'` (View events)

## Support

For support and questions, please create an issue on the GitHub repository or join the KNIRV Network community discussions.

Improvements Needed:

* The guide could benefit from a more detailed explanation of the prerequisites and the installation process.
* The API section could be expanded to include more examples and a clearer explanation of the available endpoints.
* The troubleshooting section could be improved by including more specific error messages and solutions.
* The guide could benefit from a more detailed explanation of the configuration options and how to customize them.
* The guide could benefit from a more detailed explanation of the debugging commands and how to use them.

<div class="footer-links">
<a href="#/legal/CODE_OF_CONDUCT.md" class="footer-link">Contributor Covenant Code of Conduct</a> | <a href="#/legal/PRIVACY_POLICY.md" class="footer-link">PRIVACY_POLICY.md</a> | <a href="#/legal/TERMS_AND_CONDITIONS.md" class="footer-link">TERMS AND CONDITIONS</a>

© 2025 KNIRV Network
</div>
