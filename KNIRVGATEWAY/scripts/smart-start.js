#!/usr/bin/env node

/**
 * KNIRV Gateway Smart Start
 * Automatically detects and fixes issues before starting the gateway
 */

const { execSync, spawn } = require('child_process');
const HealthChecker = require('./check-health.js');
const NexusHealthChecker = require('./check-nexus-health.js');
const { DHTStarter } = require('./start-dht.js');

class SmartStart {
    constructor() {
        this.maxRetries = 2;
        this.currentRetry = 0;
        this.dhtStarter = null;
        this.dhtEnabled = process.env.KNIRV_DHT_ENABLED !== 'false';
    }

    log(message, type = 'info') {
        const timestamp = new Date().toISOString();
        const prefix = {
            'info': '🚀',
            'warn': '⚠️ ',
            'error': '❌',
            'success': '✅',
            'fix': '🔧'
        }[type] || 'ℹ️ ';
        
        console.log(`${prefix} [SMART-START] ${message}`);
    }

    async runHealthChecks() {
        this.log('Running comprehensive health checks...');
        
        // Run main health check
        const mainChecker = new HealthChecker();
        const mainHealthy = await mainChecker.runHealthCheck();
        
        // Run NEXUS-specific health check
        const nexusChecker = new NexusHealthChecker();
        const nexusHealthy = await nexusChecker.runNexusHealthCheck();
        
        return {
            main: mainHealthy,
            nexus: nexusHealthy,
            overall: mainHealthy && nexusHealthy
        };
    }

    async attemptAutoFix() {
        if (this.currentRetry >= this.maxRetries) {
            this.log('Maximum retry attempts reached. Manual intervention required.', 'error');
            return false;
        }

        this.currentRetry++;
        this.log(`Attempting auto-fix (attempt ${this.currentRetry}/${this.maxRetries})...`, 'fix');
        
        try {
            // Run the fix script in auto mode
            execSync('./scripts/fix-netlify-cli.sh --auto', {
                stdio: 'inherit',
                timeout: 300000 // 5 minutes
            });
            
            this.log('Auto-fix completed successfully', 'success');
            return true;
            
        } catch (error) {
            this.log(`Auto-fix failed: ${error.message}`, 'error');
            return false;
        }
    }

    async initializeDHT() {
        if (!this.dhtEnabled) {
            this.log('DHT service disabled, skipping initialization');
            return true;
        }

        this.log('Initializing DHT service...');

        try {
            this.dhtStarter = new DHTStarter();
            const success = await this.dhtStarter.initialize();

            if (success) {
                this.log('DHT service initialized successfully', 'success');
                return true;
            } else {
                this.log('DHT service initialization failed', 'warn');
                // Don't fail the entire startup if DHT fails
                return true;
            }
        } catch (error) {
            this.log(`DHT initialization error: ${error.message}`, 'warn');
            // Don't fail the entire startup if DHT fails
            return true;
        }
    }

    async startDHT() {
        if (!this.dhtEnabled || !this.dhtStarter) {
            return true;
        }

        this.log('Starting DHT service...');

        try {
            const success = await this.dhtStarter.start();

            if (success) {
                this.log('DHT service started successfully', 'success');
            } else {
                this.log('DHT service failed to start', 'warn');
            }

            return true; // Don't fail startup if DHT fails
        } catch (error) {
            this.log(`DHT start error: ${error.message}`, 'warn');
            return true; // Don't fail startup if DHT fails
        }
    }

    async initializeApplications() {
        this.log('Initializing applications...');

        try {
            // Initialize forum database
            this.log('Initializing forum database...');
            execSync('npm run init-forum', {
                stdio: 'inherit',
                timeout: 60000 // 1 minute
            });

            // Initialize support desk database
            this.log('Initializing support desk database...');
            execSync('npm run init-support-desk', {
                stdio: 'inherit',
                timeout: 60000 // 1 minute
            });

            this.log('Application initialization completed successfully', 'success');
            return true;

        } catch (error) {
            this.log(`Application initialization failed: ${error.message}`, 'error');
            return false;
        }
    }

