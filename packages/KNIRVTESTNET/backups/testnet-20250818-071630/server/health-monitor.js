// KNIRV Testnet Health Monitor Server
const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = process.env.HEALTH_MONITOR_PORT || 10001;

// Enable CORS for all routes
app.use(cors());
app.use(express.json());

// Testnet service endpoints
const services = {
    knirvoracle: 'https://oracle-test.knirv.com',
    knirvchain: 'https://chain-test.knirv.com',
    knirvgraph: 'https://graph-test.knirv.com',
    knirvserver: 'https://nexus-test.knirv.com',
    knirvrouter: 'https://router-test.knirv.com',
    knirvgateway: 'https://testnet.knirv.com',
    nanda_ans: 'https://nanda-test.knirv.com'
};

// Health check function for individual services
async function checkServiceHealth(serviceName, serviceUrl) {
    const startTime = Date.now();
    
    try {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), 5000); // 5s timeout
        
        // Try different health endpoints
        const healthEndpoints = ['/health', '/status', '/'];
        let response = null;
        let healthEndpoint = '';
        
        for (const endpoint of healthEndpoints) {
            try {
                response = await fetch(`${serviceUrl}${endpoint}`, {
                    method: 'GET',
                    signal: controller.signal,
                    headers: {
                        'User-Agent': 'KNIRV-Testnet-Health-Monitor/1.0'
                    }
                });
                healthEndpoint = endpoint;
                break;
            } catch (e) {
                // Try next endpoint
                continue;
            }
        }
        
        clearTimeout(timeoutId);
        
        if (!response) {
            throw new Error('No response from any health endpoint');
        }
        
        const responseTime = Date.now() - startTime;
        const healthy = response.ok;
        
        let details = {};
        try {
            const text = await response.text();
            details = text ? (text.startsWith('{') ? JSON.parse(text) : { raw: text }) : {};
        } catch (e) {
            details = { error: 'Could not parse response' };
        }
        
        return {
            name: serviceName,
            healthy,
            status: response.status,
            responseTime,
            lastCheck: new Date().toISOString(),
            url: serviceUrl,
            endpoint: healthEndpoint,
            details
        };
        
    } catch (error) {
        const responseTime = Date.now() - startTime;
        
        return {
            name: serviceName,
            healthy: false,
            status: 0,
            responseTime,
            lastCheck: new Date().toISOString(),
            url: serviceUrl,
            error: error.message,
            details: { error: error.message }
        };
    }
}

// Check all services
async function checkAllServices() {
    const results = {};
    
    // Check each service in parallel
    const checks = Object.entries(services).map(async ([name, url]) => {
        const result = await checkServiceHealth(name, url);
        results[name] = result;
    });
    
    await Promise.all(checks);
    return results;
}

// Health status endpoint
app.get('/health-monitor/status', async (req, res) => {
    try {
        const healthResults = await checkAllServices();
        
        const overallHealthy = Object.values(healthResults).every(s => s.healthy);
        const partiallyHealthy = Object.values(healthResults).some(s => s.healthy);
        
        let overall = 'unhealthy';
        if (overallHealthy) {
            overall = 'healthy';
        } else if (partiallyHealthy) {
            overall = 'degraded';
        }
        
        res.json({
            timestamp: Date.now(),
            services: healthResults,
            overall,
            summary: {
                total: Object.keys(healthResults).length,
                healthy: Object.values(healthResults).filter(s => s.healthy).length,
                unhealthy: Object.values(healthResults).filter(s => !s.healthy).length
            }
        });
    } catch (error) {
        console.error('Health check error:', error);
        res.status(500).json({
            error: 'Health check failed',
            message: error.message,
            timestamp: Date.now()
        });
    }
});

// Serve the health monitor HTML page
app.get('/health-monitor', (req, res) => {
    res.sendFile(path.join(__dirname, '../health-monitor.html'));
});

// Root endpoint
app.get('/', (req, res) => {
    res.json({
        service: 'KNIRV Testnet Health Monitor',
        version: '1.0.0',
        endpoints: {
            status: '/health-monitor/status',
            monitor: '/health-monitor'
        },
        timestamp: Date.now()
    });
});

// Start server
app.listen(PORT, () => {
    console.log(`🏥 KNIRV Testnet Health Monitor running on port ${PORT}`);
    console.log(`📊 Health Status: http://localhost:${PORT}/health-monitor/status`);
    console.log(`🖥️  Health Monitor: http://localhost:${PORT}/health-monitor`);
});

module.exports = app;
