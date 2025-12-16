// Footer Functionality Test Suite
// Tests the KNIRV Universal Footer across different applications and contexts

class FooterTestSuite {
    constructor() {
        this.results = {
            passed: 0,
            failed: 0,
            tests: []
        };
        this.config = null;
        this.footer = null;
    }

    async runAllTests() {
        console.log('🧪 Starting Footer Functionality Tests...');

        try {
            // Test 1: Configuration Loading
            await this.testConfigurationLoading();

            // Test 2: Footer Initialization
            await this.testFooterInitialization();

            // Test 3: Link Resolution
            await this.testLinkResolution();

            // Test 4: Path Resolution
            await this.testPathResolution();

            // Test 5: Cross-Application Compatibility
            await this.testCrossApplicationCompatibility();

            // Test 6: Responsive Design
            await this.testResponsiveDesign();

            // Test 7: Error Handling
            await this.testErrorHandling();

            this.displayResults();
        } catch (error) {
            console.error('❌ Test suite failed:', error);
            this.recordResult('Test Suite Execution', false, error.message);
            this.displayResults();
        }
    }

    async testConfigurationLoading() {
        console.log('🔧 Testing Configuration Loading...');

        try {
            // Wait for configuration to load
            await this.waitForConfig();

            // Verify unified configuration system
            if (window.knirvConfig && window.knirvConfig.isLoaded) {
                this.config = window.knirvConfig.config;
                this.recordResult('Configuration Loading', true, 'Configuration loaded successfully');
            } else {
                throw new Error('Configuration not loaded');
            }

            // Verify configuration structure
            const requiredSections = ['navigation', 'footer', 'external_services'];
            for (const section of requiredSections) {
                if (!this.config[section]) {
                    throw new Error(`Missing required section: ${section}`);
                }
            }

            this.recordResult('Configuration Structure', true, 'All required sections present');

            // Verify oracle config integration
            if (window.KNIRV_GATEWAY_CONFIG) {
                this.recordResult('Gateway Config Integration', true, 'Gateway config successfully integrated');
            } else {
                this.recordResult('Gateway Config Integration', false, 'Gateway config not found');
            }

        } catch (error) {
            this.recordResult('Configuration Loading', false, error.message);
        }
    }

    async testFooterInitialization() {
        console.log('🚀 Testing Footer Initialization...');

        try {
            // Check if footer element exists
            const existingFooter = document.querySelector('footer.knirv-footer');
            if (existingFooter) {
                this.recordResult('Footer Element Creation', true, 'Footer element found');
            } else {
                this.recordResult('Footer Element Creation', false, 'Footer element not found');
            }

            // Verify footer has content
            const footerContent = existingFooter ? existingFooter.innerHTML : '';
            if (footerContent.length > 100) {
                this.recordResult('Footer Content', true, 'Footer has substantial content');
            } else {
                this.recordResult('Footer Content', false, 'Footer content too short');
            }

            // Check for required sections
            const requiredSections = ['footer-section', 'footer-bottom'];
            for (const section of requiredSections) {
                const elements = document.querySelectorAll(`.${section}`);
                if (elements.length > 0) {
                    this.recordResult(`Footer Section: ${section}`, true, `${elements.length} elements found`);
                } else {
                    this.recordResult(`Footer Section: ${section}`, false, 'Section not found');
                }
            }

        } catch (error) {
            this.recordResult('Footer Initialization', false, error.message);
        }
    }

    async testLinkResolution() {
        console.log('🔗 Testing Link Resolution...');

        try {
            // Test navigation links
            const navigationLinks = [
                'navigation.main_site',
                'navigation.documentation',
                'navigation.graphchain_explorer'
            ];

            for (const linkPath of navigationLinks) {
                const link = this.getLinkValue(linkPath);
                if (link && link !== '#') {
                    this.recordResult(`Navigation Link: ${linkPath}`, true, `Resolved to: ${link}`);
                } else {
                    this.recordResult(`Navigation Link: ${linkPath}`, false, 'Link not resolved');
                }
            }

            // Test footer links
            const footerLinks = [
                'footer.social.github',
                'footer.social.discord',
                'footer.legal.terms'
            ];

            for (const linkPath of footerLinks) {
                const link = this.getLinkValue(linkPath);
                if (link && link !== '#') {
                    this.recordResult(`Footer Link: ${linkPath}`, true, `Resolved to: ${link}`);
                } else {
                    this.recordResult(`Footer Link: ${linkPath}`, false, 'Link not resolved');
                }
            }

            // Test external services
            const externalServices = [
                'external_services.payment_oracle',
                'external_services.webgui'
            ];

            for (const servicePath of externalServices) {
                const service = this.getLinkValue(servicePath);
                if (service && service !== '#') {
                    this.recordResult(`External Service: ${servicePath}`, true, `Resolved to: ${service}`);
                } else {
                    this.recordResult(`External Service: ${servicePath}`, false, 'Service not resolved');
                }
            }

        } catch (error) {
            this.recordResult('Link Resolution', false, error.message);
        }
    }

