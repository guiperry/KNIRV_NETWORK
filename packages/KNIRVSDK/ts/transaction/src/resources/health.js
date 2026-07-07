// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.
import { APIResource } from '../core/resource';
export class Health extends APIResource {
    /**
     * Checks the health status of the blockchain node
     */
    check(options) {
        return this._client.get('/health', options);
    }
}
