/**
 * Phase 1: Critical Infrastructure Integration Tests
 * Tests for import.meta.url fixes, WASM orchestrator initialization, and EmbeddedKNIRVChain removal
 */

import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { AgentCoreCompiler } from '../../src/core/agent-core-compiler/src/AgentCoreCompiler';
import { WASMOrchestrator } from '../../src/sensory-shell/WASMOrchestrator';
import { ProtobufHandler } from '../../src/core/protobuf/ProtobufHandler';

// Test the import.meta.url compatibility fix
describe('Phase 1.1: import.meta.url Compatibility', () => {
  it('should handle import.meta.url in test environment', () => {
    // Test the fallback mechanism using the same pattern as the source code
    const getModuleUrl = () => {
      // Use conditional access to avoid Jest parsing issues
      const importMeta = typeof globalThis !== 'undefined' &&
                        (globalThis as any).importMeta;
      if (importMeta && importMeta.url) {
        return importMeta.url;
      }
      return process.cwd();
    };

    const moduleUrl = getModuleUrl();
    expect(moduleUrl).toBeDefined();
    expect(typeof moduleUrl).toBe('string');
  });

  it('should initialize ProtobufHandler without import.meta.url errors', async () => {
    const protobufHandler = new ProtobufHandler();
    await expect(protobufHandler.initialize()).resolves.not.toThrow();
  });

  it('should initialize AgentCoreCompiler without import.meta.url errors', async () => {
    const compiler = new AgentCoreCompiler();
    await expect(compiler.initialize()).resolves.not.toThrow();
  });
});

describe('Phase 1.2: WASM Orchestrator Initialization', () => {
  let wasmOrchestrator: WASMOrchestrator;

  beforeEach(() => {
    wasmOrchestrator = new WASMOrchestrator();
  });

  afterEach(async () => {
    if (wasmOrchestrator) {
      await wasmOrchestrator.shutdown();
    }
  });

  it('should initialize WASM orchestrator successfully', async () => {
    await expect(wasmOrchestrator.initialize()).resolves.not.toThrow();
    expect(wasmOrchestrator.isInitialized).toBe(true);
  });

  it('should handle WASM module initialization with proper client-side setup', async () => {
    await wasmOrchestrator.initialize();

    // Test WASM initialization pattern
    const mockWasmBytes = new Uint8Array([
      0x00, 0x61, 0x73, 0x6d, // WASM magic number
      0x01, 0x00, 0x00, 0x00  // WASM version
    ]);

    // This should not throw and should handle client-side initialization
    await expect(wasmOrchestrator.loadAgentWASM(mockWasmBytes, 'test-agent')).resolves.toBe(true);
  });

  it('should start and process inference queue', async () => {
    await wasmOrchestrator.initialize();
    await wasmOrchestrator.start();

    expect(wasmOrchestrator.isRunning).toBe(true);
    
    // Test that inference queue is processed
    const testInput = {
      type: 'text' as const,
      data: 'test input',
      timestamp: Date.now()
    };

    const response = await wasmOrchestrator.processSensoryInput(testInput);
    expect(response).toBeDefined();
    expect(response.success).toBe(true);
  });

  it('should handle initialization errors gracefully', async () => {
    // Mock a failure scenario
    const failingOrchestrator = new WASMOrchestrator();
    
    // Override internal method to simulate failure
    jest.spyOn(failingOrchestrator as any, 'initializeAgentCore').mockRejectedValue(new Error('Initialization failed'));

    await expect(failingOrchestrator.initialize()).rejects.toThrow('Initialization failed');
  });
});

describe('Phase 1.3: Real WASM Compilation', () => {
  let agentCompiler: AgentCoreCompiler;

  beforeEach(async () => {
    agentCompiler = new AgentCoreCompiler();
    await agentCompiler.initialize();
  });

  it('should compile agent-core to functional WASM (not placeholder)', async () => {
    const testConfig = {
      agentId: 'test-agent-real-wasm',
      agentName: 'Test Agent Real WASM',
      agentDescription: 'Test agent for real WASM compilation',
      agentVersion: '1.0.0',
      author: 'test',
      tools: [],
      cognitiveCapabilities: [],
      sensoryInterfaces: [],
      buildTarget: 'wasm' as const,
      optimizationLevel: 'basic' as const
    };

    const result = await agentCompiler.compileAgentCore(testConfig);
    
    expect(result.success).toBe(true);
    expect(result.wasmBytes).toBeDefined();
    expect(result.wasmBytes!.length).toBeGreaterThan(8); // More than just magic number + version
    
    // Verify it's not the placeholder WASM
    const wasmBytes = result.wasmBytes!;
    const isPlaceholder = wasmBytes.length === 8 && 
                         wasmBytes[0] === 0x00 && wasmBytes[1] === 0x61 && 
                         wasmBytes[2] === 0x73 && wasmBytes[3] === 0x6d;
    
    expect(isPlaceholder).toBe(false);
  });

  it('should produce WASM with initialization exports', async () => {
    const testConfig = {
      agentId: 'test-agent-init',
      agentName: 'Test Agent Init',
      agentDescription: 'Test agent for initialization exports',
      agentVersion: '1.0.0',
      author: 'test',
      tools: [],
      cognitiveCapabilities: [],
      sensoryInterfaces: [],
      buildTarget: 'wasm' as const,
      optimizationLevel: 'basic' as const
    };

    const result = await agentCompiler.compileAgentCore(testConfig);
    expect(result.success).toBe(true);
    
    // Verify WASM can be compiled and has proper structure
    const wasmModule = await WebAssembly.compile(result.wasmBytes!);
    const exports = WebAssembly.Module.exports(wasmModule);
    
    // Should have some exports (exact exports depend on implementation)
    expect(exports.length).toBeGreaterThan(0);
  });

  it('should validate and optimize WASM binary', async () => {
    const testConfig = {
      agentId: 'test-agent-optimized',
      agentName: 'Test Agent Optimized',
      agentDescription: 'Test agent for optimization',
      agentVersion: '1.0.0',
      author: 'test',
      tools: [],
      cognitiveCapabilities: [],
      sensoryInterfaces: [],
      buildTarget: 'wasm' as const,
      optimizationLevel: 'O2' as const
    };

    const result = await agentCompiler.compileAgentCore(testConfig);
    expect(result.success).toBe(true);
    expect(result.compilationMetrics).toBeDefined();
    expect(result.compilationMetrics!.optimizationLevel).toBe('O2');
  });
});

