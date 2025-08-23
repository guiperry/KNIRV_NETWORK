#!/usr/bin/env node

/**
 * CLI Synchronization Script
 * Synchronizes KNIRVCONTROLLER/cli with KNIRVSDK/cli for consistent functionality
 */

import { spawn } from 'child_process';
import { promises as fs } from 'fs';
import { join, dirname, relative } from 'path';
import { fileURLToPath } from 'url';
import crypto from 'crypto';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const rootDir = join(__dirname, '../..');

class CLISynchronizer {
  constructor() {
    this.controllerCLI = join(rootDir, 'KNIRVCONTROLLER', 'cli');
    this.sdkCLI = join(rootDir, 'KNIRVSDK', 'cli');
    this.syncConfig = {
      // Files to sync from CONTROLLER to SDK
      controllerToSDK: [
        'cmd/agent.go',
        'cmd/network.go',
        'cmd/system.go',
        'core/health_monitor.go',
        'core/service_registry.go',
        'ui/components/',
        'ui/screens/'
      ],
      // Files to sync from SDK to CONTROLLER
      sdkToController: [
        'cmd/economics.go',
        'cmd/wallet.go',
        'core/wallet_manager.go',
        'core/nrn_token_manager.go',
        'core/xion_wallet_manager.go',
        'pkg/inference/',
        'pkg/tui/'
      ],
      // Files to keep in sync (bidirectional)
      bidirectional: [
        'cmd/root.go',
        'cmd/utils.go',
        'config/config.go',
        'config/defaults.go',
        'core/api_client.go',
        'core/event_bus.go',
        'core/file_manager.go'
      ],
      // Files to exclude from sync
      exclude: [
        'go.mod',
        'go.sum',
        'main.go',
        'README.md',
        'Dockerfile',
        'Makefile',
        'bin/',
        'test/',
        'docs/',
        'release_assets/',
        'knirv'
      ]
    };
  }

  async run(command, args, options = {}) {
    return new Promise((resolve, reject) => {
      const process = spawn(command, args, {
        stdio: 'pipe',
        ...options
      });

      let stdout = '';
      let stderr = '';

      process.stdout.on('data', (data) => {
        stdout += data.toString();
      });

      process.stderr.on('data', (data) => {
        stderr += data.toString();
      });

      process.on('close', (code) => {
        if (code === 0) {
          resolve({ stdout, stderr });
        } else {
          reject(new Error(`Command failed with code ${code}: ${stderr}`));
        }
      });

      process.on('error', reject);
    });
  }

  async sync(direction = 'both') {
    console.log('🔄 Starting CLI synchronization...');
    console.log(`📁 Controller CLI: ${this.controllerCLI}`);
    console.log(`📁 SDK CLI: ${this.sdkCLI}`);

    try {
      // Verify both directories exist
      await this.verifyDirectories();

      // Create backup before sync
      await this.createBackup();

      // Perform synchronization based on direction
      switch (direction) {
        case 'controller-to-sdk':
          await this.syncControllerToSDK();
          break;
        case 'sdk-to-controller':
          await this.syncSDKToController();
          break;
        case 'both':
        default:
          await this.syncBidirectional();
          await this.syncControllerToSDK();
          await this.syncSDKToController();
          break;
      }

      // Update version tracking
      await this.updateVersionTracking();

      // Validate sync
      await this.validateSync();

      console.log('✅ CLI synchronization completed successfully!');

    } catch (error) {
      console.error('❌ CLI synchronization failed:', error.message);
      throw error;
    }
  }

  async verifyDirectories() {
    console.log('🔍 Verifying CLI directories...');

    try {
      await fs.access(this.controllerCLI);
      await fs.access(this.sdkCLI);
    } catch (error) {
      throw new Error('CLI directories not found. Ensure both KNIRVCONTROLLER/cli and KNIRVSDK/cli exist.');
    }

    console.log('✅ CLI directories verified');
  }

  async createBackup() {
    console.log('💾 Creating backup...');

    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const backupDir = join(rootDir, 'backups', `cli-sync-${timestamp}`);

    await fs.mkdir(backupDir, { recursive: true });

    // Backup both CLI directories
    await this.copyDirectory(this.controllerCLI, join(backupDir, 'controller-cli'));
    await this.copyDirectory(this.sdkCLI, join(backupDir, 'sdk-cli'));

    console.log(`✅ Backup created: ${backupDir}`);
  }

  async syncControllerToSDK() {
    console.log('📤 Syncing Controller CLI → SDK CLI...');

    for (const file of this.syncConfig.controllerToSDK) {
      await this.syncFile(this.controllerCLI, this.sdkCLI, file, 'controller-to-sdk');
    }
  }

  async syncSDKToController() {
    console.log('📥 Syncing SDK CLI → Controller CLI...');

    for (const file of this.syncConfig.sdkToController) {
      await this.syncFile(this.sdkCLI, this.controllerCLI, file, 'sdk-to-controller');
    }
  }

