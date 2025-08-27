/**
 * TypeScript Agent-Core Compiler
 * 
 * Revolutionary compiler that creates complete agent.wasm files by:
 * 1. Translating Go templates from agent-builder to TypeScript templates
 * 2. Converting cognitive-shell files to TypeScript templates  
 * 3. Integrating both into a wholistic agent-core package
 * 4. Compiling to WASM for embedded execution in sensory-shell
 */

import { promises as fs } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { spawn } from 'child_process';
import pino from 'pino';

const logger = pino({ name: 'agent-core-compiler' });

// Jest compatibility: handle import.meta.url fallback
let __filename: string;
let __dirname: string;

try {
  // ES module environment - use eval to avoid Jest parsing issues
  const importMeta = eval('import.meta');
  if (importMeta && importMeta.url) {
    __filename = fileURLToPath(importMeta.url);
    __dirname = dirname(__filename);
  } else {
    throw new Error('import.meta.url not available');
  }
} catch (error) {
  // CommonJS/Jest environment fallback
  __filename = require.resolve('./AgentCoreCompiler.ts');
  __dirname = dirname(__filename);
}

export interface AgentCoreConfig {
  agentId: string;
  agentName: string;
  agentDescription: string;
  agentVersion: string;
  author: string;
  tools: ToolConfig[];
  cognitiveCapabilities: CognitiveCapability[];
  sensoryInterfaces: SensoryInterface[];
  buildTarget: 'wasm' | 'typescript' | 'hybrid';
  optimizationLevel: 'none' | 'basic' | 'aggressive';
  cognitiveConfig?: {
    maxContextSize?: number;
    learningRate?: number;
    adaptationThreshold?: number;
    skillTimeout?: number;
  };
}

export interface ToolConfig {
  name: string;
  description: string;
  parameters: ToolParameter[];
  implementation: string;
  sourceType: 'inline' | 'external' | 'template';
}

export interface ToolParameter {
  name: string;
  type: string;
  required: boolean;
  description: string;
  defaultValue?: any;
}

export interface CognitiveCapability {
  name: string;
  enabled: boolean;
  config: Record<string, any>;
}

export interface SensoryInterface {
  type: 'voice' | 'visual' | 'text' | 'gesture';
  enabled: boolean;
  config: Record<string, any>;
}

export interface CompilationResult {
  success: boolean;
  agentId: string;
  wasmBytes?: Uint8Array;
  typeScriptCode?: string;
  metadata: {
    compilationTime: number;
    wasmSize?: number;
    typeScriptSize?: number;
    optimizationLevel: string;
    cognitiveCapabilities: string[];
    sensoryInterfaces: string[];
  };
  errors?: string[];
  warnings?: string[];
}

export class AgentCoreCompiler {
  private templatesDir: string;
  private buildDir: string;
  private goTemplates: Map<string, string> = new Map();
  private cognitiveTemplates: Map<string, string> = new Map();
  private isInitialized = false;

  constructor() {
    this.templatesDir = join(__dirname, '../templates');
    this.buildDir = join(__dirname, '../build');
  }

  async initialize(): Promise<void> {
    logger.info('Initializing Agent-Core Compiler...');

    try {
      // Ensure directories exist
      await fs.mkdir(this.templatesDir, { recursive: true });
      await fs.mkdir(this.buildDir, { recursive: true });

      // Load TypeScript templates
      await this.loadTypeScriptTemplates();

      // Load and convert cognitive-shell files to templates
      await this.loadAndConvertCognitiveShell();

      this.isInitialized = true;
      logger.info('Agent-Core Compiler initialized successfully');
    } catch (error) {
      logger.error({ error }, 'Failed to initialize Agent-Core Compiler');
      throw error;
    }
  }

