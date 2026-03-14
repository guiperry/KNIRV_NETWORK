#!/usr/bin/env node

/**
 * KNIRV NEXUS Unified Binary Health Checker
 * Checks NEXUS unified binary availability and health
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class NexusHealthChecker {
    constructor() {
        this.issues = [];
        this.warnings = [];
    }

    // Helper method to find the NEXUS unified binary
    getNexusBinaryPath() {
        const possibleBinaryPaths = [
            path.join(process.cwd(), '../KNIRVTESTNET/bin/knirvnexus'), // KNIRVTESTNET location
            path.join(process.cwd(), '../../KNIRVNEXUS/dist/knirv-nexus'), // KNIRVNEXUS build location
            path.join(process.cwd(), '../bin/knirvnexus'), // Alternative location
            '/usr/local/bin/knirvnexus' // System installation
        ];

        for (const testPath of possibleBinaryPaths) {
            if (fs.existsSync(testPath)) {
                return testPath;
            }
        }

        // Return first path for error reporting if none found
        return possibleBinaryPaths[0];
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
        
        console.log(`${prefix} [NEXUS] ${message}`);
    }

    async checkNexusBinary() {
        this.log('Checking NEXUS unified binary...');

        const binaryPath = this.getNexusBinaryPath();

        if (!fs.existsSync(binaryPath)) {
            this.issues.push('NEXUS unified binary not found');
            this.log(`Binary not found at: ${binaryPath}`, 'error');
            return false;
        }

        // Check if binary is executable
        try {
            const stats = fs.statSync(binaryPath);
            if (!(stats.mode & parseInt('111', 8))) {
                this.issues.push('NEXUS binary is not executable');
                return false;
            }
        } catch (error) {
            this.issues.push(`Cannot access NEXUS binary: ${error.message}`);
            return false;
        }

        // Check binary size (should be substantial due to embedded frontend)
        const stats = fs.statSync(binaryPath);
        const sizeInMB = stats.size / (1024 * 1024);

        if (sizeInMB < 5) {
            this.warnings.push(`NEXUS binary size seems small: ${sizeInMB.toFixed(2)}MB`);
        } else {
            this.log(`NEXUS binary size: ${sizeInMB.toFixed(2)}MB`, 'success');
        }

        return true;
    }

    async checkNexusService() {
        this.log('Checking NEXUS service availability...');

        // Check if NEXUS service is running
        try {
            const { execSync } = require('child_process');
            const result = execSync('curl -s --max-time 5 http://localhost:8084/health', { encoding: 'utf8' });

            if (result && (result.includes('ok') || result.includes('health') || result.includes('status'))) {
                this.log('NEXUS service is running and healthy', 'success');
                return true;
            } else {
                this.warnings.push('NEXUS service not responding or unhealthy');
                return false;
            }
        } catch (error) {
            this.warnings.push('NEXUS service not accessible (may not be running)');
            return false;
        }
    }

    async checkNexusEndpoints() {
        this.log('Checking NEXUS unified binary endpoints...');

        const endpoints = [
            { path: '/', description: 'Frontend' },
            { path: '/api/v1/health', description: 'API Health' },
            { path: '/api/v1/dve/status', description: 'DVE Status' },
            { path: '/api/v1/config', description: 'Configuration' }
        ];

        let successCount = 0;

        for (const endpoint of endpoints) {
            try {
                const { execSync } = require('child_process');
                const result = execSync(`curl -s --max-time 3 http://localhost:8084${endpoint.path}`, { encoding: 'utf8' });

                if (result && result.length > 0) {
                    this.log(`${endpoint.description} endpoint accessible`, 'success');
                    successCount++;
                } else {
                    this.warnings.push(`${endpoint.description} endpoint not responding`);
                }
            } catch (error) {
                this.warnings.push(`${endpoint.description} endpoint not accessible`);
            }
        }

        if (successCount >= endpoints.length / 2) {
            this.log(`${successCount}/${endpoints.length} endpoints accessible`, 'success');
            return true;
        } else {
            this.issues.push(`Only ${successCount}/${endpoints.length} endpoints accessible`);
            return false;
        }
        
        return true;
    }



    async runNexusHealthCheck() {
        this.log('🏥 Starting NEXUS unified binary health check...');

        const checks = [
            this.checkNexusBinary(),
            this.checkNexusService(),
            this.checkNexusEndpoints()
        ];

        await Promise.all(checks);
        
        // Report results
        if (this.issues.length === 0 && this.warnings.length === 0) {
            this.log('NEXUS unified binary health check passed! 🎉', 'success');
            return true;
        }
        
        if (this.warnings.length > 0) {
            this.log(`Found ${this.warnings.length} warnings:`, 'warn');
            this.warnings.forEach(warning => this.log(`  - ${warning}`, 'warn'));
        }
        
        if (this.issues.length > 0) {
            this.log(`Found ${this.issues.length} critical issues:`, 'error');
            this.issues.forEach(issue => this.log(`  - ${issue}`, 'error'));
            
            this.log('NEXUS unified binary health check failed', 'error');
            return false;
        }
        
        // Only warnings, still considered passing
        return true;
    }
}

// Run health check if called directly
if (require.main === module) {
    const checker = new NexusHealthChecker();
    checker.runNexusHealthCheck()
        .then(success => {
            process.exit(success ? 0 : 1);
        })
        .catch(error => {
            console.error('❌ NEXUS health check failed with error:', error.message);
            process.exit(1);
        });
}

module.exports = NexusHealthChecker;
