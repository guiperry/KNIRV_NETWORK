// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { asTextContentResult } from 'knirvchain-transaction-sdk-mcp/tools/types';

import { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { Metadata } from '../../';
import KnirvchainTransactionSDK from 'knirvchain-transaction-sdk';

export const metadata: Metadata = {
  resource: 'mcp.capability',
  operation: 'read',
  tags: [],
  httpMethod: 'get',
  httpPath: '/mcp/capability/{capability_id}',
  operationId: 'mcpGetCapability',
};

export const tool: Tool = {
  name: 'retrieve_mcp_capability',
  description: 'Retrieves a specific capability by ID',
  inputSchema: {
    type: 'object',
    properties: {
      capability_id: {
        type: 'string',
      },
    },
  },
};

export const handler = async (
  client: KnirvchainTransactionSDK,
  args: Record<string, unknown> | undefined,
) => {
  const { capability_id, ...body } = args as any;
  return asTextContentResult(await client.mcp.capability.retrieve(capability_id));
};

export default { metadata, tool, handler };
