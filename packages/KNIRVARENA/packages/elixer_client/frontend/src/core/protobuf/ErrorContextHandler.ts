/**
 * ErrorContext Handler for Phase 3.6 End-to-End Skill Invocation Lifecycle
 * 
 * Handles creation, serialization, and processing of ErrorContext protobuf messages
 * for KNIRVGRAPH discovery and skill invocation
 */

import pino from 'pino';
// Browser-compatible crypto implementation
const crypto = {
  createHash: (algorithm: string) => {
    if (algorithm !== 'sha256') {
      throw new Error(`Algorithm ${algorithm} not supported in browser`);
    }
    
    let buffer = '';
    const hashObject = {
      update: (data: string) => {
        buffer += data;
        return hashObject;
      },
      digest: (encoding: string) => {
        // Simple hash simulation for browser environment
        let hash = 0;
        for (let i = 0; i < buffer.length; i++) {
          const char = buffer.charCodeAt(i);
          hash = ((hash << 5) - hash) + char;
          hash = hash & hash; // Convert to 32bit integer
        }
        
        if (encoding === 'hex') {
          return Math.abs(hash).toString(16).padStart(64, '0');
        }
        return hash.toString();
      }
    };
    return hashObject;
  }
};

const logger = pino({ name: 'error-context-handler' });

// TypeScript interfaces matching the protobuf schema
export interface ErrorContext {
  // Agent Information
  agentId: string;
  agentVersion: string;
  baseModelId: string;

  // Environment Information
  os: string;
  architecture: string;
  runtimeEnvironment: string;

  // Error Details
  errorType: string;
  errorMessage: string;
  stackTrace?: string;
  sourceCodeSnippet?: string;

  // Task Context
  taskDescription: string;
  inputDataHash?: string;
  skillInvokedId?: string;

  // State & Metadata
  agentStateHash?: string;
  timestamp: Date;
  additionalContext?: Record<string, unknown>;
}

export interface ErrorClusterQueryRequest {
  errorContext: ErrorContext;
  maxResults?: number;
  similarityThreshold?: number;
}

export interface SkillNodeResult {
  skillUri: string;
  skillNodeId: string;
  clusterId: string;
  confidence: number;
  metadata?: Record<string, string>;
}

export interface ErrorCluster {
  clusterId: string;
  errorType: string;
  errorCount: number;
  averageSeverity: number;
  tags: string[];
  bountyAmount: number;
}

export interface ErrorClusterQueryResponse {
  status: 'QUERY_SUCCESS' | 'QUERY_FAILED' | 'QUERY_NO_MATCH' | 'QUERY_PARTIAL_MATCH';
  errorMessage?: string;
  skillNodeResult?: SkillNodeResult;
  similarClusters?: ErrorCluster[];
}

export interface ErrorNodeSubmissionRequest {
  errorContext: ErrorContext;
  bountyAmount: number;
  priority: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
}

export interface ErrorNodeSubmissionResponse {
  status: 'SUBMISSION_SUCCESS' | 'SUBMISSION_FAILED' | 'SUBMISSION_DUPLICATE' | 'SUBMISSION_INVALID';
  errorMessage?: string;
  errorNodeId?: string;
  clusterId?: string;
}

export class ErrorContextHandler {
  private initialized = false;

  async initialize(): Promise<void> {
    if (this.initialized) {
      return;
    }

    logger.info('Initializing ErrorContext handler...');
    this.initialized = true;
    logger.info('ErrorContext handler initialized successfully');
  }

  /**
   * Create an ErrorContext from an error and execution context
   */
  createErrorContext(
    error: Error,
    agentId: string,
    agentVersion: string,
    baseModelId: string,
    taskDescription: string,
    additionalContext?: Record<string, unknown>
  ): ErrorContext {
    const timestamp = new Date();
    
    // Generate hashes for input data and agent state
    const inputDataHash = this.generateInputDataHash(additionalContext?.inputData);
    const agentStateHash = this.generateAgentStateHash(additionalContext?.agentState);

    const errorContext: ErrorContext = {
      // Agent Information
      agentId,
      agentVersion,
      baseModelId,

      // Environment Information
      os: process.platform,
      architecture: process.arch,
      runtimeEnvironment: this.detectRuntimeEnvironment(),

      // Error Details
      errorType: error.constructor.name,
      errorMessage: error.message,
      stackTrace: error.stack,
      sourceCodeSnippet: this.extractSourceCodeSnippet(error.stack),

      // Task Context
      taskDescription,
      inputDataHash,
      skillInvokedId: additionalContext?.skillInvokedId as string | undefined,

      // State & Metadata
      agentStateHash,
      timestamp,
      additionalContext: this.sanitizeAdditionalContext(additionalContext)
    };

    logger.info({ 
      agentId, 
      errorType: errorContext.errorType,
      errorMessage: errorContext.errorMessage 
    }, 'Created ErrorContext');

    return errorContext;
  }

