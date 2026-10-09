/**
 * Agent Management Service
 * Handles agent upload, compilation, deployment, and lifecycle management
 */

import { arenaStore } from '../storage/arenaStore';
import { Agent, AgentMetadata } from '../types/common';

// Re-export types for convenience
export type { Agent, AgentMetadata };

// Type conversion helper
function convertDbAgentToAgent(dbAgent: Record<string, unknown> & {
  agentId: string;
  name: string;
  version: string;
  baseModelId: string;
  type: string;
  status: string;
  nrnCost: number;
  capabilities: string[];
  metadata?: Partial<AgentMetadata>;
  wasmModule?: string;
  loraAdapter?: string;
  createdAt: string;
  lastActivity?: string;
}): Agent {
  // Ensure metadata conforms to AgentMetadata interface
  const dbMetadata = dbAgent.metadata as any;
  const metadata: AgentMetadata = {
    name: dbAgent.name,
    version: dbAgent.version,
    baseModelId: dbAgent.baseModelId,
    description: dbMetadata?.description || 'No description available',
    author: dbMetadata?.author || 'Unknown',
    capabilities: dbAgent.capabilities || [],
    requirements: {
      memory: dbMetadata?.requirements?.memory || 512,
      cpu: dbMetadata?.requirements?.cpu || 1,
      storage: dbMetadata?.requirements?.storage || 100
    },
    permissions: dbMetadata?.permissions || []
  };

  return {
    agentId: dbAgent.agentId,
    name: dbAgent.name,
    version: dbAgent.version,
    baseModelId: dbAgent.baseModelId || '',
    type: dbAgent.type as 'wasm' | 'lora' | 'hybrid',
    status: dbAgent.status as 'Available' | 'Deployed' | 'Error' | 'Compiling',
    nrnCost: dbAgent.nrnCost,
    capabilities: dbAgent.capabilities || [],
    metadata,
    wasmModule: dbAgent.wasmModule,
    loraAdapter: dbAgent.loraAdapter,
    createdAt: dbAgent.createdAt || new Date().toISOString(),
    lastActivity: dbAgent.lastActivity
  };
}

// Agent and AgentMetadata interfaces are now imported from types/common.ts

export interface AgentUploadRequest {
  file: File;
  metadata: Partial<AgentMetadata>;
  type: 'wasm' | 'lora' | 'hybrid';
}

export class AgentManagementService {
  private baseUrl: string;

  constructor(baseUrl: string = 'http://localhost:3001') {
    this.baseUrl = baseUrl;
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
        version: metadata.version,
        baseModelId: metadata.baseModelId || 'default',
        type: request.type as 'wasm' | 'lora' | 'hybrid',
        status: 'Compiling' as const,
        nrnCost: this.calculateNRNCost(metadata.requirements),
        capabilities: metadata.capabilities,
        metadata: metadata as AgentMetadata,
        createdAt: new Date().toISOString()
      };

      // Save to database
      const agent = await arenaStore.createAgent(agentData as Partial<Agent>);

      // Process the uploaded file based on type
      const convertedAgent = convertDbAgentToAgent(agent as any);
      if (request.type === 'wasm') {
        await this.compileWASMAgent(convertedAgent, request.file);
      } else if (request.type === 'lora') {
        await this.compileLoRAAgent(convertedAgent, request.file);
      } else if (request.type === 'hybrid') {
        await this.compileHybridAgent(convertedAgent, request.file);
      }

      // Update status to Available after successful compilation
      const updatedAgent = await arenaStore.updateAgent(agentId, {
        status: 'Available',
        lastActivity: new Date().toISOString()
      });

      return updatedAgent ? convertDbAgentToAgent(updatedAgent as any) : convertDbAgentToAgent(agent as any);
    } catch (error) {
      console.error('Agent upload failed:', error);
      throw new Error(`Agent upload failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Get all available agents
   */
  async getAgents(): Promise<Agent[]> {
    const dbAgents = await arenaStore.listAgents();
    return dbAgents.map(agent => convertDbAgentToAgent(agent as any));
  }

  /**
   * Get deployed agents
   */
  async getDeployedAgents(): Promise<Agent[]> {
    const allAgents = await arenaStore.listAgents();
    return allAgents
      .filter(agent => agent.status === 'Deployed')
      .map(agent => convertDbAgentToAgent(agent as any));
  }

  /**
   * Get agent by ID
   */
  async getAgent(agentId: string): Promise<Agent | null> {
    const dbAgent = await arenaStore.getAgent(agentId);
    return dbAgent ? convertDbAgentToAgent(dbAgent as any) : null;
  }

  /**
   * Remove an agent
   */
  async removeAgent(agentId: string): Promise<void> {
    const agent = await arenaStore.getAgent(agentId);
    if (!agent) {
      throw new Error(`Agent ${agentId} not found`);
    }

    // Deployment runs on the paired CLI (§2); a deployed agent is recalled
    // there before it can be removed.
    if (agent.status === 'Deployed') {
      throw new Error(`Agent ${agentId} is deployed; recall it from its error node first`);
    }

    // Remove from database
    await arenaStore.deleteAgent(agentId);
  }

  private async compileWASMAgent(agent: Agent, file: File): Promise<void> {
    // Compile WASM agent from uploaded file
    const arrayBuffer = await file.arrayBuffer();
    const wasmModule = await WebAssembly.compile(arrayBuffer);

    // Validate the compiled module has required exports
    const moduleExports = WebAssembly.Module.exports(wasmModule);
    const requiredExports = ['init', 'process', 'cleanup'];
    for (const required of requiredExports) {
      if (!moduleExports.some(exp => exp.name === required)) {
        throw new Error(`WASM module missing required export: ${required}`);
      }
    }

    // Store the compiled module as base64 string in database
    const wasmBytes = new Uint8Array(arrayBuffer);
    const wasmBase64 = btoa(String.fromCharCode.apply(null, Array.from(wasmBytes)));

    await arenaStore.updateAgent(agent.agentId, {
      wasmModule: wasmBase64
    });
  }

  private async compileLoRAAgent(agent: Agent, file: File): Promise<void> {
    // Process LoRA adapter file
    const text = await file.text();
    agent.loraAdapter = text;

    // Save LoRA adapter to database
    await arenaStore.updateAgent(agent.agentId, {
      loraAdapter: text
    });
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
