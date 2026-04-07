// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.
import { APIResource } from '../core/resource';
export class UriGenerator extends APIResource {
    /**
     * Generates a new URI and announces it to the network
     */
    create(body, options) {
        return this._client.post('/uriGenerator', { body, ...options });
    }
}
