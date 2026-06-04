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
  httpPath: '/mcp/context/{context_id}',
  operationId: 'mcpGetContextRecord',
};

export const tool: Tool = {
  name: 'retrieve_mcp',
  description: 'Retrieves a specific context record by ID',
  inputSchema: {
    type: 'object',
    properties: {
      context_id: {
        type: 'string',
      },
    },
  },
};

export const handler = async (
  client: KnirvchainTransactionSDK,
  args: Record<string, unknown> | undefined,
) => {
  const { context_id, ...body } = args as any;
  return asTextContentResult(await client.mcp.retrieve(context_id));
};

export default { metadata, tool, handler };
