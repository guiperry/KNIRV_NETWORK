/**
 * ErrorContext Manager for CORTEX Agents
 * 
 * Manages error context creation and skill discovery for CORTEX agents
 * Implements Phase 3.6 End-to-End Skill Invocation Lifecycle
 */

import pino from 'pino';
import { ErrorContextHandler, ErrorContext, ErrorClusterQueryRequest, ErrorClusterQueryResponse, ErrorNodeSubmissionRequest, ErrorNodeSubmissionResponse } from '../protobuf/ErrorContextHandler';

const logger = pino({ name: 'error-context-manager' });

export interface AgentConfiguration {
  agentId: string;
  agentVersion: string;
  baseModelId: string;
  knirvgraphEndpoint: string;
  knirvRouterEndpoint: string;
  nrnWalletAddress?: string;
}

export interface SkillDiscoveryResult {
  skillFound: boolean;
  skillUri?: string;
  skillNodeId?: string;
  clusterId?: string;
  confidence?: number;
  errorNodeId?: string; // If new error was submitted
}

export interface SkillInvocationRequest {
  skillUri: string;
  agentId: string;
  nrnToken: string;
  parameters?: Record<string, unknown>;
}

export interface SkillInvocationResult {
  success: boolean;
  skillData?: unknown;
  errorMessage?: string;
  invocationId?: string;
}

export class ErrorContextManager {
  private errorContextHandler: ErrorContextHandler;
  private config: AgentConfiguration;
  private initialized = false;

  constructor(config: AgentConfiguration) {
    this.config = config;
    this.errorContextHandler = new ErrorContextHandler();
  }

  async initialize(): Promise<void> {
    if (this.initialized) {
      return;
    }

    logger.info({ agentId: this.config.agentId }, 'Initializing ErrorContext manager...');
    
    await this.errorContextHandler.initialize();
    
    this.initialized = true;
    logger.info({ agentId: this.config.agentId }, 'ErrorContext manager initialized successfully');
  }

  /**
   * Handle an error by creating context and discovering skills
   * This is the main entry point for Phase 3.6 error handling
   */
  async handleError(
    error: Error,
    taskDescription: string,
    additionalContext?: Record<string, unknown>
  ): Promise<SkillDiscoveryResult> {
    logger.info({
      agentId: this.config.agentId,
      errorType: error.constructor.name,
      taskDescription
    }, 'Handling error and discovering skills');

    try {
      // Step 1: Create ErrorContext
      const errorContext = this.errorContextHandler.createErrorContext(
        error,
        this.config.agentId,
        this.config.agentVersion,
        this.config.baseModelId,
        taskDescription,
        additionalContext
      );

      // Step 2: Query KNIRVGRAPH for similar error clusters
      const discoveryResult = await this.discoverSkillForError(errorContext);

      logger.info({
        agentId: this.config.agentId,
        skillFound: discoveryResult.skillFound,
        skillUri: discoveryResult.skillUri
      }, 'Error handling completed');

      return discoveryResult;

    } catch (handlingError) {
      logger.error({
        error: handlingError,
        agentId: this.config.agentId
      }, 'Failed to handle error');

      // Check if this is a network error or critical failure that should be propagated
      if (handlingError instanceof Error) {
        if (handlingError.message.includes('Network error') ||
            handlingError.message.includes('Invalid JSON') ||
            handlingError.message.includes('fetch')) {
          // Re-throw network and parsing errors for proper error handling
          throw handlingError;
        }
      }

      return {
        skillFound: false,
        confidence: 0
      };
    }
  }

  /**
   * Discover skills for an error through KNIRVGRAPH
   * Implements Phase 1 of the skill invocation lifecycle (Discovery)
   */
  async discoverSkillForError(errorContext: ErrorContext): Promise<SkillDiscoveryResult> {
    try {
      // Query KNIRVGRAPH for matching error clusters
      const queryRequest: ErrorClusterQueryRequest = {
        errorContext,
        maxResults: 5,
        similarityThreshold: 0.7
      };

      const queryResponse = await this.queryKNIRVGraph(queryRequest);

      if (queryResponse.status === 'QUERY_SUCCESS' && queryResponse.skillNodeResult) {
        // Skill found - return the skill URI
        return {
          skillFound: true,
          skillUri: queryResponse.skillNodeResult.skillUri,
          skillNodeId: queryResponse.skillNodeResult.skillNodeId,
          clusterId: queryResponse.skillNodeResult.clusterId,
          confidence: queryResponse.skillNodeResult.confidence
        };
      } else if (queryResponse.status === 'QUERY_NO_MATCH') {
        // No match found - submit new error node
        const submissionResult = await this.submitNewErrorNode(errorContext);
        
        if (submissionResult.status === 'SUBMISSION_SUCCESS') {
          return {
            skillFound: false,
            errorNodeId: submissionResult.errorNodeId,
            clusterId: submissionResult.clusterId
          };
        } else {
          throw new Error(`Failed to submit error node: ${submissionResult.errorMessage}`);
        }
      } else {
        throw new Error(`KNIRVGRAPH query failed: ${queryResponse.errorMessage}`);
      }

    } catch (error) {
      logger.error({ error, agentId: this.config.agentId }, 'Skill discovery failed');
      throw error;
    }
  }

