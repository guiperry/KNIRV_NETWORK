// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { APIResource } from '../core/resource';
import { APIPromise } from '../core/api-promise';
import { RequestOptions } from '../internal/request-options';

export class TransactionResource extends APIResource {
  /**
   * Submits a pre-signed transaction to the blockchain network. This endpoint is
   * used for various transaction types, including standard transfers and MCP
   * operations like capability registration (after preparation), invocation, or
   * updates. The `Transaction.type` field and `Transaction.data` structure will
   * determine how it's processed.
   *
   * @example
   * ```ts
   * const response = await client.transaction.submit({
   *   id: '0xabcdef123456...',
   *   fee: 'fee',
   *   from: 'from',
   *   public_key: 'public_key',
   *   signature: 'U3RhaW5sZXNzIHJvY2tz',
   *   timestamp: 0,
   *   type: 'type',
   *   version: 1,
   * });
   * ```
   */
  submit(body: TransactionSubmitParams, options?: RequestOptions): APIPromise<TransactionSubmitResponse> {
    return this._client.post('/transaction', { body, ...options });
  }
}

export interface Transaction {
  /**
   * Transaction hash/ID.
   */
  id: string;

  /**
   * NRN token gas fee paid for this transaction
   */
  fee: string;

  /**
   * Sender address
   */
  from: string;

  /**
   * Public key of the sender
   */
  public_key: string;

  /**
   * Cryptographic signature
   */
  signature: string;

  /**
   * Unix timestamp (nanoseconds) when the transaction was created
   */
  timestamp: number;

  /**
   * Transaction type for MCP transactions
   */
  type: string;

  version: number;

  /**
   * Transaction payload, structure depends on transaction type. For MCP ops, this
   * will be JSON of MCPRegisterCapabilityData, MCPInvokeCapabilityData, etc.
   */
  data?: Record<string, unknown>;

  /**
   * Transaction status (PENDING, SUCCESS, FAILED)
   */
  status?: string;

  /**
   * Recipient address
   */
  to?: string | null;

  /**
   * Hash of the transaction
   */
  transaction_hash?: string;

  /**
   * Amount of NRN transferred.
   */
  value?: string;
}

export interface TransactionSubmitResponse {
  message?: string;

  transactionHash?: string;
}

export interface TransactionSubmitParams {
  /**
   * Transaction hash/ID.
   */
  id: string;

  /**
   * NRN token gas fee paid for this transaction
   */
  fee: string;

  /**
   * Sender address
   */
  from: string;

  /**
   * Public key of the sender
   */
  public_key: string;

  /**
   * Cryptographic signature
   */
  signature: string;

  /**
   * Unix timestamp (nanoseconds) when the transaction was created
   */
  timestamp: number;

  /**
   * Transaction type for MCP transactions
   */
  type: string;

  version: number;

  /**
   * Transaction payload, structure depends on transaction type. For MCP ops, this
   * will be JSON of MCPRegisterCapabilityData, MCPInvokeCapabilityData, etc.
   */
  data?: Record<string, unknown>;

  /**
   * Transaction status (PENDING, SUCCESS, FAILED)
   */
  status?: string;

  /**
   * Recipient address
   */
  to?: string | null;

  /**
   * Hash of the transaction
   */
  transaction_hash?: string;

  /**
   * Amount of NRN transferred.
   */
  value?: string;
}

export declare namespace TransactionResource {
  export {
    type Transaction as Transaction,
    type TransactionSubmitResponse as TransactionSubmitResponse,
    type TransactionSubmitParams as TransactionSubmitParams,
  };
}
