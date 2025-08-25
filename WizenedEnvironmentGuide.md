# KNIRV Wizened Environment Guide

### **Complete Guide: Creating a Pre-Initialized WASM Toolchain with Wizer**

This guide will walk you through creating a hybrid, high-performance server environment. We will dramatically reduce startup times by compiling the heavy, portable components of your toolchain (Go, Rust, Python) into a single, pre-initialized WebAssembly (WASM) module using Wizer.

This WASM module will then be executed by a Go application, which will leverage the server's pre-installed `npm` for Node.js-specific tasks. This "best of both worlds" approach combines the instant startup of WASM with the power of native host tools.

#### The Core Concept: A Hybrid Approach

Instead of a slow installation script that runs on every server start, we will:

1.  **Build Time:** Compile Go, Rust, and Python into a special `wasm32-wasi` format. We will pack these tools, along with a configuration script, into a Virtual File System (VFS) inside a single `.wasm` file.
2.  **Wizer Initialization:** Use Wizer to run our configuration script *once* during the build process. Wizer takes a snapshot of the module's memory after the setup is complete, saving the configured state.
3.  **Runtime:** On the server, a lightweight Go program will instantly load and run our wizened WASM module. This Go program will then call out to the host system's `npm` to handle the final `npm install` step.

This isolates the slow, repetitive parts of your setup, making your server boot almost instantaneously.

---

### **Phase 1: Preparing Your Toolchain (Local/CI Environment)**

This phase is about gathering your assets and preparing them to be packed into the WASM module.

#### Step 1: Create the Virtual File System (VFS) Staging Directory

This directory will mirror the filesystem that will exist inside your WASM module.

```bash
# Create a root directory for our VFS
mkdir -p wasi-vfs-root

# Create the directory structure inside the VFS
mkdir -p wasi-vfs-root/bin
mkdir -p wasi-vfs-root/lib
mkdir -p wasi-vfs-root/go-workspace
mkdir -p wasi-vfs-root/scripts

echo "VFS staging directory created."
```

#### Step 2: Acquire WASI-Compatible Binaries

You need versions of your tools compiled for WebAssembly. **You cannot use the standard Linux binaries.**

1.  **Python:** Download a pre-compiled `python.wasm`. The `python-wasi-shim` project is a great source.
    ```bash
    # Download a WASI-compatible Python binary
    wget https://github.com/vmware-labs/python-wasi-shim/releases/download/v0.1.0/python.wasm
    
    # Place it in your VFS's bin directory
    mv python.wasm wasi-vfs-root/bin/python
    ```    
2.  **Shell (bash):** A shell is needed to run your setup script inside WASM.
    ```bash
    # Download a WASI-compatible bash binary
    wget https://github.com/WebAssembly/wasi-website/raw/main/src/wasi/images/bash.wasm

    # Place it in your VFS's bin directory
    mv bash.wasm wasi-vfs-root/bin/bash
    ```
3.  **Go & Rust:** You don't need to put the compilers themselves into the VFS. For this guide, we assume you will use Go and Rust to compile your *actual application*, which will then be run by the host. The goal is to provide the *runtime* environment, not necessarily a full self-hosting compiler suite inside WASM (which is a far more complex task). We will configure the *environment* for them.

#### Step 3: Create the Wizer Setup Script

This is the replacement for your `install-dependencies.sh`. It **does not install anything**. It only sets up environment variables inside the WASM module, assuming the tools are already present in the VFS.

Create a file named `wasi-vfs-root/scripts/setup.sh`:
```bash
#!/bin/bash
# KNIRV Wizened Environment Setup Script
# This script is run by Wizer ONCE at build time.

echo "🔧 Configuring KNIRV Wizened Environment inside WASM..."

# Set up the environment variables for the pre-packed toolchains.
# These paths are relative to the root of the virtual filesystem.
export GOROOT="/usr/lib/go"
export GOPATH="/go-workspace"
export CARGO_HOME="/usr/lib/cargo"
export RUSTUP_HOME="/usr/lib/rustup"

# The most important part: Add our VFS bin to the PATH.
export PATH="/bin:/usr/lib/go/bin:/usr/lib/cargo/bin:$PATH"

echo "✅ WASM environment configured."
echo "PATH is now: $PATH"
```

