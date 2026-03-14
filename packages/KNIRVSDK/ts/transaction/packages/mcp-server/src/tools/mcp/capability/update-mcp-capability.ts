// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { asTextContentResult } from 'knirvchain-transaction-sdk-mcp/tools/types';

import { Tool } from '@modelcontextprotocol/sdk/types.js';
import type { Metadata } from '../../';
import KnirvchainTransactionSDK from 'knirvchain-transaction-sdk';

export const metadata: Metadata = {
  resource: 'mcp.capability',
  operation: 'write',
  tags: [],
  httpMethod: 'post',
  httpPath: '/mcp/capability/update',
  operationId: 'mcpUpdateCapability',
};

export const tool: Tool = {
  name: 'update_mcp_capability',
  description:
    'Submits a pre-signed transaction to update an existing capability.\nThe sender must be the owner of the capability.\nThe `Transaction.data` within the signed transaction should contain `MCPUpdateCapabilityData`.\n',
  inputSchema: {
    type: 'object',
    properties: {
      signedTransaction: {
        $ref: '#/$defs/transaction',
      },
    },
    $defs: {
      transaction: {
        type: 'object',
        properties: {
          id: {
            type: 'string',
            description: 'Transaction hash/ID.',
          },
          fee: {
            type: 'string',
            description: 'NRN token gas fee paid for this transaction',
          },
          from: {
            type: 'string',
            description: 'Sender address',
          },
          public_key: {
            type: 'string',
            description: 'Public key of the sender',
          },
          signature: {
            type: 'string',
            description: 'Cryptographic signature',
          },
          timestamp: {
            type: 'integer',
            description: 'Unix timestamp (nanoseconds) when the transaction was created',
          },
          type: {
            type: 'string',
            description: 'Transaction type for MCP transactions',
          },
          version: {
            type: 'integer',
          },
          data: {
            type: 'object',
            description:
              'Transaction payload, structure depends on transaction type. For MCP ops, this will be JSON of MCPRegisterCapabilityData, MCPInvokeCapabilityData, etc.',
          },
          status: {
            type: 'string',
            description: 'Transaction status (PENDING, SUCCESS, FAILED)',
          },
          to: {
            type: 'string',
            description: 'Recipient address',
          },
          transaction_hash: {
            type: 'string',
            description: 'Hash of the transaction',
          },
          value: {
            type: 'string',
            description: 'Amount of NRN transferred.',
          },
        },
        required: ['id', 'fee', 'from', 'public_key', 'signature', 'timestamp', 'type', 'version'],
      },
    },
  },
};

export const handler = async (
  client: KnirvchainTransactionSDK,
  args: Record<string, unknown> | undefined,
) => {
  const body = args as any;
  return asTextContentResult(await client.mcp.capability.update(body));
};

export default { metadata, tool, handler };
