// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.
import { APIResource } from '../core/resource';
export class TxnPool extends APIResource {
    /**
     * Retrieves the current transaction pool
     */
    retrieve(options) {
        return this._client.get('/txn_pool', options);
    }
}
