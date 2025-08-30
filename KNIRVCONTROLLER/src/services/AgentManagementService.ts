/**
 * Agent Management Service
 * Handles agent upload, compilation, deployment, and lifecycle management
 */

import { databaseService } from '../core/services/databaseService';

export interface Agent {
  id: string;
  name: string;
  type: 'wasm' | 'lora' | 'hybrid';
  status: 'Available' | 'Deployed' | 'Error' | 'Compiling';
  nrnCost: number;
  capabilities: string[];
  metadata: AgentMetadata;
  wasmModule?: WebAssembly.Module;
  loraAdapter?: string;
  createdAt: Date;
  lastActivity?: Date;
}

export interface AgentMetadata {
  name: string;
  version: string;
  description: string;
  author: string;
  capabilities: string[];
  requirements: {
    memory: number;
    cpu: number;
    storage: number;
  };
  permissions: string[];
}

export interface AgentUploadRequest {
  file: File;
  metadata: Partial<AgentMetadata>;
  type: 'wasm' | 'lora' | 'hybrid';
}

export interface AgentDeploymentRequest {
  agentId: string;
  targetNRV?: string;
  configuration?: Record<string, unknown>;
  resources?: {
    memory?: number;
    cpu?: number;
    timeout?: number;
  };
}

export interface AgentExecutionResult {
  success: boolean;
  output?: unknown;
  error?: string;
  executionTime: number;
  resourceUsage: {
    memory: number;
    cpu: number;
  };
}

export class AgentManagementService {
  private wasmCompiler: WebAssembly.Module | null = null;
  private baseUrl: string;

  constructor(baseUrl: string = 'http://localhost:3001') {
    this.baseUrl = baseUrl;
    this.initializeWASMCompiler();
  }

  private async initializeWASMCompiler(): Promise<void> {
    try {
      // Initialize WASM compilation capabilities
      console.log('Initializing WASM compiler...');
      // This would load the WASM compiler module
    } catch (error) {
      console.error('Failed to initialize WASM compiler:', error);
    }
  }

