import { APIResource } from '../core/resource';
import { APIPromise } from '../core/api-promise';
import { RequestOptions } from '../internal/request-options';
export declare class Peers extends APIResource {
    /**
     * Retrieves the list of connected peers
     */
    list(options?: RequestOptions): APIPromise<PeerListResponse>;
}
export type PeerListResponse = Array<PeerListResponse.PeerListResponseItem>;
export declare namespace PeerListResponse {
    interface PeerListResponseItem {
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
//# sourceMappingURL=peers.d.ts.map