/**
 * Simple API Server for KNIRVCONTROLLER
 * Basic endpoints to support frontend functionality
 */

import express from 'express';
import cors from 'cors';

const app = express();
const PORT = process.env.PORT || 3001;

// Middleware
app.use(cors());
app.use(express.json());

// In-memory storage
const memoryNodes = [];
const memoryEdges = [];

// Health check
app.get('/health', (req, res) => {
  res.json({ status: 'healthy', timestamp: new Date().toISOString() });
});

// KNIRVBASE initialization endpoint
app.post('/api/knirvbase/initialize', (req, res) => {
  try {
    const { sessionId } = req.body;
    console.log('Initializing KNIRVBASE for session:', sessionId);
    
    res.json({ 
      success: true, 
      message: 'KNIRVBASE database initialized successfully',
      dataDir: '/tmp/knirvbase-data'
    });
  } catch (error) {
    console.error('Failed to initialize KNIRVBASE:', error);
    res.status(500).json({ 
      error: 'Failed to initialize KNIRVBASE',
      details: error.message
    });
  }
});

// Memory graph endpoints
app.get('/api/memory/nodes', (req, res) => {
  res.json({ nodes: memoryNodes });
});

app.get('/api/memory/edges', (req, res) => {
  res.json({ edges: memoryEdges });
});

app.post('/api/memory/nodes', (req, res) => {
  const node = { id: `node-${Date.now()}`, ...req.body };
  memoryNodes.push(node);
  res.json({ success: true, node });
});

app.post('/api/memory/edges', (req, res) => {
  const edge = { id: `edge-${Date.now()}`, ...req.body };
  memoryEdges.push(edge);
  res.json({ success: true, edge });
});

// Graph ingestion endpoints
app.post('/api/graph/error', (req, res) => {
  try {
    const { errorId, description } = req.body;
    console.log('Creating error node:', { errorId, description });
    
    const node = {
      id: errorId || `error-${Date.now()}`,
      type: 'error',
      label: description || 'Error',
      timestamp: Date.now()
    };
    
    memoryNodes.push(node);
    res.json({ success: true, node });
  } catch (error) {
    console.error('Failed to create error node:', error);
    res.status(500).json({ error: 'Failed to create error node' });
  }
});

app.post('/api/graph/context', (req, res) => {
  try {
    const { contextId, contextName, description } = req.body;
    console.log('Creating context node:', { contextId, contextName });
    
    const node = {
      id: contextId || `context-${Date.now()}`,
      type: 'capability',
      label: contextName || 'Context',
      description,
      timestamp: Date.now()
    };
    
    memoryNodes.push(node);
    res.json({ success: true, node });
  } catch (error) {
    console.error('Failed to create context node:', error);
    res.status(500).json({ error: 'Failed to create context node' });
  }
});

app.post('/api/graph/idea', (req, res) => {
  try {
    const { ideaId, ideaName, description } = req.body;
    console.log('Creating idea node:', { ideaId, ideaName });
    
    const node = {
      id: ideaId || `idea-${Date.now()}`,
      type: 'property',
      label: ideaName || 'Idea',
      description,
      timestamp: Date.now()
    };
    
    memoryNodes.push(node);
    res.json({ success: true, node });
  } catch (error) {
    console.error('Failed to create idea node:', error);
    res.status(500).json({ error: 'Failed to create idea node' });
  }
});

// HRM Bridge endpoint
app.post('/api/cognitive/hrm/init', (req, res) => {
  try {
    const { modelPath, config } = req.body;
    console.log('HRM Bridge initialized:', { modelPath, config });
    res.json({ 
      status: 'initialized',
      modelPath,
      config,
      bridgeId: `hrm_${Date.now()}`
    });
  } catch (error) {
    console.error('Failed to initialize HRM Bridge:', error);
    res.status(500).json({ 
      error: 'Failed to initialize HRM Bridge',
      details: error.message
    });
  }
});

// Cognitive engine endpoints
app.post('/api/cognitive/start', (req, res) => {
  console.log('Cognitive engine started');
  res.json({ status: 'started' });
});

app.post('/api/cognitive/stop', (req, res) => {
  console.log('Cognitive engine stopped');
  res.json({ status: 'stopped' });
});

app.post('/api/cognitive/process', (req, res) => {
  const { input, taskType, requiresSkillInvocation } = req.body;
  
  // Simulate processing
  const processingTime = Math.random() * 1000 + 500; // 500-1500ms
  const skillsInvoked = requiresSkillInvocation ? ['analysis_skill', 'processing_skill'] : [];
  
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
  console.log(`Skill ${skillId} activated`);
  res.json({ status: 'activated' });
});

app.post('/api/cognitive/skills/:skillId/deactivate', (req, res) => {
  const { skillId } = req.params;
  console.log(`Skill ${skillId} deactivated`);
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

// Terminal command endpoint
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
      output = context?.workingDirectory || '/knirv';
      break;
    case 'echo':
      output = args?.join(' ') || '';
      break;
    case 'status':
      output = `KNIRV System Status:
  Cognitive Engine: Running
  Active Agents: 0
  Active Skills: 0
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

// Status endpoint
app.get('/api/status', (req, res) => {
  res.json({
    cognitive: {
      running: true,
      metrics: {
        totalProcessingRequests: 0,
        averageProcessingTime: 0,
        skillInvocations: 0,
        learningEvents: 0,
        adaptationLevel: 0.75,
        confidenceLevel: 0.95,
        activeSkills: 0,
        contextSize: memoryNodes.length
      }
    },
    agents: {
      running: true,
      count: 0
    },
    wallet: {
      connected: true
    },
    skills: {
      count: 0
    },
    uptime: process.uptime()
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

// Error handling middleware
app.use((err, req, res, next) => {
  console.error('API Error:', err);
  res.status(500).json({ 
    error: 'Internal server error',
    message: err.message,
    timestamp: new Date().toISOString()
  });
});

app.listen(PORT, () => {
  console.log(`🚀 Simple KNIRV API Server running on port ${PORT}`);
  console.log(`🔗 Health check: http://localhost:${PORT}/health`);
});