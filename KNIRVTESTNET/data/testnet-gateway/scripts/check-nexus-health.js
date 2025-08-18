#!/usr/bin/env node

/**
 * KNIRV NEXUS Frontend Health Checker
 * Checks NEXUS frontend build integrity for unified architecture
 */

const fs = require('fs');
const path = require('path');

class NexusHealthChecker {
    constructor() {
        this.issues = [];
        this.warnings = [];
    }

    log(message, type = 'info') {
        const prefix = {
            'info': '🔍',
            'warn': '⚠️ ',
            'error': '❌',
            'success': '✅',
            'fix': '🔧'
        }[type] || 'ℹ️ ';
        
        console.log(`${prefix} [NEXUS] ${message}`);
    }

    async checkNexusDirectory() {
        this.log('Checking NEXUS frontend directory structure...');

        // In unified architecture, NEXUS frontend is in data/knirvnexus/portal
        const nexusPath = path.join(process.cwd(), '../../data/knirvnexus/portal');
        const requiredFiles = [
            'package.json',
            '.next',
            'public',
            'server.js'
        ];

        if (!fs.existsSync(nexusPath)) {
            this.log('NEXUS frontend directory does not exist', 'warn');
            this.warnings.push('NEXUS frontend directory missing - run build-nexus-frontend.sh');
            return false;
        }

        let missingFiles = [];
        for (const file of requiredFiles) {
            const filePath = path.join(nexusPath, file);
            if (!fs.existsSync(filePath)) {
                missingFiles.push(file);
            }
        }

        if (missingFiles.length > 0) {
            this.log(`Missing files: ${missingFiles.join(', ')}`, 'warn');
            this.warnings.push(`Missing NEXUS frontend files: ${missingFiles.join(', ')}`);
        } else {
            this.log('All required files present', 'success');
        }

        return missingFiles.length === 0;
    }

    async checkNexusDependencies() {
        this.log('Checking NEXUS frontend dependencies...');

        const nexusPath = path.join(process.cwd(), '../../data/knirvnexus/portal');

        if (!fs.existsSync(nexusPath)) {
            this.warnings.push('NEXUS frontend directory missing - dependencies check skipped');
            return false;
        }

        const nodeModulesPath = path.join(nexusPath, 'node_modules');
        const packageLockPath = path.join(nexusPath, 'package-lock.json');

        if (!fs.existsSync(nodeModulesPath)) {
            this.warnings.push('NEXUS frontend node_modules missing - run build-nexus-frontend.sh');
            return false;
        }

        if (!fs.existsSync(packageLockPath)) {
            this.warnings.push('NEXUS frontend package-lock.json missing');
        }

        this.log('Dependencies check passed', 'success');
        return true;
    }

    async checkNexusBuild() {
        this.log('Checking NEXUS frontend build artifacts...');
        
        const nexusPath = path.join(process.cwd(), '../../data/knirvnexus/portal');
        const nextPath = path.join(nexusPath, '.next');
        const serverPath = path.join(nextPath, 'server');
        const staticPath = path.join(nextPath, 'static');
        
        if (!fs.existsSync(nextPath)) {
            this.warnings.push('NEXUS frontend .next directory missing - needs build');
            return false;
        }
        
        if (!fs.existsSync(serverPath)) {
            this.issues.push('NEXUS frontend server build missing');
            return false;
        }
        
        if (!fs.existsSync(staticPath)) {
            this.issues.push('NEXUS frontend static assets missing');
            return false;
        }
        
        this.log('Build artifacts check passed', 'success');
        return true;
    }

    async checkBuildFreshness() {
        this.log('Checking NEXUS frontend build freshness...');
        
        const nexusPath = path.join(process.cwd(), '../../data/knirvnexus/portal');
        const nextPath = path.join(nexusPath, '.next');
        const srcPath = path.join(nexusPath, 'src');
        
        if (!fs.existsSync(nextPath)) {
            this.warnings.push('No build found to check freshness');
            return false;
        }

        try {
            const buildStat = fs.statSync(nextPath);
            const buildTime = buildStat.mtime;
            
            this.log(`Build timestamp: ${buildTime.toISOString()}`, 'info');
            
            // Check if build is older than 24 hours
            const oneDayAgo = new Date(Date.now() - 24 * 60 * 60 * 1000);
            if (buildTime < oneDayAgo) {
                this.warnings.push('NEXUS frontend build is older than 24 hours');
            } else {
                this.log('Build is fresh', 'success');
            }
            
        } catch (error) {
            this.warnings.push('Could not check build freshness');
            return false;
        }
        
        return true;
    }

    async runHealthCheck() {
        this.log('🏥 Starting NEXUS frontend health check...');
        
        await this.checkNexusDirectory();
        await this.checkNexusDependencies();
        await this.checkNexusBuild();
        await this.checkBuildFreshness();
        
        // Summary
        if (this.warnings.length > 0) {
            this.log(`Found ${this.warnings.length} warnings:`, 'warn');
            this.warnings.forEach(warning => {
                this.log(`  - ${warning}`, 'warn');
            });
        }
        
        if (this.issues.length > 0) {
            this.log(`Found ${this.issues.length} issues:`, 'error');
            this.issues.forEach(issue => {
                this.log(`  - ${issue}`, 'error');
            });
            return false;
        }
        
        if (this.warnings.length === 0) {
            this.log('✅ All NEXUS frontend health checks passed!', 'success');
        } else {
            this.log('⚠️  NEXUS frontend health check completed with warnings', 'warn');
        }
        
        return true;
    }
}

// Run health check if called directly
if (require.main === module) {
    const checker = new NexusHealthChecker();
    checker.runHealthCheck().then(healthy => {
        process.exit(healthy ? 0 : 1);
    }).catch(error => {
        console.error('Health check failed:', error);
        process.exit(1);
    });
}

module.exports = { NexusHealthChecker };
