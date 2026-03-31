"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.databaseService = exports.ChatSessions = exports.Skills = exports.Agents = void 0;
const nebuladb_1 = require("nebuladb");
const fullTextSearch_1 = require("nebuladb/dist/plugins/fullTextSearch");
const path_1 = require("path");
const fs_1 = require("fs");
const dbPath = path_1.default.resolve(process.cwd(), "data");
if (!fs_1.default.existsSync(dbPath)) {
    fs_1.default.mkdirSync(dbPath, { recursive: true });
}
const db = new nebuladb_1.NebulaDB({
    filePath: path_1.default.join(dbPath, "knirvcontroller.db"),
    // Enable automatic persistence for server-side usage
    autoload: true,
    autosave: true,
    autosaveInterval: 4000, // milliseconds
});
// Define Schemas for our core data models
const AgentSchema = db.defineSchema("Agent", {
    agentId: { type: "string", unique: true, required: true },
    name: { type: "string", required: true },
    version: { type: "string", required: true },
    baseModelId: { type: "string" },
    type: { type: "string", required: true }, // 'wasm' | 'lora' | 'hybrid'
    status: { type: "string", required: true }, // 'Available' | 'Deployed' | 'Error' | 'Compiling'
    nrnCost: { type: "number", required: true },
    capabilities: { type: "array", schema: "string" },
    metadata: {
        type: "object",
        schema: {
            name: "string",
            version: "string",
            description: "string",
            author: "string",
            capabilities: { type: "array", schema: "string" },
            requirements: {
                type: "object",
                schema: {
                    memory: "number",
                    cpu: "number",
                    storage: "number",
                },
            },
            permissions: { type: "array", schema: "string" },
        },
    },
    wasmModule: { type: "string" }, // Base64 encoded WASM module
    loraAdapter: { type: "string" }, // LoRA adapter path or identifier
    createdAt: { type: "date", default: () => new Date() },
    lastActivity: { type: "date" },
});
const SkillSchema = db.defineSchema("Skill", {
    skillId: { type: "string", unique: true, required: true },
    name: { type: "string", required: true },
    description: { type: "string" },
    // Storing LoRA adapter metadata
    loraAdapter: {
        type: "object",
        schema: {
            rank: "number",
            alpha: "number",
            weightsUri: "string", // URI to the weights file
        },
    },
    version: { type: "number", default: 1 },
    createdAt: { type: "date", default: () => new Date() },
    updatedAt: { type: "date" },
}).addPlugin(fullTextSearch_1.fullTextSearch, {
    fields: ["name", "description"] // Fields to index for searching
});
const ChatSessionSchema = db.defineSchema("ChatSession", {
    title: { type: "string", required: true },
    createdAt: { type: "date", default: () => new Date() },
    updatedAt: { type: "date" },
    messages: {
        type: "array",
        schema: {
            type: "object",
            schema: {
                id: "string",
                content: "string",
                sender: "string", // 'user' | 'ai'
                timestamp: "date",
            },
        },
    },
});
// Define and export collections
exports.Agents = db.defineCollection("agents", AgentSchema);
exports.Skills = db.defineCollection("skills", SkillSchema);
exports.ChatSessions = db.defineCollection("chat_sessions", ChatSessionSchema);
class DatabaseService {
    constructor() {
        // Expose collections for use in other services
        this.agents = exports.Agents;
        this.skills = exports.Skills;
        this.chatSessions = exports.ChatSessions;
        console.log("Database service initialized.");
    }
    static getInstance() {
        if (!DatabaseService.instance) {
            DatabaseService.instance = new DatabaseService();
        }
        return DatabaseService.instance;
    }
    // Helper methods for common operations
    async createAgent(agentData) {
        return await this.agents.insertOne(agentData);
    }
    async getAgent(agentId) {
        return await this.agents.findOne({ agentId });
    }
    async updateAgent(agentId, updateData) {
        return await this.agents.updateOne({ agentId }, updateData);
    }
    async deleteAgent(agentId) {
        return await this.agents.deleteOne({ agentId });
    }
    async listAgents() {
        return await this.agents.find({});
    }
    async createSkill(skillData) {
        return await this.skills.insertOne(skillData);
    }
    async getSkill(skillId) {
        return await this.skills.findOne({ skillId });
    }
    async updateSkill(skillId, updateData) {
        return await this.skills.updateOne({ skillId }, updateData);
    }
    async deleteSkill(skillId) {
        return await this.skills.deleteOne({ skillId });
    }
    async listSkills() {
        return await this.skills.find({});
    }
    async createChatSession(sessionData) {
        return await this.chatSessions.insertOne(sessionData);
    }
    async getChatSession(sessionId) {
        return await this.chatSessions.findOne({ _id: sessionId });
    }
    async updateChatSession(sessionId, updateData) {
        return await this.chatSessions.updateOne({ _id: sessionId }, updateData);
    }
    async deleteChatSession(sessionId) {
        return await this.chatSessions.deleteOne({ _id: sessionId });
    }
    async listChatSessions() {
        return await this.chatSessions.find({}, {
            sort: { updatedAt: -1 }
        });
    }
    // Search methods
    async searchSkills(term, limit = 10) {
        return await this.skills.search({
            term,
            limit,
            // Optional: Add fuzzy matching
            // tolerance: 1,
        });
    }
}
exports.databaseService = DatabaseService.getInstance();
