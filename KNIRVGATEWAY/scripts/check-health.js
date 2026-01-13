#!/usr/bin/env node

/**
 * KNIRV Gateway Health Checker
 * Focuses on service health and basic dependencies for the main oracle
 */

import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';

class GatewayHealthChecker {
    constructor(options = {}) {
        this.issues = [];
        this.warnings = [];
        this.buildMode = options.buildMode || false;
    }

    log(message, type = 'info') {
        const timestamp = new Date().toISOString();
        const prefix = {
            'info': 'ℹ️ ',
            'success': '✅',
            'warn': '⚠️ ',
            'error': '❌',
            'fix': '🔧'
        }[type] || 'ℹ️ ';

        console.log(`${prefix} [${timestamp}] ${message}`);
    }

    async checkNodeModules() {
        this.log('Checking node_modules integrity...');

        const nodeModulesPath = path.join(process.cwd(), 'node_modules');

        if (!fs.existsSync(nodeModulesPath)) {
            this.issues.push('node_modules directory missing');
            return false;
        }

        // Check essential dependencies for the oracle
        const essentialDeps = [
            'express',
            'cors',
            'axios',
            'uuid',
            'ws'
        ];

        for (const dep of essentialDeps) {
            const depPath = path.join(nodeModulesPath, dep);
            if (!fs.existsSync(depPath)) {
                this.issues.push(`Essential dependency missing: ${dep}`);
            }
        }

        this.log('Node modules check completed', 'success');
        return true;
    }

    async checkServiceDirectories() {
        this.log('Checking service directories...');

        const serviceDirectories = [
            'services/payment-oracle',
            'services/tunnel-registry',
            'services/operator-registry',
            'services/webgui'
        ];

        for (const serviceDir of serviceDirectories) {
            const servicePath = path.join(process.cwd(), serviceDir);
            if (!fs.existsSync(servicePath)) {
                this.issues.push(`Service directory missing: ${serviceDir}`);
                continue;
            }

            // Check if service has package.json
            const packageJsonPath = path.join(servicePath, 'package.json');
            if (!fs.existsSync(packageJsonPath)) {
                this.warnings.push(`Service ${serviceDir} missing package.json`);
                continue;
            }

            // Check if service has node_modules (if not in build mode)
            if (!this.buildMode) {
                const serviceNodeModules = path.join(servicePath, 'node_modules');
                if (!fs.existsSync(serviceNodeModules)) {
                    this.warnings.push(`Service ${serviceDir} dependencies not installed`);
                }
            }
        }

        this.log('Service directories check completed', 'success');
        return true;
    }

    async checkWebsiteDirectories() {
        this.log('Checking website directories...');

        const websiteDirectories = [
            'network-website',
            'primary-website'
        ];

        for (const websiteDir of websiteDirectories) {
            const websitePath = path.join(process.cwd(), websiteDir);
            if (!fs.existsSync(websitePath)) {
                this.issues.push(`Website directory missing: ${websiteDir}`);
                continue;
            }

            // Check if website has package.json
            const packageJsonPath = path.join(websitePath, 'package.json');
            if (!fs.existsSync(packageJsonPath)) {
                this.warnings.push(`Website ${websiteDir} missing package.json`);
                continue;
            }

            // Check if website has node_modules (if not in build mode)
            if (!this.buildMode) {
                const websiteNodeModules = path.join(websitePath, 'node_modules');
                if (!fs.existsSync(websiteNodeModules)) {
                    this.warnings.push(`Website ${websiteDir} dependencies not installed`);
                }
            }
        }

        this.log('Website directories check completed', 'success');
        return true;
    }

    async checkServerFile() {
        this.log('Checking main server file...');

        const serverPath = path.join(process.cwd(), 'server.js');
        if (!fs.existsSync(serverPath)) {
            this.issues.push('Main server.js file missing');
            return false;
        }

        // Check if NodeJSServiceManager exists
        const serviceManagerPath = path.join(process.cwd(), 'lib/services/nodejs_service_manager.js');
        if (!fs.existsSync(serviceManagerPath)) {
            this.issues.push('NodeJSServiceManager missing');
            return false;
        }

        this.log('Server file check completed', 'success');
        return true;
    }

    async checkPortAvailability() {
        if (this.buildMode) {
            this.log('Skipping port availability check in build mode');
            return true;
        }

        this.log('Checking port availability...');

        const portsToCheck = [8080]; // Main oracle port

        for (const port of portsToCheck) {
            try {
                const result = execSync(`netstat -tuln | grep :${port}`, {
                    encoding: 'utf8',
                    stdio: 'pipe'
                });

                if (result.trim()) {
                    this.warnings.push(`Port ${port} appears to be in use`);
                }
            } catch (error) {
                // Port is available (netstat returns non-zero when no matches)
                this.log(`Port ${port} is available`, 'success');
            }
        }

        return true;
    }

    async runHealthCheck() {
        this.log('🏥 Starting KNIRV Gateway health check...');

        const checks = [
            this.checkNodeModules(),
            this.checkServiceDirectories(),
            this.checkWebsiteDirectories(),
            this.checkServerFile(),
            this.checkPortAvailability()
        ];

        await Promise.all(checks);

        // Report results
        this.log('='.repeat(60));
        this.log('Health Check Results:', 'info');
        this.log('='.repeat(60));

        if (this.warnings.length > 0) {
            this.log(`Found ${this.warnings.length} warnings:`, 'warn');
            this.warnings.forEach(warning => this.log(`  - ${warning}`, 'warn'));
        }

        if (this.issues.length > 0) {
            this.log(`Found ${this.issues.length} critical issues:`, 'error');
            this.issues.forEach(issue => this.log(`  - ${issue}`, 'error'));

            this.log('='.repeat(60));
            this.log('💡 Suggested fixes:', 'info');
            this.log('  1. Run: npm install', 'info');
            this.log('  2. Run: npm run services:install', 'info');
            this.log('  3. Run: npm run websites:install', 'info');

            return false;
        } else {
            this.log('All health checks passed! 🎉', 'success');
            return true;
        }
    }
}

// CLI execution
if (import.meta.url === `file://${process.argv[1]}`) {
    const buildMode = process.argv.includes('--build-mode');
    const checker = new GatewayHealthChecker({ buildMode });

    checker.runHealthCheck().then(success => {
        process.exit(success ? 0 : 1);
    }).catch(error => {
        console.error('Health check failed:', error);
        process.exit(1);
    });
}

export default GatewayHealthChecker;