import { APIResource } from '../core/resource';
import { APIPromise } from '../core/api-promise';
import { RequestOptions } from '../internal/request-options';
export declare class Ping extends APIResource {
    /**
     * Simple ping endpoint to check if the node is responsive
     *
     * @example
     * ```ts
     * const response = await client.ping.check();
     * ```
     */
    check(options?: RequestOptions): APIPromise<string>;
}
export type PingCheckResponse = string;
export declare namespace Ping {
    export { type PingCheckResponse as PingCheckResponse };
}
//# sourceMappingURL=ping.d.ts.map