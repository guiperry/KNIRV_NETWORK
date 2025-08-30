/**
 * Tool Implementation Template: tool5
 * Generated from Go template: tool.go.template
 * Description: Test tool 5
 */

import { EventEmitter } from '../EventEmitter';
import { ToolContext, ToolResult } from '../../../../types/common';

export interface tool5Parameters {
  {{#each parameters}}
  {{name}}{{#unless required}}?{{/unless}}: {{type}};
  {{/each}}
}

export interface tool5Context extends ToolContext {
  // Tool-specific context extensions can be added here
}

export interface tool5Result extends ToolResult {
  // Tool-specific result extensions can be added here
}

/**
 * tool5 Tool Implementation
 */
export class tool5Tool extends EventEmitter {
  private name = 'tool5';
  private description = 'Test tool 5';

  async execute(
    params: tool5Parameters,
    context: tool5Context
  ): Promise<tool5Result> {
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
      return { result: "Tool 5 result: " + parameters.input };

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
export async function tool5(
  params: tool5Parameters,
  context: tool5Context
): Promise<tool5Result> {
  const tool = new tool5Tool();
  return await tool.execute(params, context);
}

export default tool5;