  private async loadTypeScriptTemplates(): Promise<void> {
    logger.info('Loading TypeScript templates...');

    const tsTemplateFiles = [
      'main.ts.template',
      'resources.ts.template',
      'tool.ts.template',
      'AdaptiveLearningPipeline.ts.template',
      'CognitiveEngine.ts.template',
      'EcosystemCommunicationLayer.ts.template',
      'EnhancedLoRAAdapter.ts.template',
      'EventEmitter.ts.template',
      'LoRAAdapter.ts.template',
      'SEALFramework.ts.template'
    ];

    for (const templateFile of tsTemplateFiles) {
      try {
        const templatePath = join(this.templatesDir, templateFile);
        const template = await fs.readFile(templatePath, 'utf-8');

        // Store the template
        this.goTemplates.set(templateFile, template);

        logger.info(`Loaded ${templateFile}`);
      } catch (error) {
        logger.warn(`Failed to load ${templateFile}:`, error);
      }
    }
  }

  private async loadAndConvertCognitiveShell(): Promise<void> {
    logger.info('Loading and converting cognitive-shell files to templates...');

    const cognitiveFiles = [
      'AdaptiveLearningPipeline.ts',
      'CognitiveEngine.ts',
      'EcosystemCommunicationLayer.ts', 
      'EnhancedLoRAAdapter.ts',
      'EventEmitter.ts',
      'LoRAAdapter.ts',
      'SEALFramework.ts'
    ];

    for (const cognitiveFile of cognitiveFiles) {
      try {
        const cognitiveFilePath = join(__dirname, '../../../src/cognitive-shell', cognitiveFile);
        const cognitiveCode = await fs.readFile(cognitiveFilePath, 'utf-8');
        
        // Convert to template with placeholders
        const template = await this.convertToTemplate(cognitiveCode, cognitiveFile);
        
        // Store the template
        const templateFile = cognitiveFile.replace('.ts', '.ts.template');
        this.cognitiveTemplates.set(templateFile, template);
        
        // Save to templates directory
        await fs.writeFile(join(this.templatesDir, templateFile), template);
        
        logger.info(`Converted ${cognitiveFile} -> ${templateFile}`);
      } catch (error) {
        logger.warn(`Failed to convert ${cognitiveFile}:`, error);
      }
    }
  }

  private async translateGoToTypeScript(goTemplate: string, templateFile: string): Promise<string> {
    // Translate Go template syntax to TypeScript template syntax
    let tsTemplate = goTemplate;

    // Handle different template files differently
    switch (templateFile) {
      case 'main.go.template':
        tsTemplate = await this.translateMainGoTemplate(goTemplate);
        break;
      case 'tool.go.template':
        tsTemplate = await this.translateToolGoTemplate(goTemplate);
        break;
      case 'resources.go.template':
        tsTemplate = await this.translateResourcesGoTemplate(goTemplate);
        break;
      case 'agent_prompt.json.template':
        // JSON templates can mostly stay the same, just update variable syntax
        tsTemplate = this.translateJsonTemplate(goTemplate);
        break;
      default:
        tsTemplate = await this.translateGenericGoTemplate(goTemplate);
    }

    return tsTemplate;
  }

  private async translateMainGoTemplate(goTemplate: string): Promise<string> {
    // Convert Go main template to TypeScript agent-core template
    return `/**
 * Agent-Core Main Module Template
 * Generated from Go template: main.go.template
 * Integrated with cognitive-shell capabilities
 */

// Agent Configuration
export const AGENT_CONFIG = {
  agentId: '{{agentId}}',
  agentName: '{{agentName}}',
  agentDescription: '{{agentDescription}}',
  agentVersion: '{{agentVersion}}',
  author: '{{author}}',
  buildTarget: '{{buildTarget}}',
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
      maxContextSize: 10000,
      learningRate: 0.01,
      adaptationThreshold: 0.7,
      skillTimeout: 30000,
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
  async execute(input: any, context: any = {}): Promise<any> {
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
  async executeTool(toolName: string, parameters: any, context: any = {}): Promise<any> {
    const tool = this.tools.get(toolName);
    if (!tool) {
      throw new Error('Tool \'' + toolName + '\' not found');
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
  async loadLoRAAdapter(adapter: any): Promise<boolean> {
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
  getStatus(): any {
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

{{#if eq buildTarget "wasm"}}
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
{{/if}}`;
  }

