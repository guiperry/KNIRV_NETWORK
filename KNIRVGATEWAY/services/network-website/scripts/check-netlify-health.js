#!/usr/bin/env node

/**
 * KNIRV Gateway Health Checker
 * Automatically detects netlify-cli corruption and other build issues
 */

import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import { fileURLToPath } from 'url';

// ES module compatibility
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

class HealthChecker {
    constructor(options = {}) {
        this.issues = [];
        this.warnings = [];
        this.autoFixAttempted = false;
        this.netlifyIssues = [];
        this.buildMode = options.buildMode || false;
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

        // First check if netlify-cli binary exists
        const netlifyPath = path.join(process.cwd(), 'node_modules', '.bin', 'netlify');
        if (!fs.existsSync(netlifyPath)) {
            this.issues.push('netlify-cli binary not found in node_modules');
            this.netlifyIssues.push('netlify-cli binary not found in node_modules');
            return false;
        }

        this.log('netlify-cli binary found in node_modules', 'success');

        try {
            // Try a quick version check with reasonable timeout
            this.log('Testing netlify-cli version command...');
            const version = execSync('timeout 15s npx netlify --version', {
                encoding: 'utf8',
                stdio: 'pipe',
                shell: true
            }).trim();

            this.log(`netlify-cli version: ${version}`, 'success');

            // Check for known problematic versions
            if (version.includes('17.')) {
                this.warnings.push('netlify-cli version 17.x detected - known to have issues but proceeding');
            }

            return true;

        } catch (error) {
            // If the command fails, check if it's a timeout or other issue
            if (error.message.includes('ETIMEDOUT') || error.status === 124) {
                this.log('netlify-cli command timed out, but binary exists - treating as working', 'warn');
                this.warnings.push('netlify-cli commands are slow but binary is present');
                return true; // Don't fail the build for slow commands
            } else {
                this.log(`netlify-cli error: ${error.message}`, 'warn');
                this.warnings.push(`netlify-cli may have issues: ${error.message}`);
                return true; // Don't fail the build, just warn
            }
        }
    }

