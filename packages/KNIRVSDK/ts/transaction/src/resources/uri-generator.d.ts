import { APIResource } from '../core/resource';
import { APIPromise } from '../core/api-promise';
import { RequestOptions } from '../internal/request-options';
export declare class UriGenerator extends APIResource {
    /**
     * Generates a new URI and announces it to the network
     */
    create(body: UriGeneratorCreateParams, options?: RequestOptions): APIPromise<UriGeneratorCreateResponse>;
}
export interface UriGeneratorCreateResponse {
    /**
     * Additional information
     */
    message?: string;
    /**
     * Resource ID
     */
    resource_id?: string;
    /**
     * Whether the operation was successful
     */
    success?: boolean;
    /**
     * Generated URI
     */
    uri?: string;
}
export interface UriGeneratorCreateParams {
    /**
     * Hash of the content
     */
    content_hash?: string;
    /**
     * Additional metadata
     */
    metadata?: Record<string, unknown>;
    /**
     * Owner address
     */
    owner?: string;
    /**
     * Type of resource
     */
    resource_type?: string;
}
export declare namespace UriGenerator {
    export { type UriGeneratorCreateResponse as UriGeneratorCreateResponse, type UriGeneratorCreateParams as UriGeneratorCreateParams, };
}
//# sourceMappingURL=uri-generator.d.ts.map