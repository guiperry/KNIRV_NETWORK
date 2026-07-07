#!/usr/bin/env node

/**
 * KNIRV Developer Portal Integration Tests
 * 
 * Comprehensive test suite for validating the developer portal functionality,
 * navigation, and integration with the main KNIRV website.
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class PortalIntegrationTester {
    constructor() {
        this.portalPath = path.join(__dirname, '../KNIRVGATEWAY/developer-portal/static');
        this.websitePath = path.join(__dirname, '../KNIRVGATEWAY');
        this.testResults = [];
        this.errors = [];
    }

    log(message, type = 'info') {
        const timestamp = new Date().toISOString();
        const logMessage = `[${timestamp}] [${type.toUpperCase()}] ${message}`;
        console.log(logMessage);
        
        if (type === 'error') {
            this.errors.push(message);
        }
    }

    async runTest(testName, testFunction) {
        this.log(`Running test: ${testName}`);
        try {
            const startTime = Date.now();
            await testFunction();
            const duration = Date.now() - startTime;
            this.testResults.push({ name: testName, status: 'PASS', duration });
            this.log(`✅ ${testName} - PASSED (${duration}ms)`, 'success');
        } catch (error) {
            this.testResults.push({ name: testName, status: 'FAIL', error: error.message });
            this.log(`❌ ${testName} - FAILED: ${error.message}`, 'error');
        }
    }

    // Test 1: Verify portal file structure
    async testPortalFileStructure() {
        const requiredFiles = [
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
            'community-support.html',
            'css/portal.css',
            'js/portal.js',
            'js/udc-management.js'
        ];

        for (const file of requiredFiles) {
            const filePath = path.join(this.portalPath, file);
            if (!fs.existsSync(filePath)) {
                throw new Error(`Required file missing: ${file}`);
            }
        }

        // Check file sizes (should not be empty)
        for (const file of requiredFiles) {
            const filePath = path.join(this.portalPath, file);
            const stats = fs.statSync(filePath);
            if (stats.size === 0) {
                throw new Error(`File is empty: ${file}`);
            }
        }
    }

    // Test 2: Validate HTML structure and navigation
    async testHTMLStructureAndNavigation() {
        const htmlFiles = [
            'index.html', 'core-concepts.html', 'getting-started.html',
            'agent-management.html', 'skill-registry.html', 'udc-management.html',
            'agent-skill-exchange.html', 'error-node-explorer.html', 'api-sdk.html',
            'wallet-management.html', 'tesnet-sandbox.html', 'community-support.html'
        ];

        for (const file of htmlFiles) {
            const filePath = path.join(this.portalPath, file);
            const content = fs.readFileSync(filePath, 'utf8');

            // Check for required HTML structure
            if (!content.includes('<!DOCTYPE html>')) {
                throw new Error(`${file}: Missing DOCTYPE declaration`);
            }

            if (!content.includes('<title>')) {
                throw new Error(`${file}: Missing title tag`);
            }

            if (!content.includes('KNIRV Developer Portal')) {
                throw new Error(`${file}: Missing KNIRV branding in title`);
            }

            // Check for navigation structure
            if (!content.includes('nav-item')) {
                throw new Error(`${file}: Missing navigation structure`);
            }

            // Check for responsive design
            if (!content.includes('viewport')) {
                throw new Error(`${file}: Missing viewport meta tag`);
            }

            // Check for CSS and JS includes
            if (!content.includes('portal.css')) {
                throw new Error(`${file}: Missing portal.css include`);
            }

            if (!content.includes('portal.js')) {
                throw new Error(`${file}: Missing portal.js include`);
            }

            // Check for Tailwind CSS
            if (!content.includes('tailwindcss.com')) {
                throw new Error(`${file}: Missing Tailwind CSS`);
            }

            // Check for Font Awesome
            if (!content.includes('font-awesome')) {
                throw new Error(`${file}: Missing Font Awesome`);
            }
        }
    }

    // Test 3: Validate navigation links consistency
    async testNavigationConsistency() {
        const htmlFiles = [
            'index.html', 'core-concepts.html', 'getting-started.html',
            'agent-management.html', 'skill-registry.html', 'udc-management.html',
            'agent-skill-exchange.html', 'error-node-explorer.html', 'api-sdk.html',
            'wallet-management.html', 'tesnet-sandbox.html', 'community-support.html'
        ];

        const expectedNavItems = [
            { href: 'index.html', text: 'Home Dashboard' },
            { href: 'core-concepts.html', text: 'Core Concepts' },
            { href: 'getting-started.html', text: 'Getting Started' },
            { href: 'agent-management.html', text: 'Agent Management' },
            { href: 'skill-registry.html', text: 'Skill Registry' },
            { href: 'udc-management.html', text: 'UDC Management' },
            { href: 'agent-skill-exchange.html', text: 'Agent/Skill Exchange' },
            { href: 'error-node-explorer.html', text: 'ErrorNode Explorer' },
            { href: 'api-sdk.html', text: 'API & SDK' },
            { href: 'wallet-management.html', text: 'KNIRV Wallets' },
            { href: 'tesnet-sandbox.html', text: 'TESNET & Sandbox' },
            { href: 'community-support.html', text: 'Community & Support' }
        ];

        for (const file of htmlFiles) {
            const filePath = path.join(this.portalPath, file);
            const content = fs.readFileSync(filePath, 'utf8');

            for (const navItem of expectedNavItems) {
                if (!content.includes(`href="${navItem.href}"`)) {
                    throw new Error(`${file}: Missing navigation link to ${navItem.href}`);
                }
                if (!content.includes(navItem.text)) {
                    throw new Error(`${file}: Missing navigation text "${navItem.text}"`);
                }
            }
        }
    }

    // Test 4: Validate CSS and JavaScript files
    async testCSSAndJavaScript() {
        // Test CSS file
        const cssPath = path.join(this.portalPath, 'css/portal.css');
        const cssContent = fs.readFileSync(cssPath, 'utf8');

        // Check for KNIRV branding colors
        if (!cssContent.includes('--knirv-primary')) {
            throw new Error('CSS: Missing KNIRV primary color variable');
        }

        if (!cssContent.includes('--knirv-secondary')) {
            throw new Error('CSS: Missing KNIRV secondary color variable');
        }

        // Check for key CSS classes
        const requiredClasses = [
            '.glass-card', '.nav-item', '.btn-primary', '.btn-secondary', '.btn-outline'
        ];

        for (const className of requiredClasses) {
            if (!cssContent.includes(className)) {
                throw new Error(`CSS: Missing required class ${className}`);
            }
        }

        // Test main JavaScript file
        const jsPath = path.join(this.portalPath, 'js/portal.js');
        const jsContent = fs.readFileSync(jsPath, 'utf8');

        // Check for main portal class
        if (!jsContent.includes('class KNIRVPortal')) {
            throw new Error('JS: Missing KNIRVPortal class');
        }

        // Check for key methods
        const requiredMethods = [
            'showNotification', 'loadUserData', 'checkNetworkStatus'
        ];

        for (const method of requiredMethods) {
            if (!jsContent.includes(method)) {
                throw new Error(`JS: Missing required method ${method}`);
            }
        }

        // Test UDC management JavaScript
        const udcJsPath = path.join(this.portalPath, 'js/udc-management.js');
        const udcJsContent = fs.readFileSync(udcJsPath, 'utf8');

        if (!udcJsContent.includes('class UDCManager')) {
            throw new Error('UDC JS: Missing UDCManager class');
        }
    }

    // Test 5: Validate main website integration
    async testMainWebsiteIntegration() {
        const indexPath = path.join(this.websitePath, 'index.html');
        const indexContent = fs.readFileSync(indexPath, 'utf8');

        // Check for developer portal links
        if (!indexContent.includes('developer-portal')) {
            throw new Error('Main website: Missing developer portal link');
        }

        // Check for navigation integration
        if (!indexContent.includes('Developer Portal')) {
            throw new Error('Main website: Missing Developer Portal in navigation');
        }

        // Check for call-to-action button (Developer Portal button)
        if (!indexContent.includes('Developer Portal') || !indexContent.includes('btn-grad-alternet')) {
            throw new Error('Main website: Missing Developer Portal CTA button');
        }
    }

    // Test 6: Validate Netlify configuration
    async testNetlifyConfiguration() {
        const netlifyConfigPath = path.join(this.websitePath, 'netlify.toml');
        const netlifyContent = fs.readFileSync(netlifyConfigPath, 'utf8');

        // Check for portal redirects
        const requiredRedirects = [
            '/portal/*', '/developer/*', '/dev-portal/*'
        ];

        for (const redirect of requiredRedirects) {
            if (!netlifyContent.includes(redirect)) {
                throw new Error(`Netlify config: Missing redirect for ${redirect}`);
            }
        }

        // Check for correct publish directory
        if (!netlifyContent.includes('developer-portal/static')) {
            throw new Error('Netlify config: Incorrect publish directory for portal');
        }
    }

    // Test 7: Validate package.json configuration
    async testPackageConfiguration() {
        const packagePath = path.join(__dirname, '../KNIRVGATEWAY/developer-portal/package.json');
        const packageContent = JSON.parse(fs.readFileSync(packagePath, 'utf8'));

        // Check for updated package name
        if (packageContent.name !== 'knirv-developer-portal') {
            throw new Error('Package.json: Incorrect package name');
        }

        // Check for version 2.0.0+
        if (!packageContent.version.startsWith('2.')) {
            throw new Error('Package.json: Version should be 2.x.x for refactored portal');
        }

        // Check for test scripts
        if (!packageContent.scripts.test) {
            throw new Error('Package.json: Missing test script');
        }

        if (!packageContent.scripts.validate) {
            throw new Error('Package.json: Missing validate script');
        }

        // Check that server.js is no longer the main entry point
        if (packageContent.main === 'server.js') {
            throw new Error('Package.json: Should not reference server.js as main entry point');
        }
    }

    // Test 8: Validate responsive design elements
    async testResponsiveDesign() {
        const htmlFiles = ['index.html', 'agent-management.html', 'skill-registry.html'];

        for (const file of htmlFiles) {
            const filePath = path.join(this.portalPath, file);
            const content = fs.readFileSync(filePath, 'utf8');

            // Check for responsive grid classes
            if (!content.includes('grid-cols-1') || !content.includes('md:grid-cols-')) {
                throw new Error(`${file}: Missing responsive grid classes`);
            }

            // Check for mobile navigation
            if (!content.includes('hidden md:block') && !content.includes('md:hidden')) {
                throw new Error(`${file}: Missing mobile navigation considerations`);
            }

            // Check for responsive spacing
            if (!content.includes('px-4') || !content.includes('sm:px-6')) {
                throw new Error(`${file}: Missing responsive spacing classes`);
            }
        }
    }

    // Test 9: Validate accessibility features
    async testAccessibilityFeatures() {
        const htmlFiles = ['index.html', 'getting-started.html', 'community-support.html'];

        for (const file of htmlFiles) {
            const filePath = path.join(this.portalPath, file);
            const content = fs.readFileSync(filePath, 'utf8');

            // Check for alt attributes on images
            const imgMatches = content.match(/<img[^>]*>/g);
            if (imgMatches) {
                for (const img of imgMatches) {
                    if (!img.includes('alt=')) {
                        throw new Error(`${file}: Image missing alt attribute: ${img}`);
                    }
                }
            }

            // Check for proper heading hierarchy
            if (content.includes('<h1>') && content.includes('<h3>') && !content.includes('<h2>')) {
                throw new Error(`${file}: Improper heading hierarchy (h1 to h3 without h2)`);
            }

            // Check for form labels
            const inputMatches = content.match(/<input[^>]*>/g);
            if (inputMatches) {
                // Should have corresponding labels
                if (!content.includes('<label') && !content.includes('aria-label')) {
                    throw new Error(`${file}: Form inputs without proper labels`);
                }
            }
        }
    }

    // Test 10: Validate KNIRV branding consistency
    async testBrandingConsistency() {
        const htmlFiles = [
            'index.html', 'core-concepts.html', 'getting-started.html',
            'agent-management.html', 'skill-registry.html'
        ];

        for (const file of htmlFiles) {
            const filePath = path.join(this.portalPath, file);
            const content = fs.readFileSync(filePath, 'utf8');

            // Check for consistent KNIRV branding
            if (!content.includes('KNIRV')) {
                throw new Error(`${file}: Missing KNIRV branding`);
            }

            // Check for D-TEN references
            if (file === 'core-concepts.html' && !content.includes('D-TEN')) {
                throw new Error(`${file}: Missing D-TEN references in core concepts`);
            }

            // Check for consistent color scheme
            if (!content.includes('bg-gray-900') && !content.includes('text-white')) {
                throw new Error(`${file}: Inconsistent color scheme`);
            }

            // Check for KNIRV-specific terminology
            const knirvTerms = ['NRN', 'ErrorNode', 'KNIRV-CHAIN', 'KNIRV-GRAPH'];
            let hasKnirvTerms = false;
            for (const term of knirvTerms) {
                if (content.includes(term)) {
                    hasKnirvTerms = true;
                    break;
                }
            }
            if (!hasKnirvTerms && file !== 'community-support.html') {
                throw new Error(`${file}: Missing KNIRV-specific terminology`);
            }
        }
    }

    // Generate test report
    generateReport() {
        const totalTests = this.testResults.length;
        const passedTests = this.testResults.filter(t => t.status === 'PASS').length;
        const failedTests = this.testResults.filter(t => t.status === 'FAIL').length;

        console.log('\n' + '='.repeat(60));
        console.log('KNIRV DEVELOPER PORTAL INTEGRATION TEST REPORT');
        console.log('='.repeat(60));
        console.log(`Total Tests: ${totalTests}`);
        console.log(`Passed: ${passedTests}`);
        console.log(`Failed: ${failedTests}`);
        console.log(`Success Rate: ${((passedTests / totalTests) * 100).toFixed(1)}%`);
        console.log('='.repeat(60));

        if (failedTests > 0) {
            console.log('\nFAILED TESTS:');
            this.testResults.filter(t => t.status === 'FAIL').forEach(test => {
                console.log(`❌ ${test.name}: ${test.error}`);
            });
        }

        if (this.errors.length > 0) {
            console.log('\nERRORS SUMMARY:');
            this.errors.forEach(error => {
                console.log(`- ${error}`);
            });
        }

        console.log('\nTEST DETAILS:');
        this.testResults.forEach(test => {
            const status = test.status === 'PASS' ? '✅' : '❌';
            const duration = test.duration ? ` (${test.duration}ms)` : '';
            console.log(`${status} ${test.name}${duration}`);
        });

        return failedTests === 0;
    }

    // Main test runner
    async runAllTests() {
        this.log('Starting KNIRV Developer Portal Integration Tests');
        this.log(`Portal path: ${this.portalPath}`);
        this.log(`Website path: ${this.websitePath}`);

        await this.runTest('Portal File Structure', () => this.testPortalFileStructure());
        await this.runTest('HTML Structure and Navigation', () => this.testHTMLStructureAndNavigation());
        await this.runTest('Navigation Consistency', () => this.testNavigationConsistency());
        await this.runTest('CSS and JavaScript', () => this.testCSSAndJavaScript());
        await this.runTest('Main Website Integration', () => this.testMainWebsiteIntegration());
        await this.runTest('Netlify Configuration', () => this.testNetlifyConfiguration());
        await this.runTest('Package Configuration', () => this.testPackageConfiguration());
        await this.runTest('Responsive Design', () => this.testResponsiveDesign());
        await this.runTest('Accessibility Features', () => this.testAccessibilityFeatures());
        await this.runTest('KNIRV Branding Consistency', () => this.testBrandingConsistency());

        const success = this.generateReport();
        process.exit(success ? 0 : 1);
    }
}

// Run tests if this file is executed directly
if (require.main === module) {
    const tester = new PortalIntegrationTester();
    tester.runAllTests().catch(error => {
        console.error('Test runner failed:', error);
        process.exit(1);
    });
}

module.exports = PortalIntegrationTester;
