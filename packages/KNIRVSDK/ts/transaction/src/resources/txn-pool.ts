// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { APIResource } from '../core/resource';
import * as TransactionAPI from './transaction';
import { APIPromise } from '../core/api-promise';
import { RequestOptions } from '../internal/request-options';

export class TxnPool extends APIResource {
  /**
   * Retrieves the current transaction pool
   */
  retrieve(options?: RequestOptions): APIPromise<TxnPoolRetrieveResponse> {
    return this._client.get('/txn_pool', options);
  }
}

export type TxnPoolRetrieveResponse = Array<TransactionAPI.Transaction>;

export declare namespace TxnPool {
  export { type TxnPoolRetrieveResponse as TxnPoolRetrieveResponse };
}
