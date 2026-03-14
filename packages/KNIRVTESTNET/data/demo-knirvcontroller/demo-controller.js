const http = require('http');
const url = require('url');
const fs = require('fs');
const path = require('path');

const PORT = process.env.PORT || 8089;

// Demo data
const demoAgents = [
    { id: 'agent-001', name: 'Demo Agent 1', status: 'active', skills: ['data-analysis', 'reporting'] },
    { id: 'agent-002', name: 'Demo Agent 2', status: 'idle', skills: ['image-processing', 'classification'] },
    { id: 'agent-003', name: 'Demo Agent 3', status: 'training', skills: ['nlp', 'sentiment-analysis'] }
];

const demoSkills = [
    { id: 'skill-001', name: 'Data Analysis', category: 'analytics', status: 'available' },
    { id: 'skill-002', name: 'Image Processing', category: 'vision', status: 'available' },
    { id: 'skill-003', name: 'NLP Processing', category: 'language', status: 'training' }
];

const demoMetrics = {
    totalAgents: 3,
    activeAgents: 1,
    totalSkills: 3,
    availableSkills: 2,
    totalInvocations: 127,
    successRate: 94.5,
    avgResponseTime: 245
};

// Simple HTML interface
const htmlInterface = `
<!DOCTYPE html>
<html>
<head>
    <title>Demo KNIRVCONTROLLER</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; }
        .metric { background: #f8f9fa; padding: 15px; border-radius: 5px; text-align: center; }
        .metric-value { font-size: 24px; font-weight: bold; color: #667eea; }
        .metric-label { color: #666; margin-top: 5px; }
        .list-item { padding: 10px; border-bottom: 1px solid #eee; display: flex; justify-content: space-between; }
        .status-active { color: #28a745; }
        .status-idle { color: #ffc107; }
        .status-training { color: #17a2b8; }
        .demo-badge { background: #ff6b6b; color: white; padding: 4px 8px; border-radius: 4px; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎮 Demo KNIRVCONTROLLER</h1>
            <p>Lightweight demo service for KNIRV testnet development</p>
            <span class="demo-badge">DEMO MODE</span>
        </div>
        
        <div class="section">
            <h2>📊 System Metrics</h2>
            <div class="metrics">
                <div class="metric">
                    <div class="metric-value">${demoMetrics.totalAgents}</div>
                    <div class="metric-label">Total Agents</div>
                </div>
                <div class="metric">
                    <div class="metric-value">${demoMetrics.activeAgents}</div>
                    <div class="metric-label">Active Agents</div>
                </div>
                <div class="metric">
                    <div class="metric-value">${demoMetrics.totalSkills}</div>
                    <div class="metric-label">Total Skills</div>
                </div>
                <div class="metric">
                    <div class="metric-value">${demoMetrics.successRate}%</div>
                    <div class="metric-label">Success Rate</div>
                </div>
            </div>
        </div>
        
        <div class="section">
            <h2>🤖 Demo Agents</h2>
            ${demoAgents.map(agent => `
                <div class="list-item">
                    <div>
                        <strong>${agent.name}</strong> (${agent.id})<br>
                        <small>Skills: ${agent.skills.join(', ')}</small>
                    </div>
                    <div class="status-${agent.status}">${agent.status.toUpperCase()}</div>
                </div>
            `).join('')}
        </div>
        
        <div class="section">
            <h2>🛠️ Demo Skills</h2>
            ${demoSkills.map(skill => `
                <div class="list-item">
                    <div>
                        <strong>${skill.name}</strong> (${skill.id})<br>
                        <small>Category: ${skill.category}</small>
                    </div>
                    <div class="status-${skill.status}">${skill.status.toUpperCase()}</div>
                </div>
            `).join('')}
        </div>
        
        <div class="section">
            <h2>🔗 API Endpoints</h2>
            <ul>
                <li><code>GET /health</code> - Health check</li>
                <li><code>GET /api/agents</code> - List agents</li>
                <li><code>GET /api/skills</code> - List skills</li>
                <li><code>GET /api/metrics</code> - System metrics</li>
                <li><code>POST /api/agents/{id}/invoke</code> - Invoke agent skill</li>
            </ul>
        </div>
    </div>
</body>
</html>
`;

const server = http.createServer((req, res) => {
    const parsedUrl = url.parse(req.url, true);
    const pathname = parsedUrl.pathname;
    
    // CORS headers
    res.setHeader('Access-Control-Allow-Origin', '*');
    res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');
    
    if (req.method === 'OPTIONS') {
        res.writeHead(200);
        res.end();
        return;
    }
    
    // Routes
    if (pathname === '/' || pathname === '/dashboard') {
        res.writeHead(200, { 'Content-Type': 'text/html' });
        res.end(htmlInterface);
    } else if (pathname === '/health') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ 
            status: 'healthy', 
            service: 'demo-knirvcontroller',
            mode: 'demo',
            timestamp: new Date().toISOString(),
            port: PORT
        }));
    } else if (pathname === '/api/agents') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ agents: demoAgents }));
    } else if (pathname === '/api/skills') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ skills: demoSkills }));
    } else if (pathname === '/api/metrics') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify(demoMetrics));
    } else if (pathname.startsWith('/api/agents/') && pathname.endsWith('/invoke')) {
        const agentId = pathname.split('/')[3];
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ 
            result: 'success', 
            agentId: agentId,
            message: 'Demo skill invocation completed',
            timestamp: new Date().toISOString()
        }));
    } else {
        res.writeHead(404, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: 'Not found', service: 'demo-knirvcontroller' }));
    }
});

server.listen(PORT, () => {
    console.log(`Demo KNIRVCONTROLLER running on port ${PORT}`);
    console.log(`Dashboard: http://localhost:${PORT}/`);
    console.log(`Health: http://localhost:${PORT}/health`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('Demo KNIRVCONTROLLER shutting down...');
    server.close(() => {
        process.exit(0);
    });
});
