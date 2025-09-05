/**
 * Tool Implementation Template: testTool
 * Generated from Go template: tool.go.template
 * Description: Test tool
 */

import { EventEmitter } from '../EventEmitter';
import { ToolParameter } from '../../../types/common';

export interface testToolParameters {
  {{#each parameters}}
  {{name}}{{#unless required}}?{{/unless}}: {{type}};
  {{/each}}
}

export interface testToolContext {
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

export interface testToolResult {
  success: boolean;
  result?: any;
  error?: string;
  executionTime: number;
  metadata?: Record<string, any>;
}

/**
 * testTool Tool Implementation
 */
export class testToolTool extends EventEmitter {
  private name = 'testTool';
  private description = 'Test tool';

  async execute(
    params: testToolParameters,
    context: testToolContext
  ): Promise<testToolResult> {
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
      return { result: "test" };

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

  getParameters(): ToolParameter[] {
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
export async function testTool(
  params: testToolParameters,
  context: testToolContext
): Promise<testToolResult> {
  const tool = new testToolTool();
  return await tool.execute(params, context);
}

export default testTool;
