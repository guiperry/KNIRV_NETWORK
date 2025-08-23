#!/usr/bin/env node

/**
 * Backend Build Script for KNIRV-CORTEX
 * Builds the backend-only version with WASM compilation pipeline
 */

import { spawn } from 'child_process';
import { promises as fs } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const rootDir = join(__dirname, '..');

class BackendBuilder {
  constructor() {
    this.distDir = join(rootDir, 'dist');
    this.backendDir = join(rootDir, 'backend');
  }

  async run(command, args, options = {}) {
    return new Promise((resolve, reject) => {
      console.log(`Running: ${command} ${args.join(' ')}`);
      const process = spawn(command, args, {
        stdio: 'inherit',
        ...options
      });

      process.on('close', (code) => {
        if (code === 0) {
          resolve(code);
        } else {
          reject(new Error(`Command failed with code ${code}`));
        }
      });

      process.on('error', reject);
    });
  }

  async build() {
    console.log('🔨 Building KNIRV-CORTEX Backend...');

    try {
      // Clean previous build
      await this.clean();

      // Compile TypeScript
      await this.compileTypeScript();

      // Copy necessary files
      await this.copyFiles();

      // Create package.json for distribution
      await this.createDistPackage();

      console.log('✅ Backend build completed successfully!');
      console.log(`📁 Distribution created in: ${this.distDir}`);

    } catch (error) {
      console.error('❌ Backend build failed:', error.message);
      process.exit(1);
    }
  }

  async clean() {
    console.log('🧹 Cleaning previous build...');
    try {
      await fs.rm(this.distDir, { recursive: true, force: true });
    } catch (error) {
      // Directory might not exist
    }
    await fs.mkdir(this.distDir, { recursive: true });
  }

  async compileTypeScript() {
    console.log('📝 Compiling TypeScript...');
    
    // Create tsconfig for backend compilation
    const tsConfig = {
      compilerOptions: {
        target: 'ES2022',
        module: 'ESNext',
        moduleResolution: 'node',
        outDir: './dist',
        rootDir: './backend',
        strict: true,
        esModuleInterop: true,
        skipLibCheck: true,
        forceConsistentCasingInFileNames: true,
        declaration: true,
        declarationMap: true,
        sourceMap: true,
        resolveJsonModule: true,
        allowSyntheticDefaultImports: true
      },
      include: ['backend/**/*'],
      exclude: ['node_modules', 'dist', 'src', 'coverage', 'test-reports']
    };

    const tsConfigPath = join(rootDir, 'tsconfig.build.json');
    await fs.writeFile(tsConfigPath, JSON.stringify(tsConfig, null, 2));

    await this.run('npx', ['tsc', '--project', 'tsconfig.build.json'], {
      cwd: rootDir
    });

    // Clean up temporary tsconfig
    await fs.unlink(tsConfigPath);
  }

  async copyFiles() {
    console.log('📋 Copying necessary files...');

    // Copy protobuf schemas
    const protoSrcDir = join(this.backendDir, 'protobuf');
    const protoDistDir = join(this.distDir, 'protobuf');
    await this.copyDirectory(protoSrcDir, protoDistDir);

    // Copy agent-core-compiler templates
    const templatesSrcDir = join(this.backendDir, 'agent-core-compiler', 'templates');
    const templatesDistDir = join(this.distDir, 'agent-core-compiler', 'templates');
    console.log('📋 Copying agent-core-compiler templates...');
    await this.copyDirectory(templatesSrcDir, templatesDistDir);

    // Copy unified server files
    console.log('📋 Copying unified server...');
    const unifiedServerSrc = join(this.backendDir, 'unifiedServer.ts');
    const unifiedServerDist = join(this.distDir, 'unifiedServer.js');
    // The TypeScript compiler will handle this, but we ensure it's included

    // Copy WASM build scripts
    const scriptsDir = join(rootDir, 'scripts');
    const distScriptsDir = join(this.distDir, 'scripts');
    await fs.mkdir(distScriptsDir, { recursive: true });
    
    const scriptFiles = ['build-wasm.sh', 'build-lora-pipeline.sh'];
    for (const file of scriptFiles) {
      try {
        await fs.copyFile(join(scriptsDir, file), join(distScriptsDir, file));
        await fs.chmod(join(distScriptsDir, file), '755');
      } catch (error) {
        console.warn(`Warning: Could not copy ${file}`);
      }
    }

    // Copy rust-wasm directory
    const rustWasmSrc = join(rootDir, 'rust-wasm');
    const rustWasmDist = join(this.distDir, 'rust-wasm');
    await this.copyDirectory(rustWasmSrc, rustWasmDist);

    // Copy README and documentation
    try {
      await fs.copyFile(join(rootDir, 'README.md'), join(this.distDir, 'README.md'));
    } catch (error) {
      console.warn('Warning: Could not copy README.md');
    }
  }