  /**
   * Serialize ErrorContext to protobuf bytes
   */
  async serializeErrorContext(errorContext: ErrorContext): Promise<Uint8Array> {
    try {
      // Convert to protobuf-compatible format
      const protobufData = {
        agent_id: errorContext.agentId,
        agent_version: errorContext.agentVersion,
        base_model_id: errorContext.baseModelId,
        os: errorContext.os,
        architecture: errorContext.architecture,
        runtime_environment: errorContext.runtimeEnvironment,
        error_type: errorContext.errorType,
        error_message: errorContext.errorMessage,
        stack_trace: errorContext.stackTrace || '',
        source_code_snippet: errorContext.sourceCodeSnippet || '',
        task_description: errorContext.taskDescription,
        input_data_hash: errorContext.inputDataHash || '',
        skill_invoked_id: errorContext.skillInvokedId || '',
        agent_state_hash: errorContext.agentStateHash || '',
        timestamp: {
          seconds: Math.floor(errorContext.timestamp.getTime() / 1000),
          nanos: (errorContext.timestamp.getTime() % 1000) * 1000000
        },
        additional_context: errorContext.additionalContext || {}
      };

      // For now, return JSON serialization as bytes
      // In production, this would use actual protobuf serialization
      const jsonString = JSON.stringify(protobufData);
      return new TextEncoder().encode(jsonString);

    } catch (error) {
      logger.error({ error }, 'Failed to serialize ErrorContext');
      throw new Error(`ErrorContext serialization failed: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  /**
   * Deserialize ErrorContext from protobuf bytes
   */
  async deserializeErrorContext(data: Uint8Array): Promise<ErrorContext> {
    try {
      // For now, use JSON deserialization
      // In production, this would use actual protobuf deserialization
      const jsonString = new TextDecoder().decode(data);
      const protobufData = JSON.parse(jsonString);

      const errorContext: ErrorContext = {
        agentId: protobufData.agent_id,
        agentVersion: protobufData.agent_version,
        baseModelId: protobufData.base_model_id,
        os: protobufData.os,
        architecture: protobufData.architecture,
        runtimeEnvironment: protobufData.runtime_environment,
        errorType: protobufData.error_type,
        errorMessage: protobufData.error_message,
        stackTrace: protobufData.stack_trace,
        sourceCodeSnippet: protobufData.source_code_snippet,
        taskDescription: protobufData.task_description,
        inputDataHash: protobufData.input_data_hash,
        skillInvokedId: protobufData.skill_invoked_id,
        agentStateHash: protobufData.agent_state_hash,
        timestamp: new Date(protobufData.timestamp.seconds * 1000 + protobufData.timestamp.nanos / 1000000),
        additionalContext: protobufData.additional_context
      };

      return errorContext;

    } catch (error) {
      logger.error({ error }, 'Failed to deserialize ErrorContext');
      throw new Error(`ErrorContext deserialization failed: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  /**
   * Generate a hash for input data
   */
  private generateInputDataHash(inputData?: unknown): string {
    if (!inputData) {
      return '';
    }

    try {
      const dataString = typeof inputData === 'string' ? inputData : JSON.stringify(inputData);
      return crypto.createHash('sha256').update(dataString).digest('hex');
    } catch (error) {
      logger.warn({ error }, 'Failed to generate input data hash');
      return '';
    }
  }

  /**
   * Generate a hash for agent state
   */
  private generateAgentStateHash(agentState?: unknown): string {
    if (!agentState) {
      return '';
    }

    try {
      const stateString = typeof agentState === 'string' ? agentState : JSON.stringify(agentState);
      return crypto.createHash('sha256').update(stateString).digest('hex');
    } catch (error) {
      logger.warn({ error }, 'Failed to generate agent state hash');
      return '';
    }
  }

  /**
   * Detect the runtime environment
   */
  private detectRuntimeEnvironment(): string {
    if (typeof window !== 'undefined') {
      return 'browser';
    } else if (typeof process !== 'undefined' && process.versions?.node) {
      return 'node';
    } else if (typeof WebAssembly !== 'undefined') {
      return 'wasm';
    } else {
      return 'unknown';
    }
  }

  /**
   * Extract source code snippet from stack trace
   */
  private extractSourceCodeSnippet(stackTrace?: string): string {
    if (!stackTrace) {
      return '';
    }

    try {
      // Extract the first meaningful line from stack trace
      const lines = stackTrace.split('\n');
      for (const line of lines) {
        if (line.includes('.ts:') || line.includes('.js:')) {
          return line.trim();
        }
      }
      return lines[1]?.trim() || '';
    } catch (error) {
      logger.warn({ error }, 'Failed to extract source code snippet');
      return '';
    }
  }

  /**
   * Sanitize additional context to remove sensitive data
   */
  private sanitizeAdditionalContext(context?: Record<string, unknown>): Record<string, unknown> {
    if (!context) {
      return {};
    }

    const sanitized = { ...context };
    
    // Remove sensitive fields
    const sensitiveFields = ['password', 'token', 'key', 'secret', 'credential'];
    for (const field of sensitiveFields) {
      if (field in sanitized) {
        delete sanitized[field];
      }
    }

    return sanitized;
  }
}