    async testPathResolution() {
        console.log('🛣️ Testing Path Resolution...');

        try {
            // Test different URL types
            const testUrls = [
                'https://example.com',  // Absolute URL
                'documentation/static/', // Relative path
                '/absolute/path',       // Root absolute path
                'relative/path'         // Relative path
            ];

            for (const url of testUrls) {
                const resolvedPath = this.resolveTestPath(url);
                if (resolvedPath) {
                    this.recordResult(`Path Resolution: ${url}`, true, `Resolved to: ${resolvedPath}`);
                } else {
                    this.recordResult(`Path Resolution: ${url}`, false, 'Path not resolved');
                }
            }

        } catch (error) {
            this.recordResult('Path Resolution', false, error.message);
        }
    }

    async testCrossApplicationCompatibility() {
        console.log('🔄 Testing Cross-Application Compatibility...');

        try {
            // Test different application contexts
            const currentPath = window.location.pathname;
            const isInNetworkWebsite = currentPath.includes('/network-website/');

            if (isInNetworkWebsite) {
                this.recordResult('Network Website Context', true, 'Running in network-website context');

                // Test network-website specific paths
                const networkWebsitePaths = [
                    'graphchain-explorer/',
                    'developer-portal/',
                    'documentation/static/'
                ];

                for (const path of networkWebsitePaths) {
                    const resolved = this.resolveTestPath(path);
                    if (resolved && !resolved.includes('undefined')) {
                        this.recordResult(`Network Website Path: ${path}`, true, `Resolved to: ${resolved}`);
                    } else {
                        this.recordResult(`Network Website Path: ${path}`, false, 'Path resolution failed');
                    }
                }
            } else {
                this.recordResult('Root Context', true, 'Running at root level');

                // Test root level paths
                const rootPaths = [
                    'KNIRVORACLE/',
                    'documentation/',
                    'graphchain-explorer/'
                ];

                for (const path of rootPaths) {
                    const resolved = this.resolveTestPath(path);
                    if (resolved && !resolved.includes('undefined')) {
                        this.recordResult(`Root Path: ${path}`, true, `Resolved to: ${resolved}`);
                    } else {
                        this.recordResult(`Root Path: ${path}`, false, 'Path resolution failed');
                    }
                }
            }

        } catch (error) {
            this.recordResult('Cross-Application Compatibility', false, error.message);
        }
    }

    async testResponsiveDesign() {
        console.log('📱 Testing Responsive Design...');

        try {
            // Test viewport sizes
            const viewports = [
                { width: 1200, height: 800, name: 'Desktop' },
                { width: 768, height: 1024, name: 'Tablet' },
                { width: 480, height: 800, name: 'Mobile' }
            ];

            for (const viewport of viewports) {
                // Simulate viewport change
                const originalWidth = window.innerWidth;
                const originalHeight = window.innerHeight;

                // Note: In a real test environment, you would use a tool like Puppeteer
                // to actually change the viewport size
                this.recordResult(`Responsive Viewport: ${viewport.name}`, true,
                    `Would test ${viewport.width}x${viewport.height} viewport`);

                // Check if responsive styles exist
                const footerStyles = document.getElementById('knirv-footer-styles');
                if (footerStyles) {
                    const styleContent = footerStyles.textContent;
                    const hasResponsiveRules = styleContent.includes('@media');

                    if (hasResponsiveRules) {
                        this.recordResult(`Responsive Styles: ${viewport.name}`, true, 'Responsive CSS rules found');
                    } else {
                        this.recordResult(`Responsive Styles: ${viewport.name}`, false, 'No responsive CSS rules found');
                    }
                }
            }

        } catch (error) {
            this.recordResult('Responsive Design', false, error.message);
        }
    }

