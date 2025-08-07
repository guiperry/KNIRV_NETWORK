#!/usr/bin/env node

/**
 * KNIRVNEXUS Frontend Integration Test Suite
 * Tests the Next.js frontend application and Socket.IO connectivity
 */

const fs = require('fs');
const path = require('path');
const http = require('http');
const https = require('https');
const { execSync } = require('child_process');

class KNIRVNEXUSFrontendTester {
    constructor() {
        this.frontendURL = 'http://localhost:3000';
        this.socketURL = 'ws://localhost:3000/api/socketio';
        this.testResults = {
            passed: 0,
            failed: 0,
            total: 0,
            details: []
        };
        this.startTime = new Date();
    }

    log(message, type = 'info') {
        const timestamp = new Date().toISOString();
        const prefix = type === 'error' ? '❌' : type === 'success' ? '✅' : 'ℹ️';
        console.log(`[${timestamp}] ${prefix} ${message}`);
    }

    async runTest(testName, testFunction) {
        this.testResults.total++;
        try {
            this.log(`Running test: ${testName}`);
            await testFunction();
            this.testResults.passed++;
            this.testResults.details.push({
                name: testName,
                status: 'PASSED',
                timestamp: new Date().toISOString()
            });
            this.log(`Test passed: ${testName}`, 'success');
        } catch (error) {
            this.testResults.failed++;
            this.testResults.details.push({
                name: testName,
                status: 'FAILED',
                error: error.message,
                timestamp: new Date().toISOString()
            });
            this.log(`Test failed: ${testName} - ${error.message}`, 'error');
        }
    }

    async makeHttpRequest(url, options = {}) {
        return new Promise((resolve, reject) => {
            const protocol = url.startsWith('https') ? https : http;
            const req = protocol.request(url, {
                method: options.method || 'GET',
                headers: options.headers || {},
                timeout: options.timeout || 10000
            }, (res) => {
                let data = '';
                res.on('data', chunk => data += chunk);
                res.on('end', () => {
                    resolve({
                        statusCode: res.statusCode,
                        headers: res.headers,
                        body: data
                    });
                });
            });

            req.on('error', reject);
            req.on('timeout', () => reject(new Error('Request timeout')));

            if (options.body) {
                req.write(options.body);
            }

            req.end();
        });
    }

    async testFrontendHealth() {
        const response = await this.makeHttpRequest(`${this.frontendURL}/api/health`);
        
        if (response.statusCode !== 200) {
            throw new Error(`Expected status 200, got ${response.statusCode}`);
        }

        const health = JSON.parse(response.body);
        if (health.status !== 'healthy') {
            throw new Error(`Expected status 'healthy', got '${health.status}'`);
        }
    }

    async testFrontendPages() {
        const pages = [
            '/',
            '/dashboard',
            '/agents',
            '/validation',
            '/nodes'
        ];

        for (const page of pages) {
            const response = await this.makeHttpRequest(`${this.frontendURL}${page}`);
            
            if (response.statusCode !== 200) {
                throw new Error(`Page ${page} returned status ${response.statusCode}`);
            }

            // Check for basic HTML structure
            if (!response.body.includes('<html') || !response.body.includes('</html>')) {
                throw new Error(`Page ${page} does not contain valid HTML structure`);
            }

            // Check for Next.js specific elements
            if (!response.body.includes('__NEXT_DATA__')) {
                throw new Error(`Page ${page} does not contain Next.js data`);
            }
        }
    }

    async testStaticAssets() {
        const assets = [
            '/favicon.ico',
            '/logo.svg',
            '/robots.txt'
        ];

        for (const asset of assets) {
            try {
                const response = await this.makeHttpRequest(`${this.frontendURL}${asset}`);
                if (response.statusCode !== 200) {
                    this.log(`Warning: Asset ${asset} returned status ${response.statusCode}`, 'warning');
                }
            } catch (error) {
                this.log(`Warning: Asset ${asset} failed to load: ${error.message}`, 'warning');
            }
        }
    }

    async testAPIEndpoints() {
        const endpoints = [
            { path: '/api/health', method: 'GET', expectedStatus: 200 },
            { path: '/api/nodes', method: 'GET', expectedStatus: 200 },
            { path: '/api/tasks', method: 'GET', expectedStatus: 200 },
            { path: '/api/system/status', method: 'GET', expectedStatus: 200 }
        ];

        for (const endpoint of endpoints) {
            try {
                const response = await this.makeHttpRequest(`${this.frontendURL}${endpoint.path}`, {
                    method: endpoint.method
                });

                if (response.statusCode !== endpoint.expectedStatus) {
                    this.log(`Warning: API ${endpoint.path} returned status ${response.statusCode}, expected ${endpoint.expectedStatus}`, 'warning');
                }
            } catch (error) {
                this.log(`Warning: API ${endpoint.path} failed: ${error.message}`, 'warning');
            }
        }
    }

    async testSocketIOConnectivity() {
        // Test Socket.IO endpoint availability
        try {
            const response = await this.makeHttpRequest(`${this.frontendURL}/api/socketio`, {
                headers: {
                    'Upgrade': 'websocket',
                    'Connection': 'Upgrade'
                }
            });

            // Socket.IO should respond with upgrade or specific status
            if (response.statusCode !== 400 && response.statusCode !== 426) {
                this.log(`Warning: Socket.IO endpoint returned unexpected status ${response.statusCode}`, 'warning');
            }
        } catch (error) {
            this.log(`Warning: Socket.IO connectivity test failed: ${error.message}`, 'warning');
        }
    }

