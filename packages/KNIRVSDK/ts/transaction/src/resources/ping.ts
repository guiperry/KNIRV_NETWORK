// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { APIResource } from '../core/resource';
import { APIPromise } from '../core/api-promise';
import { buildHeaders } from '../internal/headers';
import { RequestOptions } from '../internal/request-options';

export class Ping extends APIResource {
  /**
   * Simple ping endpoint to check if the node is responsive
   *
   * @example
   * ```ts
   * const response = await client.ping.check();
   * ```
   */
  check(options?: RequestOptions): APIPromise<string> {
    return this._client.get('/ping', {
      ...options,
      headers: buildHeaders([{ Accept: 'text/plain' }, options?.headers]),
    });
  }
}

export type PingCheckResponse = string;

export declare namespace Ping {
  export { type PingCheckResponse as PingCheckResponse };
}
