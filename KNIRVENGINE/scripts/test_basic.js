#!/usr/bin/env node

/**
 * Basic Agentic Wallet Test
 * Tests the core wallet functionality and API endpoints
 */

const http = require('http');
const { spawn } = require('child_process');
const path = require('path');

class BasicWalletTest {
    constructor() {
        this.walletServer = null;
        this.testResults = [];
    }

    async runTests() {
        console.log('🧪 Starting Basic Agentic Wallet Tests...\n');

        try {
            // Test 1: Server Startup
            await this.testServerStartup();
            
            // Test 2: Health Check
            await this.testHealthCheck();
            
            // Test 3: Supported Chains
            await this.testSupportedChains();
            
            // Test 4: Mnemonic Generation
            await this.testMnemonicGeneration();
            
            this.printTestResults();
            
        } catch (error) {
            console.error('❌ Test suite failed:', error);
        } finally {
            await this.cleanup();
        }
    }

    async testServerStartup() {
        console.log('🚀 Test 1: Server Startup');
        
        try {
            // Start wallet server
            this.walletServer = spawn('go', ['run', 'cmd/server/main.go'], {
                cwd: path.join(__dirname, '../agentic-wallet/go-backend'),
                stdio: ['pipe', 'pipe', 'pipe']
            });

            // Wait for startup
            await this.waitForStartup();
            
            this.addTestResult('Server Startup', true, 'Wallet server started successfully');
            console.log('✅ Wallet server is running');
            
        } catch (error) {
            this.addTestResult('Server Startup', false, error.message);
            console.log('❌ Wallet server startup failed:', error.message);
        }
        
        console.log('');
    }

    async testHealthCheck() {
        console.log('❤️  Test 2: Health Check');
        
        try {
            const health = await this.makeRequest('GET', 'http://localhost:8082/health');
            
            if (health.status === 'ok') {
                this.addTestResult('Health Check', true, 'Health check passed');
                console.log('✅ Health check successful');
            } else {
                throw new Error('Health check failed');
            }
            
        } catch (error) {
            this.addTestResult('Health Check', false, error.message);
            console.log('❌ Health check failed:', error.message);
        }
        
        console.log('');
    }

    async testSupportedChains() {
        console.log('🔗 Test 3: Supported Chains');
        
        try {
            const chains = await this.makeRequest('GET', 'http://localhost:8082/api/v1/multichain/chains');
            
            if (Array.isArray(chains)) {
                this.addTestResult('Supported Chains', true, `Found ${chains.length} supported chains`);
                console.log('✅ Supported chains retrieved');
                chains.forEach(chain => {
                    console.log(`   - ${chain.name || chain.chain_id || 'Unknown chain'}`);
                });
            } else {
                throw new Error('Invalid chains response');
            }
            
        } catch (error) {
            this.addTestResult('Supported Chains', false, error.message);
            console.log('❌ Supported chains test failed:', error.message);
        }
        
        console.log('');
    }

    async testMnemonicGeneration() {
        console.log('🔐 Test 4: Mnemonic Generation');
        
        try {
            const response = await this.makeRequest('POST', 'http://localhost:8082/api/v1/multichain/mnemonic/generate', {
                strength: 128 // 12 words
            });
            
            if (response.mnemonic && response.mnemonic.split(' ').length === 12) {
                this.addTestResult('Mnemonic Generation', true, 'Mnemonic generated successfully');
                console.log('✅ Mnemonic generation successful');
                console.log(`   Mnemonic: ${response.mnemonic}`);
            } else {
                throw new Error('Invalid mnemonic response');
            }
            
        } catch (error) {
            this.addTestResult('Mnemonic Generation', false, error.message);
            console.log('❌ Mnemonic generation failed:', error.message);
        }
        
        console.log('');
    }

    async waitForStartup() {
        return new Promise((resolve, reject) => {
            let attempts = 0;
            const maxAttempts = 30;
            
            const checkHealth = async () => {
                try {
                    await this.makeRequest('GET', 'http://localhost:8082/health');
                    resolve();
                } catch (error) {
                    attempts++;
                    if (attempts >= maxAttempts) {
                        reject(new Error('Wallet server failed to start within timeout'));
                    } else {
                        setTimeout(checkHealth, 1000);
                    }
                }
            };
            
            setTimeout(checkHealth, 2000); // Initial delay
        });
    }

    async makeRequest(method, url, data = null) {
        return new Promise((resolve, reject) => {
            const urlObj = new URL(url);
            const options = {
                hostname: urlObj.hostname,
                port: urlObj.port,
                path: urlObj.pathname,
                method: method,
                headers: {
                    'Content-Type': 'application/json',
                }
            };

            const req = http.request(options, (res) => {
                let body = '';
                res.on('data', (chunk) => body += chunk);
                res.on('end', () => {
                    try {
                        const response = JSON.parse(body);
                        resolve(response);
                    } catch (error) {
                        reject(new Error(`Invalid JSON response: ${body}`));
                    }
                });
            });

            req.on('error', reject);

            if (data) {
                req.write(JSON.stringify(data));
            }
            
            req.end();
        });
    }

    addTestResult(testName, passed, message) {
        this.testResults.push({ testName, passed, message });
    }

    printTestResults() {
        console.log('📊 Basic Wallet Test Results');
        console.log('============================');
        
        let passed = 0;
        let total = this.testResults.length;
        
        this.testResults.forEach(result => {
            const status = result.passed ? '✅ PASS' : '❌ FAIL';
            console.log(`${status} ${result.testName}: ${result.message}`);
            if (result.passed) passed++;
        });
        
        console.log('');
        console.log(`Total: ${passed}/${total} tests passed`);
        
        if (passed === total) {
            console.log('🎉 All tests passed! Agentic Wallet is operational.');
            console.log('');
            console.log('✨ Features Verified:');
            console.log('   • Server startup and health checks');
            console.log('   • Multichain wallet support');
            console.log('   • Mnemonic generation');
        } else {
            console.log('⚠️  Some tests failed. Please check the implementation.');
        }
    }

    async cleanup() {
        console.log('\n🧹 Cleaning up...');
        
        if (this.walletServer) {
            this.walletServer.kill();
            console.log('   Wallet server stopped');
        }
    }
}

// Run tests if this script is executed directly
if (require.main === module) {
    const test = new BasicWalletTest();
    test.runTests().catch(console.error);
}

module.exports = BasicWalletTest;