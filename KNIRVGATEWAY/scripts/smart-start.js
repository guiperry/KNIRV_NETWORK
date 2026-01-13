#!/usr/bin/env node

/**
 * KNIRV Gateway Smart Start
 * Starts the main oracle server with service management
 */

import { spawn } from 'child_process';
import GatewayHealthChecker from './check-health.js';

class GatewaySmartStart {
    constructor() {
        this.maxRetries = 3;
        this.currentRetry = 0;
        this.oracleProcess = null;
    }

    log(message, type = 'info') {
        const timestamp = new Date().toISOString();
        const prefix = {
            'info': 'ℹ️ ',
            'success': '✅',
            'warn': '⚠️ ',
            'error': '❌',
            'fix': '🔧'
        }[type] || 'ℹ️ ';
        
        console.log(`${prefix} [${timestamp}] ${message}`);
    }

    async runHealthCheck() {
        this.log('Running health check...');
        
        const checker = new GatewayHealthChecker();
        const healthResult = await checker.runHealthCheck();
        
        if (!healthResult) {
            this.log('Health check failed', 'error');
            return false;
        }
        
        this.log('Health check passed', 'success');
        return true;
    }

    async attemptAutoFix() {
        if (this.currentRetry >= this.maxRetries) {
            this.log('Maximum retry attempts reached', 'error');
            return false;
        }

        this.currentRetry++;
        this.log(`Attempting auto-fix (attempt ${this.currentRetry}/${this.maxRetries})...`, 'fix');
        
        try {
            // Install dependencies
            this.log('Installing main dependencies...', 'fix');
            const { execSync } = await import('child_process');
            execSync('npm install', {
                stdio: 'inherit',
                timeout: 300000 // 5 minutes
            });
            
            this.log('Installing service dependencies...', 'fix');
            execSync('npm run services:install', {
                stdio: 'inherit',
                timeout: 300000 // 5 minutes
            });

            this.log('Installing website dependencies...', 'fix');
            execSync('npm run websites:install', {
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

    async startGatewayServer() {
        this.log('Starting KNIRV Gateway server...');
        
        try {
            // Start the main server
            this.oracleProcess = spawn('node', ['server.js'], {
                stdio: 'inherit',
                detached: false,
                env: {
                    ...process.env,
                    NODE_ENV: process.env.NODE_ENV || 'development',
                    PORT: process.env.PORT || '8080'
                }
            });
            
            // Handle process events
            this.oracleProcess.on('error', (error) => {
                this.log(`Failed to start oracle server: ${error.message}`, 'error');
                process.exit(1);
            });
            
            this.oracleProcess.on('exit', (code) => {
                if (code !== 0) {
                    this.log(`Gateway server exited with code ${code}`, 'error');
                    process.exit(code);
                }
            });
            
            // Handle graceful shutdown
            process.on('SIGINT', () => {
                this.log('Shutting down...');
                this.shutdown();
            });

            process.on('SIGTERM', () => {
                this.log('Shutting down...');
                this.shutdown();
            });
            
            this.log('Gateway server started successfully', 'success');
            this.log('Server endpoints:', 'info');
            this.log('  - Main Gateway: http://localhost:8080', 'info');
            this.log('  - Health Check: http://localhost:8080/health', 'info');
            this.log('  - Services Status: http://localhost:8080/services/status', 'info');
            
            return true;
            
        } catch (error) {
            this.log(`Failed to start oracle server: ${error.message}`, 'error');
            return false;
        }
    }

    shutdown() {
        if (this.oracleProcess) {
            this.log('Stopping oracle server...', 'info');
            this.oracleProcess.kill('SIGINT');
        }
        process.exit(0);
    }

    async smartStart() {
        this.log('🏁 Starting KNIRV Gateway with smart health monitoring...');
        
        while (this.currentRetry <= this.maxRetries) {
            // Run health check
            const healthPassed = await this.runHealthCheck();
            
            if (healthPassed) {
                this.log('All health checks passed! Starting oracle server...', 'success');
                break;
            }
            
            this.log('Health checks failed. Attempting auto-fix...', 'warn');
            
            // Attempt auto-fix
            const fixSuccessful = await this.attemptAutoFix();
            
            if (!fixSuccessful) {
                this.log('Auto-fix failed. Cannot start oracle safely.', 'error');
                process.exit(1);
            }
            
            this.log('Auto-fix completed. Re-running health checks...', 'info');
        }

        if (this.currentRetry > this.maxRetries) {
            this.log('Maximum retry attempts exceeded. Cannot start oracle.', 'error');
            process.exit(1);
        }

        // Start the oracle server
        const startSuccessful = await this.startGatewayServer();
        if (!startSuccessful) {
            this.log('Failed to start oracle server.', 'error');
            process.exit(1);
        }

        this.log('🎉 KNIRV Gateway started successfully!', 'success');
    }
}

// CLI execution
if (import.meta.url === `file://${process.argv[1]}`) {
    const starter = new GatewaySmartStart();
    starter.smartStart().catch(error => {
        console.error('Smart start failed:', error);
        process.exit(1);
    });
}

export default GatewaySmartStart;
