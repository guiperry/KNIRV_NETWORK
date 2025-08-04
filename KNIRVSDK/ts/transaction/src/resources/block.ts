// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { APIResource } from '../core/resource';
import * as TransactionAPI from './transaction';
import { APIPromise } from '../core/api-promise';
import { RequestOptions } from '../internal/request-options';

export class BlockResource extends APIResource {
  /**
   * Submits a new block to the blockchain
   */
  submit(body: BlockSubmitParams, options?: RequestOptions): APIPromise<BlockSubmitResponse> {
    return this._client.post('/block', { body, ...options });
  }
}

export interface Block {
  /**
   * Block number in the chain
   */
  block_number?: number;

  /**
   * Hash of the block
   */
  hash?: string;

  /**
   * Nonce used for mining
   */
  nonce?: number;

  /**
   * Hash of the previous block
   */
  prevHash?: string;

  /**
   * Address of the block proposer (miner/validator)
   */
  proposer_address?: string;

  /**
   * Unix timestamp when the block was created
   */
  timestamp?: number;

  /**
   * Transactions included in this block
   */
  transactions?: Array<TransactionAPI.Transaction>;
}

export interface BlockSubmitResponse {
  message?: string;

  success?: boolean;
}

export interface BlockSubmitParams {
  /**
   * Block number in the chain
   */
  block_number?: number;

  /**
   * Hash of the block
   */
  hash?: string;

  /**
   * Nonce used for mining
   */
  nonce?: number;

  /**
   * Hash of the previous block
   */
  prevHash?: string;

  /**
   * Address of the block proposer (miner/validator)
   */
  proposer_address?: string;

  /**
   * Unix timestamp when the block was created
   */
  timestamp?: number;

  /**
   * Transactions included in this block
   */
  transactions?: Array<TransactionAPI.Transaction>;
}

export declare namespace BlockResource {
  export {
    type Block as Block,
    type BlockSubmitResponse as BlockSubmitResponse,
    type BlockSubmitParams as BlockSubmitParams,
  };
}