  /**
   * Invoke a skill through KNIRVROUTER
   * Implements Phase 2 of the skill invocation lifecycle (Invocation)
   */
  async invokeSkill(
    skillUri: string,
    nrnToken: string,
    parameters?: Record<string, unknown>
  ): Promise<SkillInvocationResult> {
    logger.info({ 
      agentId: this.config.agentId,
      skillUri,
      hasNrnToken: !!nrnToken 
    }, 'Invoking skill through KNIRVROUTER');

    try {
      const invocationRequest: SkillInvocationRequest = {
        skillUri,
        agentId: this.config.agentId,
        nrnToken,
        parameters
      };

      const result = await this.callKNIRVRouter(invocationRequest);

      logger.info({ 
        agentId: this.config.agentId,
        skillUri,
        success: result.success 
      }, 'Skill invocation completed');

      return result;

    } catch (error) {
      logger.error({ 
        error,
        agentId: this.config.agentId,
        skillUri 
      }, 'Skill invocation failed');
      
      return {
        success: false,
        errorMessage: error instanceof Error ? error.message : String(error)
      };
    }
  }

  /**
   * Complete end-to-end skill invocation lifecycle
   * Combines error handling, skill discovery, and skill invocation
   */
  async handleErrorAndInvokeSkill(
    error: Error,
    taskDescription: string,
    nrnToken: string,
    additionalContext?: Record<string, unknown>
  ): Promise<{ discoveryResult: SkillDiscoveryResult; invocationResult?: SkillInvocationResult }> {
    logger.info({ 
      agentId: this.config.agentId,
      taskDescription 
    }, 'Starting end-to-end skill invocation lifecycle');

    // Step 1: Handle error and discover skill
    const discoveryResult = await this.handleError(error, taskDescription, additionalContext);

    if (!discoveryResult.skillFound || !discoveryResult.skillUri) {
      logger.info({ 
        agentId: this.config.agentId,
        errorNodeId: discoveryResult.errorNodeId 
      }, 'No skill found - error submitted for future resolution');
      
      return { discoveryResult };
    }

    // Step 2: Invoke the discovered skill
    const invocationResult = await this.invokeSkill(
      discoveryResult.skillUri,
      nrnToken,
      additionalContext?.parameters as Record<string, unknown> | undefined
    );

    logger.info({ 
      agentId: this.config.agentId,
      skillFound: discoveryResult.skillFound,
      skillInvoked: invocationResult.success 
    }, 'End-to-end skill invocation lifecycle completed');

    return { discoveryResult, invocationResult };
  }

  /**
   * Query KNIRVGRAPH for error clusters
   */
  private async queryKNIRVGraph(request: ErrorClusterQueryRequest): Promise<ErrorClusterQueryResponse> {
    try {
      const response = await fetch(`${this.config.knirvgraphEndpoint}/api/error-clusters/query`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(request)
      });

      if (!response.ok) {
        throw new Error(`KNIRVGRAPH query failed: ${response.status} ${response.statusText}`);
      }

      return await response.json();

    } catch (error) {
      logger.error({ error }, 'Failed to query KNIRVGRAPH');
      throw error;
    }
  }

  /**
   * Submit new error node to KNIRVGRAPH
   */
  private async submitNewErrorNode(errorContext: ErrorContext): Promise<ErrorNodeSubmissionResponse> {
    try {
      const submissionRequest: ErrorNodeSubmissionRequest = {
        errorContext,
        bountyAmount: 1000000, // 1 NRN default bounty
        priority: 'MEDIUM'
      };

      const response = await fetch(`${this.config.knirvgraphEndpoint}/api/error-nodes/submit`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(submissionRequest)
      });

      if (!response.ok) {
        throw new Error(`Error node submission failed: ${response.status} ${response.statusText}`);
      }

      return await response.json();

    } catch (error) {
      logger.error({ error }, 'Failed to submit error node');
      throw error;
    }
  }

  /**
   * Call KNIRVROUTER WASM for skill invocation
   */
  private async callKNIRVRouter(request: SkillInvocationRequest): Promise<SkillInvocationResult> {
    try {
      // Use the WASM endpoint instead of the deprecated Go endpoint
      const response = await fetch(`${this.config.knirvRouterEndpoint}/wasm/invoke`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-KNIRV-Engine': 'wasm',
          'X-KNIRV-Version': '1.0.0'
        },
        body: JSON.stringify({
          invocation_id: `inv_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
          agent_id: request.agentId,
          skill_uri: request.skillUri,
          nrn_token: request.nrnToken,
          parameters: request.parameters || {},
          priority: 'normal',
          timestamp: Date.now()
        })
      });

      if (!response.ok) {
        throw new Error(`KNIRVROUTER WASM invocation failed: ${response.status} ${response.statusText}`);
      }

      const result = await response.json();

      // Log WASM response headers for debugging
      const wasmHeaders = {
        responseFormat: response.headers.get('X-KNIRV-Response-Format'),
        invocationId: response.headers.get('X-KNIRV-Invocation-ID'),
        status: response.headers.get('X-KNIRV-Status'),
        engine: response.headers.get('X-KNIRV-Engine')
      };

      logger.info({
        invocationId: result.invocation_id,
        wasmHeaders,
        executionTime: result.execution_time
      }, 'WASM skill invocation completed');

      return {
        success: result.status === 'SUCCESS',
        skillData: result.skill_data,
        errorMessage: result.error_message,
        invocationId: result.invocation_id
      };

    } catch (error) {
      logger.error({ error }, 'Failed to call KNIRVROUTER WASM');
      throw error;
    }
  }
}