---

### **Phase 2: Creating the Wizened WASM Module**

Now we will create the core `.wasm` file using a C launcher program and Wizer.

#### Step 4: Write the C Launcher

This small C program is the entry point. Wizer will execute its `init()` function, and the `main()` function will become the entry point for your actual Go application logic.

Create a file named `launcher.c`:
```c
#include <stdio.h>
#include <stdlib.h>

// This function is executed by Wizer at build time to snapshot the configured state.
void __attribute__((export_name("wizer.initialize"))) init() {
    printf("Wizer: Sourcing setup.sh to pre-configure environment...\n");
    // Execute the setup script from the virtual file system.
    system("/scripts/setup.sh");
}

// The main function. In our hybrid model, the actual Go program will provide
// this function when we link everything together. This is a placeholder.
int main() {
    // This function will be replaced by the Go application's main entry point.
    printf("WASM module initialized. Handing over to the Go host application...\n");
    return 0;
}
```

#### Step 5: Write the Main Go Application (The Host)

This is your actual server program. It will contain your API, web server logic, and will be responsible for orchestrating the WASM module and `npm`.

Create a file named `main.go`:

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
)

// This Go program will be compiled to WASI and linked with the C launcher.
// Its 'main' function will become the main entry point for the entire WASM module.

func main() {
	fmt.Println("🚀 KNIRV Go Application starting inside WASM...")

	// The environment variables from 'setup.sh' are already set thanks to Wizer.
	fmt.Printf("Go runtime sees GOROOT=%s\n", os.Getenv("GOROOT"))
	fmt.Printf("Go runtime sees PATH=%s\n", os.Getenv("PATH"))

	// Here, we would start our main application logic, like a web server.
	// NOTE: For a real server, you'd need networking support in your WASI runtime.
	// For this guide, we simulate the setup process.
	log.Println("Simulating server startup tasks...")

	// --- THIS IS A PLACEHOLDER FOR YOUR ACTUAL SERVER ---
	// For example, if you had an embedded frontend, you would start serving it here.
	// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	fmt.Fprintln(w, "Hello from the Wizened Go Server!")
	// })
	// log.Println("Starting web server on port 8080")
	// http.ListenAndServe(":8080", nil)
	// ----------------------------------------------------

	log.Println("✅ Go application setup is complete.")
	log.Println("The environment is ready.")
}

// This function demonstrates how the Go host would call out to the native npm.
// This code would live in your DEPLOYMENT script or a separate Go "runner" program,
// NOT inside the WASM module itself.
func runNpmInstallOnHost() {
	log.Println("Running 'npm install' on the host system...")
	cmd := exec.Command("npm", "install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Failed to run npm install: %v", err)
	}
	log.Println("✅ npm install completed successfully.")
}
```

#### Step 6: Compile and Pack Everything

This is the final step where we build the `.wasm` file. You will need the **WASI SDK** installed.

```bash
# 1. Compile the C launcher to a WASI object file
wasi-sdk/bin/clang --target=wasm32-wasi -c launcher.c -o launcher.o

# 2. Compile the Go application to a WASI object file
GOOS=wasip1 GOARCH=wasm go build -o main.o -buildmode=c-archive main.go

# 3. Link the C and Go objects, pack the VFS, and run Wizer
#    wasi-vfs packs the directory, links the objects, and invokes Wizer automatically.
wasi-vfs pack launcher.o main.o --mapdir /::wasi-vfs-root -o knirv-server.wasm

