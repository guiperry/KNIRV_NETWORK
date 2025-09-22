#!/usr/bin/env node

/**
 * Netlify Dependencies Patcher
 * 
 * This script patches missing dist files in netlify function dependencies
 * that are required for proper bundling by Netlify's esbuild process.
 * 
 * Issues fixed:
 * 1. bcryptjs missing dist/bcrypt.js file
 * 2. formidable missing dist/* files
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { execSync } from 'child_process';

// ES module compatibility
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

class NetlifyDepsPatcher {
    constructor() {
        this.projectRoot = path.resolve(__dirname, '..');
        this.functionsDir = path.join(this.projectRoot, 'netlify', 'functions');
        this.nodeModulesDir = path.join(this.functionsDir, 'node_modules');
        this.patchesApplied = [];
        this.errors = [];
    }

    log(message, type = 'info') {
        const timestamp = new Date().toISOString();
        const prefix = {
            info: '🔧',
            success: '✅',
            warning: '⚠️',
            error: '❌'
        }[type] || 'ℹ️';
        
        console.log(`${prefix} [${timestamp}] ${message}`);
    }

    /**
     * Check if a directory exists
     */
    dirExists(dirPath) {
        try {
            return fs.statSync(dirPath).isDirectory();
        } catch {
            return false;
        }
    }

    /**
     * Check if a file exists
     */
    fileExists(filePath) {
        try {
            return fs.statSync(filePath).isFile();
        } catch {
            return false;
        }
    }

    /**
     * Create directory recursively
     */
    ensureDir(dirPath) {
        if (!this.dirExists(dirPath)) {
            fs.mkdirSync(dirPath, { recursive: true });
            this.log(`Created directory: ${dirPath}`);
        }
    }

    /**
     * Copy file with error handling
     */
    copyFile(src, dest) {
        try {
            this.ensureDir(path.dirname(dest));
            fs.copyFileSync(src, dest);
            this.log(`Copied: ${path.basename(src)} -> ${dest}`);
            return true;
        } catch (error) {
            this.log(`Failed to copy ${src} to ${dest}: ${error.message}`, 'error');
            this.errors.push(`Copy failed: ${src} -> ${dest}: ${error.message}`);
            return false;
        }
    }

    /**
     * Copy directory recursively
     */
    copyDir(src, dest) {
        try {
            this.ensureDir(dest);
            const items = fs.readdirSync(src);
            
            for (const item of items) {
                const srcPath = path.join(src, item);
                const destPath = path.join(dest, item);
                
                if (this.dirExists(srcPath)) {
                    this.copyDir(srcPath, destPath);
                } else {
                    this.copyFile(srcPath, destPath);
                }
            }
            return true;
        } catch (error) {
            this.log(`Failed to copy directory ${src} to ${dest}: ${error.message}`, 'error');
            this.errors.push(`Directory copy failed: ${src} -> ${dest}: ${error.message}`);
            return false;
        }
    }

    /**
     * Patch bcryptjs missing dist files
     */
    patchBcryptjs() {
        this.log('Patching bcryptjs...');
        
        const bcryptjsPath = path.join(this.nodeModulesDir, 'bcryptjs');
        const srcPath = path.join(bcryptjsPath, 'src', 'bcrypt.js');
        const distDir = path.join(bcryptjsPath, 'dist');
        const distPath = path.join(distDir, 'bcrypt.js');

        if (!this.dirExists(bcryptjsPath)) {
            this.log('bcryptjs not found - skipping patch', 'warning');
            return false;
        }

        if (!this.fileExists(srcPath)) {
            this.log(`bcryptjs source file not found: ${srcPath}`, 'error');
            this.errors.push('bcryptjs source file missing');
            return false;
        }

        if (this.fileExists(distPath)) {
            this.log('bcryptjs already patched - skipping');
            return true;
        }

        this.ensureDir(distDir);
        
        if (this.copyFile(srcPath, distPath)) {
            this.patchesApplied.push('bcryptjs dist/bcrypt.js');
            this.log('bcryptjs patched successfully', 'success');
            return true;
        }

        return false;
    }

    /**
     * Patch formidable missing dist files
     */
    patchFormidable() {
        this.log('Patching formidable...');
        
        const formidablePath = path.join(this.nodeModulesDir, 'formidable');
        const srcDir = path.join(formidablePath, 'src');
        const distDir = path.join(formidablePath, 'dist');

        if (!this.dirExists(formidablePath)) {
            this.log('formidable not found - skipping patch', 'warning');
            return false;
        }

        if (!this.dirExists(srcDir)) {
            this.log(`formidable source directory not found: ${srcDir}`, 'error');
            this.errors.push('formidable source directory missing');
            return false;
        }

        // Check if key files exist to determine if already patched
        const keyFiles = ['index.js', 'Formidable.js', 'PersistentFile.js'];
        const allKeyFilesExist = keyFiles.every(file =>
            this.fileExists(path.join(distDir, file))
        );

        if (allKeyFilesExist) {
            this.log('formidable already patched - skipping');
            return true;
        }

        this.ensureDir(distDir);
        
        if (this.copyDir(srcDir, distDir)) {
            this.patchesApplied.push('formidable dist/* files');
            this.log('formidable patched successfully', 'success');
            return true;
        }

        return false;
    }

    /**
     * Install netlify function dependencies
     */
    installFunctionDeps() {
        // Skip if we're already in a postinstall context to avoid infinite loops
        if (process.env.npm_lifecycle_event === 'postinstall') {
            this.log('Skipping npm install - already in postinstall context');
            return true;
        }

        this.log('Installing netlify function dependencies...');

        const packageJsonPath = path.join(this.functionsDir, 'package.json');

        if (!this.fileExists(packageJsonPath)) {
            this.log('Creating netlify functions package.json...');
            const packageJson = {
                name: "knirvgateway-netlify-functions",
                version: "1.0.0",
                description: "Dependencies for KNIRVGATEWAY Netlify functions",
                dependencies: {
                    "axios": "^1.6.0",
                    "node-cache": "^5.1.2",
                    "bcryptjs": "^2.4.3",
                    "formidable": "^3.5.1"
                }
            };

            fs.writeFileSync(packageJsonPath, JSON.stringify(packageJson, null, 2));
            this.log('Created package.json for netlify functions', 'success');
        }

        try {
            this.log('Running npm install in network-website/netlify/functions...');
            execSync('npm install', {
                cwd: this.functionsDir,
                stdio: 'inherit'
            });
            this.log('Dependencies installed successfully', 'success');
            return true;
        } catch (error) {
            this.log(`Failed to install dependencies: ${error.message}`, 'error');
            this.errors.push(`npm install failed: ${error.message}`);
            return false;
        }
    }

    /**
     * Verify patches are working
     */
    verifyPatches() {
        this.log('Verifying patches...');
        
        const checks = [
            {
                name: 'bcryptjs dist/bcrypt.js',
                path: path.join(this.nodeModulesDir, 'bcryptjs', 'dist', 'bcrypt.js')
            },
            {
                name: 'formidable dist/index.js',
                path: path.join(this.nodeModulesDir, 'formidable', 'dist', 'index.js')
            },
            {
                name: 'formidable dist/Formidable.js',
                path: path.join(this.nodeModulesDir, 'formidable', 'dist', 'Formidable.js')
            }
        ];

        let allGood = true;
        for (const check of checks) {
            if (this.fileExists(check.path)) {
                this.log(`✓ ${check.name} exists`, 'success');
            } else {
                this.log(`✗ ${check.name} missing`, 'error');
                allGood = false;
            }
        }

        return allGood;
    }

    /**
     * Run all patches
     */
    async run() {
        this.log('🚀 Starting Netlify dependencies patcher...');
        this.log(`Project root: ${this.projectRoot}`);
        this.log(`Functions directory: ${this.functionsDir}`);

        // Ensure functions directory exists
        this.ensureDir(this.functionsDir);

        // Install dependencies first
        const depsInstalled = this.installFunctionDeps();
        if (!depsInstalled) {
            this.log('Failed to install dependencies - aborting', 'error');
            return false;
        }

        // Apply patches
        const bcryptjsPatched = this.patchBcryptjs();
        const formidablePatched = this.patchFormidable();

        // Verify patches
        const verified = this.verifyPatches();

        // Summary
        this.log('\n📊 Patch Summary:');
        this.log(`Patches applied: ${this.patchesApplied.length}`);
        for (const patch of this.patchesApplied) {
            this.log(`  ✅ ${patch}`);
        }

        if (this.errors.length > 0) {
            this.log(`Errors encountered: ${this.errors.length}`, 'warning');
            for (const error of this.errors) {
                this.log(`  ❌ ${error}`, 'error');
            }
        }

        const success = verified && this.errors.length === 0;
        this.log(`\n${success ? '🎉 All patches applied successfully!' : '⚠️ Some patches failed - check errors above'}`, success ? 'success' : 'warning');
        
        return success;
    }
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
    const patcher = new NetlifyDepsPatcher();
    patcher.run()
        .then(success => {
            process.exit(success ? 0 : 1);
        })
        .catch(error => {
            console.error('❌ Patcher failed with error:', error);
            process.exit(1);
        });
}

export { NetlifyDepsPatcher };
