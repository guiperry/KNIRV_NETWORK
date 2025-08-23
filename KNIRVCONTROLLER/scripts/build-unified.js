#!/usr/bin/env node

/**
 * Unified Build Script for KNIRV-CONTROLLER
 * Builds all components and creates unified distribution
 */

import { spawn } from 'child_process';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
import fs from 'fs/promises';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const rootDir = join(__dirname, '..');

class UnifiedBuilder {
  constructor() {
    this.buildOrder = ['receiver', 'manager', 'cli'];
    this.distDir = join(rootDir, 'dist');
  }

  async run(command, args, options = {}) {
    return new Promise((resolve, reject) => {
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

  async buildComponent(component) {
    console.log(`\n🔨 Building ${component}...`);
    const componentPath = join(rootDir, component);

    switch (component) {
      case 'receiver':
        await this.run('npm', ['run', 'build'], { cwd: componentPath });
        break;
      case 'manager':
        await this.run('npm', ['run', 'build'], { cwd: componentPath });
        break;
      case 'cli':
        await this.run('go', ['build', '-o', '../bin/knirv-cli', 'main.go'], { cwd: componentPath });
        break;
      default:
        throw new Error(`Unknown component: ${component}`);
    }

    console.log(`✅ ${component} built successfully`);
  }

  async prepareDist() {
    console.log('\n📁 Preparing distribution directory...');
    
    try {
      await fs.rm(this.distDir, { recursive: true, force: true });
    } catch (error) {
      // Directory might not exist
    }

    await fs.mkdir(this.distDir, { recursive: true });
    await fs.mkdir(join(this.distDir, 'components'), { recursive: true });
    await fs.mkdir(join(this.distDir, 'bin'), { recursive: true });
    await fs.mkdir(join(this.distDir, 'config'), { recursive: true });
  }

  async copyBuilds() {
    console.log('\n📋 Copying build artifacts...');

    // Copy receiver build
    const receiverDist = join(rootDir, 'receiver', 'dist');
    const receiverTarget = join(this.distDir, 'components', 'receiver');
    await this.copyDirectory(receiverDist, receiverTarget);

    // Copy manager build
    const managerDist = join(rootDir, 'manager', 'dist');
    const managerTarget = join(this.distDir, 'components', 'manager');
    await this.copyDirectory(managerDist, managerTarget);

    // Copy CLI binary
    const cliBinary = join(rootDir, 'bin', 'knirv-cli');
    const cliTarget = join(this.distDir, 'bin', 'knirv-cli');
    await fs.copyFile(cliBinary, cliTarget);
    await fs.chmod(cliTarget, '755');

    // Copy configuration
    const configSource = join(rootDir, 'config');
    const configTarget = join(this.distDir, 'config');
    await this.copyDirectory(configSource, configTarget);

    // Copy orchestrator
    const orchestratorSource = join(rootDir, 'scripts', 'orchestrator.js');
    const orchestratorTarget = join(this.distDir, 'orchestrator.js');
    await fs.copyFile(orchestratorSource, orchestratorTarget);
  }

  async copyDirectory(source, target) {
    try {
      await fs.mkdir(target, { recursive: true });
      const entries = await fs.readdir(source, { withFileTypes: true });

      for (const entry of entries) {
        const sourcePath = join(source, entry.name);
        const targetPath = join(target, entry.name);

        if (entry.isDirectory()) {
          await this.copyDirectory(sourcePath, targetPath);
        } else {
          await fs.copyFile(sourcePath, targetPath);
        }
      }
    } catch (error) {
      console.warn(`Warning: Could not copy ${source} to ${target}:`, error.message);
    }
  }

  async createProductionPackage() {
    console.log('\n📦 Creating production package...');

    const packageJson = {
      name: 'knirv-controller-dist',
      version: '1.0.0',
      description: 'KNIRV Controller Production Distribution',
      main: 'orchestrator.js',
      scripts: {
        start: 'node orchestrator.js',
        'start:production': 'NODE_ENV=production node orchestrator.js'
      },
      dependencies: {
        express: '^4.18.2',
        ws: '^8.14.2',
        cors: '^2.8.5',
        helmet: '^7.1.0',
        compression: '^1.7.4'
      },
      engines: {
        node: '>=20.0.0'
      }
    };

    await fs.writeFile(
      join(this.distDir, 'package.json'),
      JSON.stringify(packageJson, null, 2)
    );

    // Create startup script
    const startupScript = `#!/bin/bash
# KNIRV-CONTROLLER Production Startup Script

echo "Starting KNIRV-CONTROLLER..."

# Check Node.js version
NODE_VERSION=$(node --version | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt "20" ]; then
    echo "Error: Node.js 20+ required. Current version: $(node --version)"
    exit 1
fi

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install --production
fi

# Start the orchestrator
echo "Starting orchestrator..."
NODE_ENV=production node orchestrator.js
`;

    await fs.writeFile(join(this.distDir, 'start.sh'), startupScript);
    await fs.chmod(join(this.distDir, 'start.sh'), '755');

    // Create README
    const readme = `# KNIRV-CONTROLLER Production Distribution

## Quick Start

1. Install dependencies:
   \`\`\`bash
   npm install
   \`\`\`

2. Start the system:
   \`\`\`bash
   ./start.sh
   \`\`\`

   Or manually:
   \`\`\`bash
   node orchestrator.js
   \`\`\`

## Components

- **Manager**: Mobile-first interface (port 3001)
- **Receiver**: Cognitive shell interface (port 3002)  
- **CLI**: Interactive terminal (port 3003)
- **Orchestrator**: Component coordination (port 3000)

## Configuration

Edit \`config/unified-config.json\` to customize component settings.

## Monitoring

- Health check: http://localhost:3000/health
- Component status: WebSocket connection to ws://localhost:3000

## Requirements

- Node.js 20+
- 4GB+ RAM (for neural network operations)
- Modern browser with WebAssembly support
`;

    await fs.writeFile(join(this.distDir, 'README.md'), readme);
  }

  async build() {
    console.log('🚀 Starting unified build process...');

    try {
      await this.prepareDist();

      // Build components in order
      for (const component of this.buildOrder) {
        await this.buildComponent(component);
      }

      await this.copyBuilds();
      await this.createProductionPackage();

      console.log('\n✅ Unified build completed successfully!');
      console.log(`📁 Distribution created in: ${this.distDir}`);
      console.log('\n🚀 To run the production build:');
      console.log(`   cd ${this.distDir}`);
      console.log('   npm install');
      console.log('   ./start.sh');

    } catch (error) {
      console.error('\n❌ Build failed:', error.message);
      process.exit(1);
    }
  }
}

// Run the build
const builder = new UnifiedBuilder();
builder.build();