echo "✅ Successfully created wizened module: knirv-server.wasm"
```

---

### **Phase 3: Deployment to Render.com**

Now we deploy our hybrid application. We will use Render's Native Environment.

#### Step 7: Configure the Render Service

Create a `render.yaml` file in the root of your project. This file defines the build and start process.

```yaml
services:
  - type: web
    name: knirv-wizened-server
    env: native
    plan: free
    
    envVars:
      - key: NODE_VERSION
        value: "v18.18.0" # Specify the exact Node.js version you need
      - key: WASMTIME_VERSION
        value: v22.0.0

    buildCommand: |
      # This command runs once during deployment on the Render host
      echo "--- Building KNIRV Service ---"
      
      # 0. INSTALL NODE.JS (AND NPM)
      # This is the new, critical first step.
      echo "Installing Node.js ${NODE_VERSION}..."
      curl -L "https://nodejs.org/dist/${NODE_VERSION}/node-${NODE_VERSION}-linux-x64.tar.xz" | tar -xJ
      # Add the downloaded Node.js to the PATH for subsequent commands
      export PATH=$PWD/node-${NODE_VERSION}-linux-x64/bin:$PATH
      # Verify the installation
      node -v
      npm -v

      # 1. Install Host Dependencies (npm and Netlify CLI)
      echo "Installing host dependencies..."
      # Assuming Node.js/npm is available in Render's native env
      npm install
      npm install -g netlify-cli
      
      # 2. Build the Wizened WASM Module (commands from Step 6)
      echo "Compiling and wizening the WASM module..."
      # (Add commands to install Go/WASI SDK if needed first)
      wasi-sdk/bin/clang --target=wasm32-wasi -c launcher.c -o launcher.o
      GOOS=wasip1 GOARCH=wasm go build -o main.o -buildmode=c-archive main.go
      wasi-vfs pack launcher.o main.o --mapdir /::wasi-vfs-root -o knirv-server.wasm
      
      # 3. Download the Wasmtime Runtime
      echo "Downloading Wasmtime runtime..."
      curl -L https://github.com/bytecodealliance/wasmtime/releases/download/$WASMTIME_VERSION/wasmtime-$WASMTIME_VERSION-x86_64-linux.tar.xz | tar -xJ --strip-components=1

      https://github.com/bytecodealliance/wizer/releases/download/v9.0.0/wizer-v9.0.0-aarch64-linux.tar.xz
      
     
      echo "--- Build Complete ---"

    startCommand: |
      # This is the command that starts our long-running service
      echo "--- Starting KNIRV Service ---"
      ./wasmtime run --mapdir .:/app knirv-server.wasm
```

### **Summary of the Final Architecture**

1.  **Your Git Repository:** Contains your Go source code, `launcher.c`, the `wasi-vfs-root` directory, `package.json`, and `render.yaml`.
2.  **Render's Build Step:**
    *   Checks out your code.
    *   Runs the `buildCommand` from `render.yaml`.
    *   This compiles your Go and C code into `knirv-server.wasm`, which now contains a pre-configured environment with Python inside.
    *   It then runs `npm install` using the native `npm`.
    *   It downloads the `wasmtime` runtime.
3.  **Render's Start Step:**
    *   Executes the `startCommand`.
    *   `wasmtime` instantly loads `knirv-server.wasm`. The memory is already initialized by Wizer.
    *   Your Go `main()` function begins executing immediately, running your server logic inside the sandboxed, pre-configured WASM environment.


# The overall architecture remains the same powerful hybrid model, but now the build process is robust and correct.

| Component                 | Location                                 | Role & Justification                                                                                             |
| ------------------------- | ---------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| **Node.js + NPM**         |Downloaded and installed by `buildCommand`| **The Host Toolchain.** Provides the runtime for `npm` and `netlify-cli`. Installed first as a prerequisite.     |
| **Netlify CLI**           | Installed by `buildCommand` using `npm`  | **The Host Deployment Tool.** Needs full system access for network and auth. Runs during the build process.      |
| **`knirv-server.wasm`**   | Built by `buildCommand`                  | **The Core Environment.** Contains the wizened state for Go/Python. Boots instantly.                             |
| **`wasmtime`**            | Downloaded by `buildCommand`             | **The Executor.** A secure runtime that loads and runs the `.wasm` file to start the service.                    |
| **`render.yaml`**         | Your Git Repository                      | **The Orchestrator.** Defines the entire, correct sequence: Install Node -> Install Tools -> Build WASM -> Run.  |