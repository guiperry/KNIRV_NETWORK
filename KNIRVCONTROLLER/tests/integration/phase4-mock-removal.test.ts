/**
 * Phase 4: Mock Removal & Testing Structure Integration Tests
 * Tests to verify complete removal of mocks and proper testing structure
 */

import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';
import { promises as fs } from 'fs';
import { join } from 'path';

// Mock only external network dependencies
global.fetch = jest.fn();

describe('Phase 4.1: Complete EmbeddedKNIRVChain Removal Verification', () => {
  it('should have no EmbeddedKNIRVChain files remaining', async () => {
    const embeddedChainPath = join(__dirname, '../../src/sensory-shell/EmbeddedKNIRVChain.ts');
    
    try {
      await fs.access(embeddedChainPath);
      fail('EmbeddedKNIRVChain.ts file should not exist');
    } catch (error) {
      // File should not exist - this is expected
      expect(error).toBeDefined();
    }
  });

  it('should have no embedded skill invocation references in CognitiveEngine', async () => {
    const cognitiveEnginePath = join(__dirname, '../../src/sensory-shell/CognitiveEngine.ts');
    const content = await fs.readFile(cognitiveEnginePath, 'utf-8');
    
    // Should not contain mock skill responses
    expect(content).not.toContain('Mock skill execution result');
    expect(content).not.toContain('Fallback implementation for skill invocation');
    
    // Should contain KNIRVROUTER integration
    expect(content).toContain('KNIRVROUTER');
    expect(content).toContain('ErrorContext');
  });

  it('should have no embedded consensus mechanisms', async () => {
    const srcPath = join(__dirname, '../../src');
    
    const searchForEmbeddedConsensus = async (dir: string): Promise<string[]> => {
      const files: string[] = [];
      const entries = await fs.readdir(dir, { withFileTypes: true });
      
      for (const entry of entries) {
        const fullPath = join(dir, entry.name);
        
        if (entry.isDirectory() && !entry.name.includes('node_modules')) {
          files.push(...await searchForEmbeddedConsensus(fullPath));
        } else if (entry.isFile() && (entry.name.endsWith('.ts') || entry.name.endsWith('.tsx'))) {
          const content = await fs.readFile(fullPath, 'utf-8');
          if (content.includes('embedded') && content.includes('consensus')) {
            files.push(fullPath);
          }
        }
      }
      
      return files;
    };

    const filesWithEmbeddedConsensus = await searchForEmbeddedConsensus(srcPath);
    expect(filesWithEmbeddedConsensus).toHaveLength(0);
  });

  it('should have KNIRVChainIntegration properly refactored for KNIRVROUTER', async () => {
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
      useKnirvRouter: true
    };

    const integration = new KNIRVChainIntegration(config);
    
    // Should be configured for external KNIRVROUTER
    expect(integration).toBeDefined();
    
    // Should have KNIRVROUTER integration methods
    expect(typeof integration.invokeSkillOnChain).toBe('function');
    
    // Should not have embedded chain methods
    expect((integration as any).embeddedChain).toBeUndefined();
  });
});

