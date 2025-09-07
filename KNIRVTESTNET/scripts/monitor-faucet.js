#!/usr/bin/env node

/**
 * KNIRV Faucet Monitoring Script
 * 
 * Continuously monitors the faucet service health, economic flow,
 * and generates alerts for critical issues.
 */

const axios = require('axios');
const fs = require('fs').promises;
const path = require('path');

class FaucetMonitor {
    constructor() {
        this.config = {
            faucetUrl: process.env.FAUCET_URL || 'http://localhost:10000',
            checkInterval: parseInt(process.env.CHECK_INTERVAL) || 30000, // 30 seconds
            alertThresholds: {
                balanceWarning: 10000, // NRV
                balanceCritical: 1000,  // NRV
                responseTimeWarning: 5000, // ms
                responseTimeCritical: 10000, // ms
                successRateWarning: 80, // %
                successRateCritical: 50 // %
            },
            logFile: path.join(__dirname, '../logs/faucet-monitor.log'),
            alertFile: path.join(__dirname, '../logs/faucet-alerts.log')
        };
        
        this.isRunning = false;
        this.lastStatus = null;
        this.alertHistory = [];
        this.metrics = {
            totalChecks: 0,
            healthyChecks: 0,
            unhealthyChecks: 0,
            averageResponseTime: 0,
            lastAlert: null
        };
    }

    /**
     * Start monitoring
     */
    async start() {
        console.log('🔍 Starting KNIRV Faucet Monitor...');
        console.log(`Check interval: ${this.config.checkInterval}ms`);
        console.log(`Faucet URL: ${this.config.faucetUrl}`);
        
        this.isRunning = true;
        
        // Ensure log directories exist
        await this.ensureLogDirectories();
        
        // Start monitoring loop
        this.monitorLoop();
        
        // Handle graceful shutdown
        process.on('SIGINT', () => this.stop());
        process.on('SIGTERM', () => this.stop());
        
        console.log('✅ Monitor started. Press Ctrl+C to stop.\n');
    }

    /**
     * Stop monitoring
     */
    async stop() {
        console.log('\n🛑 Stopping monitor...');
        this.isRunning = false;
        
        // Save final metrics
        await this.saveMetrics();
        
        console.log('✅ Monitor stopped.');
        process.exit(0);
    }

    /**
     * Main monitoring loop
     */
    async monitorLoop() {
        while (this.isRunning) {
            try {
                await this.performHealthCheck();
                await this.sleep(this.config.checkInterval);
            } catch (error) {
                console.error('Monitor loop error:', error.message);
                await this.sleep(5000); // Shorter retry interval on error
            }
        }
    }

    /**
     * Perform comprehensive health check
     */
    async performHealthCheck() {
        const checkStart = Date.now();
        const timestamp = new Date().toISOString();
        
        try {
            // Check faucet health
            const healthResponse = await this.makeRequest('/api/faucet/health');
            const statusResponse = await this.makeRequest('/api/faucet/status');
            const economicResponse = await this.makeRequest('/api/faucet/economic/metrics');
            
            const responseTime = Date.now() - checkStart;
            
            const status = {
                timestamp,
                responseTime,
                health: healthResponse.data,
                status: statusResponse.data,
                economic: economicResponse.data,
                alerts: []
            };
            
            // Update metrics
            this.updateMetrics(status, responseTime);
            
            // Check for alerts
            await this.checkAlerts(status);
            
            // Log status
            await this.logStatus(status);
            
            // Display status
            this.displayStatus(status);
            
            this.lastStatus = status;
            
        } catch (error) {
            const responseTime = Date.now() - checkStart;
            
            const errorStatus = {
                timestamp,
                responseTime,
                error: error.message,
                alerts: ['CRITICAL: Faucet service unreachable']
            };
            
            this.metrics.totalChecks++;
            this.metrics.unhealthyChecks++;
            
            await this.logError(errorStatus);
            await this.sendAlert('CRITICAL', 'Faucet service unreachable', error.message);
            
            console.log(`❌ ${timestamp} - Service unreachable: ${error.message}`);
        }
    }

    /**
     * Check for alert conditions
     */
    async checkAlerts(status) {
        const alerts = [];
        
        // Balance alerts
        if (status.status.current_balance !== undefined) {
            if (status.status.current_balance <= this.config.alertThresholds.balanceCritical) {
                alerts.push('CRITICAL: Faucet balance critically low');
                await this.sendAlert('CRITICAL', 'Low Balance', `Balance: ${status.status.current_balance} NRV`);
            } else if (status.status.current_balance <= this.config.alertThresholds.balanceWarning) {
                alerts.push('WARNING: Faucet balance low');
                await this.sendAlert('WARNING', 'Low Balance', `Balance: ${status.status.current_balance} NRV`);
            }
        }
        
        // Response time alerts
        if (status.responseTime >= this.config.alertThresholds.responseTimeCritical) {
            alerts.push('CRITICAL: High response time');
            await this.sendAlert('CRITICAL', 'High Response Time', `${status.responseTime}ms`);
        } else if (status.responseTime >= this.config.alertThresholds.responseTimeWarning) {
            alerts.push('WARNING: Elevated response time');
            await this.sendAlert('WARNING', 'Elevated Response Time', `${status.responseTime}ms`);
        }
        
        // Success rate alerts
        if (status.status.success_rate_today !== undefined) {
            if (status.status.success_rate_today <= this.config.alertThresholds.successRateCritical) {
                alerts.push('CRITICAL: Low success rate');
                await this.sendAlert('CRITICAL', 'Low Success Rate', `${status.status.success_rate_today}%`);
            } else if (status.status.success_rate_today <= this.config.alertThresholds.successRateWarning) {
                alerts.push('WARNING: Reduced success rate');
                await this.sendAlert('WARNING', 'Reduced Success Rate', `${status.status.success_rate_today}%`);
            }
        }
        
        // Economic flow alerts
        if (status.economic && status.economic.economic_flow) {
            const flow = status.economic.economic_flow;
            
            if (flow.router_health !== 1) {
                alerts.push('WARNING: Router health degraded');
                await this.sendAlert('WARNING', 'Router Health', 'Router connectivity issues detected');
            }
            
            if (flow.treasury_health !== 1) {
                alerts.push('WARNING: Treasury health degraded');
                await this.sendAlert('WARNING', 'Treasury Health', 'Treasury connectivity issues detected');
            }
            
            if (flow.funding_sustainability_days < 7) {
                alerts.push('CRITICAL: Low funding sustainability');
                await this.sendAlert('CRITICAL', 'Funding Sustainability', `${flow.funding_sustainability_days} days remaining`);
            }
        }
        
        status.alerts = alerts;
    }

