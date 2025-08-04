// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.
import { APIResource } from '../core/resource';
export class BlockResource extends APIResource {
    /**
     * Submits a new block to the blockchain
     */
    submit(body, options) {
        return this._client.post('/block', { body, ...options });
    }
}
