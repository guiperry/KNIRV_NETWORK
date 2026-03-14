// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { asTextContentResult } from 'knirvchain-transaction-sdk-mcp/tools/types';

import { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { Metadata } from '../';
import KnirvchainTransactionSDK from 'knirvchain-transaction-sdk';

export const metadata: Metadata = {
  resource: 'chain',
  operation: 'read',
  tags: [],
  httpMethod: 'get',
  httpPath: '/chain',
  operationId: 'getChain',
};

export const tool: Tool = {
  name: 'retrieve_chain',
  description:
    'Retrieves the current state of the blockchain including blocks, transaction pool, and reflections',
  inputSchema: {
    type: 'object',
    properties: {},
  },
};

export const handler = async (
  client: KnirvchainTransactionSDK,
  args: Record<string, unknown> | undefined,
) => {
  return asTextContentResult(await client.chain.retrieve());
};

export default { metadata, tool, handler };