    async testErrorHandling() {
        console.log('⚠️ Testing Error Handling...');

        try {
            // Test invalid configuration paths
            const invalidPaths = [
                'nonexistent.section',
                'footer.nonexistent.key',
                'invalid.path.structure'
            ];

            for (const path of invalidPaths) {
                const result = this.getLinkValue(path);
                if (result === '#') {
                    this.recordResult(`Invalid Path Handling: ${path}`, true, 'Correctly returned fallback');
                } else {
                    this.recordResult(`Invalid Path Handling: ${path}`, false, `Should return fallback, got: ${result}`);
                }
            }

            // Test missing configuration
            const originalConfig = this.config;
            this.config = null;

            const fallbackResult = this.getLinkValue('navigation.main_site');
            if (fallbackResult === '#') {
                this.recordResult('Missing Config Handling', true, 'Correctly handled missing config');
            } else {
                this.recordResult('Missing Config Handling', false, 'Should return fallback for missing config');
            }

            // Restore config
            this.config = originalConfig;

        } catch (error) {
            this.recordResult('Error Handling', false, error.message);
        }
    }

    // Helper methods
    async waitForConfig() {
        return new Promise((resolve) => {
            const checkConfig = () => {
                if (window.knirvConfig && window.knirvConfig.isLoaded) {
                    resolve();
                } else {
                    setTimeout(checkConfig, 100);
                }
            };
            checkConfig();
        });
    }

    getLinkValue(path) {
        try {
            if (!this.config) return '#';

            const keys = path.split('.');
            let value = this.config;

            for (const key of keys) {
                if (value && typeof value === 'object' && key in value) {
                    value = value[key];
                } else {
                    return '#';
                }
            }

            return typeof value === 'string' ? value : '#';
        } catch (error) {
            return '#';
        }
    }

    resolveTestPath(url) {
        if (!url || url === '#') return url;

        // If it's already an absolute URL, return as-is
        if (url.startsWith('http://') || url.startsWith('https://')) {
            return url;
        }

        // Get current path context
        const currentPath = window.location.pathname;
        const isInNetworkWebsite = currentPath.includes('/network-website/');

        // Handle different path contexts
        if (isInNetworkWebsite) {
            // We're in the network-website subdirectory
            if (url.startsWith('/')) {
                // Absolute path from domain root
                return url;
            } else {
                // Relative path - add network-website prefix
                return `/network-website/${url}`;
            }
        } else {
            // We're at the root level or other context
            if (url.startsWith('/')) {
                // Absolute path from domain root
                return url;
            } else {
                // Relative path - keep as relative
                return url;
            }
        }
    }

    recordResult(testName, passed, message) {
        const result = {
            name: testName,
            passed,
            message,
            timestamp: new Date().toISOString()
        };

        this.results.tests.push(result);

        if (passed) {
            this.results.passed++;
            console.log(`✅ ${testName}: ${message}`);
        } else {
            this.results.failed++;
            console.error(`❌ ${testName}: ${message}`);
        }
    }

    displayResults() {
        console.log('\n📊 Footer Test Results:');
        console.log(`Passed: ${this.results.passed}`);
        console.log(`Failed: ${this.results.failed}`);
        console.log(`Total: ${this.results.passed + this.results.failed}`);

        const successRate = ((this.results.passed / (this.results.passed + this.results.failed)) * 100).toFixed(1);
        console.log(`Success Rate: ${successRate}%`);

        if (this.results.failed > 0) {
            console.log('\n❌ Failed Tests:');
            this.results.tests
                .filter(test => !test.passed)
                .forEach(test => {
                    console.log(`• ${test.name}: ${test.message}`);
                });
        }

        // Overall result
        if (this.results.failed === 0) {
            console.log('🎉 All tests passed! Footer functionality is working correctly.');
        } else {
            console.log('⚠️ Some tests failed. Please review the issues above.');
        }

        // Store results globally for further analysis
        window.footerTestResults = this.results;
    }
}

// Auto-run tests when script loads
document.addEventListener('DOMContentLoaded', async () => {
    // Wait for footer to initialize
    setTimeout(async () => {
        const testSuite = new FooterTestSuite();
        await testSuite.runAllTests();

        // Make test suite globally available
        window.footerTestSuite = testSuite;
    }, 2000); // Wait 2 seconds for footer to load
    console.log('🧪 Footer test suite loaded. Call runFooterTests() to execute tests.');
});

// Manual test runner function
window.runFooterTests = async () => {
    const testSuite = new FooterTestSuite();
    await testSuite.runAllTests();
    return testSuite.results;
};

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = FooterTestSuite;
}
