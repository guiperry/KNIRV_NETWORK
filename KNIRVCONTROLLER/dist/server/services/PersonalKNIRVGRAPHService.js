"use strict";
/**
 * Personal KNIRVGRAPH Service
 * Manages individual user's graph instance with error mapping and visualization
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
var __spreadArray = (this && this.__spreadArray) || function (to, from, pack) {
    if (pack || arguments.length === 2) for (var i = 0, l = from.length, ar; i < l; i++) {
        if (ar || !(i in from)) {
            if (!ar) ar = Array.prototype.slice.call(from, 0, i);
            ar[i] = from[i];
        }
    }
    return to.concat(ar || Array.prototype.slice.call(from));
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.personalKNIRVGRAPHService = exports.PersonalKNIRVGRAPHService = void 0;
var RxDBService_1 = require("./RxDBService");
var PersonalKNIRVGRAPHService = /** @class */ (function () {
    function PersonalKNIRVGRAPHService() {
        this.currentGraph = null;
        this.isInitialized = false;
        this.initialize();
    }
    PersonalKNIRVGRAPHService.prototype.initialize = function () {
        return __awaiter(this, void 0, void 0, function () {
            var error_1;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (this.isInitialized)
                            return [2 /*return*/];
                        _a.label = 1;
                    case 1:
                        _a.trys.push([1, 4, , 5]);
                        if (!!RxDBService_1.rxdbService.isDatabaseInitialized()) return [3 /*break*/, 3];
                        return [4 /*yield*/, RxDBService_1.rxdbService.initialize()];
                    case 2:
                        _a.sent();
                        _a.label = 3;
                    case 3:
                        this.isInitialized = true;
                        console.log('Personal KNIRVGRAPH service initialized');
                        return [3 /*break*/, 5];
                    case 4:
                        error_1 = _a.sent();
                        console.error('Failed to initialize Personal KNIRVGRAPH service:', error_1);
                        return [3 /*break*/, 5];
                    case 5: return [2 /*return*/];
                }
            });
        });
    };
    // Create a new personal graph
    PersonalKNIRVGRAPHService.prototype.createPersonalGraph = function (userId) {
        return __awaiter(this, void 0, void 0, function () {
            var graph;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        graph = {
                            id: "graph_".concat(userId, "_").concat(Date.now()),
                            userId: userId,
                            nodes: [],
                            edges: [],
                            metadata: {
                                createdAt: Date.now(),
                                lastModified: Date.now(),
                                version: 1,
                                complexity: 0
                            }
                        };
                        this.currentGraph = graph;
                        // Store in RxDB
                        return [4 /*yield*/, this.saveGraphToDatabase(graph)];
                    case 1:
                        // Store in RxDB
                        _a.sent();
                        return [2 /*return*/, graph];
                }
            });
        });
    };
    // Load user's personal graph
    PersonalKNIRVGRAPHService.prototype.loadPersonalGraph = function (userId) {
        return __awaiter(this, void 0, void 0, function () {
            var db, existing, parsedNodes, parsedEdges, graph_1, graph, error_2;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 5, , 6]);
                        if (!!this.isInitialized) return [3 /*break*/, 2];
                        return [4 /*yield*/, this.initialize()];
                    case 1:
                        _a.sent();
                        _a.label = 2;
                    case 2:
                        db = RxDBService_1.rxdbService.getDatabase();
                        return [4 /*yield*/, db.graphs.findOne({ selector: { userId: userId } }).exec()];
                    case 3:
                        existing = _a.sent();
                        if (existing) {
                            try {
                                parsedNodes = (existing.nodes || []);
                                parsedEdges = (existing.edges || []);
                                graph_1 = {
                                    id: existing.id,
                                    userId: existing.userId,
                                    nodes: parsedNodes,
                                    edges: parsedEdges,
                                    metadata: existing.metadata || {
                                        createdAt: Date.now(),
                                        lastModified: Date.now(),
                                        version: 1,
                                        complexity: 0
                                    }
                                };
                                this.currentGraph = graph_1;
                                return [2 /*return*/, graph_1];
                            }
                            catch (err) {
                                console.error('Failed parsing existing graph, creating new one:', err);
                            }
                        }
                        return [4 /*yield*/, this.createPersonalGraph(userId)];
                    case 4:
                        graph = _a.sent();
                        return [2 /*return*/, graph];
                    case 5:
                        error_2 = _a.sent();
                        console.error('Failed to load personal graph:', error_2);
                        return [2 /*return*/, null];
                    case 6: return [2 /*return*/];
                }
            });
        });
    };
    // Add error node to graph - Competitive process for SkillNode creation
    PersonalKNIRVGRAPHService.prototype.addErrorNode = function (errorData) {
        return __awaiter(this, void 0, void 0, function () {
            var node;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!this.currentGraph)
                            throw new Error('No active graph');
                        node = {
                            id: "error_".concat(errorData.errorId),
                            type: 'error',
                            label: errorData.description,
                            position: this.calculateNodePosition(),
                            data: __assign(__assign({}, errorData), { factualitySlice: errorData.factualitySlice }),
                            connections: []
                        };
                        this.currentGraph.nodes.push(node);
                        // Attempt to find related skills automatically
                        return [4 /*yield*/, this.findRelatedSkills(node)];
                    case 1:
                        // Attempt to find related skills automatically
                        _a.sent();
                        // Update graph
                        return [4 /*yield*/, this.updateGraph()];
                    case 2:
                        // Update graph
                        _a.sent();
                        return [2 /*return*/, node];
                }
            });
        });
    };
    // Add context node to graph - Creates CapabilityNodes from MCP server information
    PersonalKNIRVGRAPHService.prototype.addContextNode = function (contextData) {
        return __awaiter(this, void 0, void 0, function () {
            var capabilityData, node;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!this.currentGraph)
                            throw new Error('No active graph');
                        capabilityData = {
                            capabilityId: contextData.contextId,
                            capabilityName: contextData.contextName,
                            description: contextData.description,
                            mcpServerInfo: contextData.mcpServerInfo,
                            category: contextData.category,
                            timestamp: contextData.timestamp
                        };
                        node = {
                            id: "capability_".concat(contextData.contextId),
                            type: 'capability',
                            label: contextData.contextName,
                            position: this.calculateNodePosition(),
                            data: __assign(__assign({}, capabilityData), { capabilitySlice: contextData.capabilitySlice }),
                            connections: []
                        };
                        this.currentGraph.nodes.push(node);
                        // Find related capabilities and create connections
                        return [4 /*yield*/, this.findRelatedCapabilities(node)];
                    case 1:
                        // Find related capabilities and create connections
                        _a.sent();
                        // Update graph
                        return [4 /*yield*/, this.updateGraph()];
                    case 2:
                        // Update graph
                        _a.sent();
                        return [2 /*return*/, node];
                }
            });
        });
    };
    // Add idea node to graph - Collaborative process for PropertyNode creation
    PersonalKNIRVGRAPHService.prototype.addIdeaNode = function (ideaData) {
        return __awaiter(this, void 0, void 0, function () {
            var feasibilityReport, propertyData, node;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!this.currentGraph)
                            throw new Error('No active graph');
                        return [4 /*yield*/, this.generateFeasibilityReport(ideaData)];
                    case 1:
                        feasibilityReport = _a.sent();
                        propertyData = {
                            propertyId: ideaData.ideaId,
                            propertyName: ideaData.ideaName,
                            description: ideaData.description,
                            feasibilityReport: feasibilityReport,
                            collaborators: [this.currentGraph.userId], // Start with current user
                            timestamp: ideaData.timestamp
                        };
                        node = {
                            id: "property_".concat(ideaData.ideaId),
                            type: 'property',
                            label: ideaData.ideaName,
                            position: this.calculateNodePosition(),
                            data: __assign(__assign({}, propertyData), { feasibilitySlice: ideaData.feasibilitySlice }),
                            connections: []
                        };
                        this.currentGraph.nodes.push(node);
                        // Find potential collaborators and similar ideas
                        return [4 /*yield*/, this.findCollaborationOpportunities(node)];
                    case 2:
                        // Find potential collaborators and similar ideas
                        _a.sent();
                        // Update graph
                        return [4 /*yield*/, this.updateGraph()];
                    case 3:
                        // Update graph
                        _a.sent();
                        return [2 /*return*/, node];
                }
            });
        });
    };
    // Add skill node to graph
    PersonalKNIRVGRAPHService.prototype.addSkillNode = function (skillData) {
        return __awaiter(this, void 0, void 0, function () {
            var node;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!this.currentGraph)
                            throw new Error('No active graph');
                        node = {
                            id: "skill_".concat(skillData.skillId),
                            type: 'skill',
                            label: skillData.skillName,
                            position: this.calculateNodePosition(),
                            data: skillData,
                            connections: []
                        };
                        this.currentGraph.nodes.push(node);
                        return [4 /*yield*/, this.updateGraph()];
                    case 1:
                        _a.sent();
                        return [2 /*return*/, node];
                }
            });
        });
    };
    // Create connection between nodes
    PersonalKNIRVGRAPHService.prototype.createConnection = function (sourceId_1, targetId_1, type_1) {
        return __awaiter(this, arguments, void 0, function (sourceId, targetId, type, weight) {
            var edge, sourceNode, targetNode;
            if (weight === void 0) { weight = 1; }
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!this.currentGraph)
                            throw new Error('No active graph');
                        edge = {
                            id: "edge_".concat(sourceId, "_").concat(targetId),
                            source: sourceId,
                            target: targetId,
                            type: type,
                            weight: weight,
                            data: {
                                connectionType: type,
                                weight: weight
                            }
                        };
                        this.currentGraph.edges.push(edge);
                        sourceNode = this.currentGraph.nodes.find(function (n) { return n.id === sourceId; });
                        targetNode = this.currentGraph.nodes.find(function (n) { return n.id === targetId; });
                        if (sourceNode && !sourceNode.connections.includes(targetId)) {
                            sourceNode.connections.push(targetId);
                        }
                        if (targetNode && !targetNode.connections.includes(sourceId)) {
                            targetNode.connections.push(sourceId);
                        }
                        return [4 /*yield*/, this.updateGraph()];
                    case 1:
                        _a.sent();
                        return [2 /*return*/, edge];
                }
            });
        });
    };
    // Find skills related to an error
    PersonalKNIRVGRAPHService.prototype.findRelatedSkills = function (errorNode) {
        return __awaiter(this, void 0, void 0, function () {
            var errorText, skillMappings, _loop_1, this_1, _i, skillMappings_1, mapping;
            var _a, _b;
            return __generator(this, function (_c) {
                switch (_c.label) {
                    case 0:
                        errorText = errorNode.data.description.toLowerCase();
                        skillMappings = [
                            { pattern: /type.*error/i, skill: 'TypeScript' },
                            { pattern: /import.*error/i, skill: 'Module Management' },
                            { pattern: /network.*error/i, skill: 'Network Programming' },
                            { pattern: /async.*error/i, skill: 'Asynchronous Programming' }
                        ];
                        _loop_1 = function (mapping) {
                            var existingSkill, skillNode;
                            return __generator(this, function (_d) {
                                switch (_d.label) {
                                    case 0:
                                        if (!mapping.pattern.test(errorText)) return [3 /*break*/, 4];
                                        existingSkill = (_a = this_1.currentGraph) === null || _a === void 0 ? void 0 : _a.nodes.find(function (n) { return n.type === 'skill' && 'skillName' in n.data && n.data.skillName === mapping.skill; });
                                        if (!!existingSkill) return [3 /*break*/, 2];
                                        return [4 /*yield*/, this_1.addSkillNode({
                                                skillId: "skill_".concat(mapping.skill.toLowerCase().replace(/\s+/g, '_')),
                                                skillName: mapping.skill,
                                                description: "Skill related to ".concat(mapping.skill),
                                                category: 'programming',
                                                proficiency: 0.5
                                            })];
                                    case 1:
                                        _d.sent();
                                        _d.label = 2;
                                    case 2:
                                        skillNode = (_b = this_1.currentGraph) === null || _b === void 0 ? void 0 : _b.nodes.find(function (n) { return n.type === 'skill' && 'skillName' in n.data && n.data.skillName === mapping.skill; });
                                        if (!skillNode) return [3 /*break*/, 4];
                                        return [4 /*yield*/, this_1.createConnection(errorNode.id, skillNode.id, 'error_to_skill')];
                                    case 3:
                                        _d.sent();
                                        _d.label = 4;
                                    case 4: return [2 /*return*/];
                                }
                            });
                        };
                        this_1 = this;
                        _i = 0, skillMappings_1 = skillMappings;
                        _c.label = 1;
                    case 1:
                        if (!(_i < skillMappings_1.length)) return [3 /*break*/, 4];
                        mapping = skillMappings_1[_i];
                        return [5 /*yield**/, _loop_1(mapping)];
                    case 2:
                        _c.sent();
                        _c.label = 3;
                    case 3:
                        _i++;
                        return [3 /*break*/, 1];
                    case 4: return [2 /*return*/];
                }
            });
        });
    };
    // Helper method to find related capabilities for context nodes
    PersonalKNIRVGRAPHService.prototype.findRelatedCapabilities = function (capabilityNode) {
        return __awaiter(this, void 0, void 0, function () {
            var existingCapabilities, _i, existingCapabilities_1, existingCapability, similarity;
            var _a;
            return __generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        existingCapabilities = ((_a = this.currentGraph) === null || _a === void 0 ? void 0 : _a.nodes.filter(function (n) { return n.type === 'capability'; })) || [];
                        _i = 0, existingCapabilities_1 = existingCapabilities;
                        _b.label = 1;
                    case 1:
                        if (!(_i < existingCapabilities_1.length)) return [3 /*break*/, 4];
                        existingCapability = existingCapabilities_1[_i];
                        if (!('category' in existingCapability.data && 'category' in capabilityNode.data)) return [3 /*break*/, 3];
                        similarity = this.calculateSimilarity(existingCapability.data.category, capabilityNode.data.category);
                        if (!(similarity > 0.6)) return [3 /*break*/, 3];
                        return [4 /*yield*/, this.createConnection(capabilityNode.id, existingCapability.id, 'context_to_capability')];
                    case 2:
                        _b.sent();
                        _b.label = 3;
                    case 3:
                        _i++;
                        return [3 /*break*/, 1];
                    case 4: return [2 /*return*/];
                }
            });
        });
    };
    // Helper method to find collaboration opportunities for idea nodes
    PersonalKNIRVGRAPHService.prototype.findCollaborationOpportunities = function (propertyNode) {
        return __awaiter(this, void 0, void 0, function () {
            var existingProperties, _i, existingProperties_1, existingProperty, similarity;
            var _a;
            return __generator(this, function (_b) {
                switch (_b.label) {
                    case 0:
                        existingProperties = ((_a = this.currentGraph) === null || _a === void 0 ? void 0 : _a.nodes.filter(function (n) { return n.type === 'property'; })) || [];
                        _i = 0, existingProperties_1 = existingProperties;
                        _b.label = 1;
                    case 1:
                        if (!(_i < existingProperties_1.length)) return [3 /*break*/, 4];
                        existingProperty = existingProperties_1[_i];
                        if (!('description' in existingProperty.data && 'description' in propertyNode.data)) return [3 /*break*/, 3];
                        similarity = this.calculateSimilarity(existingProperty.data.description, propertyNode.data.description);
                        if (!(similarity > 0.4)) return [3 /*break*/, 3];
                        return [4 /*yield*/, this.createConnection(propertyNode.id, existingProperty.id, 'collaboration')];
                    case 2:
                        _b.sent();
                        _b.label = 3;
                    case 3:
                        _i++;
                        return [3 /*break*/, 1];
                    case 4: return [2 /*return*/];
                }
            });
        });
    };
    // Generate feasibility report for ideas
    PersonalKNIRVGRAPHService.prototype.generateFeasibilityReport = function (ideaData) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                // In a real implementation, this would query external APIs, databases, etc.
                // For now, return a mock feasibility report
                return [2 /*return*/, {
                        exists: Math.random() > 0.7, // 30% chance idea already exists
                        similarProjects: [
                            "Similar project 1 for ".concat(ideaData.ideaName),
                            "Related concept: ".concat(ideaData.ideaName, " variant")
                        ],
                        feasibilityScore: Math.random() * 100, // Random score 0-100
                        marketAnalysis: {
                            marketSize: Math.floor(Math.random() * 1000000),
                            competition: Math.floor(Math.random() * 50),
                            trendScore: Math.random() * 10
                        }
                    }];
            });
        });
    };
    // Calculate similarity between two strings (simple implementation)
    PersonalKNIRVGRAPHService.prototype.calculateSimilarity = function (str1, str2) {
        var words1 = str1.toLowerCase().split(/\s+/);
        var words2 = str2.toLowerCase().split(/\s+/);
        var intersection = words1.filter(function (word) { return words2.includes(word); });
        var union = __spreadArray([], new Set(__spreadArray(__spreadArray([], words1, true), words2, true)), true);
        return intersection.length / union.length;
    };
    // Calculate node position (simple algorithm)
    PersonalKNIRVGRAPHService.prototype.calculateNodePosition = function () {
        if (!this.currentGraph)
            return { x: 0, y: 0, z: 0 };
        var nodeCount = this.currentGraph.nodes.length;
        var angle = (nodeCount * 137.5) * (Math.PI / 180); // Golden angle
        var radius = Math.sqrt(nodeCount) * 50;
        return {
            x: Math.cos(angle) * radius,
            y: Math.sin(angle) * radius,
            z: nodeCount * 10
        };
    };
    // Update graph and save to database
    PersonalKNIRVGRAPHService.prototype.updateGraph = function () {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!this.currentGraph)
                            return [2 /*return*/];
                        this.currentGraph.metadata.lastModified = Date.now();
                        this.currentGraph.metadata.complexity = this.currentGraph.nodes.length + this.currentGraph.edges.length;
                        return [4 /*yield*/, this.saveGraphToDatabase(this.currentGraph)];
                    case 1:
                        _a.sent();
                        return [2 /*return*/];
                }
            });
        });
    };
    // Save graph to RxDB
    PersonalKNIRVGRAPHService.prototype.saveGraphToDatabase = function (graph) {
        return __awaiter(this, void 0, void 0, function () {
            var db, error_3;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        _a.trys.push([0, 2, , 3]);
                        db = RxDBService_1.rxdbService.getDatabase();
                        // Upsert graph into graphs collection
                        return [4 /*yield*/, db.graphs.upsert({
                                id: graph.id,
                                type: 'graph',
                                userId: graph.userId,
                                nodes: graph.nodes,
                                edges: graph.edges,
                                metadata: graph.metadata,
                                timestamp: Date.now()
                            })];
                    case 1:
                        // Upsert graph into graphs collection
                        _a.sent();
                        console.log('Graph saved to database (graphs collection)');
                        return [3 /*break*/, 3];
                    case 2:
                        error_3 = _a.sent();
                        console.error('Failed to save graph to database:', error_3);
                        return [3 /*break*/, 3];
                    case 3: return [2 /*return*/];
                }
            });
        });
    };
    // Get current graph
    PersonalKNIRVGRAPHService.prototype.getCurrentGraph = function () {
        return this.currentGraph;
    };
    // Export graph data for visualization
    PersonalKNIRVGRAPHService.prototype.exportGraphData = function () {
        if (!this.currentGraph)
            return null;
        return {
            nodes: this.currentGraph.nodes,
            edges: this.currentGraph.edges
        };
    };
    // Reset graph
    PersonalKNIRVGRAPHService.prototype.resetGraph = function () {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!this.currentGraph) return [3 /*break*/, 2];
                        this.currentGraph.nodes = [];
                        this.currentGraph.edges = [];
                        return [4 /*yield*/, this.updateGraph()];
                    case 1:
                        _a.sent();
                        _a.label = 2;
                    case 2: return [2 /*return*/];
                }
            });
        });
    };
    // Get graph statistics
    PersonalKNIRVGRAPHService.prototype.getGraphStats = function () {
        if (!this.currentGraph)
            return null;
        var nodeTypes = {};
        this.currentGraph.nodes.forEach(function (node) {
            nodeTypes[node.type] = (nodeTypes[node.type] || 0) + 1;
        });
        return {
            nodeCount: this.currentGraph.nodes.length,
            edgeCount: this.currentGraph.edges.length,
            complexity: this.currentGraph.metadata.complexity,
            nodeTypes: nodeTypes
        };
    };
    return PersonalKNIRVGRAPHService;
}());
exports.PersonalKNIRVGRAPHService = PersonalKNIRVGRAPHService;
// Export singleton instance
exports.personalKNIRVGRAPHService = new PersonalKNIRVGRAPHService();