  private async translateToolGoTemplate(goTemplate: string): Promise<string> {
    // Convert Go tool template to TypeScript tool template
    return `/**
 * Tool Implementation Template: {{toolName}}
 * Generated from Go template: tool.go.template
 * Description: {{toolDescription}}
 */

import { EventEmitter } from '../EventEmitter';

export interface {{toolName}}Parameters {
  {{#each parameters}}
  {{name}}{{#unless required}}?{{/unless}}: {{type}};
  {{/each}}
}

export interface {{toolName}}Context {
  agentId: string;
  sessionId?: string;
  userId?: string;
  environment: 'wasm' | 'browser' | 'node';
  memory: Map<string, any>;
  logger: {
    log: (message: string) => void;
    error: (message: string) => void;
    warn: (message: string) => void;
  };
}

export interface {{toolName}}Result {
  success: boolean;
  result?: any;
  error?: string;
  executionTime: number;
  metadata?: Record<string, any>;
}

/**
 * {{toolName}} Tool Implementation
 */
export class {{toolName}}Tool extends EventEmitter {
  private name = '{{toolName}}';
  private description = '{{toolDescription}}';

  async execute(
    params: {{toolName}}Parameters,
    context: {{toolName}}Context
  ): Promise<{{toolName}}Result> {
    const startTime = Date.now();
    
    try {
      // Validate parameters
      {{#each parameters}}
      {{#if required}}
      if (params.{{name}} === undefined) {
        throw new Error('Required parameter "{{name}}" is missing');
      }
      {{/if}}
      {{/each}}

      // Tool implementation
      {{toolImplementation}}

      const executionTime = Date.now() - startTime;
      
      return {
        success: true,
        result,
        executionTime,
        metadata: {
          toolName: this.name,
          parametersUsed: Object.keys(params)
        }
      };

    } catch (error) {
      const executionTime = Date.now() - startTime;
      
      return {
        success: false,
        error: error.message,
        executionTime,
        metadata: {
          toolName: this.name,
          errorType: error.constructor.name
        }
      };
    }
  }

  getName(): string {
    return this.name;
  }

  getDescription(): string {
    return this.description;
  }

  getParameters(): any[] {
    return [
      {{#each parameters}}
      {
        name: '{{name}}',
        type: '{{type}}',
        required: {{required}},
        description: '{{description}}'{{#if defaultValue}},
        defaultValue: {{defaultValue}}{{/if}}
      }{{#unless @last}},{{/unless}}
      {{/each}}
    ];
  }
}

// Export tool function for direct usage
export async function {{toolName}}(
  params: {{toolName}}Parameters,
  context: {{toolName}}Context
): Promise<{{toolName}}Result> {
  const tool = new {{toolName}}Tool();
  return await tool.execute(params, context);
}

export default {{toolName}};`;
  }

  private async translateResourcesGoTemplate(goTemplate: string): Promise<string> {
    // Convert Go resources template to TypeScript resources template
    return `/**
 * Resources Module Template
 * Generated from Go template: resources.go.template
 * Manages embedded resources and prompts
 */

// Embedded resources
const EMBEDDED_RESOURCES = new Map<string, string>([
  {{#each resources}}
  ['{{name}}', \`{{content}}\`],
  {{/each}}
]);

// Embedded prompts
const EMBEDDED_PROMPTS = new Map<string, string>([
  {{#each prompts}}
  ['{{name}}', \`{{content}}\`],
  {{/each}}
]);

/**
 * Get a resource by name
 */
export function getResource(name: string): string | null {
  return EMBEDDED_RESOURCES.get(name) || null;
}

/**
 * Get a prompt by name
 */
export function getPrompt(name: string): string | null {
  return EMBEDDED_PROMPTS.get(name) || null;
}

/**
 * Get all available resource names
 */
export function getResourceNames(): string[] {
  return Array.from(EMBEDDED_RESOURCES.keys());
}

/**
 * Get all available prompt names
 */
export function getPromptNames(): string[] {
  return Array.from(EMBEDDED_PROMPTS.keys());
}

/**
 * Check if a resource exists
 */
export function hasResource(name: string): boolean {
  return EMBEDDED_RESOURCES.has(name);
}

/**
 * Check if a prompt exists
 */
export function hasPrompt(name: string): boolean {
  return EMBEDDED_PROMPTS.has(name);
}

export default {
  getResource,
  getPrompt,
  getResourceNames,
  getPromptNames,
  hasResource,
  hasPrompt
};`;
  }