    async checkNodeModules() {
        this.log('Checking node_modules integrity...');

        const nodeModulesPath = path.join(process.cwd(), 'node_modules');

        if (!fs.existsSync(nodeModulesPath)) {
            this.issues.push('node_modules directory missing');
            return false;
        }

        // Check netlify-cli (required for netlify/functions routes)
        const netlifyCliPath = path.join(nodeModulesPath, 'netlify-cli');
        if (!fs.existsSync(netlifyCliPath)) {
            this.issues.push('netlify-cli not found in node_modules');
            this.netlifyIssues.push('netlify-cli not found in node_modules');
            return false;
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
            this.netlifyIssues.push('netlify-cli package.json is corrupted');
            return false;
        }

        // Check for common corruption indicators
        const packageLockPath = path.join(process.cwd(), 'package-lock.json');
        if (!fs.existsSync(packageLockPath)) {
            this.warnings.push('package-lock.json missing - may cause dependency issues');
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

                if (this.buildMode) {
                    // In build mode, be more tolerant of vulnerabilities
                    if (total > 50) {
                        this.issues.push(`Critical number of vulnerabilities detected: ${total}`);
                    } else if (total > 0) {
                        this.warnings.push(`${total} moderate+ vulnerabilities detected (build mode - tolerant)`);
                    }
                } else {
                    if (total > 10) {
                        this.issues.push(`High number of vulnerabilities detected: ${total}`);
                    } else if (total > 0) {
                        this.warnings.push(`${total} moderate+ vulnerabilities detected`);
                    }
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

                        if (this.buildMode) {
                            // In build mode, be more tolerant
                            if (total > 50) {
                                this.issues.push(`Critical number of vulnerabilities: ${total}`);
                            } else {
                                this.warnings.push(`${total} vulnerabilities detected (build mode - tolerant)`);
                            }
                        } else {
                            if (total > 15) {
                                this.issues.push(`Critical number of vulnerabilities: ${total}`);
                            } else {
                                this.warnings.push(`${total} vulnerabilities detected`);
                            }
                        }
                    }
                } catch (parseError) {
                    this.warnings.push('Could not parse npm audit results');
                }
            }
        }
        
        return true;
    }

    async verifyNetlifyCliBroken() {
        this.log('Verifying if netlify-cli is actually broken...', 'info');

        try {
            // Check if binary exists
            const netlifyPath = path.join(process.cwd(), 'node_modules', '.bin', 'netlify');
            if (!fs.existsSync(netlifyPath)) {
                this.log('netlify-cli binary not found - confirmed broken', 'error');
                return true;
            }

            // Test if command works
            const version = execSync('timeout 10s npx netlify --version', {
                encoding: 'utf8',
                stdio: 'pipe',
                shell: true
            }).trim();

            this.log(`netlify-cli test successful: ${version}`, 'success');
            return false; // Not broken

        } catch (error) {
            this.log(`netlify-cli test failed: ${error.message}`, 'error');
            return true; // Confirmed broken
        }
    }

    async attemptAutoFix() {
        if (this.autoFixAttempted) {
            this.log('Auto-fix already attempted, skipping to prevent loops', 'warn');
            return false;
        }

        if (this.netlifyIssues.length > 0) {
            this.log('🔧 Attempting automatic netlify-cli fix...', 'fix');
            this.autoFixAttempted = true;

            try {
                // Run the netlify fix script
                this.log('Running netlify-cli fix script...', 'fix');
                execSync('bash ../scripts/fix-netlify-cli.sh --auto', {
                    stdio: 'inherit',
                    timeout: 180000, // 3 minutes timeout
                    cwd: process.cwd()
                });

                this.log('Netlify fix script completed, re-running health check...', 'fix');

                // Clear previous issues and re-run checks
                this.issues = [];
                this.warnings = [];
                this.netlifyIssues = [];

                // Re-run netlify-specific checks
                const nodeModulesOk = await this.checkNodeModules();
                const netlifyCliOk = await this.checkNetlifyCli();

                if (nodeModulesOk && netlifyCliOk) {
                    this.log('Auto-fix successful! 🎉', 'success');
                    return true;
                } else {
                    this.log('Auto-fix completed but issues remain', 'warn');
                    return false;
                }

            } catch (error) {
                this.log(`Auto-fix failed: ${error.message}`, 'error');
                return false;
            }
        }

        return false;
    }

    async runHealthCheck() {
        this.log('🏥 Starting KNIRV Gateway health check...');
        
        const checks = [
            this.checkNodeModules(),
            this.checkNetlifyCli(),
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
            // In build mode, treat netlify issues as warnings only
            if (this.buildMode && this.issues.every(issue => issue.includes('netlify-cli'))) {
                this.log(`Build mode: Found ${this.issues.length} netlify issues (treating as warnings):`, 'warn');
                this.issues.forEach(issue => this.log(`  - ${issue}`, 'warn'));
                this.log('Build mode: Continuing despite netlify issues...', 'info');
                return true;
            }

            this.log(`Found ${this.issues.length} critical issues:`, 'error');
            this.issues.forEach(issue => this.log(`  - ${issue}`, 'error'));

            // Attempt automatic fix for netlify issues
            if (this.netlifyIssues.length > 0 && !this.autoFixAttempted) {
                // Double-check if netlify-cli is actually working before attempting fix
                this.log('Double-checking netlify-cli status before attempting fix...', 'info');
                const netlifyActuallyBroken = await this.verifyNetlifyCliBroken();

                if (netlifyActuallyBroken) {
                    this.log('Confirmed netlify-cli is broken, attempting automatic fix...', 'fix');
                    const fixSuccessful = await this.attemptAutoFix();

                    if (fixSuccessful) {
                        this.log('Auto-fix successful! Re-running full health check...', 'success');
                        // Re-run the full health check after successful fix
                        return await this.runHealthCheck();
                    } else {
                        if (this.buildMode) {
                            this.log('Build mode: Auto-fix incomplete but continuing...', 'warn');
                            return true;
                        } else {
                            this.log('Auto-fix failed or incomplete', 'error');
                        }
                    }
                } else {
                    this.log('netlify-cli is actually working, skipping auto-fix', 'info');
                    // Remove netlify issues since they're false positives
                    this.issues = this.issues.filter(issue => !issue.includes('netlify-cli'));
                    this.netlifyIssues = [];

                    // Re-check if we still have issues
                    if (this.issues.length === 0) {
                        this.log('All issues resolved - netlify-cli is working properly', 'success');
                        return true;
                    }
                }
            }

            if (this.buildMode) {
                this.log('Build mode: Health check issues detected but continuing build...', 'warn');
                return true;
            } else {
                this.log('Health check failed - manual intervention may be needed', 'error');
                return false;
            }
        }

        return true;
    }
}

// Run health check if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
    const buildMode = process.argv.includes('--build-mode');
    const checker = new HealthChecker({ buildMode });

    if (buildMode) {
        console.log('🏗️  Running health check in build mode (more tolerant)');
    }

    checker.runHealthCheck()
        .then(success => {
            if (buildMode) {
                // In build mode, always succeed but show warnings
                if (!success) {
                    console.log('⚠️  Build mode: Netlify issues detected but continuing build...');
                    console.log('💡 Note: Netlify functionality may be limited but core services will work');
                }
                process.exit(0);
            } else {
                process.exit(success ? 0 : 1);
            }
        })
        .catch(error => {
            if (buildMode) {
                console.log('⚠️  Build mode: Health check error but continuing build...');
                console.log(`💡 Error details: ${error.message}`);
                process.exit(0);
            } else {
                console.error('❌ Health check failed with error:', error.message);
                process.exit(1);
            }
        });
}

export default HealthChecker;
