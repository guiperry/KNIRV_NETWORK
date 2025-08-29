#!/usr/bin/env node

/**
 * KNIRV Developer Portal Validation Script
 * 
 * Quick validation script to check portal structure and basic functionality
 * before running comprehensive integration tests.
 */

const fs = require('fs');
const path = require('path');

class PortalValidator {
    constructor() {
        this.portalPath = path.join(__dirname, '../KNIRVGATEWAY/agent-developer-portal/static');
        this.websitePath = path.join(__dirname, '../KNIRVGATEWAY');
        this.issues = [];
        this.warnings = [];
    }

    log(message, type = 'info') {
        const colors = {
            info: '\x1b[36m',    // Cyan
            success: '\x1b[32m', // Green
            warning: '\x1b[33m', // Yellow
            error: '\x1b[31m',   // Red
            reset: '\x1b[0m'     // Reset
        };

        console.log(`${colors[type]}[${type.toUpperCase()}] ${message}${colors.reset}`);
    }

    addIssue(message) {
        this.issues.push(message);
        this.log(message, 'error');
    }

    addWarning(message) {
        this.warnings.push(message);
        this.log(message, 'warning');
    }

    // Check if portal directory exists and has correct structure
    validatePortalStructure() {
        this.log('Validating portal directory structure...');

        if (!fs.existsSync(this.portalPath)) {
            this.addIssue(`Portal directory does not exist: ${this.portalPath}`);
            return false;
        }

        const requiredDirs = ['css', 'js'];
        for (const dir of requiredDirs) {
            const dirPath = path.join(this.portalPath, dir);
            if (!fs.existsSync(dirPath)) {
                this.addIssue(`Required directory missing: ${dir}`);
            }
        }

        return true;
    }

    // Validate all required HTML pages exist
    validateHTMLPages() {
        this.log('Validating HTML pages...');

        const requiredPages = [
            'index.html',
            'core-concepts.html',
            'getting-started.html',
            'agent-management.html',
            'skill-registry.html',
            'udc-management.html',
            'agent-skill-exchange.html',
            'error-node-explorer.html',
            'api-sdk.html',
            'wallet-management.html',
            'tesnet-sandbox.html',
            'community-support.html'
        ];

        let validPages = 0;
        for (const page of requiredPages) {
            const pagePath = path.join(this.portalPath, page);
            if (fs.existsSync(pagePath)) {
                const stats = fs.statSync(pagePath);
                if (stats.size > 0) {
                    validPages++;
                } else {
                    this.addWarning(`Page exists but is empty: ${page}`);
                }
            } else {
                this.addIssue(`Required page missing: ${page}`);
            }
        }

        this.log(`Found ${validPages}/${requiredPages.length} valid HTML pages`, 'success');
        return validPages === requiredPages.length;
    }

    // Validate CSS and JavaScript files
    validateAssets() {
        this.log('Validating CSS and JavaScript assets...');

        const requiredAssets = [
            'css/portal.css',
            'js/portal.js',
            'js/udc-management.js'
        ];

        let validAssets = 0;
        for (const asset of requiredAssets) {
            const assetPath = path.join(this.portalPath, asset);
            if (fs.existsSync(assetPath)) {
                const stats = fs.statSync(assetPath);
                if (stats.size > 0) {
                    validAssets++;
                } else {
                    this.addWarning(`Asset exists but is empty: ${asset}`);
                }
            } else {
                this.addIssue(`Required asset missing: ${asset}`);
            }
        }

        this.log(`Found ${validAssets}/${requiredAssets.length} valid assets`, 'success');
        return validAssets === requiredAssets.length;
    }

    // Check for basic HTML structure in key pages
    validateHTMLStructure() {
        this.log('Validating HTML structure...');

        const keyPages = ['index.html', 'getting-started.html', 'agent-management.html'];
        let validStructures = 0;

        for (const page of keyPages) {
            const pagePath = path.join(this.portalPath, page);
            if (fs.existsSync(pagePath)) {
                const content = fs.readFileSync(pagePath, 'utf8');
                
                const checks = [
                    { test: content.includes('<!DOCTYPE html>'), name: 'DOCTYPE' },
                    { test: content.includes('<title>'), name: 'Title tag' },
                    { test: content.includes('KNIRV'), name: 'KNIRV branding' },
                    { test: content.includes('nav-item'), name: 'Navigation structure' },
                    { test: content.includes('portal.css'), name: 'CSS include' },
                    { test: content.includes('portal.js'), name: 'JS include' }
                ];

                let pageValid = true;
                for (const check of checks) {
                    if (!check.test) {
                        this.addWarning(`${page}: Missing ${check.name}`);
                        pageValid = false;
                    }
                }

                if (pageValid) {
                    validStructures++;
                }
            }
        }

        this.log(`${validStructures}/${keyPages.length} pages have valid structure`, 'success');
        return validStructures === keyPages.length;
    }

    // Validate main website integration
    validateWebsiteIntegration() {
        this.log('Validating main website integration...');

        const indexPath = path.join(this.websitePath, 'index.html');
        if (!fs.existsSync(indexPath)) {
            this.addIssue('Main website index.html not found');
            return false;
        }

        const content = fs.readFileSync(indexPath, 'utf8');
        const integrationChecks = [
            { test: content.includes('agent-developer-portal'), name: 'Portal link' },
            { test: content.includes('Developer Portal'), name: 'Portal navigation' },
            { test: content.includes('Developer Portal') && content.includes('btn'), name: 'CTA button' }
        ];

        let validIntegrations = 0;
        for (const check of integrationChecks) {
            if (check.test) {
                validIntegrations++;
            } else {
                this.addWarning(`Main website missing: ${check.name}`);
            }
        }

        this.log(`${validIntegrations}/${integrationChecks.length} integration points found`, 'success');
        return validIntegrations >= 2; // At least 2 out of 3 should be present
    }

