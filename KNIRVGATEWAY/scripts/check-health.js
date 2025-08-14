#!/usr/bin/env node

/**
 * KNIRV Gateway Health Checker
 * Automatically detects netlify-cli corruption and other build issues
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class HealthChecker {
    constructor() {
        this.issues = [];
        this.warnings = [];
        this.autoFixAttempted = false;
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

    async checkNetlifyCli() {
        this.log('Checking netlify-cli health...');
        
        try {
            // Check if netlify-cli is installed
            const version = execSync('npx netlify --version', { 
                encoding: 'utf8', 
                timeout: 10000,
                stdio: 'pipe'
            }).trim();
            
            this.log(`netlify-cli version: ${version}`, 'success');
            
            // Check for known problematic versions
            if (version.includes('17.')) {
                this.issues.push('netlify-cli version 17.x detected - known to have corruption issues');
                return false;
            }
            
            // Test basic functionality
            execSync('npx netlify --help', { 
                encoding: 'utf8', 
                timeout: 5000,
                stdio: 'pipe'
            });
            
            return true;
            
        } catch (error) {
            this.issues.push(`netlify-cli is corrupted or not working: ${error.message}`);
            return false;
        }
    }

    async checkNodeModules() {
        this.log('Checking node_modules integrity...');
        
        const nodeModulesPath = path.join(process.cwd(), 'node_modules');
        const netlifyCliPath = path.join(nodeModulesPath, 'netlify-cli');
        
        if (!fs.existsSync(nodeModulesPath)) {
            this.issues.push('node_modules directory missing');
            return false;
        }
        
        if (!fs.existsSync(netlifyCliPath)) {
            this.issues.push('netlify-cli not found in node_modules');
            return false;
        }
        
        // Check for common corruption indicators
        const packageLockPath = path.join(process.cwd(), 'package-lock.json');
        if (!fs.existsSync(packageLockPath)) {
            this.warnings.push('package-lock.json missing - may cause dependency issues');
        }
        
        // Check netlify-cli package.json
        try {
            const netlifyPackageJson = path.join(netlifyCliPath, 'package.json');
            if (fs.existsSync(netlifyPackageJson)) {
                const packageData = JSON.parse(fs.readFileSync(netlifyPackageJson, 'utf8'));
                this.log(`netlify-cli package version: ${packageData.version}`, 'success');
            }
        } catch (error) {
            this.issues.push('netlify-cli package.json is corrupted');
            return false;
        }
        
        return true;
    }

    async checkNexusPortal() {
        this.log('Checking NEXUS portal build status...');
        
        const nexusPath = path.join(process.cwd(), 'nexus-portal');
        const distPath = path.join(nexusPath, 'dist');
        const indexPath = path.join(distPath, 'index.html');
        
        if (!fs.existsSync(nexusPath)) {
            this.issues.push('nexus-portal directory missing');
            return false;
        }
        
        if (!fs.existsSync(distPath)) {
            this.warnings.push('nexus-portal dist directory missing - needs build');
            return false;
        }
        
        if (!fs.existsSync(indexPath)) {
            this.warnings.push('nexus-portal index.html missing - needs build');
            return false;
        }
        
        // Check if build is recent (within last hour)
        try {
            const stats = fs.statSync(indexPath);
            const ageMs = Date.now() - stats.mtime.getTime();
            const ageMinutes = Math.floor(ageMs / (1000 * 60));
            
            if (ageMinutes > 60) {
                this.warnings.push(`nexus-portal build is ${ageMinutes} minutes old - may need refresh`);
            } else {
                this.log(`nexus-portal build is ${ageMinutes} minutes old`, 'success');
            }
        } catch (error) {
            this.warnings.push('Could not check nexus-portal build age');
        }
        
        return true;
    }

    async checkDependencyConflicts() {
        this.log('Checking for dependency conflicts...');
        
        try {
            // Run npm audit to check for issues
            const auditResult = execSync('npm audit --audit-level=moderate --json', { 
                encoding: 'utf8',
                timeout: 30000,
                stdio: 'pipe'
            });
            
            const audit = JSON.parse(auditResult);
            
            if (audit.metadata && audit.metadata.vulnerabilities) {
                const vulns = audit.metadata.vulnerabilities;
                const total = vulns.moderate + vulns.high + vulns.critical;
                
                if (total > 10) {
                    this.issues.push(`High number of vulnerabilities detected: ${total}`);
                } else if (total > 0) {
                    this.warnings.push(`${total} moderate+ vulnerabilities detected`);
                }
            }
            
        } catch (error) {
            // npm audit returns non-zero exit code when vulnerabilities found
            if (error.stdout) {
                try {
                    const audit = JSON.parse(error.stdout);
                    if (audit.metadata && audit.metadata.vulnerabilities) {
                        const vulns = audit.metadata.vulnerabilities;
                        const total = vulns.moderate + vulns.high + vulns.critical;
                        
                        if (total > 15) {
                            this.issues.push(`Critical number of vulnerabilities: ${total}`);
                        } else {
                            this.warnings.push(`${total} vulnerabilities detected`);
                        }
                    }
                } catch (parseError) {
                    this.warnings.push('Could not parse npm audit results');
                }
            }
        }
        
        return true;
    }

    async runHealthCheck() {
        this.log('🏥 Starting KNIRV Gateway health check...');
        
        const checks = [
            this.checkNodeModules(),
            this.checkNetlifyCli(),
            this.checkNexusPortal(),
            this.checkDependencyConflicts()
        ];
        
        await Promise.all(checks);
        
        // Report results
        if (this.issues.length === 0 && this.warnings.length === 0) {
            this.log('All health checks passed! 🎉', 'success');
            return true;
        }
        
        if (this.warnings.length > 0) {
            this.log(`Found ${this.warnings.length} warnings:`, 'warn');
            this.warnings.forEach(warning => this.log(`  - ${warning}`, 'warn'));
        }
        
        if (this.issues.length > 0) {
            this.log(`Found ${this.issues.length} critical issues:`, 'error');
            this.issues.forEach(issue => this.log(`  - ${issue}`, 'error'));
            
            this.log('Health check failed - automatic fix may be needed', 'error');
            return false;
        }
        
        return true;
    }
}

// Run health check if called directly
if (require.main === module) {
    const checker = new HealthChecker();
    checker.runHealthCheck()
        .then(success => {
            process.exit(success ? 0 : 1);
        })
        .catch(error => {
            console.error('❌ Health check failed with error:', error.message);
            process.exit(1);
        });
}

module.exports = HealthChecker;
