#!/usr/bin/env node

/**
 * KNIRV Gateway Function Dependencies Installer
 * Ensures Netlify Function dependencies are installed before build
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class FunctionDependencyInstaller {
    constructor() {
        this.functionsDir = path.join(__dirname, '..', 'netlify', 'functions');
        this.packageJsonPath = path.join(this.functionsDir, 'package.json');
        this.nodeModulesPath = path.join(this.functionsDir, 'node_modules');
    }

    log(message, type = 'info') {
        const timestamp = new Date().toISOString();
        const prefix = {
            info: '🔍',
            success: '✅',
            warning: '⚠️',
            error: '❌'
        }[type] || '📝';
        
        console.log(`${prefix} [${timestamp}] ${message}`);
    }

    async ensureFunctionDependencies() {
        this.log('🔧 Ensuring Netlify Function dependencies are installed...');

        // Check if functions directory exists
        if (!fs.existsSync(this.functionsDir)) {
            this.log('Functions directory not found, skipping dependency installation', 'warning');
            return true;
        }

        // Check if package.json exists
        if (!fs.existsSync(this.packageJsonPath)) {
            this.log('Functions package.json not found, skipping dependency installation', 'warning');
            return true;
        }

        // Check if node_modules exists and has content
        const needsInstall = !fs.existsSync(this.nodeModulesPath) || 
                           fs.readdirSync(this.nodeModulesPath).length === 0;

        if (needsInstall) {
            this.log('Installing function dependencies...', 'info');
            try {
                const originalCwd = process.cwd();
                process.chdir(this.functionsDir);
                
                // Install dependencies
                execSync('npm install', { stdio: 'inherit' });
                
                process.chdir(originalCwd);
                this.log('Function dependencies installed successfully', 'success');
            } catch (error) {
                this.log(`Failed to install function dependencies: ${error.message}`, 'error');
                return false;
            }
        } else {
            this.log('Function dependencies already installed', 'success');
        }

        return true;
    }

    async run() {
        this.log('🚀 Starting Function Dependency Installation...');
        this.log('='.repeat(50));

        const success = await this.ensureFunctionDependencies();

        this.log('='.repeat(50));
        
        if (success) {
            this.log('Function dependency installation completed successfully!', 'success');
            process.exit(0);
        } else {
            this.log('Function dependency installation failed!', 'error');
            process.exit(1);
        }
    }
}

// Main execution
if (require.main === module) {
    const installer = new FunctionDependencyInstaller();
    installer.run().catch(console.error);
}

module.exports = FunctionDependencyInstaller;