    // Validate Netlify configuration
    validateNetlifyConfig() {
        this.log('Validating Netlify configuration...');

        const netlifyPath = path.join(this.websitePath, 'netlify.toml');
        if (!fs.existsSync(netlifyPath)) {
            this.addIssue('netlify.toml not found');
            return false;
        }

        const content = fs.readFileSync(netlifyPath, 'utf8');
        const configChecks = [
            { test: content.includes('/portal/*'), name: 'Portal redirect' },
            { test: content.includes('/developer/*'), name: 'Developer redirect' },
            { test: content.includes('agent-developer-portal/static'), name: 'Publish directory' }
        ];

        let validConfigs = 0;
        for (const check of configChecks) {
            if (check.test) {
                validConfigs++;
            } else {
                this.addWarning(`Netlify config missing: ${check.name}`);
            }
        }

        this.log(`${validConfigs}/${configChecks.length} Netlify configurations found`, 'success');
        return validConfigs >= 2;
    }

    // Check package.json configuration
    validatePackageConfig() {
        this.log('Validating package.json configuration...');

        const packagePath = path.join(__dirname, '../KNIRVGATEWAY/agent-developer-portal/package.json');
        if (!fs.existsSync(packagePath)) {
            this.addIssue('package.json not found');
            return false;
        }

        try {
            const packageContent = JSON.parse(fs.readFileSync(packagePath, 'utf8'));
            
            const packageChecks = [
                { test: packageContent.name === 'knirv-developer-portal', name: 'Correct package name' },
                { test: packageContent.version.startsWith('2.'), name: 'Version 2.x.x' },
                { test: packageContent.scripts && packageContent.scripts.test, name: 'Test script' },
                { test: packageContent.main !== 'server.js', name: 'No server.js reference' }
            ];

            let validPackageConfigs = 0;
            for (const check of packageChecks) {
                if (check.test) {
                    validPackageConfigs++;
                } else {
                    this.addWarning(`Package.json: ${check.name} check failed`);
                }
            }

            this.log(`${validPackageConfigs}/${packageChecks.length} package configurations valid`, 'success');
            return validPackageConfigs >= 3;
        } catch (error) {
            this.addIssue(`Failed to parse package.json: ${error.message}`);
            return false;
        }
    }

    // Check for legacy files that should be removed
    validateLegacyCleanup() {
        this.log('Checking for legacy files...');

        const legacyFiles = [
            path.join(__dirname, '../KNIRVGATEWAY/agent-developer-portal/server.js'),
            path.join(this.portalPath, 'dashboard.html'),
            path.join(this.portalPath, 'inventory.html'),
            path.join(this.portalPath, 'dex.html'),
            path.join(this.portalPath, 'explorer.html')
        ];

        let legacyFound = 0;
        for (const file of legacyFiles) {
            if (fs.existsSync(file)) {
                this.addWarning(`Legacy file still exists: ${path.basename(file)}`);
                legacyFound++;
            }
        }

        if (legacyFound === 0) {
            this.log('No legacy files found - good cleanup!', 'success');
        }

        return legacyFound === 0;
    }

    // Generate validation report
    generateReport() {
        console.log('\n' + '='.repeat(50));
        console.log('KNIRV DEVELOPER PORTAL VALIDATION REPORT');
        console.log('='.repeat(50));

        if (this.issues.length === 0 && this.warnings.length === 0) {
            this.log('✅ All validations passed! Portal is ready for testing.', 'success');
        } else {
            if (this.issues.length > 0) {
                console.log(`\n❌ CRITICAL ISSUES (${this.issues.length}):`);
                this.issues.forEach(issue => console.log(`  - ${issue}`));
            }

            if (this.warnings.length > 0) {
                console.log(`\n⚠️  WARNINGS (${this.warnings.length}):`);
                this.warnings.forEach(warning => console.log(`  - ${warning}`));
            }
        }

        console.log('\n' + '='.repeat(50));
        return this.issues.length === 0;
    }

    // Main validation runner
    async validate() {
        this.log('Starting KNIRV Developer Portal Validation');
        this.log(`Portal path: ${this.portalPath}`);

        const validations = [
            () => this.validatePortalStructure(),
            () => this.validateHTMLPages(),
            () => this.validateAssets(),
            () => this.validateHTMLStructure(),
            () => this.validateWebsiteIntegration(),
            () => this.validateNetlifyConfig(),
            () => this.validatePackageConfig(),
            () => this.validateLegacyCleanup()
        ];

        let passedValidations = 0;
        for (const validation of validations) {
            if (validation()) {
                passedValidations++;
            }
        }

        this.log(`\nValidation Summary: ${passedValidations}/${validations.length} checks passed`);

        const success = this.generateReport();
        
        if (success) {
            this.log('\n🚀 Portal is ready for comprehensive integration testing!', 'success');
            this.log('Run: npm test (from agent-developer-portal directory)', 'info');
        } else {
            this.log('\n🔧 Please fix the issues above before running integration tests.', 'warning');
        }

        return success;
    }
}

// Run validation if this file is executed directly
if (require.main === module) {
    const validator = new PortalValidator();
    validator.validate().then(success => {
        process.exit(success ? 0 : 1);
    }).catch(error => {
        console.error('Validation failed:', error);
        process.exit(1);
    });
}

module.exports = PortalValidator;