    /**
     * Send alert
     */
    async sendAlert(level, type, message) {
        const alert = {
            timestamp: new Date().toISOString(),
            level,
            type,
            message
        };
        
        // Prevent duplicate alerts within 5 minutes
        const recentAlert = this.alertHistory.find(a => 
            a.type === type && 
            a.level === level && 
            Date.now() - new Date(a.timestamp).getTime() < 300000
        );
        
        if (recentAlert) {
            return;
        }
        
        this.alertHistory.push(alert);
        this.metrics.lastAlert = alert;
        
        // Log alert
        await this.logAlert(alert);
        
        console.log(`🚨 ${level}: ${type} - ${message}`);
    }

    /**
     * Update monitoring metrics
     */
    updateMetrics(status, responseTime) {
        this.metrics.totalChecks++;
        
        if (status.health && status.health.status === 'healthy') {
            this.metrics.healthyChecks++;
        } else {
            this.metrics.unhealthyChecks++;
        }
        
        // Update average response time
        this.metrics.averageResponseTime = 
            (this.metrics.averageResponseTime * (this.metrics.totalChecks - 1) + responseTime) / 
            this.metrics.totalChecks;
    }

    /**
     * Display current status
     */
    displayStatus(status) {
        const healthIcon = status.health?.status === 'healthy' ? '✅' : '❌';
        const balanceIcon = status.status?.current_balance > 5000 ? '💰' : '⚠️';
        
        console.log(`${healthIcon} ${status.timestamp} - Health: ${status.health?.status || 'unknown'} | ` +
                   `Balance: ${balanceIcon} ${status.status?.current_balance || 0} NRV | ` +
                   `Response: ${status.responseTime}ms | ` +
                   `Success Rate: ${status.status?.success_rate_today || 0}%`);
        
        if (status.alerts.length > 0) {
            status.alerts.forEach(alert => console.log(`  🚨 ${alert}`));
        }
    }

    /**
     * Make HTTP request
     */
    async makeRequest(path) {
        return await axios.get(`${this.config.faucetUrl}${path}`, {
            timeout: 10000,
            validateStatus: (status) => status < 500
        });
    }

    /**
     * Ensure log directories exist
     */
    async ensureLogDirectories() {
        const logDir = path.dirname(this.config.logFile);
        await fs.mkdir(logDir, { recursive: true });
    }

    /**
     * Log status to file
     */
    async logStatus(status) {
        try {
            const logEntry = `${status.timestamp} | HEALTH: ${status.health?.status} | ` +
                           `BALANCE: ${status.status?.current_balance} | ` +
                           `RESPONSE: ${status.responseTime}ms | ` +
                           `SUCCESS_RATE: ${status.status?.success_rate_today}%\n`;
            
            await fs.appendFile(this.config.logFile, logEntry);
        } catch (error) {
            console.error('Failed to log status:', error.message);
        }
    }

    /**
     * Log error to file
     */
    async logError(errorStatus) {
        try {
            const logEntry = `${errorStatus.timestamp} | ERROR: ${errorStatus.error} | ` +
                           `RESPONSE: ${errorStatus.responseTime}ms\n`;
            
            await fs.appendFile(this.config.logFile, logEntry);
        } catch (error) {
            console.error('Failed to log error:', error.message);
        }
    }

    /**
     * Log alert to file
     */
    async logAlert(alert) {
        try {
            const logEntry = `${alert.timestamp} | ${alert.level} | ${alert.type} | ${alert.message}\n`;
            await fs.appendFile(this.config.alertFile, logEntry);
        } catch (error) {
            console.error('Failed to log alert:', error.message);
        }
    }

    /**
     * Save metrics to file
     */
    async saveMetrics() {
        try {
            const metricsFile = path.join(__dirname, '../logs/faucet-metrics.json');
            await fs.writeFile(metricsFile, JSON.stringify(this.metrics, null, 2));
        } catch (error) {
            console.error('Failed to save metrics:', error.message);
        }
    }

    /**
     * Sleep for specified milliseconds
     */
    async sleep(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

// Run monitor if this file is executed directly
if (require.main === module) {
    const monitor = new FaucetMonitor();
    monitor.start().catch(error => {
        console.error('Monitor failed to start:', error);
        process.exit(1);
    });
}

module.exports = { FaucetMonitor };
