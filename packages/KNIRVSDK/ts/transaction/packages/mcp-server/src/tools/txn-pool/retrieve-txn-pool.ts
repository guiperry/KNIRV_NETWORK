// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { asTextContentResult } from 'knirvchain-transaction-sdk-mcp/tools/types';

import { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { Metadata } from '../';
import KnirvchainTransactionSDK from 'knirvchain-transaction-sdk';

export const metadata: Metadata = {
  resource: 'txn_pool',
  operation: 'read',
  tags: [],
  httpMethod: 'get',
  httpPath: '/txn_pool',
  operationId: 'getTransactionPool',
};

export const tool: Tool = {
  name: 'retrieve_txn_pool',
  description: 'Retrieves the current transaction pool',
  inputSchema: {
    type: 'object',
    properties: {},
  },
};

export const handler = async (
  client: KnirvchainTransactionSDK,
  args: Record<string, unknown> | undefined,
) => {
  return asTextContentResult(await client.txnPool.retrieve());
};

export default { metadata, tool, handler };
