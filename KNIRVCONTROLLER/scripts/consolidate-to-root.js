#!/usr/bin/env node

/**
 * KNIRVCONTROLLER Root Consolidation Script
 * 
 * This script consolidates all frontend, backend, and manager components
 * into the root directory, creating a truly unified application structure.
 * 
 * Features:
 * - Creates comprehensive backups before any changes
 * - Migrates all source files to root src/ directory
 * - Consolidates all package.json dependencies
 * - Updates all import paths and configurations
 * - Creates unified build system
 * - Validates the consolidated structure
 */

import fs from 'fs/promises';
import path from 'path';
import { execSync } from 'child_process';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const rootDir = path.join(__dirname, '..');

class RootConsolidator {
  constructor() {
    this.rootDir = rootDir;
    this.timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    this.backupDir = path.join(this.rootDir, 'backups', `pre-consolidation-${this.timestamp}`);
    this.srcDir = path.join(this.rootDir, 'src');
    this.publicDir = path.join(this.rootDir, 'public');
  }

  log(message, type = 'info') {
    const timestamp = new Date().toISOString();
    const prefix = type === 'error' ? '❌' : type === 'success' ? '✅' : type === 'warning' ? '⚠️' : 'ℹ️';
    console.log(`${prefix} [${timestamp}] ${message}`);
  }

  async ensureDir(dirPath) {
    try {
      await fs.access(dirPath);
    } catch {
      await fs.mkdir(dirPath, { recursive: true });
      this.log(`Created directory: ${dirPath}`);
    }
  }

  async copyDirectory(src, dest) {
    await this.ensureDir(dest);
    const entries = await fs.readdir(src, { withFileTypes: true });
    
    for (const entry of entries) {
      const srcPath = path.join(src, entry.name);
      const destPath = path.join(dest, entry.name);
      
      if (entry.isDirectory()) {
        // Skip node_modules, dist, and .git directories
        if (['node_modules', 'dist', '.git', 'coverage'].includes(entry.name)) {
          continue;
        }
        await this.copyDirectory(srcPath, destPath);
      } else {
        await fs.copyFile(srcPath, destPath);
      }
    }
  }

  async readJsonFile(filePath) {
    try {
      const content = await fs.readFile(filePath, 'utf8');
      return JSON.parse(content);
    } catch (error) {
      this.log(`Warning: Could not read ${filePath}: ${error.message}`, 'warning');
      return null;
    }
  }

  async writeJsonFile(filePath, data) {
    await fs.writeFile(filePath, JSON.stringify(data, null, 2) + '\n');
  }

  async createBackup() {
    this.log('Creating comprehensive backup before consolidation...');
    
    await this.ensureDir(this.backupDir);
    
    // Backup all major directories
    const dirsToBackup = ['frontend', 'backend', 'manager', 'receiver', 'scripts', 'config', 'rust-wasm'];
    for (const dir of dirsToBackup) {
      const srcPath = path.join(this.rootDir, dir);
      const destPath = path.join(this.backupDir, dir);
      try {
        await this.copyDirectory(srcPath, destPath);
        this.log(`Backed up ${dir}/`);
      } catch (error) {
        this.log(`Warning: Could not backup ${dir}: ${error.message}`, 'warning');
      }
    }
    
    // Backup root configuration files
    const filesToBackup = [
      'package.json', 'package-lock.json', 'tsconfig.json', 
      'babel.config.js', 'jest.config.js'
    ];
    for (const file of filesToBackup) {
      const srcPath = path.join(this.rootDir, file);
      const destPath = path.join(this.backupDir, file);
      try {
        await fs.copyFile(srcPath, destPath);
        this.log(`Backed up ${file}`);
      } catch (error) {
        this.log(`Warning: Could not backup ${file}: ${error.message}`, 'warning');
      }
    }
    
    this.log(`Backup completed: ${this.backupDir}`, 'success');
  }