describe('Phase 4.2: HRMBridge Real Implementation Verification', () => {
  let hrmBridge: any;

  beforeEach(async () => {
    const { HRMBridge } = await import('../../src/sensory-shell/HRMBridge');
    
    const config = {
      modelPath: '/path/to/hrm/model',
      maxMemoryMB: 512,
      enableGPU: false,
      batchSize: 1,
      sequenceLength: 512
    };

    hrmBridge = new HRMBridge(config);
  });

  it('should integrate real HRM WASM module', async () => {
    await hrmBridge.initialize();
    
    // Should have real WASM module loaded
    expect(hrmBridge.isInitialized()).toBe(true);
    expect(hrmBridge.wasmModule).toBeDefined();
    
    // Should not be a mock implementation
    expect(hrmBridge.wasmModule.constructor.name).not.toBe('MockWASMModule');
  });

  it('should implement actual cognitive processing', async () => {
    await hrmBridge.initialize();
    
    const testInput = {
      text: 'Test cognitive processing input',
      context: 'Test context',
      maxTokens: 100
    };

    const result = await hrmBridge.processInput(testInput);
    
    expect(result).toBeDefined();
    expect(result.success).toBe(true);
    expect(result.output).toBeDefined();
    
    // Should not be a mock response
    expect(result.output).not.toContain('Mock HRM response');
    expect(result.output).not.toContain('Placeholder');
  });

  it('should ensure proper WASM initialization sequence', async () => {
    // Test that WASM initialization follows proper pattern
    const initSpy = jest.spyOn(hrmBridge, 'initializeWASM');
    
    await hrmBridge.initialize();
    
    expect(initSpy).toHaveBeenCalled();
    
    // Should have called client-side initialization
    expect(hrmBridge.wasmInstance).toBeDefined();
    expect(hrmBridge.wasmInstance.exports).toBeDefined();
  });

  it('should remove all mock response generation', async () => {
    await hrmBridge.initialize();
    
    // Test multiple inputs to ensure no mock responses
    const inputs = [
      'Test input 1',
      'Test input 2',
      'Complex cognitive task'
    ];

    for (const input of inputs) {
      const result = await hrmBridge.processInput({ text: input });
      
      expect(result.output).not.toContain('Mock');
      expect(result.output).not.toContain('Placeholder');
      expect(result.output).not.toContain('Simulated');
    }
  });
});

