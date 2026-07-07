// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { asTextContentResult } from 'knirvchain-transaction-sdk-mcp/tools/types';

import { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { Metadata } from '../';
import KnirvchainTransactionSDK from 'knirvchain-transaction-sdk';

export const metadata: Metadata = {
  resource: 'ping',
  operation: 'read',
  tags: [],
  httpMethod: 'get',
  httpPath: '/ping',
  operationId: 'ping',
};

export const tool: Tool = {
  name: 'check_ping',
  description: 'Simple ping endpoint to check if the node is responsive',
  inputSchema: {
    type: 'object',
    properties: {},
  },
};

export const handler = async (
  client: KnirvchainTransactionSDK,
  args: Record<string, unknown> | undefined,
) => {
  return asTextContentResult(await client.ping.check());
};

export default { metadata, tool, handler };
