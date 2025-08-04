// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { asTextContentResult } from 'knirvchain-transaction-sdk-mcp/tools/types';

import { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { Metadata } from '../';
import KnirvchainTransactionSDK from 'knirvchain-transaction-sdk';

export const metadata: Metadata = {
  resource: 'info',
  operation: 'read',
  tags: [],
  httpMethod: 'get',
  httpPath: '/info',
  operationId: 'getServerInfo',
};

export const tool: Tool = {
  name: 'retrieve_info',
  description: 'Retrieves information about the blockchain node',
  inputSchema: {
    type: 'object',
    properties: {},
  },
};

export const handler = async (
  client: KnirvchainTransactionSDK,
  args: Record<string, unknown> | undefined,
) => {
  return asTextContentResult(await client.info.retrieve());
};

export default { metadata, tool, handler };