  async syncBidirectional() {
    console.log('🔄 Syncing bidirectional files...');

    for (const file of this.syncConfig.bidirectional) {
      // Compare timestamps and sync the newer version
      const controllerPath = join(this.controllerCLI, file);
      const sdkPath = join(this.sdkCLI, file);

      try {
        const controllerStat = await fs.stat(controllerPath);
        const sdkStat = await fs.stat(sdkPath);

        if (controllerStat.mtime > sdkStat.mtime) {
          await this.syncFile(this.controllerCLI, this.sdkCLI, file, 'bidirectional');
          console.log(`  📤 ${file} (Controller → SDK)`);
        } else if (sdkStat.mtime > controllerStat.mtime) {
          await this.syncFile(this.sdkCLI, this.controllerCLI, file, 'bidirectional');
          console.log(`  📥 ${file} (SDK → Controller)`);
        } else {
          console.log(`  ✅ ${file} (already in sync)`);
        }
      } catch (error) {
        console.warn(`  ⚠️  ${file} (file not found in one location)`);
      }
    }
  }

  async syncFile(sourceDir, targetDir, filePath, direction) {
    const sourcePath = join(sourceDir, filePath);
    const targetPath = join(targetDir, filePath);

    try {
      const stat = await fs.stat(sourcePath);

      if (stat.isDirectory()) {
        await this.copyDirectory(sourcePath, targetPath);
      } else {
        // Ensure target directory exists
        await fs.mkdir(dirname(targetPath), { recursive: true });
        await fs.copyFile(sourcePath, targetPath);
      }

      console.log(`  ✅ ${filePath}`);
    } catch (error) {
      console.warn(`  ⚠️  ${filePath} (${error.message})`);
    }
  }

  async copyDirectory(source, target) {
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
  }

  async updateVersionTracking() {
    console.log('📝 Updating version tracking...');

    const versionInfo = {
      timestamp: new Date().toISOString(),
      syncedFiles: [
        ...this.syncConfig.controllerToSDK,
        ...this.syncConfig.sdkToController,
        ...this.syncConfig.bidirectional
      ],
      checksums: {}
    };

    // Calculate checksums for synced files
    for (const file of versionInfo.syncedFiles) {
      try {
        const controllerPath = join(this.controllerCLI, file);
        const sdkPath = join(this.sdkCLI, file);

        const controllerHash = await this.calculateFileHash(controllerPath);
        const sdkHash = await this.calculateFileHash(sdkPath);

        versionInfo.checksums[file] = {
          controller: controllerHash,
          sdk: sdkHash,
          inSync: controllerHash === sdkHash
        };
      } catch (error) {
        // File might not exist in both locations
      }
    }

    // Save version info
    const versionPath = join(rootDir, 'KNIRVCONTROLLER', 'cli-sync-version.json');
    await fs.writeFile(versionPath, JSON.stringify(versionInfo, null, 2));

    console.log(`✅ Version tracking updated: ${versionPath}`);
  }

  async calculateFileHash(filePath) {
    try {
      const content = await fs.readFile(filePath);
      return crypto.createHash('sha256').update(content).digest('hex');
    } catch (error) {
      return null;
    }
  }

  async validateSync() {
    console.log('🔍 Validating synchronization...');

    let syncErrors = 0;
    let totalFiles = 0;

    // Check if critical files are in sync
    const criticalFiles = [
      'cmd/root.go',
      'core/api_client.go',
      'config/config.go'
    ];

    for (const file of criticalFiles) {
      totalFiles++;
      const controllerPath = join(this.controllerCLI, file);
      const sdkPath = join(this.sdkCLI, file);

      try {
        const controllerHash = await this.calculateFileHash(controllerPath);
        const sdkHash = await this.calculateFileHash(sdkPath);

        if (controllerHash !== sdkHash) {
          console.warn(`  ⚠️  ${file} - checksums don't match`);
          syncErrors++;
        } else {
          console.log(`  ✅ ${file} - in sync`);
        }
      } catch (error) {
        console.warn(`  ❌ ${file} - validation failed: ${error.message}`);
        syncErrors++;
      }
    }

    if (syncErrors === 0) {
      console.log('✅ All critical files validated successfully');
    } else {
      console.warn(`⚠️  ${syncErrors}/${totalFiles} files have sync issues`);
    }
  }

  async getStatus() {
    console.log('📊 CLI Synchronization Status');
    console.log('================================');

    try {
      const versionPath = join(rootDir, 'KNIRVCONTROLLER', 'cli-sync-version.json');
      const versionInfo = JSON.parse(await fs.readFile(versionPath, 'utf-8'));

      console.log(`Last sync: ${versionInfo.timestamp}`);
      console.log(`Synced files: ${versionInfo.syncedFiles.length}`);

      let inSyncCount = 0;
      let outOfSyncCount = 0;

      for (const [file, checksums] of Object.entries(versionInfo.checksums)) {
        if (checksums.inSync) {
          inSyncCount++;
        } else {
          outOfSyncCount++;
          console.log(`  ⚠️  ${file} - out of sync`);
        }
      }

      console.log(`In sync: ${inSyncCount}`);
      console.log(`Out of sync: ${outOfSyncCount}`);

    } catch (error) {
      console.log('No sync history found. Run sync first.');
    }
  }
}

// CLI interface
const args = process.argv.slice(2);
const command = args[0] || 'sync';
const direction = args[1] || 'both';

const synchronizer = new CLISynchronizer();

switch (command) {
  case 'sync':
    synchronizer.sync(direction).catch(error => {
      console.error('Sync failed:', error.message);
      process.exit(1);
    });
    break;
  case 'status':
    synchronizer.getStatus().catch(error => {
      console.error('Status check failed:', error.message);
      process.exit(1);
    });
    break;
  default:
    console.log('Usage: node cli-sync.js [sync|status] [both|controller-to-sdk|sdk-to-controller]');
    process.exit(1);
}