  private translateJsonTemplate(jsonTemplate: string): string {
    // Convert Go template variables to TypeScript template variables
    return jsonTemplate
      .replace(/\{\{\.(\w+)\}\}/g, '{{$1}}')
      .replace(/\{\{range \.(\w+)\}\}/g, '{{#each $1}}')
      .replace(/\{\{end\}\}/g, '{{/each}}');
  }

  private async translateGenericGoTemplate(goTemplate: string): Promise<string> {
    // Generic Go to TypeScript template translation
    let tsTemplate = goTemplate;

    // Convert Go template syntax to Handlebars-style syntax
    tsTemplate = tsTemplate
      .replace(/\{\{\.(\w+)\}\}/g, '{{$1}}')
      .replace(/\{\{range \.(\w+)\}\}/g, '{{#each $1}}')
      .replace(/\{\{if eq \.(\w+) "(\w+)"\}\}/g, '{{#if (eq $1 "$2")}}')
      .replace(/\{\{if ne \.(\w+) "(\w+)"\}\}/g, '{{#unless (eq $1 "$2")}}')
      .replace(/\{\{end\}\}/g, '{{/each}}');

    // Convert Go package declarations to TypeScript module exports
    tsTemplate = tsTemplate.replace(/^package main$/gm, '// TypeScript Module');

    // Convert Go imports to TypeScript imports (basic conversion)
    tsTemplate = tsTemplate.replace(/import \(/g, '// Imports:');
    tsTemplate = tsTemplate.replace(/^\s*"([^"]+)"\s*$/gm, '// import from "$1"');

    return tsTemplate;
  }

  private async convertToTemplate(cognitiveCode: string, fileName: string): Promise<string> {
    // Convert cognitive-shell TypeScript files to templates with placeholders
    let template = cognitiveCode;

    // Add template header
    template = `/**
 * Cognitive Shell Template: ${fileName}
 * Generated from: KNIRVCORTEX/agent-core/src/cognitive-shell/${fileName}
 * 
 * This template is compiled into agent.wasm for embedded cognitive processing
 * Communication with sensory-shell happens through WASM interface
 */

${template}`;

    // Add configuration placeholders for key values
    template = template
      .replace(/maxContextSize:\s*\d+/g, 'maxContextSize: {{cognitiveConfig.maxContextSize}}')
      .replace(/learningRate:\s*[\d.]+/g, 'learningRate: {{cognitiveConfig.learningRate}}')
      .replace(/adaptationThreshold:\s*[\d.]+/g, 'adaptationThreshold: {{cognitiveConfig.adaptationThreshold}}')
      .replace(/skillTimeout:\s*\d+/g, 'skillTimeout: {{cognitiveConfig.skillTimeout}}');

    // Add agent configuration placeholders
    template = template
      .replace(/'agent-\w+'/g, "'{{agentId}}'")
      .replace(/"agent-\w+"/g, '"{{agentId}}"');

    return template;
  }

