/**
 * KNIRV Controller API Server
 * Provides backend endpoints for all Phase 1 services
 */

import express from 'express';
import cors from 'cors';
import { WebSocketServer } from 'ws';
import { createServer } from 'http';

const app = express();
const server = createServer(app);
const wss = new WebSocketServer({ server });

// Middleware
app.use(cors());
app.use(express.json({ limit: '50mb' }));
app.use(express.urlencoded({ extended: true }));

// In-memory storage for demo (replace with real database in production)
const agents = new Map();
// const transactions = new Map();
const cognitiveState = {
  isRunning: false,
  metrics: {
    totalProcessingRequests: 0,
    averageProcessingTime: 0,
    skillInvocations: 0,
    learningEvents: 0,
    adaptationLevel: 0.75,
    confidenceLevel: 0.95,
    activeSkills: 0,
    contextSize: 0
  },
  activeSkills: new Set(),
  context: new Map()
};

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ status: 'healthy', timestamp: new Date().toISOString() });
});

// System status endpoint
app.get('/api/status', (req, res) => {
  res.json({
    cognitive: {
      running: cognitiveState.isRunning,
      metrics: cognitiveState.metrics
    },
    agents: {
      running: true,
      count: agents.size
    },
    wallet: {
      connected: true
    },
    skills: {
      count: cognitiveState.activeSkills.size
    },
    uptime: process.uptime()
  });
});