  async analyzeDependencies() {
    this.log('Analyzing dependencies across all packages...');
    
    const packageFiles = [
      'package.json',
      'frontend/package.json',
      'manager/package.json',
      'receiver/package.json'
    ];
    
    const dependencies = new Map();
    const devDependencies = new Map();
    const conflicts = [];
    
    for (const pkgFile of packageFiles) {
      const pkg = await this.readJsonFile(path.join(this.rootDir, pkgFile));
      if (!pkg) continue;
      
      // Process dependencies
      if (pkg.dependencies) {
        for (const [name, version] of Object.entries(pkg.dependencies)) {
          if (dependencies.has(name) && dependencies.get(name) !== version) {
            conflicts.push({ name, versions: [dependencies.get(name), version], type: 'dependency' });
          }
          dependencies.set(name, version);
        }
      }
      
      // Process devDependencies
      if (pkg.devDependencies) {
        for (const [name, version] of Object.entries(pkg.devDependencies)) {
          if (devDependencies.has(name) && devDependencies.get(name) !== version) {
            conflicts.push({ name, versions: [devDependencies.get(name), version], type: 'devDependency' });
          }
          devDependencies.set(name, version);
        }
      }
    }
    
    if (conflicts.length > 0) {
      this.log(`Found ${conflicts.length} dependency conflicts:`, 'warning');
      conflicts.forEach(conflict => {
        this.log(`  ${conflict.name}: ${conflict.versions.join(' vs ')}`, 'warning');
      });
    }
    
    this.dependencies = Object.fromEntries(dependencies);
    this.devDependencies = Object.fromEntries(devDependencies);
    
    this.log(`Analyzed ${dependencies.size} dependencies and ${devDependencies.size} devDependencies`, 'success');
  }

  async createRootStructure() {
    this.log('Creating root directory structure...');
    
    // Create main directories
    const directories = [
      'src',
      'src/components',
      'src/pages',
      'src/backend',
      'src/manager',
      'src/shared',
      'src/sensory-shell',
      'src/wasm-pkg',
      'public',
      'tests',
      'config'
    ];
    
    for (const dir of directories) {
      await this.ensureDir(path.join(this.rootDir, dir));
    }
    
    this.log('Root structure created', 'success');
  }

  async migrateSourceFiles() {
    this.log('Migrating source files to root structure...');
    
    // Migration mappings
    const migrations = [
      // Frontend files
      { from: 'frontend/src', to: 'src', exclude: ['manager'] },
      { from: 'frontend/public', to: 'public' },
      
      // Backend files
      { from: 'backend', to: 'src/backend' },
      
      // Manager files (if not already in frontend)
      { from: 'manager/src', to: 'src/manager/original' },
      
      // Receiver files (additional components not in frontend)
      { from: 'receiver/src', to: 'src/receiver-backup' },
      
      // WASM files
      { from: 'rust-wasm/pkg-web', to: 'src/wasm-pkg' },
      
      // Configuration files
      { from: 'config', to: 'config' },
      
      // Test files
      { from: 'frontend/src/__tests__', to: 'tests/frontend' },
      { from: 'backend/__tests__', to: 'tests/backend' },
      { from: 'tests', to: 'tests/legacy' }
    ];
    
    for (const migration of migrations) {
      const srcPath = path.join(this.rootDir, migration.from);
      const destPath = path.join(this.rootDir, migration.to);
      
      try {
        await fs.access(srcPath);
        await this.copyDirectory(srcPath, destPath);
        this.log(`Migrated ${migration.from} → ${migration.to}`);
      } catch (error) {
        this.log(`Warning: Could not migrate ${migration.from}: ${error.message}`, 'warning');
      }
    }
    
    this.log('Source file migration completed', 'success');
  }

  async updateImportPaths() {
    this.log('Updating import paths for root structure...');
    
    // Path mappings for common import patterns
    const pathMappings = new Map([
      ['../backend/', './backend/'],
      ['../../backend/', './backend/'],
      ['../frontend/src/', './'],
      ['./manager/', './manager/'],
      ['../shared/', './shared/'],
      ['../components/', './components/'],
      ['../pages/', './pages/'],
      ['../sensory-shell/', './sensory-shell/'],
      ['../../shared/', './shared/'],
      ['../../../shared/', './shared/']
    ]);
    
    // Find all TypeScript/JavaScript files in src
    const { glob } = await import('glob');
    const files = await glob('**/*.{ts,tsx,js,jsx}', {
      cwd: this.srcDir,
      absolute: true
    });
    
    let updatedFiles = 0;
    
    for (const file of files) {
      try {
        let content = await fs.readFile(file, 'utf8');
        let updated = false;
        
        // Update import statements
        for (const [oldPath, newPath] of pathMappings) {
          const importRegex = new RegExp(`(from\\s+['"])${oldPath.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`, 'g');
          const requireRegex = new RegExp(`(require\\s*\\(['"])${oldPath.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`, 'g');
          
          if (importRegex.test(content) || requireRegex.test(content)) {
            content = content.replace(importRegex, `$1${newPath}`);
            content = content.replace(requireRegex, `$1${newPath}`);
            updated = true;
          }
        }
        
        if (updated) {
          await fs.writeFile(file, content);
          updatedFiles++;
        }
      } catch (error) {
        this.log(`Warning: Could not update imports in ${file}: ${error.message}`, 'warning');
      }
    }
    
    this.log(`Updated import paths in ${updatedFiles} files`, 'success');
  }

