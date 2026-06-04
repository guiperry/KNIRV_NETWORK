// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { APIResource } from '../core/resource';
import * as BlockAPI from './block';
import * as TransactionAPI from './transaction';
import { APIPromise } from '../core/api-promise';
import { RequestOptions } from '../internal/request-options';

export class Chain extends APIResource {
  /**
   * Retrieves the current state of the blockchain including blocks, transaction
   * pool, and reflections
   */
  retrieve(options?: RequestOptions): APIPromise<ChainRetrieveResponse> {
    return this._client.get('/chain', options);
  }
}

export interface ChainRetrieveResponse {
  blocks?: Array<BlockAPI.Block>;

  /**
   * The blockchain's address
   */
  chain_address?: string;

  /**
   * Unique identifier for the blockchain
   */
  chain_id?: string;

  /**
   * Map of reflection URLs
   */
  reflections?: Record<string, boolean>;

  transaction_pool?: Array<TransactionAPI.Transaction>;
}

export declare namespace Chain {
  export { type ChainRetrieveResponse as ChainRetrieveResponse };
}
