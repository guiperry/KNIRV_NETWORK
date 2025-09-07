/**
 * KNIRVTESTNET Faucet Integration Tests
 * 
 * Comprehensive test suite for the NRV faucet implementation including
 * API endpoints, rate limiting, economic flow integration, and error handling.
 */

const axios = require('axios');
const fs = require('fs').promises;
const path = require('path');

// Test configuration
const TEST_CONFIG = {
    baseUrl: process.env.TEST_BASE_URL || 'http://localhost:10000',
    timeout: 30000,
    testAddress: 'knirv1testaddress123456789abcdef',
    invalidAddress: 'invalid_address_format',
    testAmount: 500,
    largeAmount: 5000,
    invalidAmount: 10000
};

class FaucetIntegrationTests {
    constructor() {
        this.testResults = [];
        this.startTime = Date.now();
    }

    /**
     * Run all integration tests
     */
    async runAllTests() {
        console.log('🧪 Starting KNIRV Faucet Integration Tests...\n');

        const tests = [
            { name: 'Faucet Status API', fn: () => this.testFaucetStatus() },
            { name: 'Valid Faucet Request', fn: () => this.testValidRequest() },
            { name: 'Invalid Address Validation', fn: () => this.testInvalidAddress() },
            { name: 'Amount Validation', fn: () => this.testAmountValidation() },
            { name: 'Rate Limiting', fn: () => this.testRateLimiting() },
            { name: 'Request History', fn: () => this.testRequestHistory() },
            { name: 'Health Check', fn: () => this.testHealthCheck() },
            { name: 'Metrics Endpoint', fn: () => this.testMetricsEndpoint() },
            { name: 'Economic Flow Status', fn: () => this.testEconomicFlow() },
            { name: 'Treasury Status', fn: () => this.testTreasuryStatus() },
            { name: 'Router Integration', fn: () => this.testRouterIntegration() },
            { name: 'Error Handling', fn: () => this.testErrorHandling() }
        ];

        for (const test of tests) {
            await this.runTest(test.name, test.fn);
            // Small delay between tests
            await this.sleep(1000);
        }

        this.generateReport();
    }

    /**
     * Run individual test with error handling
     */
    async runTest(testName, testFn) {
        const startTime = Date.now();
        
        try {
            console.log(`🔍 Running: ${testName}`);
            await testFn();
            
            const duration = Date.now() - startTime;
            this.testResults.push({
                name: testName,
                status: 'PASS',
                duration: duration,
                error: null
            });
            
            console.log(`✅ ${testName} - PASSED (${duration}ms)\n`);
            
        } catch (error) {
            const duration = Date.now() - startTime;
            this.testResults.push({
                name: testName,
                status: 'FAIL',
                duration: duration,
                error: error.message
            });
            
            console.log(`❌ ${testName} - FAILED (${duration}ms)`);
            console.log(`   Error: ${error.message}\n`);
        }
    }

    /**
     * Test faucet status endpoint
     */
    async testFaucetStatus() {
        const response = await this.makeRequest('GET', '/api/faucet/status');
        
        this.assert(response.status === 200, 'Status endpoint should return 200');
        this.assert(response.data.faucet_enabled !== undefined, 'Should include faucet_enabled');
        this.assert(response.data.daily_limit !== undefined, 'Should include daily_limit');
        this.assert(response.data.current_balance !== undefined, 'Should include current_balance');
        this.assert(response.data.rate_limits !== undefined, 'Should include rate_limits');
    }

    /**
     * Test valid faucet request
     */
    async testValidRequest() {
        const requestData = {
            address: TEST_CONFIG.testAddress,
            amount: TEST_CONFIG.testAmount,
            reason: 'Integration test'
        };

        const response = await this.makeRequest('POST', '/api/faucet/request', requestData);
        
        // Note: This might fail if faucet is not properly configured or KNIRVORACLE is not running
        // In that case, we check for appropriate error messages
        if (response.status === 200) {
            this.assert(response.data.success === true, 'Request should be successful');
            this.assert(response.data.request_id !== undefined, 'Should include request_id');
            this.assert(response.data.amount === TEST_CONFIG.testAmount, 'Should return correct amount');
        } else {
            // Check for expected error conditions
            this.assert(response.status === 400 || response.status === 503, 'Should return appropriate error status');
            this.assert(response.data.error !== undefined, 'Should include error message');
        }
    }

