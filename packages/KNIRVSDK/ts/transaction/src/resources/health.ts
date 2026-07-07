// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { APIResource } from '../core/resource';
import { APIPromise } from '../core/api-promise';
import { RequestOptions } from '../internal/request-options';

export class Health extends APIResource {
  /**
   * Checks the health status of the blockchain node
   */
  check(options?: RequestOptions): APIPromise<HealthCheckResponse> {
    return this._client.get('/health', options);
  }
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
