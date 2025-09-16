/**
 * KNIRV Controller API Server
 * Provides backend endpoints for all Phase 1 services
 */

import express from 'express';
import cors from 'cors';
import { WebSocketServer } from 'ws';
import { createServer } from 'http';
import { apiKeyService, ApiKey } from '../services/ApiKeyService';

const app = express();
const server = createServer(app);
const wss = new WebSocketServer({ server });

// Middleware
app.use(cors());
app.use(express.json({ limit: '50mb' }));
app.use(express.urlencoded({ extended: true }));

// API Key Authentication Middleware
const authenticateApiKey = async (req: express.Request, res: express.Response, next: express.NextFunction) => {
  // Skip authentication for health check and public endpoints
  if (req.path === '/health' || req.path === '/api/status' || req.path.startsWith('/public/')) {
    return next();
  }

  const apiKey = req.headers['x-api-key'] as string || req.headers['authorization']?.replace('Bearer ', '');

  if (!apiKey) {
    return res.status(401).json({
      error: 'API key required',
      message: 'Please provide an API key in the X-API-Key header or Authorization header'
    });
  }

  try {
    const validatedKey = await apiKeyService.validateApiKey(apiKey);
    if (!validatedKey) {
      return res.status(401).json({
        error: 'Invalid API key',
        message: 'The provided API key is invalid or expired'
      });
    }

    // Check rate limits
    const rateLimitCheck = await apiKeyService.checkRateLimit(validatedKey);
    if (!rateLimitCheck.allowed) {
      return res.status(429).json({
        error: 'Rate limit exceeded',
        message: 'API rate limit exceeded',
        resetTime: rateLimitCheck.resetTime
      });
    }

    // Record usage
    await apiKeyService.recordUsage({
      keyId: validatedKey.id,
      endpoint: req.path,
      method: req.method,
      timestamp: new Date(),
      ipAddress: req.ip || 'unknown',
      userAgent: req.headers['user-agent'] || 'unknown',
      responseStatus: 200, // Will be updated in response
      responseTime: 0 // Will be calculated
    });

    // Attach API key info to request
    (req as express.Request & { apiKey: ApiKey }).apiKey = validatedKey;
    next();
  } catch (error) {
    console.error('API key authentication error:', error);
    return res.status(500).json({
      error: 'Authentication error',
      message: 'Internal server error during authentication'
    });
  }
};

// Permission checking middleware
const requirePermission = (permission: string) => {
  return (req: express.Request, res: express.Response, next: express.NextFunction) => {
    const apiKey = (req as express.Request & { apiKey: ApiKey }).apiKey;

    if (!apiKey) {
      return res.status(401).json({ error: 'Authentication required' });
    }

    if (!apiKeyService.hasPermission(apiKey, permission)) {
      return res.status(403).json({
        error: 'Insufficient permissions',
        message: `This API key does not have the required permission: ${permission}`,
        required: permission,
        available: apiKey.permissions
      });
    }

    next();
  };
};

// Apply authentication middleware to API routes
app.use('/api', authenticateApiKey);

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

// API Key Management Endpoints
app.post('/api/keys', requirePermission('admin:all'), async (req, res) => {
  try {
    const { name, description, permissions, expiresAt, rateLimit } = req.body;

    if (!name || !description || !permissions) {
      return res.status(400).json({
        error: 'Missing required fields',
        required: ['name', 'description', 'permissions']
      });
    }

    const apiKey = await apiKeyService.createApiKey({
      name,
      description,
      permissions,
      expiresAt: expiresAt ? new Date(expiresAt) : undefined,
      rateLimit
    });

    res.json({
      success: true,
      apiKey: {
        ...apiKey,
        key: apiKey.key // Include the actual key only in creation response
      }
    });
  } catch (error) {
    console.error('Failed to create API key:', error);
    res.status(500).json({ error: 'Failed to create API key' });
  }
});

app.get('/api/keys', requirePermission('admin:all'), async (req, res) => {
  try {
    const apiKeys = await apiKeyService.getApiKeys();
    // Don't return the actual keys for security
    const safeKeys = apiKeys.map(key => ({
      ...key,
      key: key.key.substring(0, 8) + '...' // Show only first 8 characters
    }));

    res.json({ apiKeys: safeKeys });
  } catch (error) {
    console.error('Failed to get API keys:', error);
    res.status(500).json({ error: 'Failed to get API keys' });
  }
});