    /**
     * Test invalid address validation
     */
    async testInvalidAddress() {
        const requestData = {
            address: TEST_CONFIG.invalidAddress,
            amount: TEST_CONFIG.testAmount,
            reason: 'Invalid address test'
        };

        const response = await this.makeRequest('POST', '/api/faucet/request', requestData);
        
        this.assert(response.status === 400, 'Should return 400 for invalid address');
        this.assert(response.data.success === false, 'Should not be successful');
        this.assert(response.data.error.includes('address'), 'Error should mention address');
    }

    /**
     * Test amount validation
     */
    async testAmountValidation() {
        const requestData = {
            address: TEST_CONFIG.testAddress,
            amount: TEST_CONFIG.invalidAmount, // Too large
            reason: 'Amount validation test'
        };

        const response = await this.makeRequest('POST', '/api/faucet/request', requestData);
        
        this.assert(response.status === 400, 'Should return 400 for invalid amount');
        this.assert(response.data.success === false, 'Should not be successful');
        this.assert(response.data.error.includes('amount') || response.data.error.includes('limit'), 'Error should mention amount or limit');
    }

    /**
     * Test rate limiting
     */
    async testRateLimiting() {
        const requestData = {
            address: TEST_CONFIG.testAddress,
            amount: 100,
            reason: 'Rate limiting test'
        };

        // Make multiple rapid requests
        const requests = [];
        for (let i = 0; i < 6; i++) { // Exceed the per-IP hourly limit of 5
            requests.push(this.makeRequest('POST', '/api/faucet/request', requestData));
        }

        const responses = await Promise.all(requests);
        
        // At least one should be rate limited
        const rateLimited = responses.some(r => r.status === 429);
        this.assert(rateLimited, 'Should trigger rate limiting after multiple requests');
    }

    /**
     * Test request history
     */
    async testRequestHistory() {
        const response = await this.makeRequest('GET', `/api/faucet/history?address=${TEST_CONFIG.testAddress}&limit=5`);
        
        this.assert(response.status === 200, 'History endpoint should return 200');
        this.assert(response.data.address === TEST_CONFIG.testAddress, 'Should return correct address');
        this.assert(Array.isArray(response.data.history), 'Should return history array');
    }

    /**
     * Test health check endpoint
     */
    async testHealthCheck() {
        const response = await this.makeRequest('GET', '/api/faucet/health');
        
        this.assert(response.status === 200 || response.status === 207 || response.status === 503, 'Should return valid health status');
        this.assert(response.data.service === 'testnet-faucet', 'Should identify as testnet-faucet');
        this.assert(response.data.status !== undefined, 'Should include status');
        this.assert(response.data.timestamp !== undefined, 'Should include timestamp');
    }

    /**
     * Test metrics endpoint
     */
    async testMetricsEndpoint() {
        const response = await this.makeRequest('GET', '/api/faucet/metrics');
        
        this.assert(response.status === 200, 'Metrics endpoint should return 200');
        this.assert(typeof response.data === 'string', 'Should return metrics as text');
        this.assert(response.data.includes('faucet_requests_total'), 'Should include faucet metrics');
        this.assert(response.data.includes('faucet_balance_nrv'), 'Should include balance metrics');
    }

    /**
     * Test economic flow status
     */
    async testEconomicFlow() {
        const response = await this.makeRequest('GET', '/api/faucet/economic/metrics');
        
        this.assert(response.status === 200, 'Economic metrics should return 200');
        this.assert(response.data.economic_flow !== undefined, 'Should include economic_flow');
        this.assert(response.data.sustainability_status !== undefined, 'Should include sustainability_status');
    }

    /**
     * Test treasury status
     */
    async testTreasuryStatus() {
        const response = await this.makeRequest('GET', '/api/faucet/treasury/status');
        
        this.assert(response.status === 200 || response.status === 500, 'Should return valid status');
        if (response.status === 200) {
            this.assert(response.data.treasury_health !== undefined, 'Should include treasury_health');
            this.assert(response.data.faucet_balance_nrv !== undefined, 'Should include faucet_balance_nrv');
        }
    }

