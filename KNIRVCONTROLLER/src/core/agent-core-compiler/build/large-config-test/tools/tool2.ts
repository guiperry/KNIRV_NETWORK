/**
 * Tool Implementation Template: tool2
 * Generated from Go template: tool.go.template
 * Description: Test tool 2
 */

import { EventEmitter } from '../EventEmitter';

export interface tool2Parameters {
  {{#each parameters}}
  {{name}}{{#unless required}}?{{/unless}}: {{type}};
  {{/each}}
}

export interface tool2Context {
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

export interface tool2Result {
  success: boolean;
  result?: any;
  error?: string;
  executionTime: number;
  metadata?: Record<string, any>;
}

/**
 * tool2 Tool Implementation
 */
export class tool2Tool extends EventEmitter {
  private name = 'tool2';
  private description = 'Test tool 2';

  async execute(
    params: tool2Parameters,
    context: tool2Context
  ): Promise<tool2Result> {
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
      return { result: "Tool 2 result: " + parameters.input };

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
export async function tool2(
  params: tool2Parameters,
  context: tool2Context
): Promise<tool2Result> {
  const tool = new tool2Tool();
  return await tool.execute(params, context);
}

export default tool2;
