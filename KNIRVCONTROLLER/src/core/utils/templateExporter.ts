/**
 * Template Exporter Utility
 * Exports agent-core-compiler templates to OS application data directory
 * Following the pattern from KNIRVENGINE for template management
 */

import fs from 'fs/promises';
import path from 'path';
import os from 'os';
import pino from 'pino';

const logger = pino({
  name: 'template-exporter',
  level: process.env.LOG_LEVEL || 'info'
});

export class TemplateExporter {
  private appName = 'KNIRV-Controller';
  private appDataDir: string;
  private templatesDir: string;
  private sourceTemplatesDir: string;

  constructor() {
    this.appDataDir = this.getAppDataDir();
    this.templatesDir = path.join(this.appDataDir, 'templates');
    this.sourceTemplatesDir = path.join(process.cwd(), 'src', 'core', 'agent-core-compiler', 'templates');
  }

  /**
   * Get OS-specific application data directory
   * Linux: ~/.config/KNIRV-Controller
   * Windows: %APPDATA%\KNIRV-Controller
   * macOS: ~/Library/Application Support/KNIRV-Controller
   */
  private getAppDataDir(): string {
    const platform = os.platform();
    const homeDir = os.homedir();

    switch (platform) {
      case 'win32':
        return path.join(process.env.APPDATA || path.join(homeDir, 'AppData', 'Roaming'), this.appName);
      case 'darwin':
        return path.join(homeDir, 'Library', 'Application Support', this.appName);
      default: // linux and others
        return path.join(process.env.XDG_CONFIG_HOME || path.join(homeDir, '.config'), this.appName);
    }
  }

  /**
   * Ensure directory exists, create if it doesn't
   */
  private async ensureDirectory(dirPath: string): Promise<void> {
    try {
      await fs.access(dirPath);
    } catch {
      await fs.mkdir(dirPath, { recursive: true });
      logger.info(`Created directory: ${dirPath}`);
    }
  }

  /**
   * Copy a file from source to destination
   */
  private async copyFile(srcPath: string, destPath: string): Promise<void> {
    try {
      const content = await fs.readFile(srcPath);
      await fs.writeFile(destPath, content);
      logger.debug(`Copied: ${path.basename(srcPath)}`);
    } catch (error) {
      logger.error(`Failed to copy ${srcPath} to ${destPath}: ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }

  /**
   * Copy directory recursively
   */
  private async copyDirectory(srcDir: string, destDir: string): Promise<void> {
    await this.ensureDirectory(destDir);
    
    const entries = await fs.readdir(srcDir, { withFileTypes: true });
    
    for (const entry of entries) {
      const srcPath = path.join(srcDir, entry.name);
      const destPath = path.join(destDir, entry.name);
      
      if (entry.isDirectory()) {
        await this.copyDirectory(srcPath, destPath);
      } else {
        await this.copyFile(srcPath, destPath);
      }
    }
  }

  /**
   * Check if templates need updating by comparing modification times
   */
  private async needsUpdate(): Promise<boolean> {
    try {
      const sourceStats = await fs.stat(this.sourceTemplatesDir);
      const destStats = await fs.stat(this.templatesDir);
      
      // Update if source is newer than destination
      return sourceStats.mtime > destStats.mtime;
    } catch {
      // If destination doesn't exist or error accessing, we need to update
      return true;
    }
  }

  /**
   * Export templates to application data directory
   */
  public async exportTemplates(): Promise<void> {
    logger.info('Starting template export process...');
    
    try {
      // Check if source templates directory exists
      try {
        await fs.access(this.sourceTemplatesDir);
      } catch {
        logger.warn(`Source templates directory not found: ${this.sourceTemplatesDir}`);
        return;
      }

      // Check if update is needed
      if (!(await this.needsUpdate())) {
        logger.info('Templates are up to date, skipping export');
        return;
      }

      // Ensure app data directory exists
      await this.ensureDirectory(this.appDataDir);
      
      // Create backup of existing templates if they exist
      const backupDir = path.join(this.appDataDir, 'templates-backup');
      try {
        await fs.access(this.templatesDir);
        logger.info('Creating backup of existing templates...');
        await fs.rm(backupDir, { recursive: true, force: true });
        await fs.rename(this.templatesDir, backupDir);
      } catch {
        // No existing templates to backup
      }

      // Copy templates
      logger.info(`Copying templates from ${this.sourceTemplatesDir} to ${this.templatesDir}`);
      await this.copyDirectory(this.sourceTemplatesDir, this.templatesDir);

      // Verify copy
      const sourceFiles = await this.getFileCount(this.sourceTemplatesDir);
      const destFiles = await this.getFileCount(this.templatesDir);
      
      logger.info(`Template export completed:`);
      logger.info(`  Source files: ${sourceFiles}`);
      logger.info(`  Exported files: ${destFiles}`);
      logger.info(`  Destination: ${this.templatesDir}`);

      if (sourceFiles !== destFiles) {
        logger.warn('File count mismatch - some files may not have been copied');
      }

    } catch (error) {
      logger.error(`Template export failed: ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }

  /**
   * Get total file count in directory recursively
   */
  private async getFileCount(dirPath: string): Promise<number> {
    let count = 0;
    
    try {
      const entries = await fs.readdir(dirPath, { withFileTypes: true });
      
      for (const entry of entries) {
        if (entry.isDirectory()) {
          count += await this.getFileCount(path.join(dirPath, entry.name));
        } else {
          count++;
        }
      }
    } catch {
      // Directory doesn't exist or can't be read
    }
    
    return count;
  }

  /**
   * Get the templates directory path
   */
  public getTemplatesPath(): string {
    return this.templatesDir;
  }

  /**
   * Get the app data directory path
   */
  public getAppDataPath(): string {
    return this.appDataDir;
  }

  /**
   * Clean up old backups (keep only the most recent)
   */
  public async cleanupBackups(): Promise<void> {
    try {
      const backupDir = path.join(this.appDataDir, 'templates-backup');
      await fs.access(backupDir);
      
      // For now, just log that backup exists
      // In the future, could implement rotation of multiple backups
      logger.info(`Template backup available at: ${backupDir}`);
    } catch {
      // No backup to clean up
    }
  }
}
