# KNIRVSHELL User Guide

## Introduction

KNIRVSHELL is a command-line interface (CLI) for interacting with the KNIRV Network. It simplifies management of various KNIRV services, including blockchain operations, AI integration, and real-time communication.

## Getting Started

### Installation

KNIRVSHELL offers several installation methods:

**1. Pre-built Binaries (Recommended):**

1. Download the appropriate binary for your operating system from the [GitHub Releases page](https://github.com/guiperry/KNIRV_Network/releases).
2. Extract the archive.
3. Move the `knirv` binary to a directory included in your system's `PATH` environment variable.  (Instructions for Linux/macOS and Windows are provided below).
4. (Linux/macOS only) Make the binary executable using `chmod +x knirv`.
5. Verify installation with `knirv version`.

**Installation Instructions for Specific Platforms:**

* **Linux/macOS:**
	+ System-wide (requires `sudo`): `sudo mv knirv /usr/local/bin/`
	+ User-specific: `mkdir -p ~/.local/bin && mv knirv ~/.local/bin/ && echo 'export PATH=$PATH:~/.local/bin' >> ~/.bashrc && source ~/.bashrc` (or `.zshrc` for Zsh)
* **Windows:**
	1. Create a directory (e.g., `C:\Program Files\KnirvCLI`).
	2. Move `knirv.exe` to this directory.
	3. Add the directory to your `PATH` environment variable (search for "environment variables" in the Windows search bar).

**Alternative Installation Methods:**

* **Installation Scripts:** The release packages include installation scripts (`install.sh` for Linux/macOS, `install.ps1` for Windows) that automate the process. Run these scripts after extracting the archive.
* **Package Managers:** KNIRVSHELL is available through various package managers:
	+ **macOS (Homebrew):** `brew tap guiperry/knirv && brew install knirv`
	+ **Linux (Debian/Ubuntu):** (See README for detailed instructions involving adding a repository and installing)
	+ **Linux (Fedora/RHEL):** (See README for detailed instructions involving adding a repository and installing)
	+ **Windows (Chocolatey):** `choco install knirv`
	+ **Windows (Scoop):** `scoop bucket add guiperry https://github.com/guiperry/scoop-bucket.git && scoop install knirv`
* **Build from Source:** (For developers) Clone the repository, run `go mod tidy`, then `go build -o knirv`.

### Configuration

1. Copy the sample configuration: `cp config/sample.yaml ~/.knirv/config.yaml`
2. Edit `~/.knirv/config.yaml` to set API keys and service URLs.  You'll need API keys for the services you intend to use.
3. Optionally, set environment variables like `KNIRV_ROOT_API_KEY` for additional configuration.

## Basic Usage

### Command Mode

Run commands directly, for example: `knirv wallet list` or `knirv economics balance --wallet my-wallet`. Use `knirv help` for a list of commands.

### Interactive Shell Mode

Start an interactive shell with `knirv`. This provides tab completion and command history. Type `help` for assistance, and `exit` or `quit` to leave.

## Key Features and Commands

KNIRVSHELL integrates with several KNIRV Network services:

* **KNIRVROOT:** Blockchain operations (use `knirv system` commands).
* **KNIRVGATEWAY:** API gateway access (implicitly used by many commands).
* **KNIRVNEXUS:** DVE rental and inference API (use `knirv mcp` commands).
* **KNIRVGRAPH:** NRV system operations (use `knirv mcp nrv` commands).

**Wallet and Economics:** Manage NRN tokens using `knirv economics` commands (balance, transfer, faucet).

**NRV System:** Submit and query ErrorNodes and SkillNodes using `knirv mcp nrv` commands.

**MCP (Multi-Capability Provider):** Manage AI capabilities and servers using `knirv mcp` commands.

## Troubleshooting

* **Service Discovery Fails:** Check network connectivity, service URLs in the configuration file, and ensure services are running.
* **Authentication Errors:** Verify API keys and environment variables.
* **WebSocket Issues:** Check firewall settings and WebSocket endpoints.

Enable debug logging (`export LOG_LEVEL=debug`) for more detailed error messages.

## Further Information

Refer to the main KNIRV Network documentation for more advanced usage and details.

Improvements needed:

* Add more detailed explanations for each installation method.
* Provide clear instructions for setting up environment variables.
* Consider adding a section on common use cases and examples.
* Update the troubleshooting section to include more specific error messages and solutions.
* Consider adding a section on security best practices and recommendations.

<div class="footer-links">
<a href="#/legal/CODE_OF_CONDUCT.md" class="footer-link">Contributor Covenant Code of Conduct</a> | <a href="#/legal/PRIVACY_POLICY.md" class="footer-link">PRIVACY_POLICY.md</a> | <a href="#/legal/TERMS_AND_CONDITIONS.md" class="footer-link">TERMS AND CONDITIONS</a>

© 2025 KNIRV Network
</div>
