// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { asTextContentResult } from 'knirvchain-transaction-sdk-mcp/tools/types';

import { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { Metadata } from '../';
import KnirvchainTransactionSDK from 'knirvchain-transaction-sdk';

export const metadata: Metadata = {
  resource: 'uri_generator',
  operation: 'write',
  tags: [],
  httpMethod: 'post',
  httpPath: '/uriGenerator',
  operationId: 'generateAndAnnounceURI',
};

export const tool: Tool = {
  name: 'create_uri_generator',
  description: 'Generates a new URI and announces it to the network',
  inputSchema: {
    type: 'object',
    properties: {
      content_hash: {
        type: 'string',
        description: 'Hash of the content',
      },
      metadata: {
        type: 'object',
        description: 'Additional metadata',
      },
      owner: {
        type: 'string',
        description: 'Owner address',
      },
      resource_type: {
        type: 'string',
        description: 'Type of resource',
      },
    },
  },
};

export const handler = async (
  client: KnirvchainTransactionSDK,
  args: Record<string, unknown> | undefined,
) => {
  const body = args as any;
  return asTextContentResult(await client.uriGenerator.create(body));
};

export default { metadata, tool, handler };