// Agent Management Endpoints
app.post('/api/agents/deploy', (req, res) => {
  const { agentId, targetNRV } = req.body;
  
  const deploymentId = `deployment_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  
  // Simulate deployment
  setTimeout(() => {
    console.log(`Agent ${agentId} deployed to ${targetNRV || 'default'}`);
  }, 1000);
  
  res.json({ deploymentId, status: 'deploying' });
});

app.post('/api/agents/:agentId/execute', (req, res) => {
  // const { agentId } = req.params;
  const { skillId, parameters } = req.body;
  
  // Simulate skill execution
  const output = {
    result: `Skill ${skillId} executed successfully`,
    parameters,
    timestamp: new Date().toISOString()
  };
  
  res.json({
    output,
    resourceUsage: { memory: 64, cpu: 0.5 }
  });
});

app.post('/api/agents/:agentId/undeploy', (req, res) => {
  const { agentId } = req.params;
  console.log(`Agent ${agentId} undeployed`);
  res.json({ status: 'undeployed' });
});

// Cognitive Engine Endpoints
app.post('/api/cognitive/start', (req, res) => {
  cognitiveState.isRunning = true;
  console.log('Cognitive engine started');
  res.json({ status: 'started' });
});

app.post('/api/cognitive/stop', (req, res) => {
  cognitiveState.isRunning = false;
  console.log('Cognitive engine stopped');
  res.json({ status: 'stopped' });
});

app.post('/api/cognitive/process', (req, res) => {
  const { input, taskType, requiresSkillInvocation } = req.body;
  
  if (!cognitiveState.isRunning) {
    return res.status(400).json({ error: 'Cognitive engine is not running' });
  }
  
  // Simulate processing
  const processingTime = Math.random() * 1000 + 500; // 500-1500ms
  const skillsInvoked = requiresSkillInvocation ? ['analysis_skill', 'processing_skill'] : [];
  
  cognitiveState.metrics.totalProcessingRequests++;
  cognitiveState.metrics.skillInvocations += skillsInvoked.length;
  
  const response = {
    output: `Processed: ${input}. Task type: ${taskType}`,
    confidence: 0.95,
    skillsInvoked,
    processingTime,
    contextUpdates: { lastInput: input, timestamp: Date.now() },
    adaptationTriggered: Math.random() > 0.8
  };
  
  res.json(response);
});

app.post('/api/cognitive/skills/:skillId/execute', (req, res) => {
  const { skillId } = req.params;
  const { parameters, context } = req.body;
  
  // Simulate skill execution
  const output = {
    skillId,
    result: `Skill ${skillId} executed with parameters: ${JSON.stringify(parameters)}`,
    context,
    timestamp: new Date().toISOString()
  };
  
  res.json({
    output,
    resourceUsage: { memory: 32, cpu: 0.3 }
  });
});

app.post('/api/cognitive/skills/:skillId/activate', (req, res) => {
  const { skillId } = req.params;
  cognitiveState.activeSkills.add(skillId);
  cognitiveState.metrics.activeSkills = cognitiveState.activeSkills.size;
  res.json({ status: 'activated' });
});

app.post('/api/cognitive/skills/:skillId/deactivate', (req, res) => {
  const { skillId } = req.params;
  cognitiveState.activeSkills.delete(skillId);
  cognitiveState.metrics.activeSkills = cognitiveState.activeSkills.size;
  res.json({ status: 'deactivated' });
});

app.post('/api/cognitive/learning/start', (req, res) => {
  console.log('Learning mode started');
  res.json({ status: 'learning_started' });
});

app.post('/api/cognitive/adaptation/save', (req, res) => {
  const { context, activeSkills, metrics, timestamp } = req.body;
  console.log('Adaptation saved:', { context, activeSkills, metrics, timestamp });
  res.json({ status: 'adaptation_saved' });
});

app.post('/api/cognitive/hrm/init', (req, res) => {
  const { modelPath, config } = req.body;
  console.log('HRM Bridge initialized:', { modelPath, config });
  res.json({ 
    status: 'initialized',
    modelPath,
    config,
    bridgeId: `hrm_${Date.now()}`
  });
});

// Terminal Command Endpoints
app.post('/api/terminal/execute', (req, res) => {
  const { command, args, context } = req.body;
  
  // Simulate command execution
  let output = '';
  let exitCode = 0;
  
  switch (command) {
    case 'ls':
      output = 'agents/\nskills/\nconfig/\nlogs/\ndata/\nREADME.md';
      break;
    case 'pwd':
      output = context.workingDirectory || '/knirv';
      break;
    case 'echo':
      output = args.join(' ');
      break;
    case 'status':
      output = `KNIRV System Status:
  Cognitive Engine: ${cognitiveState.isRunning ? 'Running' : 'Stopped'}
  Active Agents: ${agents.size}
  Active Skills: ${cognitiveState.activeSkills.size}
  Uptime: ${Math.floor(process.uptime())}s`;
      break;
    default:
      output = '';
      exitCode = 127;
      break;
  }
  
  res.json({
    success: exitCode === 0,
    output,
    exitCode,
    executionTime: Math.random() * 100 + 50
  });
});

// WebSocket for real-time updates
wss.on('connection', (ws) => {
  console.log('WebSocket client connected');
  
  // Send initial status
  ws.send(JSON.stringify({
    type: 'status',
    data: {
      cognitive: cognitiveState,
      agents: Array.from(agents.values()),
      timestamp: Date.now()
    }
  }));
  
  ws.on('message', (message) => {
    try {
      const data = JSON.parse(message.toString());
      console.log('WebSocket message received:', data);
      
      // Echo back for now
      ws.send(JSON.stringify({
        type: 'response',
        data: { received: data, timestamp: Date.now() }
      }));
    } catch (error) {
      console.error('WebSocket message error:', error);
    }
  });
  
  ws.on('close', () => {
    console.log('WebSocket client disconnected');
  });
});

// Error handling middleware
app.use((err: unknown, req: express.Request, res: express.Response, _next: express.NextFunction) => {
  console.error('API Error:', err);
  res.status(500).json({ 
    error: 'Internal server error',
    message: err.message,
    timestamp: new Date().toISOString()
  });
});

// 404 handler
app.use((req, res) => {
  res.status(404).json({ 
    error: 'Not found',
    path: req.path,
    timestamp: new Date().toISOString()
  });
});

const PORT = process.env.PORT || 3001;

server.listen(PORT, () => {
  console.log(`🚀 KNIRV Controller API Server running on port ${PORT}`);
  console.log(`📡 WebSocket server ready for real-time updates`);
  console.log(`🔗 Health check: http://localhost:${PORT}/health`);
});

export default app;
