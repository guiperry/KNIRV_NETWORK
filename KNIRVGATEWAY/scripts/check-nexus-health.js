#!/usr/bin/env node

/**
 * KNIRV NEXUS Portal Health Checker
 * Specifically checks NEXUS portal build integrity
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class NexusHealthChecker {
    constructor() {
        this.issues = [];
        this.warnings = [];
    }

    // Helper method to find the correct nexus-portal path
    getNexusPath() {
        const possibleNexusPaths = [
            path.join(process.cwd(), 'nexus-portal'), // Original location
            path.join(process.cwd(), '../data/knirvnexus/portal'), // KNIRVTESTNET location
            path.join(process.cwd(), 'data/knirvnexus/portal') // Alternative location
        ];

        for (const testPath of possibleNexusPaths) {
            if (fs.existsSync(testPath)) {
                return testPath;
            }
        }

        // Return original path for error reporting if none found
        return possibleNexusPaths[0];
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

    async checkNexusDirectory() {
        this.log('Checking NEXUS portal directory structure...');

        const nexusPath = this.getNexusPath();
        const requiredFiles = [
            'package.json',
            'vite.config.ts',
            'src/main.tsx',
            'src/App.tsx',
            'dashboard.html'
        ];

        // Check if running in testnet mode
        const isTestnetMode = process.env.TESTNET_MODE === 'true' || process.env.NODE_ENV === 'testnet';

        if (!fs.existsSync(nexusPath)) {
            if (isTestnetMode) {
                this.log('nexus-portal directory not found - skipping (testnet mode)', 'info');
                return true;
            } else {
                this.issues.push('nexus-portal directory does not exist');
                return false;
            }
        }
        
        for (const file of requiredFiles) {
            const filePath = path.join(nexusPath, file);
            if (!fs.existsSync(filePath)) {
                this.issues.push(`Required file missing: ${file}`);
            }
        }
        
        return this.issues.length === 0;
    }

    async checkNexusDependencies() {
        this.log('Checking NEXUS portal dependencies...');

        const nexusPath = this.getNexusPath();
        const nodeModulesPath = path.join(nexusPath, 'node_modules');
        const packageLockPath = path.join(nexusPath, 'package-lock.json');
        
        if (!fs.existsSync(nodeModulesPath)) {
            this.warnings.push('nexus-portal node_modules missing - will install');
            return false;
        }
        
        if (!fs.existsSync(packageLockPath)) {
            this.warnings.push('nexus-portal package-lock.json missing');
        }
        
        // Check for critical dependencies
        const criticalDeps = ['react', 'vite', 'typescript'];
        for (const dep of criticalDeps) {
            const depPath = path.join(nodeModulesPath, dep);
            if (!fs.existsSync(depPath)) {
                this.issues.push(`Critical dependency missing: ${dep}`);
            }
        }
        
        return this.issues.length === 0;
    }

    async checkNexusBuild() {
        this.log('Checking NEXUS portal build artifacts...');

        const nexusPath = this.getNexusPath();
        const distPath = path.join(nexusPath, 'dist');
        const indexPath = path.join(distPath, 'index.html');
        const assetsPath = path.join(distPath, 'assets');
        
        if (!fs.existsSync(distPath)) {
            this.warnings.push('nexus-portal dist directory missing - needs build');
            return false;
        }
        
        if (!fs.existsSync(indexPath)) {
            this.issues.push('nexus-portal index.html missing from dist');
            return false;
        }
        
        if (!fs.existsSync(assetsPath)) {
            this.issues.push('nexus-portal assets directory missing from dist');
            return false;
        }
        
        // Check for CSS and JS files
        try {
            const assetFiles = fs.readdirSync(assetsPath);
            const hasCss = assetFiles.some(file => file.endsWith('.css'));
            const hasJs = assetFiles.some(file => file.endsWith('.js'));
            
            if (!hasCss) {
                this.issues.push('No CSS files found in nexus-portal build');
            }
            
            if (!hasJs) {
                this.issues.push('No JavaScript files found in nexus-portal build');
            }
            
            this.log(`Found ${assetFiles.length} asset files`, 'success');
            
        } catch (error) {
            this.issues.push('Could not read nexus-portal assets directory');
            return false;
        }
        
        return true;
    }

    async checkBuildFreshness() {
        this.log('Checking NEXUS portal build freshness...');

        const nexusPath = this.getNexusPath();
        const distPath = path.join(nexusPath, 'dist');
        const indexPath = path.join(distPath, 'index.html');
        const srcPath = path.join(nexusPath, 'src');
        
        if (!fs.existsSync(indexPath)) {
            return false; // Already handled in checkNexusBuild
        }
        
        try {
            const buildTime = fs.statSync(indexPath).mtime.getTime();
            
            // Check if any source files are newer than the build
            const checkSourceFiles = (dir) => {
                const files = fs.readdirSync(dir);
                for (const file of files) {
                    const filePath = path.join(dir, file);
                    const stat = fs.statSync(filePath);
                    
                    if (stat.isDirectory()) {
                        if (checkSourceFiles(filePath)) return true;
                    } else if (stat.mtime.getTime() > buildTime) {
                        this.warnings.push(`Source file ${file} is newer than build`);
                        return true;
                    }
                }
                return false;
            };
            
            if (fs.existsSync(srcPath)) {
                checkSourceFiles(srcPath);
            }
            
            // Check build age
            const ageMs = Date.now() - buildTime;
            const ageMinutes = Math.floor(ageMs / (1000 * 60));
            
            if (ageMinutes > 120) { // 2 hours
                this.warnings.push(`NEXUS portal build is ${ageMinutes} minutes old`);
            } else {
                this.log(`Build is ${ageMinutes} minutes old`, 'success');
            }
            
        } catch (error) {
            this.warnings.push('Could not check build freshness');
        }
        
        return true;
    }

    async testViteBuild() {
        this.log('Testing Vite build capability...');

        const nexusPath = this.getNexusPath();
        
        try {
            // Test if vite can be invoked
            const result = execSync('npm run build-only --dry-run', {
                cwd: nexusPath,
                encoding: 'utf8',
                timeout: 10000,
                stdio: 'pipe'
            });
            
            this.log('Vite build test passed', 'success');
            return true;
            
        } catch (error) {
            this.issues.push(`Vite build test failed: ${error.message}`);
            return false;
        }
    }

    async runNexusHealthCheck() {
        this.log('🏥 Starting NEXUS portal health check...');

        // Check if running in testnet mode
        const isTestnetMode = process.env.TESTNET_MODE === 'true' || process.env.NODE_ENV === 'testnet';
        const nexusPath = this.getNexusPath();

        // If in testnet mode and nexus-portal doesn't exist, skip all checks
        if (isTestnetMode && !fs.existsSync(nexusPath)) {
            this.log('Testnet mode detected with no nexus-portal - skipping all checks', 'info');
            return true;
        }

        const checks = [
            this.checkNexusDirectory(),
            this.checkNexusDependencies(),
            this.checkNexusBuild(),
            this.checkBuildFreshness()
        ];

        await Promise.all(checks);
        
        // Report results
        if (this.issues.length === 0 && this.warnings.length === 0) {
            this.log('NEXUS portal health check passed! 🎉', 'success');
            return true;
        }
        
        if (this.warnings.length > 0) {
            this.log(`Found ${this.warnings.length} warnings:`, 'warn');
            this.warnings.forEach(warning => this.log(`  - ${warning}`, 'warn'));
        }
        
        if (this.issues.length > 0) {
            this.log(`Found ${this.issues.length} critical issues:`, 'error');
            this.issues.forEach(issue => this.log(`  - ${issue}`, 'error'));
            
            this.log('NEXUS portal health check failed', 'error');
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
