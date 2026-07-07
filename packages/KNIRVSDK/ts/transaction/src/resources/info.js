// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.
import { APIResource } from '../core/resource';
export class Info extends APIResource {
    /**
     * Retrieves information about the blockchain node
     */
    retrieve(options) {
        return this._client.get('/info', options);
    }
}
