"use strict";
/**
 * KNIRV Controller API Server
 * Provides backend endpoints for all Phase 1 services
 */
var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g = Object.create((typeof Iterator === "function" ? Iterator : Object).prototype);
    return g.next = verb(0), g["throw"] = verb(1), g["return"] = verb(2), typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (g && (g = 0, op[0] && (_ = 0)), _) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
Object.defineProperty(exports, "__esModule", { value: true });
var express_1 = require("express");
var cors_1 = require("cors");
var ws_1 = require("ws");
var http_1 = require("http");
var ApiKeyService_1 = require("../services/ApiKeyService");
var app = (0, express_1.default)();
var server = (0, http_1.createServer)(app);
var wss = new ws_1.WebSocketServer({ server: server });
// Middleware
app.use((0, cors_1.default)());
app.use(express_1.default.json({ limit: '50mb' }));
app.use(express_1.default.urlencoded({ extended: true }));
// API Key Authentication Middleware
var authenticateApiKey = function (req, res, next) { return __awaiter(void 0, void 0, void 0, function () {
    var apiKey, validatedKey, rateLimitCheck, error_1;
    var _a;
    return __generator(this, function (_b) {
        switch (_b.label) {
            case 0:
                // Skip authentication for health check and public endpoints
                if (req.path === '/health' || req.path === '/api/status' || req.path.startsWith('/public/')) {
                    return [2 /*return*/, next()];
                }
                apiKey = req.headers['x-api-key'] || ((_a = req.headers['authorization']) === null || _a === void 0 ? void 0 : _a.replace('Bearer ', ''));
                if (!apiKey) {
                    return [2 /*return*/, res.status(401).json({
                            error: 'API key required',
                            message: 'Please provide an API key in the X-API-Key header or Authorization header'
                        })];
                }
                _b.label = 1;
            case 1:
                _b.trys.push([1, 5, , 6]);
                return [4 /*yield*/, ApiKeyService_1.apiKeyService.validateApiKey(apiKey)];
            case 2:
                validatedKey = _b.sent();
                if (!validatedKey) {
                    return [2 /*return*/, res.status(401).json({
                            error: 'Invalid API key',
                            message: 'The provided API key is invalid or expired'
                        })];
                }
                return [4 /*yield*/, ApiKeyService_1.apiKeyService.checkRateLimit(validatedKey)];
            case 3:
                rateLimitCheck = _b.sent();
                if (!rateLimitCheck.allowed) {
                    return [2 /*return*/, res.status(429).json({
                            error: 'Rate limit exceeded',
                            message: 'API rate limit exceeded',
                            resetTime: rateLimitCheck.resetTime
                        })];
                }
                // Record usage
                return [4 /*yield*/, ApiKeyService_1.apiKeyService.recordUsage({
                        keyId: validatedKey.id,
                        endpoint: req.path,
                        method: req.method,
                        timestamp: new Date(),
                        ipAddress: req.ip || 'unknown',
                        userAgent: req.headers['user-agent'] || 'unknown',
                        responseStatus: 200, // Will be updated in response
                        responseTime: 0 // Will be calculated
                    })];
            case 4:
                // Record usage
                _b.sent();
                // Attach API key info to request
                req.apiKey = validatedKey;
                next();
                return [3 /*break*/, 6];
            case 5:
                error_1 = _b.sent();
                console.error('API key authentication error:', error_1);
                return [2 /*return*/, res.status(500).json({
                        error: 'Authentication error',
                        message: 'Internal server error during authentication'
                    })];
            case 6: return [2 /*return*/];
        }
    });
}); };
// Permission checking middleware
var requirePermission = function (permission) {
    return function (req, res, next) {
        var apiKey = req.apiKey;
        if (!apiKey) {
            return res.status(401).json({ error: 'Authentication required' });
        }
        if (!ApiKeyService_1.apiKeyService.hasPermission(apiKey, permission)) {
            return res.status(403).json({
                error: 'Insufficient permissions',
                message: "This API key does not have the required permission: ".concat(permission),
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
var agents = new Map();
// const transactions = new Map();
var cognitiveState = {
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
app.get('/health', function (req, res) {
    res.json({ status: 'healthy', timestamp: new Date().toISOString() });
});
// API Key Management Endpoints
app.post('/api/keys', requirePermission('admin:all'), function (req, res) { return __awaiter(void 0, void 0, void 0, function () {
    var _a, name_1, description, permissions, expiresAt, rateLimit, apiKey, error_2;
    return __generator(this, function (_b) {
        switch (_b.label) {
            case 0:
                _b.trys.push([0, 2, , 3]);
                _a = req.body, name_1 = _a.name, description = _a.description, permissions = _a.permissions, expiresAt = _a.expiresAt, rateLimit = _a.rateLimit;
                if (!name_1 || !description || !permissions) {
                    return [2 /*return*/, res.status(400).json({
                            error: 'Missing required fields',
                            required: ['name', 'description', 'permissions']
                        })];
                }
                return [4 /*yield*/, ApiKeyService_1.apiKeyService.createApiKey({
                        name: name_1,
                        description: description,
                        permissions: permissions,
                        expiresAt: expiresAt ? new Date(expiresAt) : undefined,
                        rateLimit: rateLimit
                    })];
            case 1:
                apiKey = _b.sent();
                res.json({
                    success: true,
                    apiKey: __assign(__assign({}, apiKey), { key: apiKey.key // Include the actual key only in creation response
                     })
                });
                return [3 /*break*/, 3];
            case 2:
                error_2 = _b.sent();
                console.error('Failed to create API key:', error_2);
                res.status(500).json({ error: 'Failed to create API key' });
                return [3 /*break*/, 3];
            case 3: return [2 /*return*/];
        }
    });
}); });
app.get('/api/keys', requirePermission('admin:all'), function (req, res) { return __awaiter(void 0, void 0, void 0, function () {
    var apiKeys, safeKeys, error_3;
    return __generator(this, function (_a) {
        switch (_a.label) {
            case 0:
                _a.trys.push([0, 2, , 3]);
                return [4 /*yield*/, ApiKeyService_1.apiKeyService.getApiKeys()];
            case 1:
                apiKeys = _a.sent();
                safeKeys = apiKeys.map(function (key) { return (__assign(__assign({}, key), { key: key.key.substring(0, 8) + '...' // Show only first 8 characters
                 })); });
                res.json({ apiKeys: safeKeys });
                return [3 /*break*/, 3];
            case 2:
                error_3 = _a.sent();
                console.error('Failed to get API keys:', error_3);
                res.status(500).json({ error: 'Failed to get API keys' });
                return [3 /*break*/, 3];
            case 3: return [2 /*return*/];
        }
    });
}); });
app.delete('/api/keys/:keyId', requirePermission('admin:all'), function (req, res) { return __awaiter(void 0, void 0, void 0, function () {
    var keyId, error_4;
    return __generator(this, function (_a) {
        switch (_a.label) {
            case 0:
                _a.trys.push([0, 2, , 3]);
                keyId = req.params.keyId;
                return [4 /*yield*/, ApiKeyService_1.apiKeyService.deleteApiKey(keyId)];
            case 1:
                _a.sent();
                res.json({ success: true, message: 'API key deleted successfully' });
                return [3 /*break*/, 3];
            case 2:
                error_4 = _a.sent();
                console.error('Failed to delete API key:', error_4);
                res.status(500).json({ error: 'Failed to delete API key' });
                return [3 /*break*/, 3];
            case 3: return [2 /*return*/];
        }
    });
}); });
app.get('/api/keys/:keyId/usage', requirePermission('admin:all'), function (req, res) { return __awaiter(void 0, void 0, void 0, function () {
    var keyId, usage, error_5;
    return __generator(this, function (_a) {
        switch (_a.label) {
            case 0:
                _a.trys.push([0, 2, , 3]);
                keyId = req.params.keyId;
                return [4 /*yield*/, ApiKeyService_1.apiKeyService.getApiKeyUsage(keyId)];
            case 1:
                usage = _a.sent();
                res.json({ usage: usage });
                return [3 /*break*/, 3];
            case 2:
                error_5 = _a.sent();
                console.error('Failed to get API key usage:', error_5);
                res.status(500).json({ error: 'Failed to get API key usage' });
                return [3 /*break*/, 3];
            case 3: return [2 /*return*/];
        }
    });
}); });
app.get('/api/permissions', requirePermission('admin:all'), function (req, res) {
    var permissions = ApiKeyService_1.apiKeyService.getAvailablePermissions();
    res.json({ permissions: permissions });
});
// System status endpoint
app.get('/api/status', function (req, res) {
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
var PersonalKNIRVGRAPHService_1 = require("../services/PersonalKNIRVGRAPHService");
app.post('/api/graph/error', authenticateApiKey, requirePermission('write:graph'), function (req, res) { return __awaiter(void 0, void 0, void 0, function () {
    var _a, errorId, errorType, description, context, timestamp, factualitySlice, node, err_1;
    return __generator(this, function (_b) {
        switch (_b.label) {
            case 0:
                _b.trys.push([0, 2, , 3]);
                _a = req.body, errorId = _a.errorId, errorType = _a.errorType, description = _a.description, context = _a.context, timestamp = _a.timestamp, factualitySlice = _a.factualitySlice;
                if (!errorId || !description)
                    return [2 /*return*/, res.status(400).json({ error: 'Missing required fields: errorId or description' })];
                return [4 /*yield*/, PersonalKNIRVGRAPHService_1.personalKNIRVGRAPHService.addErrorNode({ errorId: errorId, errorType: errorType || 'user-submitted', description: description, context: context || {}, timestamp: timestamp || Date.now(), factualitySlice: factualitySlice })];
            case 1:
                node = _b.sent();
                res.json({ success: true, node: node });
                return [3 /*break*/, 3];
            case 2:
                err_1 = _b.sent();
                console.error('Failed to create error node:', err_1);
                res.status(500).json({ error: 'Failed to create error node' });
                return [3 /*break*/, 3];
            case 3: return [2 /*return*/];
        }
    });
}); });
app.post('/api/graph/context', authenticateApiKey, requirePermission('write:graph'), function (req, res) { return __awaiter(void 0, void 0, void 0, function () {
    var _a, contextId, contextName, description, mcpServerInfo, category, timestamp, capabilitySlice, node, err_2;
    return __generator(this, function (_b) {
        switch (_b.label) {
            case 0:
                _b.trys.push([0, 2, , 3]);
                _a = req.body, contextId = _a.contextId, contextName = _a.contextName, description = _a.description, mcpServerInfo = _a.mcpServerInfo, category = _a.category, timestamp = _a.timestamp, capabilitySlice = _a.capabilitySlice;
                if (!contextId || !contextName)
                    return [2 /*return*/, res.status(400).json({ error: 'Missing required fields: contextId or contextName' })];
                return [4 /*yield*/, PersonalKNIRVGRAPHService_1.personalKNIRVGRAPHService.addContextNode({ contextId: contextId, contextName: contextName, description: description || '', mcpServerInfo: mcpServerInfo || {}, category: category || 'integration', timestamp: timestamp || Date.now(), capabilitySlice: capabilitySlice })];
            case 1:
                node = _b.sent();
                res.json({ success: true, node: node });
                return [3 /*break*/, 3];
            case 2:
                err_2 = _b.sent();
                console.error('Failed to create context node:', err_2);
                res.status(500).json({ error: 'Failed to create context node' });
                return [3 /*break*/, 3];
            case 3: return [2 /*return*/];
        }
    });
}); });
app.post('/api/graph/idea', authenticateApiKey, requirePermission('write:graph'), function (req, res) { return __awaiter(void 0, void 0, void 0, function () {
    var _a, ideaId, ideaName, description, timestamp, feasibilitySlice, node, err_3;
    return __generator(this, function (_b) {
        switch (_b.label) {
            case 0:
                _b.trys.push([0, 2, , 3]);
                _a = req.body, ideaId = _a.ideaId, ideaName = _a.ideaName, description = _a.description, timestamp = _a.timestamp, feasibilitySlice = _a.feasibilitySlice;
                if (!ideaId || !ideaName)
                    return [2 /*return*/, res.status(400).json({ error: 'Missing required fields: ideaId or ideaName' })];
                return [4 /*yield*/, PersonalKNIRVGRAPHService_1.personalKNIRVGRAPHService.addIdeaNode({ ideaId: ideaId, ideaName: ideaName, description: description || '', timestamp: timestamp || Date.now(), feasibilitySlice: feasibilitySlice })];
            case 1:
                node = _b.sent();
                res.json({ success: true, node: node });
                return [3 /*break*/, 3];
            case 2:
                err_3 = _b.sent();
                console.error('Failed to create idea node:', err_3);
                res.status(500).json({ error: 'Failed to create idea node' });
                return [3 /*break*/, 3];
            case 3: return [2 /*return*/];
        }
    });
}); });
// Agent Management Endpoints
app.post('/api/agents/deploy', requirePermission('write:agents'), function (req, res) {
    var _a = req.body, agentId = _a.agentId, targetNRV = _a.targetNRV;
    var deploymentId = "deployment_".concat(Date.now(), "_").concat(Math.random().toString(36).substr(2, 9));
    // Simulate deployment
    setTimeout(function () {
        console.log("Agent ".concat(agentId, " deployed to ").concat(targetNRV || 'default'));
    }, 1000);
    res.json({ deploymentId: deploymentId, status: 'deploying' });
});
app.post('/api/agents/:agentId/execute', requirePermission('write:agents'), function (req, res) {
    // const { agentId } = req.params;
    var _a = req.body, skillId = _a.skillId, parameters = _a.parameters;
    // Simulate skill execution
    var output = {
        result: "Skill ".concat(skillId, " executed successfully"),
        parameters: parameters,
        timestamp: new Date().toISOString()
    };
    res.json({
        output: output,
        resourceUsage: { memory: 64, cpu: 0.5 }
    });
});
app.post('/api/agents/:agentId/undeploy', function (req, res) {
    var agentId = req.params.agentId;
    console.log("Agent ".concat(agentId, " undeployed"));
    res.json({ status: 'undeployed' });
});
// Cognitive Engine Endpoints
app.post('/api/cognitive/start', function (req, res) {
    cognitiveState.isRunning = true;
    console.log('Cognitive engine started');
    res.json({ status: 'started' });
});
app.post('/api/cognitive/stop', function (req, res) {
    cognitiveState.isRunning = false;
    console.log('Cognitive engine stopped');
    res.json({ status: 'stopped' });
});
app.post('/api/cognitive/process', function (req, res) {
    var _a = req.body, input = _a.input, taskType = _a.taskType, requiresSkillInvocation = _a.requiresSkillInvocation;
    if (!cognitiveState.isRunning) {
        return res.status(400).json({ error: 'Cognitive engine is not running' });
    }
    // Simulate processing
    var processingTime = Math.random() * 1000 + 500; // 500-1500ms
    var skillsInvoked = requiresSkillInvocation ? ['analysis_skill', 'processing_skill'] : [];
    cognitiveState.metrics.totalProcessingRequests++;
    cognitiveState.metrics.skillInvocations += skillsInvoked.length;
    var response = {
        output: "Processed: ".concat(input, ". Task type: ").concat(taskType),
        confidence: 0.95,
        skillsInvoked: skillsInvoked,
        processingTime: processingTime,
        contextUpdates: { lastInput: input, timestamp: Date.now() },
        adaptationTriggered: Math.random() > 0.8
    };
    res.json(response);
});
app.post('/api/cognitive/skills/:skillId/execute', function (req, res) {
    var skillId = req.params.skillId;
    var _a = req.body, parameters = _a.parameters, context = _a.context;
    // Simulate skill execution
    var output = {
        skillId: skillId,
        result: "Skill ".concat(skillId, " executed with parameters: ").concat(JSON.stringify(parameters)),
        context: context,
        timestamp: new Date().toISOString()
    };
    res.json({
        output: output,
        resourceUsage: { memory: 32, cpu: 0.3 }
    });
});
app.post('/api/cognitive/skills/:skillId/activate', function (req, res) {
    var skillId = req.params.skillId;
    cognitiveState.activeSkills.add(skillId);
    cognitiveState.metrics.activeSkills = cognitiveState.activeSkills.size;
    res.json({ status: 'activated' });
});
app.post('/api/cognitive/skills/:skillId/deactivate', function (req, res) {
    var skillId = req.params.skillId;
    cognitiveState.activeSkills.delete(skillId);
    cognitiveState.metrics.activeSkills = cognitiveState.activeSkills.size;
    res.json({ status: 'deactivated' });
});
app.post('/api/cognitive/learning/start', function (req, res) {
    console.log('Learning mode started');
    res.json({ status: 'learning_started' });
});
app.post('/api/cognitive/adaptation/save', function (req, res) {
    var _a = req.body, context = _a.context, activeSkills = _a.activeSkills, metrics = _a.metrics, timestamp = _a.timestamp;
    console.log('Adaptation saved:', { context: context, activeSkills: activeSkills, metrics: metrics, timestamp: timestamp });
    res.json({ status: 'adaptation_saved' });
});
app.post('/api/cognitive/hrm/init', function (req, res) {
    var _a = req.body, modelPath = _a.modelPath, config = _a.config;
    console.log('HRM Bridge initialized:', { modelPath: modelPath, config: config });
    res.json({
        status: 'initialized',
        modelPath: modelPath,
        config: config,
        bridgeId: "hrm_".concat(Date.now())
    });
});
// Terminal Command Endpoints
app.post('/api/terminal/execute', function (req, res) {
    var _a = req.body, command = _a.command, args = _a.args, context = _a.context;
    // Simulate command execution
    var output = '';
    var exitCode = 0;
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
            output = "KNIRV System Status:\n  Cognitive Engine: ".concat(cognitiveState.isRunning ? 'Running' : 'Stopped', "\n  Active Agents: ").concat(agents.size, "\n  Active Skills: ").concat(cognitiveState.activeSkills.size, "\n  Uptime: ").concat(Math.floor(process.uptime()), "s");
            break;
        default:
            output = '';
            exitCode = 127;
            break;
    }
    res.json({
        success: exitCode === 0,
        output: output,
        exitCode: exitCode,
        executionTime: Math.random() * 100 + 50
    });
});
// WebSocket for real-time updates
wss.on('connection', function (ws) {
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
    ws.on('message', function (message) {
        try {
            var data = JSON.parse(message.toString());
            console.log('WebSocket message received:', data);
            // Echo back for now
            ws.send(JSON.stringify({
                type: 'response',
                data: { received: data, timestamp: Date.now() }
            }));
        }
        catch (error) {
            console.error('WebSocket message error:', error);
        }
    });
    ws.on('close', function () {
        console.log('WebSocket client disconnected');
    });
});
// Error handling middleware
app.use(function (err, req, res, _next) {
    console.error('API Error:', err);
    res.status(500).json({
        error: 'Internal server error',
        message: err instanceof Error ? err.message : String(err),
        timestamp: new Date().toISOString()
    });
});
// 404 handler
app.use(function (req, res) {
    res.status(404).json({
        error: 'Not found',
        path: req.path,
        timestamp: new Date().toISOString()
    });
});
var PORT = process.env.PORT || 3001;
server.listen(PORT, function () {
    console.log("\uD83D\uDE80 KNIRV Controller API Server running on port ".concat(PORT));
    console.log("\uD83D\uDCE1 WebSocket server ready for real-time updates");
    console.log("\uD83D\uDD17 Health check: http://localhost:".concat(PORT, "/health"));
});
exports.default = app;
