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
  httpPath: '/mcp/capability/{capability_id}/invocations',
  operationId: 'mcpListCapabilityInvocations',
};

export const tool: Tool = {
  name: 'retrieve_invocations_mcp_capability',
  description: 'Lists all invocations of a specific capability',
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
  return asTextContentResult(await client.mcp.capability.retrieveInvocations(capability_id));
};

export default { metadata, tool, handler };
