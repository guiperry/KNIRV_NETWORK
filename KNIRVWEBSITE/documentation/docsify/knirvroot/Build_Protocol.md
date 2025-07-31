

---

**Source**: KNIRVROOT/docs/protocols/Build_Protocol.md

# KNIRVCHAIN Build System

This document describes the build system for KNIRVCHAIN, including role-specific builds and cross-compilation.

## Node Roles

KNIRVCHAIN supports multiple node roles, each with specific functionality:

- **Root**: A root node that initializes a new chain
- **Bootnode**: A bootnode that helps with dev discovery
- **Developer** (Peer): A dev node that participates in the network
- **Client**: A client-only node with reduced functionality

## Role-Specific Builds

KNIRVCHAIN now supports role-specific builds using Go build tags. This allows for creating binaries that are optimized for specific roles without requiring runtime configuration.

### Building for Specific Roles

You can build KNIRVCHAIN for specific roles using the Makefile targets:

```bash
# Build for client role (default)
make build

# Build for root role
make build/root

# Build for bootnode role
make build/bootnode

# Build for developer role
make build/developer

# Build for all roles
make build/all-roles
```

### Cross-Compilation

You can cross-compile KNIRVCHAIN for multiple platforms and roles:

```bash
# Cross-compile for all platforms (client role)
make build/all

# Cross-compile for all platforms (root role)
make build/all/root

# Cross-compile for all platforms (bootnode role)
make build/all/bootnode

# Cross-compile for all platforms (developer role)
make build/all/developer

# Cross-compile for all platforms and all roles
make build/all-platforms-all-roles
```

### Manual Building

You can also build manually using Go build tags:

```bash
# Build for client role
go build -tags=client -o KNIRVCHAIN

# Build for root role
go build -tags=root -o KNIRVCHAIN_root

# Build for bootnode role
go build -tags=bootnode -o KNIRVCHAIN_bootnode

# Build for developer role
go build -tags=developer -o KNIRVCHAIN_developer
```

## User Interfaces

KNIRVCHAIN supports multiple user interfaces:

- **Terminal UI**: A text-based interface using Bubbletea
- **Fyne GUI**: A graphical interface using the Fyne toolkit (Windows)
- **WebView GUI**: A web-based interface using WebView (macOS, Linux)

The terminal UI is enabled by default for all roles but can be disabled with the `--no-terminal` flag.

## Command-Line Flags

KNIRVCHAIN supports the following command-line flags:

- `--terminal`: Enable the terminal UI (if not already enabled by default)
- `--no-terminal`: Disable the terminal UI
- `--gui`: Enable the graphical UI
- `--role=<role>`: Override the role (Root, Bootnode, Peer, Client)
- `--root`: Run as a root node
- `--bootnode`: Run as a bootnode
- `--dev`: Run as a dev node
- `--client-only`: Run as a client-only node

## Build System Implementation

The build system is implemented using:

1. **Role-specific entry point files**:
   - `main_root.go`: Entry point for Root nodes
   - `main_bootnode.go`: Entry point for Bootnode nodes
   - `main_developer.go`: Entry point for Developer (Peer) nodes
   - `main_client.go`: Entry point for Client nodes

2. **Makefile targets** for different roles and platforms

3. **Cross-compilation script** (`scripts/cross-compile.sh`) that supports role-specific builds

## Advantages of Role-Specific Builds

1. **Reduced binary size**: Each binary only includes the code needed for its specific role
2. **Simplified configuration**: No need to configure the role at runtime
3. **Improved security**: Root nodes can be built separately with additional security measures
4. **Better user experience**: Each binary is optimized for its specific use case
5. **Clearer deployment**: Easier to deploy the right binary for each role

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