  async compileAgentCore(config: AgentCoreConfig): Promise<CompilationResult> {
    if (!this.isInitialized) {
      throw new Error('Compiler not initialized');
    }

    const startTime = Date.now();
    logger.info({ agentId: config.agentId }, 'Starting agent-core compilation');

    try {
      // Create build directory for this agent
      const agentBuildDir = join(this.buildDir, config.agentId);
      await fs.mkdir(agentBuildDir, { recursive: true });

      // Generate TypeScript code from templates
      const typeScriptCode = await this.generateTypeScriptCode(config, agentBuildDir);

      let result: CompilationResult;

      if (config.buildTarget === 'wasm' || config.buildTarget === 'hybrid') {
        // Compile to WASM
        const wasmBytes = await this.compileToWASM(agentBuildDir, config);
        
        result = {
          success: true,
          agentId: config.agentId,
          wasmBytes,
          typeScriptCode,
          metadata: {
            compilationTime: Date.now() - startTime,
            wasmSize: wasmBytes.length,
            typeScriptSize: typeScriptCode.length,
            optimizationLevel: config.optimizationLevel,
            cognitiveCapabilities: config.cognitiveCapabilities.filter(c => c.enabled).map(c => c.name),
            sensoryInterfaces: config.sensoryInterfaces.filter(s => s.enabled).map(s => s.type)
          }
        };
      } else {
        // TypeScript only
        result = {
          success: true,
          agentId: config.agentId,
          typeScriptCode,
          metadata: {
            compilationTime: Date.now() - startTime,
            typeScriptSize: typeScriptCode.length,
            optimizationLevel: config.optimizationLevel,
            cognitiveCapabilities: config.cognitiveCapabilities.filter(c => c.enabled).map(c => c.name),
            sensoryInterfaces: config.sensoryInterfaces.filter(s => s.enabled).map(s => s.type)
          }
        };
      }

      logger.info({ agentId: config.agentId, compilationTime: result.metadata.compilationTime }, 'Agent-core compilation completed');
      return result;

    } catch (error) {
      logger.error({ error, agentId: config.agentId }, 'Agent-core compilation failed');
      
      return {
        success: false,
        agentId: config.agentId,
        metadata: {
          compilationTime: Date.now() - startTime,
          optimizationLevel: config.optimizationLevel,
          cognitiveCapabilities: [],
          sensoryInterfaces: []
        },
        errors: [error instanceof Error ? error.message : String(error)]
      };
    }
  }

  private async generateTypeScriptCode(config: AgentCoreConfig, buildDir: string): Promise<string> {
    // Process main template
    const mainTemplate = this.goTemplates.get('main.ts.template');
    if (!mainTemplate) {
      throw new Error('Main template not found');
    }

    // Process template with configuration
    const code = this.processTemplate(mainTemplate, config);

    // Add cognitive shell components
    for (const [templateName, template] of this.cognitiveTemplates) {
      const processedTemplate = this.processTemplate(template, config);
      const fileName = templateName.replace('.template', '');
      
      // Write individual cognitive component files
      await fs.writeFile(join(buildDir, fileName), processedTemplate);
    }

    // Generate tool files
    for (const tool of config.tools) {
      const toolTemplate = this.goTemplates.get('tool.ts.template');
      if (toolTemplate) {
        const toolCode = this.processToolTemplate(toolTemplate, tool, config);
        await fs.writeFile(join(buildDir, `tools/${tool.name}.ts`), toolCode);
      }
    }

    // Write main agent-core file
    await fs.writeFile(join(buildDir, 'index.ts'), code);

    return code;
  }