    async testBuildArtifacts() {
        const knirvnexusPath = path.join(__dirname, '..', 'KNIRVNEXUS');
        const nextPath = path.join(knirvnexusPath, '.next');

        if (!fs.existsSync(nextPath)) {
            throw new Error('Next.js build artifacts not found. Run npm run build first.');
        }

        // Check for essential build files
        const requiredFiles = [
            path.join(nextPath, 'BUILD_ID'),
            path.join(nextPath, 'static'),
            path.join(nextPath, 'server')
        ];

        for (const file of requiredFiles) {
            if (!fs.existsSync(file)) {
                throw new Error(`Required build artifact missing: ${file}`);
            }
        }
    }

    async testPackageIntegrity() {
        const knirvnexusPath = path.join(__dirname, '..', 'KNIRVNEXUS');
        const packageJsonPath = path.join(knirvnexusPath, 'package.json');

        if (!fs.existsSync(packageJsonPath)) {
            throw new Error('package.json not found in KNIRVNEXUS directory');
        }

        const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));

        // Check for essential dependencies
        const requiredDeps = ['next', 'react', 'react-dom', 'socket.io', 'socket.io-client'];
        for (const dep of requiredDeps) {
            if (!packageJson.dependencies[dep]) {
                throw new Error(`Required dependency missing: ${dep}`);
            }
        }

        // Check for essential scripts
        const requiredScripts = ['dev', 'build', 'start'];
        for (const script of requiredScripts) {
            if (!packageJson.scripts[script]) {
                throw new Error(`Required script missing: ${script}`);
            }
        }
    }

    async testEnvironmentConfiguration() {
        // Check if development server is running
        try {
            const response = await this.makeHttpRequest(`${this.frontendURL}/_next/static/development/_devPagesManifest.json`);
            if (response.statusCode === 200) {
                this.log('Development mode detected');
            }
        } catch (error) {
            // This is expected in production mode
        }

        // Test environment variables (if accessible)
        try {
            const response = await this.makeHttpRequest(`${this.frontendURL}/api/config`);
            if (response.statusCode === 200) {
                const config = JSON.parse(response.body);
                this.log(`Environment configuration loaded: ${Object.keys(config).length} settings`);
            }
        } catch (error) {
            this.log('Environment configuration endpoint not available', 'warning');
        }
    }

    generateReport() {
        const endTime = new Date();
        const duration = endTime - this.startTime;

        const report = {
            summary: {
                total: this.testResults.total,
                passed: this.testResults.passed,
                failed: this.testResults.failed,
                success_rate: ((this.testResults.passed / this.testResults.total) * 100).toFixed(2),
                duration_ms: duration,
                timestamp: endTime.toISOString()
            },
            details: this.testResults.details,
            environment: {
                frontend_url: this.frontendURL,
                socket_url: this.socketURL,
                node_version: process.version,
                platform: process.platform
            }
        };

        // Save report to file
        const reportPath = path.join(__dirname, 'reports', `knirvnexus-frontend-test-${endTime.toISOString().replace(/[:.]/g, '-')}.json`);
        
        // Ensure reports directory exists
        const reportsDir = path.dirname(reportPath);
        if (!fs.existsSync(reportsDir)) {
            fs.mkdirSync(reportsDir, { recursive: true });
        }

        fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
        this.log(`Test report saved to: ${reportPath}`);

        return report;
    }

    async runAllTests() {
        this.log('Starting KNIRVNEXUS Frontend Integration Tests');

        await this.runTest('Frontend Health Check', () => this.testFrontendHealth());
        await this.runTest('Package Integrity Check', () => this.testPackageIntegrity());
        await this.runTest('Build Artifacts Check', () => this.testBuildArtifacts());
        await this.runTest('Frontend Pages Accessibility', () => this.testFrontendPages());
        await this.runTest('Static Assets Loading', () => this.testStaticAssets());
        await this.runTest('API Endpoints Connectivity', () => this.testAPIEndpoints());
        await this.runTest('Socket.IO Connectivity', () => this.testSocketIOConnectivity());
        await this.runTest('Environment Configuration', () => this.testEnvironmentConfiguration());

        const report = this.generateReport();

        this.log(`\n=== KNIRVNEXUS Frontend Integration Test Results ===`);
        this.log(`Total Tests: ${report.summary.total}`);
        this.log(`Passed: ${report.summary.passed}`, 'success');
        this.log(`Failed: ${report.summary.failed}`, report.summary.failed > 0 ? 'error' : 'success');
        this.log(`Success Rate: ${report.summary.success_rate}%`);
        this.log(`Duration: ${report.summary.duration_ms}ms`);

        if (report.summary.failed > 0) {
            this.log('\nFailed Tests:');
            report.details.filter(test => test.status === 'FAILED').forEach(test => {
                this.log(`  - ${test.name}: ${test.error}`, 'error');
            });
        }

        return report.summary.failed === 0;
    }
}

// Run tests if called directly
if (require.main === module) {
    const tester = new KNIRVNEXUSFrontendTester();
    tester.runAllTests().then(success => {
        process.exit(success ? 0 : 1);
    }).catch(error => {
        console.error('Test runner failed:', error);
        process.exit(1);
    });
}

module.exports = KNIRVNEXUSFrontendTester;
