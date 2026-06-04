// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.
import { APIResource } from '../core/resource';
import { buildHeaders } from '../internal/headers';
export class Ping extends APIResource {
    /**
     * Simple ping endpoint to check if the node is responsive
     *
     * @example
     * ```ts
     * const response = await client.ping.check();
     * ```
     */
    check(options) {
        return this._client.get('/ping', {
            ...options,
            headers: buildHeaders([{ Accept: 'text/plain' }, options?.headers]),
        });
    }
}
