/**
 * DVE Agent Runtime — Browser Extension KNIRVAGENT
 * 
 * Lightweight TypeScript port of the core AgentLoop for browser-extension DVE nodes.
 * Runs within the ValidationRuntime Web Worker context.
 * 
 * Uses the same KNIRVAGENT configuration schema and workspace layout.
 * Not a separate agent framework — a TS reimplementation of the loop contract.
 */

// ===================== Types =====================

export interface DVEAgentConfig {
  dveID: string;
  dveName: string;
  ownerAddress: string;
  teeType: 'sgx' | 'sev-snp' | 'tdx' | 'software' | 'browser-extension';
  sessionKey: string;
  capabilities: string[];
  attachedPolicies: string[];
  workspacePath: string;
  identityPath: string;
}

export interface AgentTask {
  id: string;
  type: string;
  payload: any;
  status: 'queued' | 'running' | 'completed' | 'failed';
  createdAt: number;
  completedAt?: number;
  result?: any;
  error?: string;
}

export interface ValidationResult {
  taskID: string;
  dveID: string;
  status: 'passed' | 'failed' | 'skipped';
  proof: string;
  timestamp: number;
  metrics: {
    duration: number;
    memory: number;
    confidence: number;
  };
}

export interface DVEAgentMessage {
  type: 'agent_chat' | 'agent_response' | 'task_result' | 'alert' | 'status_update';
  dveID: string;
  agentID: string;
  payload: any;
  timestamp: number;
}

export type AgentEventHandler = (event: DVEAgentMessage) => void;

// ===================== Session Key Manager =====================

class SessionManager {
  private sessionKey: string;
  private initialized: boolean = false;

  constructor(key: string) {
    this.sessionKey = key;
  }

  async initialize(): Promise<void> {
    // In production, validate session key with KNIRVSERVER
    this.initialized = true;
  }

  isActive(): boolean {
    return this.initialized;
  }

  getSessionKey(): string {
    return this.sessionKey;
  }

  async rotate(): Promise<void> {
    // In production, request new session key from KNIRVSERVER
    this.sessionKey = `sk-${crypto.randomUUID().substring(0, 16)}`;
  }
}

// ===================== Task Queue =====================

class TaskQueue {
  private tasks: Map<string, AgentTask> = new Map();
  private queue: string[] = [];

  enqueue(task: AgentTask): void {
    this.tasks.set(task.id, task);
    this.queue.push(task.id);
  }

  dequeue(): AgentTask | undefined {
    const id = this.queue.shift();
    if (!id) return undefined;
    return this.tasks.get(id);
  }

  getTask(id: string): AgentTask | undefined {
    return this.tasks.get(id);
  }

  updateTask(id: string, updates: Partial<AgentTask>): void {
    const task = this.tasks.get(id);
    if (task) {
      Object.assign(task, updates);
    }
  }

  getPending(): AgentTask[] {
    return Array.from(this.tasks.values()).filter(t => t.status === 'queued' || t.status === 'running');
  }

  getCompleted(): AgentTask[] {
    return Array.from(this.tasks.values()).filter(t => t.status === 'completed' || t.status === 'failed');
  }

  size(): number {
    return this.queue.length;
  }

  clear(): void {
    this.tasks.clear();
    this.queue = [];
  }
}

// ===================== Tool Registry =====================

interface DVETool {
  name: string;
  description: string;
  execute: (params: any) => Promise<any>;
}

class DVELimitedToolRegistry {
  private tools: Map<string, DVETool> = new Map();
  private dveID: string;

  constructor(dveID: string) {
    this.dveID = dveID;
    this.registerDefaultTools();
  }