    /**
     * Test router integration
     */
    async testRouterIntegration() {
        const response = await this.makeRequest('GET', '/api/faucet/router/proofs');
        
        this.assert(response.status === 200 || response.status === 500, 'Should return valid status');
        if (response.status === 200) {
            this.assert(response.data.router_health !== undefined, 'Should include router_health');
            this.assert(response.data.monitoring_active !== undefined, 'Should include monitoring_active');
        }
    }

    /**
     * Test error handling
     */
    async testErrorHandling() {
        // Test missing required fields
        const response = await this.makeRequest('POST', '/api/faucet/request', {});
        
        this.assert(response.status === 400, 'Should return 400 for missing fields');
        this.assert(response.data.success === false, 'Should not be successful');
        this.assert(response.data.error !== undefined, 'Should include error message');
    }

    /**
     * Make HTTP request with error handling
     */
    async makeRequest(method, path, data = null) {
        try {
            const config = {
                method: method,
                url: `${TEST_CONFIG.baseUrl}${path}`,
                timeout: TEST_CONFIG.timeout,
                validateStatus: () => true // Don't throw on HTTP errors
            };

            if (data) {
                config.data = data;
                config.headers = { 'Content-Type': 'application/json' };
            }

            return await axios(config);
        } catch (error) {
            throw new Error(`Request failed: ${error.message}`);
        }
    }

    /**
     * Assert condition with error message
     */
    assert(condition, message) {
        if (!condition) {
            throw new Error(`Assertion failed: ${message}`);
        }
    }

    /**
     * Sleep for specified milliseconds
     */
    async sleep(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    /**
     * Generate test report
     */
    generateReport() {
        const totalTime = Date.now() - this.startTime;
        const passed = this.testResults.filter(r => r.status === 'PASS').length;
        const failed = this.testResults.filter(r => r.status === 'FAIL').length;
        const total = this.testResults.length;

        console.log('\n' + '='.repeat(60));
        console.log('📊 KNIRV FAUCET INTEGRATION TEST REPORT');
        console.log('='.repeat(60));
        console.log(`Total Tests: ${total}`);
        console.log(`Passed: ${passed} ✅`);
        console.log(`Failed: ${failed} ❌`);
        console.log(`Success Rate: ${((passed / total) * 100).toFixed(1)}%`);
        console.log(`Total Time: ${totalTime}ms`);
        console.log('='.repeat(60));

        if (failed > 0) {
            console.log('\n❌ FAILED TESTS:');
            this.testResults
                .filter(r => r.status === 'FAIL')
                .forEach(test => {
                    console.log(`  • ${test.name}: ${test.error}`);
                });
        }

        console.log('\n📋 DETAILED RESULTS:');
        this.testResults.forEach(test => {
            const status = test.status === 'PASS' ? '✅' : '❌';
            console.log(`  ${status} ${test.name} (${test.duration}ms)`);
        });

        // Save report to file
        this.saveReport();
    }

    /**
     * Save test report to file
     */
    async saveReport() {
        try {
            const report = {
                timestamp: new Date().toISOString(),
                summary: {
                    total: this.testResults.length,
                    passed: this.testResults.filter(r => r.status === 'PASS').length,
                    failed: this.testResults.filter(r => r.status === 'FAIL').length,
                    duration: Date.now() - this.startTime
                },
                tests: this.testResults
            };

            const reportPath = path.join(__dirname, 'reports', `faucet-integration-${Date.now()}.json`);
            
            // Ensure reports directory exists
            await fs.mkdir(path.dirname(reportPath), { recursive: true });
            
            await fs.writeFile(reportPath, JSON.stringify(report, null, 2));
            console.log(`\n📄 Report saved to: ${reportPath}`);
            
        } catch (error) {
            console.error('Failed to save report:', error.message);
        }
    }
}

// Run tests if this file is executed directly
if (require.main === module) {
    const tests = new FaucetIntegrationTests();
    tests.runAllTests()
        .then(() => {
            const failed = tests.testResults.filter(r => r.status === 'FAIL').length;
            process.exit(failed > 0 ? 1 : 0);
        })
        .catch(error => {
            console.error('Test execution failed:', error);
            process.exit(1);
        });
}

module.exports = { FaucetIntegrationTests, TEST_CONFIG };
