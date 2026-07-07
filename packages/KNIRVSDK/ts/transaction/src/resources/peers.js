// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.
import { APIResource } from '../core/resource';
export class Peers extends APIResource {
    /**
     * Retrieves the list of connected peers
     */
    list(options) {
        return this._client.get('/peers', options);
    }
}
