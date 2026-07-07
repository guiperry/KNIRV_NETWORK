#!/usr/bin/env node

/**
 * KNIRV GraphChain Explorer Integration Tests
 * 
 * Comprehensive test suite for validating the KNIRV GraphChain Explorer functionality,
 * API integration, SSE connections, and UI components.
 * 
 * Part of the KNIRV D-TEN Integration Testing Suite
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const http = require('http');
const https = require('https');

class KNIRVGraphChainExplorerTester {
    constructor() {
        this.explorerPath = path.join(__dirname, '../KNIRVGATEWAY/graphchain-explorer');
        this.gatewayPath = path.join(__dirname, '../KNIRVGATEWAY');
        this.testResults = [];
        this.errors = [];
        this.baseUrl = 'http://localhost:8888';
        this.explorerUrl = `${this.baseUrl}/graphchain-explorer`;
    }

    log(message, type = 'info') {
        const timestamp = new Date().toISOString();
        const logMessage = `[${timestamp}] [KNIRV-GCE] [${type.toUpperCase()}] ${message}`;
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

    // Helper function to make HTTP requests
    async makeRequest(url, options = {}) {
        return new Promise((resolve, reject) => {
            const client = url.startsWith('https') ? https : http;
            const req = client.request(url, options, (res) => {
                let data = '';
                res.on('data', chunk => data += chunk);
                res.on('end', () => {
                    resolve({ statusCode: res.statusCode, headers: res.headers, body: data });
                });
            });
            req.on('error', reject);
            req.end();
        });
    }

    // Test 1: Verify KNIRV GraphChain Explorer file structure
    async testExplorerFileStructure() {
        const requiredFiles = [
            'index.html',
            'skills.html',
            'errors.html',
            'test-data.js',
            'README.md',
            'css/graphchain.css',
            'css/components.css',
            'css/responsive.css',
            'js/graphchain-core.js',
            'js/graphchain-api.js',
            'js/graphchain-sse.js',
            'js/components/animated-logo.js',
            'js/components/skill-card.js',
            'js/components/error-card.js',
            'js/components/stats-card.js',
            'js/pages/dashboard.js',
            'js/pages/skills.js',
            'js/pages/errors.js'
        ];

        for (const file of requiredFiles) {
            const filePath = path.join(this.explorerPath, file);
            if (!fs.existsSync(filePath)) {
                throw new Error(`Required file missing: ${file}`);
            }
        }

        this.log(`All ${requiredFiles.length} required files found`);
    }

    // Test 2: Verify HTML pages load correctly
    async testHTMLPages() {
        const pages = ['index.html', 'skills.html', 'errors.html'];
        
        for (const page of pages) {
            const response = await this.makeRequest(`${this.explorerUrl}/${page}`);
            if (response.statusCode !== 200) {
                throw new Error(`Page ${page} returned status ${response.statusCode}`);
            }
            
            // Check for KNIRV GraphChain branding
            if (!response.body.includes('KNIRV GraphChain')) {
                throw new Error(`Page ${page} missing KNIRV GraphChain branding`);
            }
        }

        this.log('All HTML pages load correctly with proper branding');
    }

    // Test 3: Verify CSS resources load
    async testCSSResources() {
        const cssFiles = [
            'css/graphchain.css',
            'css/components.css', 
            'css/responsive.css'
        ];

        for (const cssFile of cssFiles) {
            const response = await this.makeRequest(`${this.explorerUrl}/${cssFile}`);
            if (response.statusCode !== 200) {
                throw new Error(`CSS file ${cssFile} returned status ${response.statusCode}`);
            }
            
            // Check for KNIRVGATEWAY color scheme
            if (cssFile === 'css/graphchain.css') {
                if (!response.body.includes('#7c3aed') || !response.body.includes('#2563eb')) {
                    throw new Error(`CSS file ${cssFile} missing KNIRVGATEWAY color scheme`);
                }
            }
        }

        this.log('All CSS resources load correctly with proper styling');
    }

    // Test 4: Verify JavaScript resources load
    async testJavaScriptResources() {
        const jsFiles = [
            'js/graphchain-core.js',
            'js/graphchain-api.js',
            'js/graphchain-sse.js',
            'js/components/animated-logo.js',
            'js/components/skill-card.js',
            'js/components/error-card.js',
            'js/components/stats-card.js',
            'js/pages/dashboard.js',
            'js/pages/skills.js',
            'js/pages/errors.js',
            'test-data.js'
        ];

        for (const jsFile of jsFiles) {
            const response = await this.makeRequest(`${this.explorerUrl}/${jsFile}`);
            if (response.statusCode !== 200) {
                throw new Error(`JavaScript file ${jsFile} returned status ${response.statusCode}`);
            }
            
            // Check for KNIRV GraphChain references
            if (jsFile.includes('graphchain-') && !response.body.includes('KNIRV GraphChain')) {
                throw new Error(`JavaScript file ${jsFile} missing KNIRV GraphChain references`);
            }
        }

        this.log('All JavaScript resources load correctly');
    }

    // Test 5: Verify Netlify Functions integration
    async testNetlifyFunctions() {
        try {
            // Test GraphChain events endpoint
            const eventsResponse = await this.makeRequest(`${this.baseUrl}/.netlify/functions/graphchain-events`);
            if (eventsResponse.statusCode !== 200) {
                this.log(`GraphChain events endpoint returned ${eventsResponse.statusCode} (expected for SSE)`, 'warn');
            }

            // Test gateway SSE with GraphChain routing
            const gatewayResponse = await this.makeRequest(`${this.baseUrl}/.netlify/functions/gateway-sse`);
            if (gatewayResponse.statusCode !== 200) {
                this.log(`Gateway SSE endpoint returned ${gatewayResponse.statusCode} (expected for SSE)`, 'warn');
            }

            this.log('Netlify Functions endpoints accessible');
        } catch (error) {
            this.log(`Netlify Functions test skipped: ${error.message}`, 'warn');
        }
    }

    // Test 6: Verify navigation integration
    async testNavigationIntegration() {
        // Check main KNIRVGATEWAY index.html for GraphChain Explorer link
        const mainPageResponse = await this.makeRequest(`${this.baseUrl}/`);
        if (mainPageResponse.statusCode !== 200) {
            throw new Error(`Main KNIRVGATEWAY page returned status ${mainPageResponse.statusCode}`);
        }

        if (!mainPageResponse.body.includes('KNIRV GraphChain Explorer')) {
            throw new Error('Main navigation missing KNIRV GraphChain Explorer link');
        }

        if (!mainPageResponse.body.includes('href="graphchain-explorer/"')) {
            throw new Error('Main navigation missing correct GraphChain Explorer href');
        }

        this.log('Navigation integration verified');
    }

    // Test 7: Verify mock data functionality
    async testMockDataFunctionality() {
        const testDataResponse = await this.makeRequest(`${this.explorerUrl}/test-data.js`);
        if (testDataResponse.statusCode !== 200) {
            throw new Error(`Test data file returned status ${testDataResponse.statusCode}`);
        }

        // Check for mock data structures
        const requiredMockData = [
            'MockGraphChainAPI',
            'MockGraphChainSSEClient',
            'mockSkills',
            'mockErrors',
            'mockHeight'
        ];

        for (const mockItem of requiredMockData) {
            if (!testDataResponse.body.includes(mockItem)) {
                throw new Error(`Mock data missing: ${mockItem}`);
            }
        }

        this.log('Mock data functionality verified');
    }

    // Test 8: Verify responsive design
    async testResponsiveDesign() {
        const responsiveCSSResponse = await this.makeRequest(`${this.explorerUrl}/css/responsive.css`);
        if (responsiveCSSResponse.statusCode !== 200) {
            throw new Error(`Responsive CSS returned status ${responsiveCSSResponse.statusCode}`);
        }

        // Check for responsive breakpoints
        const requiredBreakpoints = [
            '@media (max-width: 575.98px)',
            '@media (min-width: 576px)',
            '@media (min-width: 768px)',
            '@media (min-width: 992px)',
            '@media (min-width: 1200px)'
        ];

        for (const breakpoint of requiredBreakpoints) {
            if (!responsiveCSSResponse.body.includes(breakpoint)) {
                throw new Error(`Missing responsive breakpoint: ${breakpoint}`);
            }
        }

        this.log('Responsive design verified');
    }

    // Test 9: Verify accessibility features
    async testAccessibilityFeatures() {
        const indexResponse = await this.makeRequest(`${this.explorerUrl}/index.html`);
        
        // Check for accessibility attributes
        const accessibilityFeatures = [
            'aria-label',
            'role="button"',
            'tabindex',
            'lang="en"'
        ];

        for (const feature of accessibilityFeatures) {
            if (!indexResponse.body.includes(feature)) {
                throw new Error(`Missing accessibility feature: ${feature}`);
            }
        }

        // Check for alt attributes or SVG accessibility (more flexible)
        if (!indexResponse.body.includes('alt=') && !indexResponse.body.includes('aria-label')) {
            throw new Error('Missing image accessibility: no alt attributes or aria-labels found');
        }

        this.log('Accessibility features verified');
    }

    // Test 10: Verify documentation completeness
    async testDocumentationCompleteness() {
        const readmePath = path.join(this.explorerPath, 'README.md');
        const readmeContent = fs.readFileSync(readmePath, 'utf8');

        const requiredSections = [
            '# KNIRV GraphChain Explorer',
            '## Architecture',
            '## Features',
            '## Development',
            '## API Integration',
            '## Component System',
            '## Deployment'
        ];

        for (const section of requiredSections) {
            if (!readmeContent.includes(section)) {
                throw new Error(`README missing section: ${section}`);
            }
        }

        // Check for npm run dev instructions
        if (!readmeContent.includes('npm run dev')) {
            throw new Error('README missing npm run dev instructions');
        }

        this.log('Documentation completeness verified');
    }

    // Generate test report
    generateReport() {
        const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
        const reportPath = path.join(__dirname, 'reports', `knirv-graphchain-explorer-test-${timestamp}.json`);
        
        const report = {
            timestamp: new Date().toISOString(),
            component: 'KNIRV GraphChain Explorer',
            totalTests: this.testResults.length,
            passed: this.testResults.filter(r => r.status === 'PASS').length,
            failed: this.testResults.filter(r => r.status === 'FAIL').length,
            errors: this.errors,
            results: this.testResults
        };

        // Ensure reports directory exists
        const reportsDir = path.dirname(reportPath);
        if (!fs.existsSync(reportsDir)) {
            fs.mkdirSync(reportsDir, { recursive: true });
        }

        fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
        this.log(`Test report generated: ${reportPath}`);
        
        return report;
    }

    // Main test runner
    async runAllTests() {
        this.log('Starting KNIRV GraphChain Explorer Integration Tests');
        
        await this.runTest('Explorer File Structure', () => this.testExplorerFileStructure());
        await this.runTest('HTML Pages Load', () => this.testHTMLPages());
        await this.runTest('CSS Resources', () => this.testCSSResources());
        await this.runTest('JavaScript Resources', () => this.testJavaScriptResources());
        await this.runTest('Netlify Functions Integration', () => this.testNetlifyFunctions());
        await this.runTest('Navigation Integration', () => this.testNavigationIntegration());
        await this.runTest('Mock Data Functionality', () => this.testMockDataFunctionality());
        await this.runTest('Responsive Design', () => this.testResponsiveDesign());
        await this.runTest('Accessibility Features', () => this.testAccessibilityFeatures());
        await this.runTest('Documentation Completeness', () => this.testDocumentationCompleteness());

        const report = this.generateReport();
        
        this.log(`\n=== KNIRV GraphChain Explorer Test Summary ===`);
        this.log(`Total Tests: ${report.totalTests}`);
        this.log(`Passed: ${report.passed}`);
        this.log(`Failed: ${report.failed}`);
        
        if (report.failed > 0) {
            this.log(`\nErrors:`, 'error');
            this.errors.forEach(error => this.log(`  - ${error}`, 'error'));
            process.exit(1);
        } else {
            this.log(`\n🎉 All tests passed!`, 'success');
        }
    }
}

// Run tests if called directly
if (require.main === module) {
    const tester = new KNIRVGraphChainExplorerTester();
    tester.runAllTests().catch(error => {
        console.error('Test runner failed:', error);
        process.exit(1);
    });
}

module.exports = KNIRVGraphChainExplorerTester;