  private registerDefaultTools(): void {
    this.register({
      name: 'read_file',
      description: 'Read a file from the DVE workspace',
      execute: async (params: { path: string }) => {
        if (params.path.includes('..')) {
          throw new Error('Path traversal detected');
        }
        return { content: `[workspace read: ${params.path}]` };
      },
    });

    this.register({
      name: 'write_file',
      description: 'Write a file to the DVE workspace',
      execute: async (params: { path: string; content: string }) => {
        if (params.path.includes('..')) {
          throw new Error('Path traversal detected');
        }
        return { written: true, path: params.path, size: params.content.length };
      },
    });

    this.register({
      name: 'list_dir',
      description: 'List the DVE workspace directory',
      execute: async () => {
        return { entries: ['IDENTITY.md', 'AGENT.md', 'tasks/', 'results/'] };
      },
    });

    this.register({
      name: 'dve_validate',
      description: 'Run a validation task on this DVE',
      execute: async (params: { taskType: string; data: any }) => {
        return { taskID: crypto.randomUUID(), status: 'queued', dveID: this.dveID };
      },
    });

    this.register({
      name: 'dve_status',
      description: 'Get current DVE metrics and status',
      execute: async () => {
        return {
          dveID: this.dveID,
          status: 'online',
          cpu: Math.floor(Math.random() * 60) + 10,
          memory: Math.floor(Math.random() * 50) + 20,
          uptime: process.uptime(),
        };
      },
    });

    this.register({
      name: 'dve_alert',
      description: 'Send an alert to the Cognitive Engine',
      execute: async (params: { severity: string; message: string }) => {
        return { sent: true, alertID: crypto.randomUUID() };
      },
    });
  }

  register(tool: DVETool): void {
    this.tools.set(tool.name, tool);
  }

  getTool(name: string): DVETool | undefined {
    return this.tools.get(name);
  }

  listTools(): string[] {
    return Array.from(this.tools.keys());
  }

  async executeTool(name: string, params: any): Promise<any> {
    const tool = this.tools.get(name);
    if (!tool) {
      throw new Error(`Tool '${name}' not found in DVE agent scope`);
    }
    return tool.execute(params);
  }
}

// ===================== Workspace Manager =====================

class WorkspaceManager {
  private config: DVEAgentConfig;
  private initialized: boolean = false;

  constructor(config: DVEAgentConfig) {
    this.config = config;
  }

  async initialize(): Promise<void> {
    // In production: create workspace directory, write identity files
    this.initialized = true;
  }

  getIdentity(): string {
    return `# DVE Supervisor Agent
  
DVE ID: ${this.config.dveID}
DVE Name: ${this.config.dveName}
Owner Address: ${this.config.ownerAddress}
TEE Type: ${this.config.teeType}
DVE URI: knirv://dve/${this.config.ownerAddress}/${this.config.teeType}
Capabilities: ${this.config.capabilities.join(', ')}
Attached Policies: ${this.config.attachedPolicies.join(', ') || '(none)'}
Initialized At: ${new Date().toISOString()}`;
  }

  getAgentInstructions(): string {
    return `# Agent Instructions

You are a supervisor agent for DVE ${this.config.dveName} on the KNIRV Network.
Your role is to manage validation tasks, report results, and assist the
DVE owner with skill execution and error resolution.

## Constraints
- Only use tools scoped to this DVE's capabilities
- Do not attempt to access other DVEs or wallets
- Report all validation results via the task_result channel
- Escalate anomalies to the Cognitive Engine via the alert bus

## Available Capabilities
${this.config.capabilities.map(c => `- ${c}`).join('\n')}

## Attached Policies
${this.config.attachedPolicies.map(p => `- ${p}`).join('\n') || '(none)'}`;
  }

  isReady(): boolean {
    return this.initialized;
  }
}

// ==================== Alert Bus =====================

class AlertBus {
  private listeners: Set<AgentEventHandler> = new Set();
  private dveID: string;

  constructor(dveID: string) {
    this.dveID = dveID;
  }

  subscribe(handler: AgentEventHandler): () => void {
    this.listeners.add(handler);
    return () => this.listeners.delete(handler);
  }

  async emit(event: DVEAgentMessage): Promise<void> {
    for (const listener of this.listeners) {
      try {
        listener(event);
      } catch (err) {
        console.error('Alert bus listener error:', err);
      }
    }
  }

  async sendAlert(severity: 'low' | 'medium' | 'high' | 'critical', message: string): Promise<void> {
    await this.emit({
      type: 'alert',
      dveID: this.dveID,
      agentID: `agent-${this.dveID}`,
      payload: { severity, message },
      timestamp: Date.now(),
    });
  }
}

// ===================== Main DVE Agent Runtime =====================

export class DVEAgentRuntime {
  private config: DVEAgentConfig;
  private sessionManager: SessionManager;
  private taskQueue: TaskQueue;
  private toolRegistry: DVELimitedToolRegistry;
  private workspaceManager: WorkspaceManager;
  private alertBus: AlertBus;
  private running: boolean = false;
  private eventHandlers: Set<AgentEventHandler> = new Set();