  async copyDirectory(source, target) {
    try {
      await fs.mkdir(target, { recursive: true });
      const entries = await fs.readdir(source, { withFileTypes: true });

      for (const entry of entries) {
        const sourcePath = join(source, entry.name);
        const targetPath = join(target, entry.name);

        if (entry.isDirectory()) {
          // Skip node_modules and target directories
          if (entry.name === 'node_modules' || entry.name === 'target') {
            continue;
          }
          await this.copyDirectory(sourcePath, targetPath);
        } else {
          await fs.copyFile(sourcePath, targetPath);
        }
      }
    } catch (error) {
      console.warn(`Warning: Could not copy ${source} to ${target}:`, error.message);
    }
  }

  async createDistPackage() {
    console.log('📦 Creating distribution package.json...');

    const packageJson = {
      name: 'knirv-cortex-backend',
      version: '1.0.0',
      description: 'KNIRV-CORTEX Backend - LoRA Adapter Processing Engine',
      type: 'module',
      main: 'index.js',
      scripts: {
        start: 'node index.js',
        'start:production': 'NODE_ENV=production node index.js',
        'build:wasm': './scripts/build-wasm.sh',
        'test:wasm': './scripts/test-wasm.sh'
      },
      dependencies: {
        '@napi-rs/wasm-runtime': '^1.0.3',
        'protobufjs': '^7.2.5',
        'express': '^4.18.2',
        'ws': '^8.14.2',
        'cors': '^2.8.5',
        'helmet': '^7.1.0',
        'compression': '^1.7.4',
        'pino': '^8.16.2',
        'pino-pretty': '^10.2.3'
      },
      engines: {
        node: '>=18.0.0'
      },
      keywords: [
        'knirv',
        'cortex',
        'lora',
        'adapters',
        'wasm',
        'backend'
      ],
      author: 'KNIRV Network',
      license: 'MIT'
    };

    await fs.writeFile(
      join(this.distDir, 'package.json'),
      JSON.stringify(packageJson, null, 2)
    );

    // Create startup script
    const startupScript = `#!/bin/bash
# KNIRV-CORTEX Backend Startup Script

echo "Starting KNIRV-CORTEX Backend..."

# Check Node.js version
NODE_VERSION=$(node --version | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt "18" ]; then
    echo "Error: Node.js 18+ required. Current version: $(node --version)"
    exit 1
fi

# Check if Rust toolchain is available
if ! command -v rustc &> /dev/null; then
    echo "Warning: Rust toolchain not found. WASM compilation will not be available."
    echo "Install Rust from: https://rustup.rs/"
fi

# Check if wasm-pack is available
if ! command -v wasm-pack &> /dev/null; then
    echo "Warning: wasm-pack not found. WASM compilation will not be available."
    echo "Install wasm-pack: curl https://rustwasm.github.io/wasm-pack/installer/init.sh -sSf | sh"
fi

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install --production
fi

# Start the backend
echo "Starting KNIRV-CORTEX backend..."
NODE_ENV=production node index.js
`;

    await fs.writeFile(join(this.distDir, 'start.sh'), startupScript);
    await fs.chmod(join(this.distDir, 'start.sh'), '755');

    // Create README for distribution
    const readme = `# KNIRV-CORTEX Backend

Revolutionary LoRA Adapter Processing Engine - Backend Only

## Overview

This is the backend-only version of KNIRV-CORTEX that implements the revolutionary concept where **skills ARE LoRA adapters** containing weights and biases that directly modify agent-core neural network behavior.

## Key Features

- **LoRA Adapter Compilation**: Convert solutions+errors from KNIRVGRAPH into neural network weights
- **Skill Invocation**: Load and apply LoRA adapters to agent-cores via protobuf serialization
- **WASM Compilation**: Compile Rust code to WebAssembly for embedded execution
- **Protobuf Serialization**: Efficient serialization of LoRA adapters and skill responses

## Quick Start

1. Install dependencies:
   \`\`\`bash
   npm install
   \`\`\`

2. Start the backend:
   \`\`\`bash
   ./start.sh
   \`\`\`

   Or manually:
   \`\`\`bash
   node index.js
   \`\`\`

## API Endpoints

- **POST /lora/compile** - Compile LoRA adapter from skill data
- **POST /lora/invoke** - Invoke LoRA adapter (returns protobuf)
- **POST /wasm/compile** - Compile WASM module
- **GET /health** - Health check
- **GET /api/status** - System status

## Requirements

- Node.js 18+
- Rust toolchain (for WASM compilation)
- wasm-pack (for WebAssembly builds)

## Architecture

This backend implements the revolutionary transformation where:
1. **Solutions + Errors** → **Training Data**
2. **Training Data** → **LoRA Weights & Biases**
3. **LoRA Adapters** → **Skills**
4. **Skills** → **Agent-Core Modifications**

## Configuration

Set environment variables:
- \`PORT\` - Server port (default: 3004)
- \`LOG_LEVEL\` - Logging level (default: info)
- \`NODE_ENV\` - Environment (development/production)
`;

    await fs.writeFile(join(this.distDir, 'README.md'), readme);
  }
}

// Run the build
const builder = new BackendBuilder();
builder.build();