  /**
   * Upload and compile a new agent
   */
  async uploadAgent(request: AgentUploadRequest): Promise<Agent> {
    try {
      const agentId = this.generateAgentId();
      
      // Create agent metadata
      const metadata: AgentMetadata = {
        name: request.metadata.name || request.file.name,
        version: request.metadata.version || '1.0.0',
        description: request.metadata.description || '',
        author: request.metadata.author || 'Unknown',
        capabilities: request.metadata.capabilities || [],
        requirements: request.metadata.requirements || {
          memory: 64,
          cpu: 1,
          storage: 10
        },
        permissions: request.metadata.permissions || []
      };

      // Create agent record
      const agentData = {
        agentId,
        name: metadata.name,
        type: request.type,
        status: 'Compiling',
        nrnCost: this.calculateNRNCost(metadata.requirements),
        capabilities: metadata.capabilities,
        metadata,
        createdAt: new Date()
      };

      // Save to database
      const agent = await databaseService.createAgent(agentData);

      // Process the uploaded file based on type
      if (request.type === 'wasm') {
        await this.compileWASMAgent(agent, request.file);
      } else if (request.type === 'lora') {
        await this.compileLoRAAgent(agent, request.file);
      } else if (request.type === 'hybrid') {
        await this.compileHybridAgent(agent, request.file);
      }

      // Update status to Available after successful compilation
      const updatedAgent = await databaseService.updateAgent(agentId, {
        status: 'Available',
        lastActivity: new Date()
      });

      return updatedAgent || agent;
    } catch (error) {
      console.error('Agent upload failed:', error);
      throw new Error(`Agent upload failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Deploy an agent to a target environment
   */
  async deployAgent(request: AgentDeploymentRequest): Promise<string> {
    const agent = await databaseService.getAgent(request.agentId);
    if (!agent) {
      throw new Error(`Agent ${request.agentId} not found`);
    }

    if (agent.status !== 'Available') {
      throw new Error(`Agent ${request.agentId} is not available for deployment`);
    }

    try {
      // Update agent status in database
      await databaseService.updateAgent(request.agentId, {
        status: 'Deployed',
        lastActivity: new Date()
      });

      // Send deployment request to backend
      const response = await fetch(`${this.baseUrl}/api/agents/deploy`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          agentId: request.agentId,
          targetNRV: request.targetNRV,
          configuration: request.configuration,
          resources: request.resources
        })
      });

      if (!response.ok) {
        throw new Error(`Deployment failed: ${response.statusText}`);
      }

      const result = await response.json();
      return result.deploymentId;
    } catch (error) {
      // Revert status on failure
      await databaseService.updateAgent(request.agentId, {
        status: 'Available'
      });
      throw error;
    }
  }

  /**
   * Execute a skill on a deployed agent
   */
  async executeSkill(agentId: string, skillId: string, parameters: Record<string, unknown>): Promise<AgentExecutionResult> {
    const agent = await databaseService.getAgent(agentId);
    if (!agent || agent.status !== 'Deployed') {
      throw new Error(`Agent ${agentId} is not deployed`);
    }

    const startTime = Date.now();

    try {
      const response = await fetch(`${this.baseUrl}/api/agents/${agentId}/execute`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          skillId,
          parameters
        })
      });

      if (!response.ok) {
        throw new Error(`Skill execution failed: ${response.statusText}`);
      }

      const result = await response.json();
      const executionTime = Date.now() - startTime;

      agent.lastActivity = new Date();

      return {
        success: true,
        output: result.output,
        executionTime,
        resourceUsage: result.resourceUsage || { memory: 0, cpu: 0 }
      };
    } catch (error) {
      const executionTime = Date.now() - startTime;
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error',
        executionTime,
        resourceUsage: { memory: 0, cpu: 0 }
      };
    }
  }

  /**
   * Get all available agents
   */
  async getAgents(): Promise<Agent[]> {
    return await databaseService.listAgents();
  }

  /**
   * Get deployed agents
   */
  async getDeployedAgents(): Promise<Agent[]> {
    const allAgents = await databaseService.listAgents();
    return allAgents.filter(agent => agent.status === 'Deployed');
  }

  /**
   * Get agent by ID
   */
  async getAgent(agentId: string): Promise<Agent | null> {
    return await databaseService.getAgent(agentId);
  }

  /**
   * Remove an agent
   */
  async removeAgent(agentId: string): Promise<void> {
    const agent = await databaseService.getAgent(agentId);
    if (!agent) {
      throw new Error(`Agent ${agentId} not found`);
    }

    // Undeploy if deployed
    if (agent.status === 'Deployed') {
      await this.undeployAgent(agentId);
    }

    // Remove from database
    await databaseService.deleteAgent(agentId);
  }

  /**
   * Undeploy an agent
   */
  async undeployAgent(agentId: string): Promise<void> {
    const agent = await databaseService.getAgent(agentId);
    if (!agent || agent.status !== 'Deployed') {
      throw new Error(`Agent ${agentId} is not deployed`);
    }

    try {
      await fetch(`${this.baseUrl}/api/agents/${agentId}/undeploy`, {
        method: 'POST'
      });

      await databaseService.updateAgent(agentId, {
        status: 'Available',
        lastActivity: new Date()
      });
    } catch (error) {
      console.error('Failed to undeploy agent:', error);
      throw error;
    }
  }

  private async compileWASMAgent(agent: Agent, file: File): Promise<void> {
    // Compile WASM agent from uploaded file
    const arrayBuffer = await file.arrayBuffer();
    const wasmModule = await WebAssembly.compile(arrayBuffer);
    agent.wasmModule = wasmModule;
  }

  private async compileLoRAAgent(agent: Agent, file: File): Promise<void> {
    // Process LoRA adapter file
    const text = await file.text();
    agent.loraAdapter = text;
  }

  private async compileHybridAgent(agent: Agent, file: File): Promise<void> {
    // Process hybrid agent (both WASM and LoRA components)
    await this.compileWASMAgent(agent, file);
    // Additional hybrid processing would go here
  }

  private generateAgentId(): string {
    return `agent_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private calculateNRNCost(requirements: AgentMetadata['requirements']): number {
    // Calculate NRN cost based on resource requirements
    const baseCost = 10;
    const memoryCost = requirements.memory * 0.1;
    const cpuCost = requirements.cpu * 5;
    const storageCost = requirements.storage * 0.05;
    
    return Math.ceil(baseCost + memoryCost + cpuCost + storageCost);
  }
}

// Export singleton instance
export const agentManagementService = new AgentManagementService();