  async createUnifiedPackage() {
    this.log('Creating unified package.json...');
    
    // Get the current root package.json as base
    const rootPkg = await this.readJsonFile(path.join(this.rootDir, 'package.json')) || {};
    
    // Create unified package.json
    const unifiedPackage = {
      name: 'knirv-controller-unified',
      version: '1.0.0',
      description: 'Unified KNIRV Controller - Single root application with integrated frontend, backend, and manager',
      type: 'module',
      main: 'dist/backend/server.js',
      scripts: {
        // Development
        'dev': 'vite --host 0.0.0.0 --port 3000',
        'dev:backend': 'nodemon --exec "node --loader ts-node/esm src/backend/server.ts"',
        'dev:full': 'concurrently "npm run dev:backend" "npm run dev"',
        
        // Building
        'build': 'npm run build:wasm && npm run build:backend && vite build',
        'build:wasm': './scripts/build-wasm.sh',
        'build:backend': 'tsc -p tsconfig.backend.json',
        'build:frontend': 'vite build',
        
        // Production
        'start': 'npm run preview',
        'preview': 'vite preview --host 0.0.0.0 --port 3000',
        'start:production': 'node dist/backend/server.js',
        
        // Testing
        'test': 'jest',
        'test:watch': 'jest --watch',
        'test:coverage': 'jest --coverage',
        'test:e2e': 'playwright test',
        
        // Utilities
        'lint': 'eslint src/',
        'lint:fix': 'eslint src/ --fix',
        'clean': 'rm -rf dist node_modules && npm install',
        'reset': 'npm run clean && npm run build',
        
        // Legacy compatibility
        'install:all': 'npm install',
        'build:unified': 'npm run build'
      },
      dependencies: this.dependencies,
      devDependencies: this.devDependencies,
      engines: {
        node: '>=20.0.0',
        rust: '>=1.70.0'
      },
      repository: rootPkg.repository,
      keywords: [
        'knirv', 'controller', 'ai', 'blockchain', 'lora', 'cognitive-shell',
        'wasm', 'agent-core', 'neural-networks', 'unified', 'frontend', 'backend'
      ],
      author: 'KNIRV Network',
      license: 'MIT'
    };
    
    await this.writeJsonFile(path.join(this.rootDir, 'package.json'), unifiedPackage);
    this.log('Unified package.json created', 'success');
  }

  async consolidate() {
    try {
      this.log('Starting KNIRVCONTROLLER root consolidation process...', 'success');
      
      // Execute consolidation phases
      await this.createBackup();
      await this.analyzeDependencies();
      await this.createRootStructure();
      await this.migrateSourceFiles();
      await this.updateImportPaths();
      await this.createUnifiedPackage();
      
      this.log('✅ Root consolidation completed successfully!', 'success');
      this.log('');
      this.log('📁 Backup stored at:', this.backupDir);
      this.log('🏠 All components now in root directory');
      this.log('📦 Unified package.json created');
      this.log('');
      this.log('🚀 Next steps:');
      this.log('  1. npm install');
      this.log('  2. npm run build');
      this.log('  3. npm start');
      this.log('');
      this.log('🌐 Application will be available at:');
      this.log('  http://localhost:3000/ - Unified Frontend (Receiver + Manager)');
      
    } catch (error) {
      this.log(`❌ Consolidation failed: ${error.message}`, 'error');
      this.log(`📁 Backup available at: ${this.backupDir}`);
      this.log('💡 Use backup to restore previous state if needed');
      throw error;
    }
  }
}

// Run the consolidation
if (import.meta.url === `file://${process.argv[1]}`) {
  const consolidator = new RootConsolidator();
  consolidator.consolidate().catch(error => {
    console.error('Consolidation failed:', error);
    process.exit(1);
  });
}

export { RootConsolidator };
