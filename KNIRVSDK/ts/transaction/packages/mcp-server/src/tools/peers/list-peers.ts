// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { asTextContentResult } from 'knirvchain-transaction-sdk-mcp/tools/types';

import { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { Metadata } from '../';
import KnirvchainTransactionSDK from 'knirvchain-transaction-sdk';

export const metadata: Metadata = {
  resource: 'peers',
  operation: 'read',
  tags: [],
  httpMethod: 'get',
  httpPath: '/peers',
  operationId: 'getPeers',
};

export const tool: Tool = {
  name: 'list_peers',
  description: 'Retrieves the list of connected peers',
  inputSchema: {
    type: 'object',
    properties: {},
  },
};

export const handler = async (
  client: KnirvchainTransactionSDK,
  args: Record<string, unknown> | undefined,
) => {
  return asTextContentResult(await client.peers.list());
};

export default { metadata, tool, handler };
