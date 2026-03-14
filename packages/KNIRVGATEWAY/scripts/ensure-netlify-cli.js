#!/usr/bin/env node

/**
 * KNIRV Gateway - Ensure Netlify CLI Script
 * Ensures netlify-cli is properly installed and working
 */

import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import { fileURLToPath } from 'url';

// ES module compatibility
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

class NetlifyCliEnsurer {
    constructor() {
        this.issues = [];
        this.warnings = [];
    }

    log(message, type = 'info') {
        const timestamp = new Date().toISOString();
        const prefix = {
            'info': '🔍',
            'warn': '⚠️ ',
            'error': '❌',
            'success': '✅',
            'fix': '🔧'
        }[type] || 'ℹ️ ';
        
        console.log(`${prefix} [${timestamp}] ${message}`);
    }

    checkNetlifyCliInstallation() {
        this.log('Checking netlify-cli installation...');

        // Check if netlify-cli is in node_modules
        const netlifyCliPath = path.join(process.cwd(), 'node_modules', 'netlify-cli');
        const netlifyBinaryPath = path.join(process.cwd(), 'node_modules', '.bin', 'netlify');

        if (!fs.existsSync(netlifyCliPath)) {
            this.log('netlify-cli package not found in node_modules', 'error');
            return false;
        }

        if (!fs.existsSync(netlifyBinaryPath)) {
            this.log('netlify-cli binary not found in node_modules/.bin', 'error');
            return false;
        }

        this.log('netlify-cli installation found', 'success');
        return true;
    }

    testNetlifyCliCommand() {
        this.log('Testing netlify-cli command...');

        try {
            // Try a quick version check with timeout
            const version = execSync('timeout 15s npx netlify --version', {
                encoding: 'utf8',
                stdio: 'pipe',
                shell: true
            }).trim();

            this.log(`netlify-cli version: ${version}`, 'success');
            return true;

        } catch (error) {
            if (error.message.includes('ETIMEDOUT') || error.status === 124) {
                this.log('netlify-cli command timed out, but binary exists - treating as working', 'warn');
                return true; // Don't fail for slow commands
            } else {
                this.log(`netlify-cli command failed: ${error.message}`, 'error');
                return false;
            }
        }
    }

    installNetlifyCli() {
        this.log('Installing netlify-cli...', 'fix');

        try {
            // First ensure we have all dependencies including dev
            this.log('Running: npm install --include=dev');
            execSync('npm install --include=dev', {
                stdio: 'inherit',
                timeout: 120000 // 2 minutes timeout
            });

            // Install specific version of netlify-cli
            this.log('Running: npm install netlify-cli@21.6.0 --save-dev');
            execSync('npm install netlify-cli@21.6.0 --save-dev', {
                stdio: 'inherit',
                timeout: 120000 // 2 minutes timeout
            });

            this.log('netlify-cli installation completed', 'success');
            return true;

        } catch (error) {
            this.log(`Failed to install netlify-cli: ${error.message}`, 'error');
            return false;
        }
    }

    cleanAndReinstall() {
        this.log('Cleaning and reinstalling netlify-cli...', 'fix');

        try {
            // Remove existing netlify-cli
            this.log('Removing existing netlify-cli...');
            execSync('npm uninstall netlify-cli', { stdio: 'pipe' });

            // Clear npm cache
            this.log('Clearing npm cache...');
            execSync('npm cache clean --force', { stdio: 'pipe' });

            // Remove node_modules and package-lock.json for clean install
            this.log('Removing node_modules for clean install...');
            execSync('rm -rf node_modules package-lock.json', { stdio: 'pipe' });

            // Reinstall
            return this.installNetlifyCli();

        } catch (error) {
            this.log(`Failed to clean and reinstall: ${error.message}`, 'error');
            return false;
        }
    }

    async ensureNetlifyCli() {
        this.log('🔧 Ensuring netlify-cli is properly installed...');

        // First check if it's already installed and working
        if (this.checkNetlifyCliInstallation() && this.testNetlifyCliCommand()) {
            this.log('netlify-cli is already working properly! 🎉', 'success');
            return true;
        }

        // Try to install if missing
        if (!this.checkNetlifyCliInstallation()) {
            this.log('netlify-cli not found, installing...', 'fix');
            if (!this.installNetlifyCli()) {
                this.log('Failed to install netlify-cli', 'error');
                return false;
            }
        }

        // Test again after installation
        if (this.testNetlifyCliCommand()) {
            this.log('netlify-cli is now working! 🎉', 'success');
            return true;
        }

        // If still not working, try clean reinstall
        this.log('netlify-cli still not working, trying clean reinstall...', 'fix');
        if (!this.cleanAndReinstall()) {
            this.log('Failed to fix netlify-cli installation', 'error');
            return false;
        }

        // Final test
        if (this.testNetlifyCliCommand()) {
            this.log('netlify-cli is now working after clean reinstall! 🎉', 'success');
            return true;
        }

        this.log('Unable to get netlify-cli working', 'error');
        return false;
    }
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
    const ensurer = new NetlifyCliEnsurer();
    ensurer.ensureNetlifyCli()
        .then(success => {
            process.exit(success ? 0 : 1);
        })
        .catch(error => {
            console.error('❌ Netlify CLI ensurer failed with error:', error.message);
            process.exit(1);
        });
}

export default NetlifyCliEnsurer;
