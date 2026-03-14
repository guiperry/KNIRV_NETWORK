// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { APIResource } from '../../core/resource';
import * as CapabilityAPI from './capability';
import {
  BaseDescriptor,
  Capability,
  CapabilityDescriptor,
  CapabilityInvokeParams,
  CapabilityInvokeResponse,
  CapabilityPrepareRegistrationParams,
  CapabilityPrepareRegistrationResponse,
  CapabilityRetrieveInvocationsResponse,
  CapabilityUpdateParams,
  CapabilityUpdateResponse,
} from './capability';
import { APIPromise } from '../../core/api-promise';
import { RequestOptions } from '../../internal/request-options';
import { path } from '../../internal/utils/path';

export class Mcp extends APIResource {
  capability: CapabilityAPI.Capability = new CapabilityAPI.Capability(this._client);

  /**
   * Retrieves a specific context record by ID
   */
  retrieve(contextID: string, options?: RequestOptions): APIPromise<ContextRecord> {
    return this._client.get(path`/mcp/context/${contextID}`, options);
  }

  /**
   * Lists all registered capabilities
   */
  retrieveCapabilities(
    query: McpRetrieveCapabilitiesParams | null | undefined = {},
    options?: RequestOptions,
  ): APIPromise<McpRetrieveCapabilitiesResponse> {
    return this._client.get('/mcp/capabilities', { query, ...options });
  }

  /**
   * Lists all context records
   */
  retrieveContexts(
    query: McpRetrieveContextsParams | null | undefined = {},
    options?: RequestOptions,
  ): APIPromise<McpRetrieveContextsResponse> {
    return this._client.get('/mcp/contexts', { query, ...options });
  }
}

export interface ContextRecord {
  /**
   * Unique identifier for the context record
   */
  id?: string;

  /**
   * Identifier of the MCP capability involved
   */
  capabilityID?: string;

  createdAt?: string;

  /**
   * Primitive-specific information
   */
  details?: Record<string, unknown>;

  /**
   * Public key or address of the entity initiating the context
   */
  initiator?: string;

  /**
   * Optional hash of the input data to the interaction
   */
  inputHash?: string;

  /**
   * Type of interaction
   */
  interactionType?:
    | 'CAPABILITY_REGISTRATION'
    | 'CAPABILITY_INVOCATION'
    | 'CAPABILITY_UPDATE'
    | 'GENERAL_MESSAGE'
    | 'TOOL_INVOCATION'
    | 'PROMPT_USAGE'
    | 'RESOURCE_ACCESS'
    | 'PLUGIN_EXECUTION'
    | 'SAMPLING_REQUEST_SENT'
    | 'SAMPLING_RESPONSE_RECEIVED'
    | 'MEMORY_WRITE';

  /**
   * Optional hash of the output data from the interaction
   */
  outputHash?: string;

  /**
   * Cryptographic signature of the ContextRecord content by the initiator
   */
  signature?: string;

  status?: 'pending' | 'success' | 'failed' | 'processing';

  /**
   * Timestamp of the event
   */
  timestamp?: number;

  updatedAt?: string;
}

export type McpRetrieveCapabilitiesResponse = Array<CapabilityAPI.CapabilityDescriptor>;

export type McpRetrieveContextsResponse = Array<ContextRecord>;

export interface McpRetrieveCapabilitiesParams {
  /**
   * Filter by owner address
   */
  owner?: string;

  /**
   * Filter by capability type
   */
  type?: 'RESOURCE' | 'TOOL' | 'PROMPT' | 'MEMORY_SERVICE';
}

export interface McpRetrieveContextsParams {
  /**
   * Filter by capability ID
   */
  capabilityId?: string;

  /**
   * Filter by initiator address
   */
  initiator?: string;

  /**
   * Filter by interaction type
   */
  interactionType?:
    | 'TOOL_INVOCATION'
    | 'PROMPT_USAGE'
    | 'RESOURCE_ACCESS'
    | 'PLUGIN_EXECUTION'
    | 'SAMPLING_REQUEST_SENT'
    | 'SAMPLING_RESPONSE_RECEIVED'
    | 'MEMORY_WRITE'
    | 'CAPABILITY_REGISTRATION';
}

Mcp.Capability = Capability;

export declare namespace Mcp {
  export {
    type ContextRecord as ContextRecord,
    type McpRetrieveCapabilitiesResponse as McpRetrieveCapabilitiesResponse,
    type McpRetrieveContextsResponse as McpRetrieveContextsResponse,
    type McpRetrieveCapabilitiesParams as McpRetrieveCapabilitiesParams,
    type McpRetrieveContextsParams as McpRetrieveContextsParams,
  };

  export {
    Capability as Capability,
    type BaseDescriptor as BaseDescriptor,
    type CapabilityDescriptor as CapabilityDescriptor,
    type CapabilityUpdateResponse as CapabilityUpdateResponse,
    type CapabilityInvokeResponse as CapabilityInvokeResponse,
    type CapabilityPrepareRegistrationResponse as CapabilityPrepareRegistrationResponse,
    type CapabilityRetrieveInvocationsResponse as CapabilityRetrieveInvocationsResponse,
    type CapabilityUpdateParams as CapabilityUpdateParams,
    type CapabilityInvokeParams as CapabilityInvokeParams,
    type CapabilityPrepareRegistrationParams as CapabilityPrepareRegistrationParams,
  };
}
