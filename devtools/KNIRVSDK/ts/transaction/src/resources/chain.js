// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.
import { APIResource } from '../core/resource';
export class Chain extends APIResource {
    /**
     * Retrieves the current state of the blockchain including blocks, transaction
     * pool, and reflections
     */
    retrieve(options) {
        return this._client.get('/chain', options);
    }
}
