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
  httpPath: '/mcp/capabilities',
  operationId: 'mcpListCapabilities',
};

export const tool: Tool = {
  name: 'retrieve_capabilities_mcp',
  description: 'Lists all registered capabilities',
  inputSchema: {
    type: 'object',
    properties: {
      owner: {
        type: 'string',
        description: 'Filter by owner address',
      },
      type: {
        type: 'string',
        description: 'Filter by capability type',
        enum: ['RESOURCE', 'TOOL', 'PROMPT', 'MEMORY_SERVICE'],
      },
    },
  },
};

export const handler = async (
  client: KnirvchainTransactionSDK,
  args: Record<string, unknown> | undefined,
) => {
  const body = args as any;
  return asTextContentResult(await client.mcp.retrieveCapabilities(body));
};

export default { metadata, tool, handler };
