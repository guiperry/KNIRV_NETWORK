#!/usr/bin/env node

/**
 * KNIRV Network Website Smart Netlify Start
 * Automatically detects and fixes issues before starting Netlify dev
 */

import { execSync, spawn } from 'child_process';
import HealthChecker from './check-netlify-health.js';

class SmartNetlifyStart {
    constructor() {
        this.maxRetries = 2;
        this.currentRetry = 0;
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
        this.log('Running Netlify health checks...');

        // Run Netlify health check
        const netlifyChecker = new HealthChecker();
        const netlifyHealthy = await netlifyChecker.runHealthCheck();

        return {
            netlify: netlifyHealthy,
            overall: netlifyHealthy
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

    async prepareNetlifyEnvironment() {
        this.log('Preparing Netlify development environment...');

        try {
            // Ensure netlify functions are built
            this.log('Building Netlify functions...');
            execSync('npm run build:netlify-functions', {
                stdio: 'inherit',
                timeout: 120000 // 2 minutes
            });

            this.log('Netlify environment prepared successfully', 'success');
            return true;
        } catch (error) {
            this.log(`Netlify environment preparation failed: ${error.message}`, 'error');
            return false;
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
        if (netlifyProcess) {
            netlifyProcess.kill('SIGINT');
        }

        process.exit(0);
    }

    async smartStart() {
        this.log('🏁 Starting KNIRV Network Website with smart Netlify monitoring...');

        while (this.currentRetry <= this.maxRetries) {
            // Run health checks
            const health = await this.runHealthChecks();

            if (health.overall) {
                this.log('All health checks passed! Starting Netlify dev...', 'success');
                break;
            }

            this.log('Health checks failed. Issues detected.', 'warn');

            if (!health.netlify) {
                this.log('Netlify issues detected', 'error');
            }

            // Attempt auto-fix
            const fixSuccessful = await this.attemptAutoFix();

            if (!fixSuccessful) {
                this.log('Auto-fix failed. Cannot start Netlify dev safely.', 'error');
                process.exit(1);
            }

            this.log('Auto-fix completed. Re-running health checks...', 'info');
        }

        if (this.currentRetry > this.maxRetries) {
            this.log('Maximum retry attempts exceeded. Cannot start Netlify dev.', 'error');
            process.exit(1);
        }

        // Prepare Netlify environment
        const prepareSuccessful = await this.prepareNetlifyEnvironment();
        if (!prepareSuccessful) {
            this.log('Netlify environment preparation failed. Cannot start dev server.', 'error');
            process.exit(1);
        }

        // Start the Netlify development server
        const startSuccessful = await this.startNetlifyDev();
        if (!startSuccessful) {
            this.log('Failed to start Netlify development server.', 'error');
            process.exit(1);
        }

        this.log('🎉 KNIRV Network Website started successfully with Netlify dev!', 'success');
    }
}

// CLI execution
if (import.meta.url === `file://${process.argv[1]}`) {
    const smartStart = new SmartNetlifyStart();
    smartStart.smartStart().catch(error => {
        console.error('❌ Smart Netlify start failed with error:', error.message);
        process.exit(1);
    });
}

export default SmartNetlifyStart;
