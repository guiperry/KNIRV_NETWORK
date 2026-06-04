import { APIResource } from '../core/resource';
import { APIPromise } from '../core/api-promise';
import { RequestOptions } from '../internal/request-options';
export declare class Health extends APIResource {
    /**
     * Checks the health status of the blockchain node
     */
    check(options?: RequestOptions): APIPromise<HealthCheckResponse>;
}
export interface HealthCheckResponse {
    /**
     * Blockchain subsystem status
     */
    blockchain?: boolean;
    /**
     * Database subsystem status
     */
    database?: boolean;
    /**
     * Additional status information
     */
    message?: string;
    /**
     * Network subsystem status
     */
    network?: boolean;
    /**
     * Overall health status
     */
    status?: 'healthy' | 'unhealthy';
}
export declare namespace Health {
    export { type HealthCheckResponse as HealthCheckResponse };
}
//# sourceMappingURL=health.d.ts.map