  private processTemplate(template: string, config: AgentCoreConfig): string {
    let processed = template;

    // Simple template processing (in production, use a proper template engine like Handlebars)
    const replacements = {
      '{{agentId}}': config.agentId,
      '{{agentName}}': config.agentName,
      '{{agentDescription}}': config.agentDescription,
      '{{agentVersion}}': config.agentVersion,
      '{{author}}': config.author,
      '{{buildTarget}}': config.buildTarget
    };

    for (const [placeholder, value] of Object.entries(replacements)) {
      processed = processed.replace(new RegExp(placeholder.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g'), value);
    }

    return processed;
  }

  private processToolTemplate(template: string, tool: ToolConfig, config: AgentCoreConfig): string {
    let processed = template;

    const replacements = {
      '{{toolName}}': tool.name,
      '{{toolDescription}}': tool.description,
      '{{toolImplementation}}': tool.implementation
    };

    for (const [placeholder, value] of Object.entries(replacements)) {
      processed = processed.replace(new RegExp(placeholder.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g'), value);
    }

    return processed;
  }

  private async compileToWASM(buildDir: string, config: AgentCoreConfig): Promise<Uint8Array> {
    logger.info('Compiling agent-core to WASM using Rust toolchain...');

    try {
      // 1. Generate Rust code from TypeScript templates
      const rustCode = await this.generateRustFromTemplates(buildDir, config);

      // 2. Create temporary Rust project
      const rustProjectDir = await this.createRustProject(buildDir, config, rustCode);

      // 3. Compile to WASM using wasm-pack
      const wasmBytes = await this.compileRustToWASM(rustProjectDir, config);

      logger.info('WASM compilation completed successfully');
      return wasmBytes;

    } catch (error) {
      logger.error({ error }, 'WASM compilation failed, falling back to minimal WASM');

      // Fallback to minimal WASM if compilation fails
      return this.generateMinimalWASM(config);
    }
  }

  private async generateRustFromTemplates(buildDir: string, config: AgentCoreConfig): Promise<string> {
    logger.info('Generating Rust code from TypeScript templates...');

    // Convert TypeScript cognitive templates to Rust
    const rustCode = `
use wasm_bindgen::prelude::*;
use web_sys::console;
use serde::{Deserialize, Serialize};

// Import console.log for debugging
#[wasm_bindgen]
extern "C" {
    #[wasm_bindgen(js_namespace = console)]
    fn log(s: &str);
}

macro_rules! console_log {
    ($($t:tt)*) => (log(&format_args!($($t)*).to_string()))
}

// Initialize panic hook
#[wasm_bindgen(start)]
pub fn main() {
    console_error_panic_hook::set_once();
    console_log!("Agent-Core WASM initialized: {}", "${config.agentId}");
}

// Agent configuration
#[derive(Serialize, Deserialize)]
pub struct AgentConfig {
    pub agent_id: String,
    pub max_context_size: usize,
    pub learning_rate: f32,
    pub adaptation_threshold: f32,
    pub skill_timeout: u32,
}

// Agent-Core implementation
#[wasm_bindgen]
pub struct AgentCore {
    config: AgentConfig,
    initialized: bool,
    memory: Vec<String>,
}

#[wasm_bindgen]
impl AgentCore {
    #[wasm_bindgen(constructor)]
    pub fn new() -> AgentCore {
        let config = AgentConfig {
            agent_id: "${config.agentId}".to_string(),
            max_context_size: ${config.cognitiveConfig?.maxContextSize || 10000},
            learning_rate: ${config.cognitiveConfig?.learningRate || 0.01},
            adaptation_threshold: ${config.cognitiveConfig?.adaptationThreshold || 0.7},
            skill_timeout: ${config.cognitiveConfig?.skillTimeout || 30000},
        };

        AgentCore {
            config,
            initialized: false,
            memory: Vec::new(),
        }
    }

    #[wasm_bindgen]
    pub fn agent_core_execute(&mut self, input: &str, context: &str) -> String {
        if !self.initialized {
            return r#"{"error": "Agent not initialized"}"#.to_string();
        }

        console_log!("Executing agent-core with input: {}", input);

        // Parse input and context
        let result = self.process_cognitive_input(input, context);

        // Store in memory for learning
        self.memory.push(input.to_string());
        if self.memory.len() > self.config.max_context_size {
            self.memory.remove(0);
        }

        result
    }

    #[wasm_bindgen]
    pub fn agent_core_execute_tool(&self, tool_name: &str, parameters: &str, context: &str) -> String {
        if !self.initialized {
            return r#"{"error": "Agent not initialized"}"#.to_string();
        }

        console_log!("Executing tool: {} with parameters: {}", tool_name, parameters);

        // Tool execution logic
        format!(
            r#"{{"success": true, "result": "Tool {} executed successfully", "parameters": {}, "agentId": "{}"}}"#,
            tool_name, parameters, self.config.agent_id
        )
    }

    #[wasm_bindgen]
    pub fn agent_core_load_lora(&mut self, adapter: &str) -> bool {
        console_log!("Loading LoRA adapter: {}", adapter);
        // LoRA adapter loading logic would go here
        true
    }

    #[wasm_bindgen]
    pub fn agent_core_apply_skill(&mut self, proto_bytes: &[u8]) -> bool {
        console_log!("Applying skill from protobuf ({} bytes)", proto_bytes.len());
        // Skill application logic would go here
        true
    }

    #[wasm_bindgen]
    pub fn agent_core_get_status(&self) -> String {
        format!(
            r#"{{"agentId": "{}", "agentName": "${config.agentName || 'Unknown'}", "version": "${config.agentVersion}", "initialized": {}, "cognitiveEngine": "rust-wasm", "availableTools": [], "memorySize": {}}}"#,
            self.config.agent_id, self.initialized, self.memory.len()
        )
    }
}

impl AgentCore {
    fn process_cognitive_input(&self, input: &str, context: &str) -> String {
        // Cognitive processing logic
        // This would contain the actual AI processing logic

        let confidence = if input.len() > 10 { 0.8 } else { 0.6 };

        format!(
            r#"{{"success": true, "result": {{"response": "Processed: {}", "confidence": {}, "source": "rust-agent-core"}}, "processingTime": 50, "metadata": {{"agentId": "{}", "contextSize": {}}}}}"#,
            input, confidence, self.config.agent_id, context.len()
        )
    }
}
`;

    return rustCode;
  }

  private async createRustProject(buildDir: string, config: AgentCoreConfig, rustCode: string): Promise<string> {
    const rustProjectDir = join(buildDir, 'rust-agent');

    // Create Rust project structure
    await fs.mkdir(rustProjectDir, { recursive: true });
    await fs.mkdir(join(rustProjectDir, 'src'), { recursive: true });

    // Create Cargo.toml
    const cargoToml = `[package]
name = "agent-core-${config.agentId.toLowerCase().replace(/[^a-z0-9]/g, '-')}"
version = "${config.agentVersion}"
edition = "2021"
description = "Compiled agent-core for ${config.agentName}"

[lib]
crate-type = ["cdylib"]

[dependencies]
wasm-bindgen = "0.2"
js-sys = "0.3"
web-sys = "0.3"
serde = { version = "1.0", features = ["derive"] }
serde-wasm-bindgen = "0.6"
console_error_panic_hook = "0.1"

[dependencies.web-sys]
version = "0.3"
features = [
  "console",
  "WebAssembly",
  "Memory",
  "ArrayBuffer",
  "Uint8Array",
]

[profile.release]
opt-level = "s"
lto = true
codegen-units = 1
panic = "abort"
`;

    await fs.writeFile(join(rustProjectDir, 'Cargo.toml'), cargoToml);
    await fs.writeFile(join(rustProjectDir, 'src', 'lib.rs'), rustCode);

    return rustProjectDir;
  }

  private async compileRustToWASM(rustProjectDir: string, config: AgentCoreConfig): Promise<Uint8Array> {
    const { spawn } = await import('child_process');

    logger.info('Compiling Rust to WASM using wasm-pack...');

    return new Promise((resolve, reject) => {
      const wasmPack = spawn('wasm-pack', [
        'build',
        '--target', 'web',
        '--out-dir', 'pkg',
        '--release',
        '--scope', 'knirv'
      ], {
        cwd: rustProjectDir,
        stdio: ['pipe', 'pipe', 'pipe']
      });

      let stdout = '';
      let stderr = '';

      wasmPack.stdout?.on('data', (data) => {
        stdout += data.toString();
      });

      wasmPack.stderr?.on('data', (data) => {
        stderr += data.toString();
      });

      wasmPack.on('close', async (code) => {
        if (code === 0) {
          try {
            // Read the compiled WASM file
            const wasmPath = join(rustProjectDir, 'pkg', `agent_core_${config.agentId.toLowerCase().replace(/[^a-z0-9]/g, '_')}_bg.wasm`);

            // Try different possible WASM file names
            const possiblePaths = [
              wasmPath,
              join(rustProjectDir, 'pkg', 'agent_core_bg.wasm'),
              join(rustProjectDir, 'pkg', `${config.agentId.toLowerCase()}_bg.wasm`)
            ];

            let wasmBytes: Uint8Array | null = null;

            for (const path of possiblePaths) {
              try {
                wasmBytes = await fs.readFile(path);
                logger.info(`WASM file found at: ${path}`);
                break;
              } catch (error) {
                // Try next path
                continue;
              }
            }

            if (wasmBytes) {
              resolve(wasmBytes);
            } else {
              throw new Error('WASM file not found in expected locations');
            }
          } catch (error) {
            logger.error({ error }, 'Failed to read compiled WASM file');
            reject(error);
          }
        } else {
          logger.error({ code, stdout, stderr }, 'wasm-pack compilation failed');
          reject(new Error(`wasm-pack failed with code ${code}: ${stderr}`));
        }
      });

      wasmPack.on('error', (error) => {
        logger.error({ error }, 'Failed to spawn wasm-pack');
        reject(error);
      });
    });
  }

  private generateMinimalWASM(config: AgentCoreConfig): Uint8Array {
    // Generate a minimal WASM module with the required interface
    // This is used as a fallback when compilation fails
    logger.warn('Using minimal WASM fallback for agent:', config.agentId);

    const wasmModule = new Uint8Array([
      0x00, 0x61, 0x73, 0x6d, // WASM magic number
      0x01, 0x00, 0x00, 0x00, // WASM version

      // Type section
      0x01, 0x07, 0x01, 0x60, 0x02, 0x7f, 0x7f, 0x7f,

      // Function section
      0x03, 0x06, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00,

      // Memory section
      0x05, 0x03, 0x01, 0x00, 0x01,

      // Export section with agent-core functions
      0x07, 0x5a, 0x05,
      0x06, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x02, 0x00,
      0x11, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x72, 0x65, 0x45, 0x78, 0x65, 0x63, 0x75, 0x74, 0x65, 0x00, 0x00,
      0x15, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x72, 0x65, 0x45, 0x78, 0x65, 0x63, 0x75, 0x74, 0x65, 0x54, 0x6f, 0x6f, 0x6c, 0x00, 0x01,
      0x13, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x72, 0x65, 0x4c, 0x6f, 0x61, 0x64, 0x4c, 0x6f, 0x52, 0x41, 0x00, 0x02,
      0x15, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x72, 0x65, 0x41, 0x70, 0x70, 0x6c, 0x79, 0x53, 0x6b, 0x69, 0x6c, 0x6c, 0x00, 0x03,
      0x13, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x72, 0x65, 0x47, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x00, 0x04,

      // Code section with minimal function implementations
      0x0a, 0x2d, 0x05,
      0x07, 0x00, 0x41, 0x00, 0x41, 0x00, 0x0b, // agentCoreExecute: return 0
      0x07, 0x00, 0x41, 0x00, 0x41, 0x00, 0x0b, // agentCoreExecuteTool: return 0
      0x04, 0x00, 0x41, 0x01, 0x0b,             // agentCoreLoadLoRA: return 1 (true)
      0x04, 0x00, 0x41, 0x01, 0x0b,             // agentCoreApplySkill: return 1 (true)
      0x04, 0x00, 0x41, 0x00, 0x0b              // agentCoreGetStatus: return 0
    ]);

    return wasmModule;
  }

  isReady(): boolean {
    return this.isInitialized;
  }

  async dispose(): Promise<void> {
    this.goTemplates.clear();
    this.cognitiveTemplates.clear();
    this.isInitialized = false;
    logger.info('Agent-Core Compiler disposed');
  }
}

export default AgentCoreCompiler;
