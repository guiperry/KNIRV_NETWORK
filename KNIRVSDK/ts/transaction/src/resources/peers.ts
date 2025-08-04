// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { APIResource } from '../core/resource';
import { APIPromise } from '../core/api-promise';
import { RequestOptions } from '../internal/request-options';

export class Peers extends APIResource {
  /**
   * Retrieves the list of connected peers
   */
  list(options?: RequestOptions): APIPromise<PeerListResponse> {
    return this._client.get('/peers', options);
  }
}

export type PeerListResponse = Array<PeerListResponse.PeerListResponseItem>;

export namespace PeerListResponse {
  export interface PeerListResponseItem {
    /**
     * Peer ID
     */
    id?: string;

    /**
     * Peer address
     */
    address?: string;

    /**
     * Unix timestamp when the peer connected
     */
    connected_since?: number;
  }
}

export declare namespace Peers {
  export { type PeerListResponse as PeerListResponse };
}