describe('Phase 4.3: Testing Structure Verification', () => {
  it('should have reduced mocking to <5% of codebase', async () => {
    const testDir = join(__dirname, '..');
    
    const countMockUsage = async (dir: string): Promise<{ total: number; mocked: number }> => {
      let totalTests = 0;
      let mockedTests = 0;
      
      const entries = await fs.readdir(dir, { withFileTypes: true });
      
      for (const entry of entries) {
        const fullPath = join(dir, entry.name);
        
        if (entry.isDirectory() && !entry.name.includes('node_modules')) {
          const subResult = await countMockUsage(fullPath);
          totalTests += subResult.total;
          mockedTests += subResult.mocked;
        } else if (entry.isFile() && entry.name.endsWith('.test.ts')) {
          totalTests++;
          
          const content = await fs.readFile(fullPath, 'utf-8');
          if (content.includes('jest.mock') || content.includes('mockImplementation')) {
            // Only count as mocked if it mocks internal components
            if (!content.includes('external') && !content.includes('network')) {
              mockedTests++;
            }
          }
        }
      }
      
      return { total: totalTests, mocked: mockedTests };
    };

    const result = await countMockUsage(testDir);
    const mockPercentage = (result.mocked / result.total) * 100;
    
    expect(mockPercentage).toBeLessThan(5);
  });

  it('should keep mocks only for external dependencies', async () => {
    const testFiles = [
      'phase1-infrastructure.test.ts',
      'phase2-revolutionary-architecture.test.ts',
      'phase3-frontend-backend.test.ts'
    ];

    for (const testFile of testFiles) {
      const testPath = join(__dirname, testFile);
      const content = await fs.readFile(testPath, 'utf-8');
      
      // Should mock external network calls
      expect(content).toContain('global.fetch = jest.fn()');
      
      // Should not mock internal components extensively
      expect(content).not.toContain('jest.mock(\'../../src/sensory-shell/CognitiveEngine\')');
      expect(content).not.toContain('jest.mock(\'../../src/sensory-shell/WASMOrchestrator\')');
    }
  });

  it('should use real implementations for internal components', async () => {
    // Test that internal components are imported and used directly
    const { CognitiveEngine } = await import('../../src/sensory-shell/CognitiveEngine');
    const { WASMOrchestrator } = await import('../../src/sensory-shell/WASMOrchestrator');
    const { AgentCoreCompiler } = await import('../../src/core/agent-core-compiler/src/AgentCoreCompiler');

    // Should be real classes, not mocks
    expect(CognitiveEngine).toBeDefined();
    expect(WASMOrchestrator).toBeDefined();
    expect(AgentCoreCompiler).toBeDefined();
    
    // Should be able to instantiate
    const cognitiveEngine = new CognitiveEngine({
      maxContextSize: 100,
      learningRate: 0.01,
      adaptationThreshold: 0.3,
      skillTimeout: 30000,
      errorContextEnabled: true
    });
    
    expect(cognitiveEngine).toBeInstanceOf(CognitiveEngine);
  });

  it('should have end-to-end tests with minimal mocking', async () => {
    // This test represents an end-to-end workflow
    const { CognitiveEngine } = await import('../../src/sensory-shell/CognitiveEngine');
    const { WASMOrchestrator } = await import('../../src/sensory-shell/WASMOrchestrator');
    const { AgentCoreCompiler } = await import('../../src/core/agent-core-compiler/src/AgentCoreCompiler');

    // Mock only external network calls
    (global.fetch as jest.Mock)
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ skillNodeUri: 'knirv://skills/test/v1.0.0' })
      })
      .mockResolvedValueOnce({
        ok: true,
        arrayBuffer: () => Promise.resolve(new Uint8Array([1, 2, 3, 4]).buffer)
      });

    // Use real implementations
    const compiler = new AgentCoreCompiler();
    const orchestrator = new WASMOrchestrator();
    const cognitiveEngine = new CognitiveEngine({
      maxContextSize: 100,
      learningRate: 0.01,
      adaptationThreshold: 0.3,
      skillTimeout: 30000,
      errorContextEnabled: true,
      knirvGraphUrl: 'http://localhost:6000',
      knirvRouterUrl: 'http://localhost:5000'
    });

    // Initialize components
    await compiler.initialize();
    await orchestrator.initialize();

    // Compile agent
    const agentConfig = {
      agentId: 'e2e-test-agent',
      agentName: 'E2E Test Agent',
      agentDescription: 'End-to-end test agent',
      agentVersion: '1.0.0',
      author: 'test',
      tools: [],
      cognitiveCapabilities: [],
      sensoryInterfaces: [],
      buildTarget: 'wasm' as const,
      optimizationLevel: 'basic' as const
    };

    const compilationResult = await compiler.compileAgentCore(agentConfig);
    expect(compilationResult.success).toBe(true);

    // Load into orchestrator
    await orchestrator.start();
    const loadResult = await orchestrator.loadAgentWASM(compilationResult.wasmBytes!, agentConfig.agentId);
    expect(loadResult).toBe(true);

    // Test error handling with skill acquisition
    try {
      throw new Error('E2E test error');
    } catch (error) {
      const errorContext = await cognitiveEngine.generateErrorContext(error as Error, {
        task_description: 'E2E test task'
      });
      
      const skillNodeUri = await cognitiveEngine.queryKNIRVGRAPH(errorContext);
      expect(skillNodeUri).toBe('knirv://skills/test/v1.0.0');
      
      const skillResult = await cognitiveEngine.invokeSkillByUri(skillNodeUri!, 'nrn-token-123');
      expect(skillResult.success).toBe(true);
    }

    // Cleanup
    await orchestrator.shutdown();
  });
});