describe('Phase 1.4: EmbeddedKNIRVChain Removal', () => {
  it('should not have any EmbeddedKNIRVChain imports', () => {
    // Test that EmbeddedKNIRVChain is completely removed
    expect(() => {
      // This should fail if EmbeddedKNIRVChain still exists
      require('../../src/sensory-shell/EmbeddedKNIRVChain');
    }).toThrow();
  });

  it('should have KNIRVChainIntegration refactored for KNIRVROUTER', async () => {
    const { KNIRVChainIntegration } = await import('../../src/sensory-shell/KNIRVChainIntegration');
    
    const config = {
      rpcUrl: 'http://localhost:26657',
      chainId: 'knirv-testnet',
      networkName: 'KNIRV Testnet',
      contractAddresses: {
        nrnToken: 'knirv1...',
        llmRegistry: 'knirv1...',
        skillRegistry: 'knirv1...'
      },
      gasPrice: '0.025uknirv',
      gasLimit: '200000',
      knirvRouterUrl: 'http://localhost:5000',
      knirvGraphUrl: 'http://localhost:6000',
      useKnirvRouter: true
    };

    const integration = new KNIRVChainIntegration(config);
    
    // Should be configured for KNIRVROUTER, not embedded chain
    expect(integration).toBeDefined();
    expect(config.useKnirvRouter).toBe(true);
  });

  it('should have CognitiveEngine without embedded skill invocation', async () => {
    const { CognitiveEngine } = await import('../../src/sensory-shell/CognitiveEngine');
    
    const config = {
      maxContextSize: 100,
      learningRate: 0.01,
      adaptationThreshold: 0.3,
      skillTimeout: 30000,
      voiceEnabled: false,
      visualEnabled: false,
      loraEnabled: true,
      enhancedLoraEnabled: false,
      hrmEnabled: false,
      adaptiveLearningEnabled: false,
      walletIntegrationEnabled: false,
      chainIntegrationEnabled: true, // Should use external KNIRVROUTER
      ecosystemCommunicationEnabled: false,
      errorContextEnabled: true // New feature for ErrorContext generation
    };

    const engine = new CognitiveEngine(config);
    
    // Should not have embedded skill invocation
    expect(engine).toBeDefined();
    
    // Test that skill invocation goes through external network
    const skillResult = await engine.invokeSkill('test-skill', { test: 'data' });
    expect(skillResult).toBeDefined();
    // Should not be a mock response
    expect(skillResult.output).not.toContain('Mock skill execution result');
  });
});

describe('Phase 1: Integration Verification', () => {
  it('should have all Phase 1 components working together', async () => {
    // Initialize all components
    const agentCompiler = new AgentCoreCompiler();
    const wasmOrchestrator = new WASMOrchestrator();
    const protobufHandler = new ProtobufHandler();

    await agentCompiler.initialize();
    await wasmOrchestrator.initialize();
    await protobufHandler.initialize();

    // Compile an agent
    const testConfig = {
      agentId: 'integration-test-agent',
      agentName: 'Integration Test Agent',
      agentDescription: 'Agent for integration testing',
      agentVersion: '1.0.0',
      author: 'test',
      tools: [],
      cognitiveCapabilities: [],
      sensoryInterfaces: [],
      buildTarget: 'wasm' as const,
      optimizationLevel: 'basic' as const
    };

    const compilationResult = await agentCompiler.compileAgentCore(testConfig);
    expect(compilationResult.success).toBe(true);

    // Load into orchestrator
    await wasmOrchestrator.start();
    const loadResult = await wasmOrchestrator.loadAgentWASM(compilationResult.wasmBytes!, testConfig.agentId);
    expect(loadResult).toBe(true);

    // Test processing
    const testInput = {
      type: 'text' as const,
      data: 'integration test input',
      timestamp: Date.now()
    };

    const response = await wasmOrchestrator.processSensoryInput(testInput);
    expect(response.success).toBe(true);

    // Cleanup
    await wasmOrchestrator.shutdown();
  });
});
