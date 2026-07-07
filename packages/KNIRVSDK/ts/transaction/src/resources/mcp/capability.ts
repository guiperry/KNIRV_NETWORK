// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { APIResource } from '../../core/resource';
import * as CapabilityAPI from './capability';
import * as TransactionAPI from '../transaction';
import * as McpAPI from './mcp';
import { APIPromise } from '../../core/api-promise';
import { RequestOptions } from '../../internal/request-options';
import { path } from '../../internal/utils/path';

export class Capability extends APIResource {
  /**
   * Retrieves a specific capability by ID
   */
  retrieve(capabilityID: string, options?: RequestOptions): APIPromise<CapabilityDescriptor> {
    return this._client.get(path`/mcp/capability/${capabilityID}`, options);
  }

  /**
   * Submits a pre-signed transaction to update an existing capability. The sender
   * must be the owner of the capability. The `Transaction.data` within the signed
   * transaction should contain `MCPUpdateCapabilityData`.
   */
  update(body: CapabilityUpdateParams, options?: RequestOptions): APIPromise<CapabilityUpdateResponse> {
    return this._client.post('/mcp/capability/update', { body, ...options });
  }

  /**
   * Submits a pre-signed transaction to invoke an existing capability. The
   * `Transaction.data` within the signed transaction should contain
   * `MCPInvokeCapabilityData`.
   */
  invoke(body: CapabilityInvokeParams, options?: RequestOptions): APIPromise<CapabilityInvokeResponse> {
    return this._client.post('/mcp/capability/invoke', { body, ...options });
  }

  /**
   * Allows a client to get the necessary data (e.g., a hash or structured unsigned
   * transaction) that needs to be signed to register a new MCP capability. The
   * client signs this data locally and then submits the complete, signed transaction
   * via the general /transaction endpoint.
   */
  prepareRegistration(
    body: CapabilityPrepareRegistrationParams,
    options?: RequestOptions,
  ): APIPromise<CapabilityPrepareRegistrationResponse> {
    return this._client.post('/mcp/capability/prepare_registration', { body, ...options });
  }

  /**
   * Lists all invocations of a specific capability
   */
  retrieveInvocations(
    capabilityID: string,
    options?: RequestOptions,
  ): APIPromise<CapabilityRetrieveInvocationsResponse> {
    return this._client.get(path`/mcp/capability/${capabilityID}/invocations`, options);
  }
}

export interface BaseDescriptor {
  /**
   * Unique identifier
   */
  id?: string;

  /**
   * Type of capability
   */
  capabilityType?: 'RESOURCE' | 'TOOL' | 'PROMPT' | 'MEMORY_SERVICE';

  /**
   * Custom metadata
   */
  customMetadata?: Record<string, unknown>;

  /**
   * Description of the capability
   */
  description?: string;

  /**
   * NRN token gas fee for invoking/using this capability
   */
  gasFeeNRN?: number;

  /**
   * Human-readable name
   */
  name?: string;

  /**
   * Public key or address of the owner/root
   */
  owner?: string;

  /**
   * Creation/registration timestamp
   */
  timestamp?: number;

  /**
   * Version string
   */
  version?: string;
}

export type CapabilityDescriptor =
  | CapabilityDescriptor.ResourceDescriptor
  | CapabilityDescriptor.ToolDescriptor
  | CapabilityDescriptor.PromptDescriptor
  | CapabilityDescriptor.MemoryServiceDescriptor;

export namespace CapabilityDescriptor {
  export interface ResourceDescriptor extends CapabilityAPI.BaseDescriptor {
    /**
     * SHA256 hash of the Capability package
     */
    contentHash?: string;

    /**
     * Specific kind of resource
     */
    resourceType?:
      | 'FILE'
      | 'API'
      | 'PLUGIN'
      | 'GENERATED_DOCUMENT'
      | 'DATASET'
      | 'MODEL_ARTIFACT'
      | 'SERVICE';

    schema?: ResourceDescriptor.Schema;
  }

  export namespace ResourceDescriptor {
    export interface Schema {
      /**
       * For APIs - endpoint, auth details, etc.
       */
      accessInfo?: Record<string, unknown>;

      /**
       * Relative path to the main executable file within the downloaded package
       */
      executableFile?: string;

      /**
       * Array of URIs where the Capability package can be downloaded
       */
      locationHints?: Array<string>;

      /**
       * Relative path to the manifest file within the downloaded package
       */
      manifestFile?: string;

      /**
       * Optional hint for plugin's output/config directory
       */
      outputDirectoryHint?: string;

      /**
       * Brief, human-readable summary of the plugin's purpose
       */
      summary?: string;
    }
  }

  export interface ToolDescriptor extends CapabilityAPI.BaseDescriptor {
    /**
     * Reference to a Plugin, API endpoint, or embedded logic
     */
    executionPointer?: string;

    /**
     * JSON Schema for tool input
     */
    inputSchemaJSON?: string;

    /**
     * JSON Schema for tool output
     */
    outputSchemaJSON?: string;
  }

  export interface PromptDescriptor extends CapabilityAPI.BaseDescriptor {
    /**
     * JSON Schema for template parameters
     */
    parametersSchemaJSON?: string;

    /**
     * The prompt template string
     */
    template?: string;
  }

  export interface MemoryServiceDescriptor extends CapabilityAPI.BaseDescriptor {
    /**
     * Describes the structure/ontology of the memory graph if applicable
     */
    graphSchema?: string;
  }
}

export interface CapabilityUpdateResponse {
  message?: string;

  transactionHash?: string;
}

export interface CapabilityInvokeResponse {
  contextID?: string;

  message?: string;

  transactionHash?: string;
}

export interface CapabilityPrepareRegistrationResponse {
  /**
   * A server-side hash identifying this pending registration preparation.
   */
  pendingTransactionHash: string;

  /**
   * A structured representation of the unsigned transaction the client is expected
   * to form and sign.
   */
  unsignedTransaction: CapabilityPrepareRegistrationResponse.UnsignedTransaction;
}

export namespace CapabilityPrepareRegistrationResponse {
  /**
   * A structured representation of the unsigned transaction the client is expected
   * to form and sign.
   */
  export interface UnsignedTransaction {
    /**
     * Base64 encoded JSON bytes of MCPRegisterCapabilityData.
     */
    data?: string;

    fee?: string;

    from?: string;

    timestamp?: number;

    to?: string | null;

    type?: string;

    /**
     * Hash of the payload that needs to be signed.
     */
    unsignedTransactionPayloadHash?: string;

    value?: string;
  }
}

export type CapabilityRetrieveInvocationsResponse = Array<McpAPI.ContextRecord>;

export interface CapabilityUpdateParams {
  signedTransaction: TransactionAPI.Transaction;
}

export interface CapabilityInvokeParams {
  signedTransaction: TransactionAPI.Transaction;
}

export interface CapabilityPrepareRegistrationParams {
  /**
   * Network fee for the registration transaction.
   */
  fee: number;

  /**
   * Address of the account that will own the capability.
   */
  ownerAddress: string;

  descriptor?: CapabilityDescriptor;

  /**
   * Additional information
   */
  message?: string;
}

export declare namespace Capability {
  export {
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