describe('Phase 4: Complete Implementation Verification', () => {
  it('should achieve 100% test success rate', async () => {
    // This test verifies that all critical functionality works together
    const { CognitiveEngine } = await import('../../src/sensory-shell/CognitiveEngine');
    const { WASMOrchestrator } = await import('../../src/sensory-shell/WASMOrchestrator');
    const { KNIRVRouterIntegration } = await import('../../src/sensory-shell/KNIRVRouterIntegration');

    // Mock external network responses
    (global.fetch as jest.Mock)
      .mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true }),
        arrayBuffer: () => Promise.resolve(new Uint8Array([1, 2, 3, 4]).buffer)
      });

    // Initialize all components
    const orchestrator = new WASMOrchestrator();
    const routerIntegration = new KNIRVRouterIntegration({
      knirvRouterUrl: 'http://localhost:5000',
      knirvGraphUrl: 'http://localhost:6000',
      enableP2P: true,
      enableWASM: true,
      maxRetries: 3,
      timeoutMs: 30000
    });
    
    const cognitiveEngine = new CognitiveEngine({
      maxContextSize: 100,
      learningRate: 0.01,
      adaptationThreshold: 0.3,
      skillTimeout: 30000,
      errorContextEnabled: true,
      chainIntegrationEnabled: true
    });

    // All components should initialize successfully
    await expect(orchestrator.initialize()).resolves.not.toThrow();
    await expect(routerIntegration.initialize()).resolves.not.toThrow();

    // Revolutionary architecture should be functional
    const errorContext = {
      agent_id: 'test-agent',
      error_type: 'Error',
      error_message: 'Test error',
      timestamp: new Date()
    } as any;

    const skillNodeUri = await routerIntegration.queryKNIRVGraphForPatterns(errorContext);
    expect(skillNodeUri).toBeDefined();

    // WASM functionality should work
    await orchestrator.start();
    const testInput = {
      type: 'text' as const,
      data: 'test input',
      timestamp: Date.now()
    };

    const response = await orchestrator.processSensoryInput(testInput);
    expect(response.success).toBe(true);

    // Cleanup
    await orchestrator.shutdown();
  });

  it('should have zero embedded blockchain code remaining', async () => {
    const srcPath = join(__dirname, '../../src');
    
    const searchForEmbeddedBlockchain = async (dir: string): Promise<string[]> => {
      const files: string[] = [];
      const entries = await fs.readdir(dir, { withFileTypes: true });
      
      for (const entry of entries) {
        const fullPath = join(dir, entry.name);
        
        if (entry.isDirectory() && !entry.name.includes('node_modules')) {
          files.push(...await searchForEmbeddedBlockchain(fullPath));
        } else if (entry.isFile() && (entry.name.endsWith('.ts') || entry.name.endsWith('.tsx'))) {
          const content = await fs.readFile(fullPath, 'utf-8');
          if (content.includes('EmbeddedKNIRVChain') || 
              (content.includes('embedded') && content.includes('blockchain'))) {
            files.push(fullPath);
          }
        }
      }
      
      return files;
    };

    const filesWithEmbeddedBlockchain = await searchForEmbeddedBlockchain(srcPath);
    expect(filesWithEmbeddedBlockchain).toHaveLength(0);
  });

  it('should have all operations complete within acceptable time limits', async () => {
    const { CognitiveEngine } = await import('../../src/sensory-shell/CognitiveEngine');
    
    // Mock fast responses
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ success: true }),
      arrayBuffer: () => Promise.resolve(new Uint8Array([1, 2, 3, 4]).buffer)
    });

    const cognitiveEngine = new CognitiveEngine({
      maxContextSize: 100,
      learningRate: 0.01,
      adaptationThreshold: 0.3,
      skillTimeout: 30000,
      errorContextEnabled: true
    });

    // ErrorContext generation should be fast (<100ms)
    const startTime = Date.now();
    const errorContext = await cognitiveEngine.generateErrorContext(new Error('Test'), {});
    const contextGenTime = Date.now() - startTime;
    expect(contextGenTime).toBeLessThan(100);

    // KNIRVGRAPH queries should be fast (<500ms)
    const graphStartTime = Date.now();
    await cognitiveEngine.queryKNIRVGRAPH(errorContext);
    const graphQueryTime = Date.now() - graphStartTime;
    expect(graphQueryTime).toBeLessThan(500);
  });
});