app.delete('/api/keys/:keyId', requirePermission('admin:all'), async (req, res) => {
  try {
    const { keyId } = req.params;
    await apiKeyService.deleteApiKey(keyId);
    res.json({ success: true, message: 'API key deleted successfully' });
  } catch (error) {
    console.error('Failed to delete API key:', error);
    res.status(500).json({ error: 'Failed to delete API key' });
  }
});

app.get('/api/keys/:keyId/usage', requirePermission('admin:all'), async (req, res) => {
  try {
    const { keyId } = req.params;
    const usage = await apiKeyService.getApiKeyUsage(keyId);
    res.json({ usage });
  } catch (error) {
    console.error('Failed to get API key usage:', error);
    res.status(500).json({ error: 'Failed to get API key usage' });
  }
});

app.get('/api/permissions', requirePermission('admin:all'), (req, res) => {
  const permissions = apiKeyService.getAvailablePermissions();
  res.json({ permissions });
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

// Graph ingestion endpoints: Error, Context, Idea
import { personalKNIRVGRAPHService } from '../services/PersonalKNIRVGRAPHService';
import { FactualitySlice } from '../slices/factualitySlice';
import { FeasibilityReport } from '../slices/feasibilitySlice';

app.post('/api/graph/error', authenticateApiKey, requirePermission('write:graph'), async (req, res) => {
  try {
  const { errorId, errorType, description, context, timestamp, factualitySlice } = req.body as { errorId?: string; errorType?: string; description?: string; context?: Record<string, unknown>; timestamp?: number; factualitySlice?: FactualitySlice };
  if (!errorId || !description) return res.status(400).json({ error: 'Missing required fields: errorId or description' });

  const node = await personalKNIRVGRAPHService.addErrorNode({ errorId, errorType: errorType || 'user-submitted', description, context: context || {}, timestamp: timestamp || Date.now(), factualitySlice });

    res.json({ success: true, node });
  } catch (err) {
    console.error('Failed to create error node:', err);
    res.status(500).json({ error: 'Failed to create error node' });
  }
});

app.post('/api/graph/context', authenticateApiKey, requirePermission('write:graph'), async (req, res) => {
  try {
  const { contextId, contextName, description, mcpServerInfo, category, timestamp, capabilitySlice } = req.body as { contextId?: string; contextName?: string; description?: string; mcpServerInfo?: Record<string, unknown>; category?: string; timestamp?: number; capabilitySlice?: FactualitySlice };
  if (!contextId || !contextName) return res.status(400).json({ error: 'Missing required fields: contextId or contextName' });

  const node = await personalKNIRVGRAPHService.addContextNode({ contextId, contextName, description: description || '', mcpServerInfo: mcpServerInfo || {}, category: category || 'integration', timestamp: timestamp || Date.now(), capabilitySlice });

    res.json({ success: true, node });
  } catch (err) {
    console.error('Failed to create context node:', err);
    res.status(500).json({ error: 'Failed to create context node' });
  }
});

app.post('/api/graph/idea', authenticateApiKey, requirePermission('write:graph'), async (req, res) => {
  try {
  const { ideaId, ideaName, description, timestamp, feasibilitySlice } = req.body as { ideaId?: string; ideaName?: string; description?: string; timestamp?: number; feasibilitySlice?: FeasibilityReport };
  if (!ideaId || !ideaName) return res.status(400).json({ error: 'Missing required fields: ideaId or ideaName' });

  const node = await personalKNIRVGRAPHService.addIdeaNode({ ideaId, ideaName, description: description || '', timestamp: timestamp || Date.now(), feasibilitySlice });

    res.json({ success: true, node });
  } catch (err) {
    console.error('Failed to create idea node:', err);
    res.status(500).json({ error: 'Failed to create idea node' });
  }
});

// Agent Management Endpoints
app.post('/api/agents/deploy', requirePermission('write:agents'), (req, res) => {
  const { agentId, targetNRV } = req.body;
  
  const deploymentId = `deployment_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  
  // Simulate deployment
  setTimeout(() => {
    console.log(`Agent ${agentId} deployed to ${targetNRV || 'default'}`);
  }, 1000);
  
  res.json({ deploymentId, status: 'deploying' });
});

app.post('/api/agents/:agentId/execute', requirePermission('write:agents'), (req, res) => {
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
    message: err instanceof Error ? err.message : String(err),
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
