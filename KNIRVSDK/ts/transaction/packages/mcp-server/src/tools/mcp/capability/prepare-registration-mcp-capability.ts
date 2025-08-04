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
  httpPath: '/mcp/capability/prepare_registration',
  operationId: 'mcpPrepareCapabilityRegistration',
};

export const tool: Tool = {
  name: 'prepare_registration_mcp_capability',
  description:
    'Allows a client to get the necessary data (e.g., a hash or structured unsigned transaction)\nthat needs to be signed to register a new MCP capability.\nThe client signs this data locally and then submits the complete, signed transaction\nvia the general /transaction endpoint.\n',
  inputSchema: {
    type: 'object',
    properties: {
      fee: {
        type: 'integer',
        description: 'Network fee for the registration transaction.',
      },
      ownerAddress: {
        type: 'string',
        description: 'Address of the account that will own the capability.',
      },
      descriptor: {
        $ref: '#/$defs/capability_descriptor',
      },
      message: {
        type: 'string',
        description: 'Additional information',
      },
    },
    $defs: {
      capability_descriptor: {
        anyOf: [
          {
            allOf: [
              {
                $ref: '#/$defs/base_descriptor',
              },
            ],
          },
          {
            allOf: [
              {
                $ref: '#/$defs/base_descriptor',
              },
            ],
          },
          {
            allOf: [
              {
                $ref: '#/$defs/base_descriptor',
              },
            ],
          },
          {
            allOf: [
              {
                $ref: '#/$defs/base_descriptor',
              },
            ],
          },
        ],
      },
      base_descriptor: {
        type: 'object',
        properties: {
          id: {
            type: 'string',
            description: 'Unique identifier',
          },
          capabilityType: {
            type: 'string',
            description: 'Type of capability',
            enum: ['RESOURCE', 'TOOL', 'PROMPT', 'MEMORY_SERVICE'],
          },
          customMetadata: {
            type: 'object',
            description: 'Custom metadata',
          },
          description: {
            type: 'string',
            description: 'Description of the capability',
          },
          gasFeeNRN: {
            type: 'integer',
            description: 'NRN token gas fee for invoking/using this capability',
          },
          name: {
            type: 'string',
            description: 'Human-readable name',
          },
          owner: {
            type: 'string',
            description: 'Public key or address of the owner/root',
          },
          timestamp: {
            type: 'integer',
            description: 'Creation/registration timestamp',
          },
          version: {
            type: 'string',
            description: 'Version string',
          },
        },
        required: [],
      },
    },
  },
};

export const handler = async (
  client: KnirvchainTransactionSDK,
  args: Record<string, unknown> | undefined,
) => {
  const body = args as any;
  return asTextContentResult(await client.mcp.capability.prepareRegistration(body));
};

export default { metadata, tool, handler };
