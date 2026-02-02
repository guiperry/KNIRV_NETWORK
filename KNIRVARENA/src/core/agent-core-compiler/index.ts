/**
 * Agent-Core Compiler Module
 * 
 * TypeScript-written agent-core compiler that creates complete agent.wasm files
 * with embedded cognitive capabilities
 */

export { AgentCoreCompiler } from './src/AgentCoreCompiler';

export interface CompilationConfig {
  agentId: string;
  templates: string[];
  optimizationLevel?: 'O0' | 'O1' | 'O2' | 'O3';
  targetPlatform?: 'web' | 'node' | 'universal';
  enableDebug?: boolean;
}

export interface CompilationMetrics {
  compilationTime: number;
  wasmSize: number;
  optimizationLevel: string;
  templatesUsed: string[];
  memoryUsage: number;
}

export interface TemplateValidationResult {
  isValid: boolean;
  errors: string[];
  warnings: string[];
}
