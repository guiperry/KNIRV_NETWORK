// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { asTextContentResult } from 'knirvchain-transaction-sdk-mcp/tools/types';

import { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { Metadata } from '../';
import KnirvchainTransactionSDK from 'knirvchain-transaction-sdk';

export const metadata: Metadata = {
  resource: 'mcp',
  operation: 'read',
  tags: [],
  httpMethod: 'get',
  httpPath: '/mcp/contexts',
  operationId: 'mcpListContextRecords',
};

export const tool: Tool = {
  name: 'retrieve_contexts_mcp',
  description: 'Lists all context records',
  inputSchema: {
    type: 'object',
    properties: {
      capabilityId: {
        type: 'string',
        description: 'Filter by capability ID',
      },
      initiator: {
        type: 'string',
        description: 'Filter by initiator address',
      },
      interactionType: {
        type: 'string',
        description: 'Filter by interaction type',
        enum: [
          'TOOL_INVOCATION',
          'PROMPT_USAGE',
          'RESOURCE_ACCESS',
          'PLUGIN_EXECUTION',
          'SAMPLING_REQUEST_SENT',
          'SAMPLING_RESPONSE_RECEIVED',
          'MEMORY_WRITE',
          'CAPABILITY_REGISTRATION',
        ],
      },
    },
  },
};

export const handler = async (
  client: KnirvchainTransactionSDK,
  args: Record<string, unknown> | undefined,
) => {
  const body = args as any;
  return asTextContentResult(await client.mcp.retrieveContexts(body));
};

export default { metadata, tool, handler };