    async buildProject() {
        this.log('Building project...');

        try {
            execSync('npm run build', {
                stdio: 'inherit',
                timeout: 300000 // 5 minutes
            });

            this.log('Build completed successfully', 'success');
            return true;

        } catch (error) {
            this.log(`Build failed: ${error.message}`, 'error');
            return false;
        }
    }

    async startNetlifyDev() {
        this.log('Starting Netlify development server...');
        
        try {
            // Start netlify dev in a child process
            const netlifyProcess = spawn('npx', ['netlify', 'dev'], {
                stdio: 'inherit',
                detached: false
            });
            
            // Handle process events
            netlifyProcess.on('error', (error) => {
                this.log(`Failed to start netlify dev: ${error.message}`, 'error');
                process.exit(1);
            });
            
            netlifyProcess.on('exit', (code) => {
                if (code !== 0) {
                    this.log(`netlify dev exited with code ${code}`, 'error');
                    process.exit(code);
                }
            });
            
            // Handle graceful shutdown
            process.on('SIGINT', () => {
                this.log('Shutting down...');
                this.shutdown(netlifyProcess);
            });

            process.on('SIGTERM', () => {
                this.log('Shutting down...');
                this.shutdown(netlifyProcess);
            });
            
            return true;
            
        } catch (error) {
            this.log(`Failed to start netlify dev: ${error.message}`, 'error');
            return false;
        }
    }

    async shutdown(netlifyProcess) {
        if (this.dhtStarter) {
            try {
                await this.dhtStarter.shutdown();
            } catch (error) {
                this.log(`Error shutting down DHT: ${error.message}`, 'error');
            }
        }

        if (netlifyProcess) {
            netlifyProcess.kill('SIGINT');
        }

        process.exit(0);
    }

    async smartStart() {
        this.log('🏁 Starting KNIRV Gateway with smart health monitoring...');
        
        while (this.currentRetry <= this.maxRetries) {
            // Run health checks
            const health = await this.runHealthChecks();
            
            if (health.overall) {
                this.log('All health checks passed! Starting gateway...', 'success');
                break;
            }
            
            this.log('Health checks failed. Issues detected.', 'warn');
            
            if (!health.main) {
                this.log('Main gateway issues detected', 'error');
            }
            
            if (!health.nexus) {
                this.log('NEXUS portal issues detected', 'error');
            }
            
            // Attempt auto-fix
            const fixSuccessful = await this.attemptAutoFix();
            
            if (!fixSuccessful) {
                this.log('Auto-fix failed. Cannot start gateway safely.', 'error');
                process.exit(1);
            }
            
            this.log('Auto-fix completed. Re-running health checks...', 'info');
        }

        // Initialize DHT service
        const dhtInitSuccessful = await this.initializeDHT();
        if (!dhtInitSuccessful) {
            this.log('DHT initialization failed. Continuing without DHT...', 'warn');
        }

        // Initialize applications
        const initSuccessful = await this.initializeApplications();
        if (!initSuccessful) {
            this.log('Application initialization failed. Cannot start gateway.', 'error');
            process.exit(1);
        }

        // Build the project
        const buildSuccessful = await this.buildProject();
        if (!buildSuccessful) {
            this.log('Build failed. Cannot start gateway.', 'error');
            process.exit(1);
        }

        // Start DHT service
        const dhtStartSuccessful = await this.startDHT();
        if (!dhtStartSuccessful) {
            this.log('DHT service failed to start. Continuing without DHT...', 'warn');
        }

        // Start the development server
        const startSuccessful = await this.startNetlifyDev();
        if (!startSuccessful) {
            this.log('Failed to start development server.', 'error');
            process.exit(1);
        }
    }
}

// Run smart start if called directly
if (require.main === module) {
    const smartStart = new SmartStart();
    smartStart.smartStart().catch(error => {
        console.error('❌ Smart start failed with error:', error.message);
        process.exit(1);
    });
}

module.exports = SmartStart;
