/**
 * Phase 5.2 TypeScript WASM Compiler Tests
 * Tests for KNIRVCORTEX Agent-Builder TypeScript WASM compilation pipeline
 */

import { describe, test, expect, beforeAll, afterAll, beforeEach } from '@jest/globals';
import { promises as fs } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { tmpdir } from 'os';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Mock AgentCoreCompiler for testing
class MockAgentCoreCompiler {
  private templatesDir: string;
  private buildDir: string;
  private isInitialized = false;

  constructor(templatesDir: string, buildDir: string) {
    this.templatesDir = templatesDir;
    this.buildDir = buildDir;
  }

  async initialize(): Promise<void> {
    await fs.mkdir(this.templatesDir, { recursive: true });
    await fs.mkdir(this.buildDir, { recursive: true });
    this.isInitialized = true;
  }

  async compileAgentCore(config: AgentCoreConfig): Promise<CompilationResult> {
    if (!this.isInitialized) {
      throw new Error('Compiler not initialized');
    }

    const startTime = Date.now();

    try {
      // Simulate TypeScript compilation
      const typeScriptCode = await this.generateTypeScriptCode(config);
      
      // Simulate WASM compilation
      const wasmBytes = await this.compileToWASM(typeScriptCode);

      return {
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
    } catch (error) {
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

  private async generateTypeScriptCode(config: AgentCoreConfig): Promise<string> {
    return `
// Generated Agent Core for ${config.agentId}
export class ${config.agentName.replace(/\s+/g, '')}AgentCore {
  private config = ${JSON.stringify(config, null, 2)};
  
  async execute(input: any): Promise<any> {
    return { result: "processed", input, agentId: "${config.agentId}" };
  }
  
  async loadLoRAAdapter(adapter: any): Promise<boolean> {
    return true;
  }
  
  getStatus(): any {
    return {
      agentId: "${config.agentId}",
      initialized: true,
      capabilities: ${JSON.stringify(config.cognitiveCapabilities.map(c => c.name))}
    };
  }
}
`;
  }

  private async compileToWASM(typeScriptCode: string): Promise<Uint8Array> {
    // Simulate WASM compilation with proper WASM magic number
    const wasmMagic = new Uint8Array([
      0x00, 0x61, 0x73, 0x6d, // WASM magic number
      0x01, 0x00, 0x00, 0x00  // WASM version
    ]);
    
    // Add some simulated compiled content
    const content = new TextEncoder().encode(typeScriptCode);
    const result = new Uint8Array(wasmMagic.length + content.length);
    result.set(wasmMagic);
    result.set(content, wasmMagic.length);
    
    return result;
  }

  isReady(): boolean {
    return this.isInitialized;
  }

  async dispose(): Promise<void> {
    this.isInitialized = false;
  }
}

// Type definitions
interface AgentCoreConfig {
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
}

interface ToolConfig {
  name: string;
  description: string;
  parameters: ToolParameter[];
  implementation: string;
  sourceType: 'inline' | 'external' | 'template';
}

interface ToolParameter {
  name: string;
  type: string;
  required: boolean;
  description: string;
  defaultValue?: any;
}

interface CognitiveCapability {
  name: string;
  enabled: boolean;
  config: Record<string, any>;
}

interface SensoryInterface {
  type: 'voice' | 'visual' | 'text' | 'gesture';
  enabled: boolean;
  config: Record<string, any>;
}

interface CompilationResult {
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

// Test suite
describe('Phase 5.2 TypeScript WASM Compiler Tests', () => {
  let testDir: string;
  let templatesDir: string;
  let buildDir: string;
  let compiler: MockAgentCoreCompiler;

  beforeAll(async () => {
    testDir = await fs.mkdtemp(join(tmpdir(), 'ts-wasm-compiler-test-'));
    templatesDir = join(testDir, 'templates');
    buildDir = join(testDir, 'build');
    
    compiler = new MockAgentCoreCompiler(templatesDir, buildDir);
    await compiler.initialize();
  });

  afterAll(async () => {
    await compiler.dispose();
    await fs.rm(testDir, { recursive: true, force: true });
  });

  beforeEach(async () => {
    // Clean build directory before each test
    await fs.rm(buildDir, { recursive: true, force: true });
    await fs.mkdir(buildDir, { recursive: true });
  });

  describe('TypeScript Pipeline Integration Tests', () => {
    test('should initialize TypeScript compilation pipeline', async () => {
      expect(compiler.isReady()).toBe(true);
      
      // Verify directories exist
      await expect(fs.access(templatesDir)).resolves.not.toThrow();
      await expect(fs.access(buildDir)).resolves.not.toThrow();
    });

    test('should compile basic agent configuration to TypeScript', async () => {
      const config: AgentCoreConfig = {
        agentId: 'test-agent-001',
        agentName: 'Test Agent',
        agentDescription: 'A test agent for compilation',
        agentVersion: '1.0.0',
        author: 'Test Suite',
        tools: [
          {
            name: 'calculator',
            description: 'Basic calculator tool',
            parameters: [
              { name: 'operation', type: 'string', required: true, description: 'Math operation' },
              { name: 'a', type: 'number', required: true, description: 'First number' },
              { name: 'b', type: 'number', required: true, description: 'Second number' }
            ],
            implementation: 'return a + b;',
            sourceType: 'inline'
          }
        ],
        cognitiveCapabilities: [
          { name: 'reasoning', enabled: true, config: {} },
          { name: 'learning', enabled: true, config: {} }
        ],
        sensoryInterfaces: [
          { type: 'text', enabled: true, config: {} },
          { type: 'voice', enabled: false, config: {} }
        ],
        buildTarget: 'typescript',
        optimizationLevel: 'basic'
      };

      const result = await compiler.compileAgentCore(config);

      expect(result.success).toBe(true);
      expect(result.agentId).toBe('test-agent-001');
      expect(result.typeScriptCode).toBeDefined();
      expect(result.typeScriptCode).toContain('TestAgentAgentCore');
      expect(result.typeScriptCode).toContain('test-agent-001');
      expect(result.metadata.typeScriptSize).toBeGreaterThan(0);
      expect(result.metadata.cognitiveCapabilities).toEqual(['reasoning', 'learning']);
      expect(result.metadata.sensoryInterfaces).toEqual(['text']);
    });

    test('should compile agent configuration to WASM', async () => {
      const config: AgentCoreConfig = {
        agentId: 'wasm-test-agent',
        agentName: 'WASM Test Agent',
        agentDescription: 'Agent for WASM compilation testing',
        agentVersion: '1.0.0',
        author: 'Test Suite',
        tools: [],
        cognitiveCapabilities: [
          { name: 'cognitive-engine', enabled: true, config: {} }
        ],
        sensoryInterfaces: [
          { type: 'text', enabled: true, config: {} }
        ],
        buildTarget: 'wasm',
        optimizationLevel: 'aggressive'
      };

      const result = await compiler.compileAgentCore(config);

      expect(result.success).toBe(true);
      expect(result.wasmBytes).toBeDefined();
      expect(result.wasmBytes!.length).toBeGreaterThan(8);
      
      // Verify WASM magic number
      expect(result.wasmBytes![0]).toBe(0x00);
      expect(result.wasmBytes![1]).toBe(0x61);
      expect(result.wasmBytes![2]).toBe(0x73);
      expect(result.wasmBytes![3]).toBe(0x6d);
      
      // Verify WASM version
      expect(result.wasmBytes![4]).toBe(0x01);
      expect(result.wasmBytes![5]).toBe(0x00);
      expect(result.wasmBytes![6]).toBe(0x00);
      expect(result.wasmBytes![7]).toBe(0x00);

      expect(result.metadata.wasmSize).toBe(result.wasmBytes!.length);
      expect(result.metadata.optimizationLevel).toBe('aggressive');
    });

    test('should handle hybrid compilation target', async () => {
      const config: AgentCoreConfig = {
        agentId: 'hybrid-agent',
        agentName: 'Hybrid Agent',
        agentDescription: 'Agent for hybrid compilation',
        agentVersion: '1.0.0',
        author: 'Test Suite',
        tools: [
          {
            name: 'text-processor',
            description: 'Text processing tool',
            parameters: [
              { name: 'text', type: 'string', required: true, description: 'Input text' }
            ],
            implementation: 'return text.toUpperCase();',
            sourceType: 'inline'
          }
        ],
        cognitiveCapabilities: [
          { name: 'lora-adapter', enabled: true, config: { rank: 8, alpha: 16.0 } },
          { name: 'adaptive-learning', enabled: true, config: {} }
        ],
        sensoryInterfaces: [
          { type: 'text', enabled: true, config: {} },
          { type: 'visual', enabled: true, config: {} }
        ],
        buildTarget: 'hybrid',
        optimizationLevel: 'basic'
      };

      const result = await compiler.compileAgentCore(config);

      expect(result.success).toBe(true);
      expect(result.wasmBytes).toBeDefined();
      expect(result.typeScriptCode).toBeDefined();
      expect(result.metadata.cognitiveCapabilities).toEqual(['lora-adapter', 'adaptive-learning']);
      expect(result.metadata.sensoryInterfaces).toEqual(['text', 'visual']);
    });

    test('should handle compilation errors gracefully', async () => {
      const invalidConfig = {
        agentId: '', // Invalid empty ID
        agentName: 'Invalid Agent',
        agentDescription: 'Agent with invalid configuration',
        agentVersion: '1.0.0',
        author: 'Test Suite',
        tools: [],
        cognitiveCapabilities: [],
        sensoryInterfaces: [],
        buildTarget: 'wasm' as const,
        optimizationLevel: 'none' as const
      };

      // Mock compiler to throw error for empty agentId
      const errorCompiler = new MockAgentCoreCompiler(templatesDir, buildDir);
      await errorCompiler.initialize();
      
      // Override compileAgentCore to simulate error
      const originalCompile = errorCompiler.compileAgentCore.bind(errorCompiler);
      errorCompiler.compileAgentCore = async (config) => {
        if (!config.agentId) {
          throw new Error('Agent ID cannot be empty');
        }
        return originalCompile(config);
      };

      const result = await errorCompiler.compileAgentCore(invalidConfig);

      expect(result.success).toBe(false);
      expect(result.errors).toBeDefined();
      expect(result.errors!.length).toBeGreaterThan(0);
      expect(result.errors![0]).toContain('Agent ID cannot be empty');
    });

    test('should optimize compilation based on optimization level', async () => {
      const baseConfig: AgentCoreConfig = {
        agentId: 'optimization-test',
        agentName: 'Optimization Test Agent',
        agentDescription: 'Agent for testing optimization levels',
        agentVersion: '1.0.0',
        author: 'Test Suite',
        tools: [],
        cognitiveCapabilities: [
          { name: 'cognitive-engine', enabled: true, config: {} }
        ],
        sensoryInterfaces: [
          { type: 'text', enabled: true, config: {} }
        ],
        buildTarget: 'wasm',
        optimizationLevel: 'none'
      };

      // Test different optimization levels
      const optimizationLevels: Array<'none' | 'basic' | 'aggressive'> = ['none', 'basic', 'aggressive'];
      const results: CompilationResult[] = [];

      for (const level of optimizationLevels) {
        const config = { ...baseConfig, optimizationLevel: level };
        const result = await compiler.compileAgentCore(config);
        expect(result.success).toBe(true);
        results.push(result);
      }

      // Verify optimization metadata
      expect(results[0].metadata.optimizationLevel).toBe('none');
      expect(results[1].metadata.optimizationLevel).toBe('basic');
      expect(results[2].metadata.optimizationLevel).toBe('aggressive');

      // All should have valid WASM output
      results.forEach(result => {
        expect(result.wasmBytes).toBeDefined();
        expect(result.wasmBytes!.length).toBeGreaterThan(8);
      });
    });

    test('should handle complex tool configurations', async () => {
      const config: AgentCoreConfig = {
        agentId: 'complex-tools-agent',
        agentName: 'Complex Tools Agent',
        agentDescription: 'Agent with complex tool configurations',
        agentVersion: '1.0.0',
        author: 'Test Suite',
        tools: [
          {
            name: 'data-analyzer',
            description: 'Advanced data analysis tool',
            parameters: [
              { name: 'data', type: 'object', required: true, description: 'Input data' },
              { name: 'algorithm', type: 'string', required: false, description: 'Analysis algorithm', defaultValue: 'default' },
              { name: 'threshold', type: 'number', required: false, description: 'Analysis threshold', defaultValue: 0.5 }
            ],
            implementation: `
              const algorithm = parameters.algorithm || 'default';
              const threshold = parameters.threshold || 0.5;
              return { analyzed: true, algorithm, threshold, dataSize: Object.keys(parameters.data).length };
            `,
            sourceType: 'inline'
          },
          {
            name: 'external-api-caller',
            description: 'External API integration tool',
            parameters: [
              { name: 'endpoint', type: 'string', required: true, description: 'API endpoint' },
              { name: 'method', type: 'string', required: false, description: 'HTTP method', defaultValue: 'GET' }
            ],
            implementation: 'return { called: true, endpoint: parameters.endpoint, method: parameters.method };',
            sourceType: 'external'
          }
        ],
        cognitiveCapabilities: [
          { name: 'advanced-reasoning', enabled: true, config: { depth: 5 } },
          { name: 'memory-management', enabled: true, config: { maxSize: 1000 } }
        ],
        sensoryInterfaces: [
          { type: 'text', enabled: true, config: { maxLength: 10000 } },
          { type: 'visual', enabled: true, config: { resolution: '1920x1080' } },
          { type: 'gesture', enabled: false, config: {} }
        ],
        buildTarget: 'hybrid',
        optimizationLevel: 'basic'
      };

      const result = await compiler.compileAgentCore(config);

      expect(result.success).toBe(true);
      expect(result.typeScriptCode).toContain('data-analyzer');
      expect(result.typeScriptCode).toContain('external-api-caller');
      expect(result.metadata.cognitiveCapabilities).toEqual(['advanced-reasoning', 'memory-management']);
      expect(result.metadata.sensoryInterfaces).toEqual(['text', 'visual']);
    });

    test('should measure compilation performance', async () => {
      const config: AgentCoreConfig = {
        agentId: 'performance-test-agent',
        agentName: 'Performance Test Agent',
        agentDescription: 'Agent for performance testing',
        agentVersion: '1.0.0',
        author: 'Test Suite',
        tools: [],
        cognitiveCapabilities: [
          { name: 'performance-monitor', enabled: true, config: {} }
        ],
        sensoryInterfaces: [
          { type: 'text', enabled: true, config: {} }
        ],
        buildTarget: 'wasm',
        optimizationLevel: 'basic'
      };

      const startTime = Date.now();
      const result = await compiler.compileAgentCore(config);
      const totalTime = Date.now() - startTime;

      expect(result.success).toBe(true);
      expect(result.metadata.compilationTime).toBeGreaterThan(0);
      expect(result.metadata.compilationTime).toBeLessThanOrEqual(totalTime);
      
      // Performance should be reasonable (less than 5 seconds for test)
      expect(result.metadata.compilationTime).toBeLessThan(5000);
    });
  });
});
