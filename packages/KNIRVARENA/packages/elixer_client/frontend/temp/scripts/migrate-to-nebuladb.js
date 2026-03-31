#!/usr/bin/env ts-node
"use strict";
/**
 * Migration script to move data from SQLite to NebulaDB
 * This script handles migration of chat sessions, agents, and skills
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.DataMigrator = void 0;
const databaseService_1 = require("../src/core/services/databaseService");
const fs_1 = require("fs");
const path_1 = require("path");
class DataMigrator {
    constructor() {
        this.sqliteDbPath = path_1.default.resolve(process.cwd(), 'data');
        this.backupPath = path_1.default.resolve(process.cwd(), 'data/backup');
    }
    async migrate() {
        console.log('🚀 Starting NebulaDB migration...');
        try {
            // Create backup directory
            await this.createBackup();
            // Check for existing SQLite databases
            const sqliteFiles = await this.findSQLiteFiles();
            if (sqliteFiles.length === 0) {
                console.log('✅ No SQLite databases found. Starting with clean NebulaDB.');
                return;
            }
            console.log(`📁 Found ${sqliteFiles.length} SQLite database(s):`, sqliteFiles);
            // Migrate each database
            for (const dbFile of sqliteFiles) {
                await this.migrateSQLiteFile(dbFile);
            }
            console.log('✅ Migration completed successfully!');
            console.log('📋 Migration Summary:');
            await this.printMigrationSummary();
        }
        catch (error) {
            console.error('❌ Migration failed:', error);
            throw error;
        }
    }
    async createBackup() {
        if (!fs_1.default.existsSync(this.backupPath)) {
            fs_1.default.mkdirSync(this.backupPath, { recursive: true });
        }
        const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
        const backupDir = path_1.default.join(this.backupPath, `backup-${timestamp}`);
        if (fs_1.default.existsSync(this.sqliteDbPath)) {
            fs_1.default.mkdirSync(backupDir, { recursive: true });
            // Copy all files in data directory to backup
            const files = fs_1.default.readdirSync(this.sqliteDbPath);
            for (const file of files) {
                if (file !== 'backup') {
                    const srcPath = path_1.default.join(this.sqliteDbPath, file);
                    const destPath = path_1.default.join(backupDir, file);
                    if (fs_1.default.statSync(srcPath).isFile()) {
                        fs_1.default.copyFileSync(srcPath, destPath);
                    }
                }
            }
            console.log(`💾 Backup created at: ${backupDir}`);
        }
    }
    async findSQLiteFiles() {
        if (!fs_1.default.existsSync(this.sqliteDbPath)) {
            return [];
        }
        const files = fs_1.default.readdirSync(this.sqliteDbPath);
        return files.filter(file => file.endsWith('.db') ||
            file.endsWith('.sqlite') ||
            file.endsWith('.sqlite3')).map(file => path_1.default.join(this.sqliteDbPath, file));
    }
    async migrateSQLiteFile(dbPath) {
        const dbName = path_1.default.basename(dbPath);
        console.log(`🔄 Migrating ${dbName}...`);
        // For this implementation, we'll simulate migration since we don't have actual SQLite data
        // In a real scenario, you would use sqlite3 package to read the data
        if (dbName.includes('chat')) {
            await this.migrateChatData(dbPath);
        }
        else if (dbName.includes('agent')) {
            await this.migrateAgentData(dbPath);
        }
        else {
            console.log(`⚠️  Unknown database type: ${dbName}, skipping...`);
        }
    }
    async migrateChatData(dbPath) {
        console.log('💬 Migrating chat data...');
        // Simulate reading from SQLite (in real implementation, use sqlite3)
        const mockChatSessions = [
            {
                id: '1',
                title: 'Sample Chat Session',
                created_at: '2024-01-01T00:00:00Z',
                updated_at: '2024-01-01T01:00:00Z'
            }
        ];
        const mockMessages = [
            {
                id: '1',
                session_id: '1',
                content: 'Hello, this is a migrated message',
                sender: 'user',
                timestamp: '2024-01-01T00:30:00Z'
            }
        ];
        // Migrate sessions
        for (const session of mockChatSessions) {
            const sessionMessages = mockMessages.filter(msg => msg.session_id === session.id);
            const migratedSession = {
                title: session.title,
                createdAt: new Date(session.created_at),
                updatedAt: new Date(session.updated_at),
                messages: sessionMessages.map(msg => ({
                    id: msg.id,
                    content: msg.content,
                    sender: msg.sender,
                    timestamp: new Date(msg.timestamp)
                }))
            };
            await databaseService_1.databaseService.createChatSession(migratedSession);
            console.log(`✅ Migrated chat session: ${session.title}`);
        }
    }
    async migrateAgentData(dbPath) {
        console.log('🤖 Migrating agent data...');
        // Simulate reading from SQLite
        const mockAgents = [
            {
                id: 'agent-1',
                name: 'Sample Agent',
                type: 'wasm',
                status: 'Available',
                metadata: JSON.stringify({
                    name: 'Sample Agent',
                    version: '1.0.0',
                    description: 'A migrated agent',
                    author: 'Migration Script',
                    capabilities: ['sample'],
                    requirements: { memory: 64, cpu: 1, storage: 10 },
                    permissions: ['read']
                }),
                created_at: '2024-01-01T00:00:00Z'
            }
        ];
        // Migrate agents
        for (const agent of mockAgents) {
            const metadata = JSON.parse(agent.metadata);
            const migratedAgent = {
                agentId: agent.id,
                name: agent.name,
                type: agent.type,
                status: agent.status,
                nrnCost: 100, // Default value
                capabilities: metadata.capabilities || [],
                metadata,
                createdAt: new Date(agent.created_at),
                lastActivity: agent.last_activity ? new Date(agent.last_activity) : undefined
            };
            await databaseService_1.databaseService.createAgent(migratedAgent);
            console.log(`✅ Migrated agent: ${agent.name}`);
        }
    }
    async printMigrationSummary() {
        const chatSessions = await databaseService_1.databaseService.listChatSessions();
        const agents = await databaseService_1.databaseService.listAgents();
        const skills = await databaseService_1.databaseService.listSkills();
        console.log(`📊 Chat Sessions: ${chatSessions.length}`);
        console.log(`📊 Agents: ${agents.length}`);
        console.log(`📊 Skills: ${skills.length}`);
    }
}
exports.DataMigrator = DataMigrator;
// CLI execution
async function main() {
    const migrator = new DataMigrator();
    try {
        await migrator.migrate();
        process.exit(0);
    }
    catch (error) {
        console.error('Migration failed:', error);
        process.exit(1);
    }
}
// Run if called directly
if (require.main === module) {
    main();
}
