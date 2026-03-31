/**
 * Agent-Core Main Module Template
 * Generated from Go template: main.go.template
 * Integrated with cognitive-shell capabilities
 */

// Agent Configuration
export const AGENT_CONFIG = {
  agentId: 'test-agent-real-wasm',
  agentName: 'Test Agent Real WASM',
  agentDescription: 'Test agent for real WASM compilation',
  agentVersion: '1.0.0',
  author: 'test',
  buildTarget: 'wasm',
  factsUrl: '{{factsUrl}}',
  privateFactsUrl: '{{privateFactsUrl}}',
  adaptiveRouterUrl: '{{adaptiveRouterUrl}}',
  ttl: {{ttl}},
  signature: '{{signature}}'
};

// Import cognitive capabilities
import { CognitiveEngine } from './CognitiveEngine';
import { AdaptiveLearningPipeline } from './AdaptiveLearningPipeline';
import { SEALFramework } from './SEALFramework';
import { LoRAAdapter } from './LoRAAdapter';
import { EventEmitter } from './EventEmitter';

// Import tool implementations
{{#each tools}}
import { {{name}} } from './tools/{{name}}';
{{/each}}

/**
 * Agent-Core Main Class
 * Integrates Go template functionality with cognitive-shell capabilities
 */
export class AgentCore extends EventEmitter {
  private cognitiveEngine: CognitiveEngine;
  private adaptiveLearning: AdaptiveLearningPipeline;
  private sealFramework: SEALFramework;
  private tools: Map<string, Function> = new Map();
  private memory: Map<string, any> = new Map();
  private isInitialized = false;

  constructor() {
    super();
    this.initializeComponents();
  }

  private async initializeComponents(): Promise<void> {
    // Initialize cognitive engine with agent configuration
    this.cognitiveEngine = new CognitiveEngine({
      maxContextSize: {{cognitiveConfig.maxContextSize}},
      learningRate: {{cognitiveConfig.learningRate}},
      adaptationThreshold: {{cognitiveConfig.adaptationThreshold}},
      skillTimeout: {{cognitiveConfig.skillTimeout}},
      voiceEnabled: {{cognitiveCapabilities.voice}},
      visualEnabled: {{cognitiveCapabilities.visual}},
      loraEnabled: {{cognitiveCapabilities.lora}},
      enhancedLoraEnabled: {{cognitiveCapabilities.enhancedLora}},
      hrmEnabled: false, // Disabled in agent-core
      wasmAgentsEnabled: false, // We ARE the WASM agent
      typeScriptCompilerEnabled: false, // Compilation happens at build time
      adaptiveLearningEnabled: {{cognitiveCapabilities.adaptiveLearning}},
      walletIntegrationEnabled: {{cognitiveCapabilities.wallet}},
      chainIntegrationEnabled: {{cognitiveCapabilities.chain}},
      ecosystemCommunicationEnabled: {{cognitiveCapabilities.ecosystem}}
    });

    // Initialize adaptive learning
    this.adaptiveLearning = new AdaptiveLearningPipeline();

    // Initialize SEAL framework
    this.sealFramework = new SEALFramework();

    // Register tools
    this.registerTools();

    await this.cognitiveEngine.initialize();
    this.isInitialized = true;
  }

  private registerTools(): void {
    {{#each tools}}
    this.tools.set('{{name}}', {{name}});
    {{/each}}
  }

  /**
   * Main execution method - called from sensory-shell
   */
  async execute(input: unknown, context: unknown = {}): Promise<any> {
    if (!this.isInitialized) {
      throw new Error('Agent-Core not initialized');
    }

    try {
      // Process through cognitive engine
      const result = await this.cognitiveEngine.processInput(input, context.inputType || 'text');
      
      // Apply adaptive learning
      await this.adaptiveLearning.learn(input, result, context);
      
      return result;
    } catch (error) {
      this.emit('execution_error', { error: error.message, input, context });
      throw error;
    }
  }

  /**
   * Tool execution method
   */
  async executeTool(toolName: string, parameters: unknown, context: unknown = {}): Promise<any> {
    const tool = this.tools.get(toolName);
    if (!tool) {
      throw new Error(`Tool '${toolName}' not found`);
    }

    try {
      return await tool(parameters, context);
    } catch (error) {
      this.emit('tool_error', { toolName, error: error.message, parameters });
      throw error;
    }
  }

  /**
   * Load LoRA adapter (for skill modification)
   */
  async loadLoRAAdapter(adapter: unknown): Promise<boolean> {
    try {
      // Apply LoRA adapter to cognitive engine
      return await this.cognitiveEngine.loadLoRAAdapterToWASMAgent(adapter);
    } catch (error) {
      this.emit('lora_error', { error: error.message, adapter });
      return false;
    }
  }

  /**
   * Get agent status
   */
  getStatus(): unknown {
    return {
      agentId: AGENT_CONFIG.agentId,
      agentName: AGENT_CONFIG.agentName,
      version: AGENT_CONFIG.agentVersion,
      initialized: this.isInitialized,
      cognitiveEngine: this.cognitiveEngine ? 'ready' : 'not_ready',
      availableTools: Array.from(this.tools.keys()),
      memorySize: this.memory.size
    };
  }

  /**
   * Cleanup resources
   */
  async dispose(): Promise<void> {
    if (this.cognitiveEngine) {
      await this.cognitiveEngine.dispose();
    }
    this.tools.clear();
    this.memory.clear();
    this.isInitialized = false;
  }
}

// Export for WASM integration
export const agentCore = new AgentCore();
export default agentCore;

{{#if (eq buildTarget "wasm")}}
// WASM export functions for sensory-shell communication
declare global {
  var agentCoreExecute: (input: string, context: string) => Promise<string>;
  var agentCoreExecuteTool: (toolName: string, parameters: string, context: string) => Promise<string>;
  var agentCoreLoadLoRA: (adapter: string) => Promise<boolean>;
  var agentCoreGetStatus: () => string;
}

// WASM interface functions
globalThis.agentCoreExecute = async (input: string, context: string = '{}'): Promise<string> => {
  try {
    const parsedInput = JSON.parse(input);
    const parsedContext = JSON.parse(context);
    const result = await agentCore.execute(parsedInput, parsedContext);
    return JSON.stringify(result);
  } catch (error) {
    return JSON.stringify({ error: error.message });
  }
};

globalThis.agentCoreExecuteTool = async (toolName: string, parameters: string, context: string = '{}'): Promise<string> => {
  try {
    const parsedParams = JSON.parse(parameters);
    const parsedContext = JSON.parse(context);
    const result = await agentCore.executeTool(toolName, parsedParams, parsedContext);
    return JSON.stringify(result);
  } catch (error) {
    return JSON.stringify({ error: error.message });
  }
};

globalThis.agentCoreLoadLoRA = async (adapter: string): Promise<boolean> => {
  try {
    const parsedAdapter = JSON.parse(adapter);
    return await agentCore.loadLoRAAdapter(parsedAdapter);
  } catch (error) {
    return false;
  }
};

globalThis.agentCoreGetStatus = (): string => {
  try {
    return JSON.stringify(agentCore.getStatus());
  } catch (error) {
    return JSON.stringify({ error: error.message });
  }
};
{{/if}}
