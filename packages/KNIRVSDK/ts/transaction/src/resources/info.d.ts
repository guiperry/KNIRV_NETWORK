import { APIResource } from '../core/resource';
import { APIPromise } from '../core/api-promise';
import { RequestOptions } from '../internal/request-options';
export declare class Info extends APIResource {
    /**
     * Retrieves information about the blockchain node
     */
    retrieve(options?: RequestOptions): APIPromise<InfoRetrieveResponse>;
}
export interface InfoRetrieveResponse {
    /**
     * Current blockchain height
     */
    chain_height?: number;
    /**
     * Whether the node is currently mining
     */
    is_mining?: boolean;
    /**
     * Unique identifier for the node
     */
    node_id?: string;
    /**
     * Number of connected peers
     */
    peer_count?: number;
    /**
     * Node uptime in seconds
     */
    uptime?: number;
    /**
     * Software version
     */
    version?: string;
    /**
     * Node's wallet address
     */
    wallet_address?: string;
}
export declare namespace Info {
    export { type InfoRetrieveResponse as InfoRetrieveResponse };
}
//# sourceMappingURL=info.d.ts.map