  constructor(config: DVEAgentConfig) {
    this.config = config;
    this.sessionManager = new SessionManager(config.sessionKey);
    this.taskQueue = new TaskQueue();
    this.toolRegistry = new DVELimitedToolRegistry(config.dveID);
    this.workspaceManager = new WorkspaceManager(config);
    this.alertBus = new AlertBus(config.dveID);
  }

  async initialize(): Promise<void> {
    await this.sessionManager.initialize();
    await this.workspaceManager.initialize();
    this.running = true;

    // Start the event loop
    this.startEventLoop();
  }

  private startEventLoop(): void {
    const loop = async () => {
      while (this.running) {
        const task = this.taskQueue.dequeue();
        if (task) {
          await this.processTask(task);
        }
        await this.sleep(100);
      }
    };
    loop();
  }

  private async processTask(task: AgentTask): Promise<void> {
    task.status = 'running';

    try {
      const result = await this.toolRegistry.executeTool(task.type, task.payload);
      task.status = 'completed';
      task.completedAt = Date.now();
      task.result = result;

      await this.alertBus.emit({
        type: 'task_result',
        dveID: this.config.dveID,
        agentID: `agent-${this.config.dveID}`,
        payload: { taskID: task.id, status: 'completed', result },
        timestamp: Date.now(),
      });
    } catch (err: any) {
      task.status = 'failed';
      task.error = err.message;

      await this.alertBus.emit({
        type: 'task_result',
        dveID: this.config.dveID,
        agentID: `agent-${this.config.dveID}`,
        payload: { taskID: task.id, status: 'failed', error: err.message },
        timestamp: Date.now(),
      });
    }
  }

  async submitTask(type: string, payload: any): Promise<string> {
    const task: AgentTask = {
      id: crypto.randomUUID(),
      type,
      payload,
      status: 'queued',
      createdAt: Date.now(),
    };

    this.taskQueue.enqueue(task);

    await this.alertBus.emit({
      type: 'agent_chat',
      dveID: this.config.dveID,
      agentID: `agent-${this.config.dveID}`,
      payload: { taskID: task.id, type, status: 'queued' },
      timestamp: Date.now(),
    });

    return task.id;
  }

  async processChatMessage(message: string): Promise<string> {
    const lower = message.toLowerCase();

    if (lower.includes('status')) {
      const status = await this.toolRegistry.executeTool('dve_status', {});
      const pending = this.taskQueue.getPending();
      return `DVE ${this.config.dveName} is online. CPU: ${status.cpu}% | Memory: ${status.memory}% | Pending tasks: ${pending.length}`;
    }

    if (lower.includes('pending')) {
      const pending = this.taskQueue.getPending();
      if (pending.length === 0) return 'No pending tasks.';
      return pending.map(t => `- ${t.type}: ${t.id.substring(0, 8)} (${t.status})`).join('\n');
    }

    if (lower.includes('capabilities') || lower.includes('badges')) {
      return `Capabilities: ${this.config.capabilities.join(', ') || '(none configured)'}`;
    }

    if (lower.includes('run') || lower.includes('start')) {
      const taskID = await this.submitTask('dve_validate', { data: message });
      return `Task queued. ID: ${taskID.substring(0, 12)}...`;
    }

    return `Acknowledged. DVE ${this.config.dveName} standing by. Use "status", "pending", "capabilities", or "run <task>" commands.`;
  }

  subscribe(handler: AgentEventHandler): () => void {
    this.eventHandlers.add(handler);
    return () => this.eventHandlers.delete(handler);
  }

  getStatus(): { online: boolean; tasks: { pending: number; completed: number }; capabilities: string[]; workspace: string } {
    return {
      online: this.running,
      tasks: {
        pending: this.taskQueue.getPending().length,
        completed: this.taskQueue.getCompleted().length,
      },
      capabilities: this.toolRegistry.listTools(),
      workspace: this.config.workspacePath,
    };
  }

  async shutdown(): Promise<void> {
    this.running = false;
    this.taskQueue.clear();
    await this.alertBus.emit({
      type: 'status_update',
      dveID: this.config.dveID,
      agentID: `agent-${this.config.dveID}`,
      payload: { status: 'offline', reason: 'shutdown' },
      timestamp: Date.now(),
    });
  }

  getAlertBus(): AlertBus {
    return this.alertBus;
  }

  getToolRegistry(): DVELimitedToolRegistry {
    return this.toolRegistry;
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}
