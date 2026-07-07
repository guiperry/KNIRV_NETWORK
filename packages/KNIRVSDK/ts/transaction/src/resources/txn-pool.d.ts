import { APIResource } from '../core/resource';
import * as TransactionAPI from './transaction';
import { APIPromise } from '../core/api-promise';
import { RequestOptions } from '../internal/request-options';
export declare class TxnPool extends APIResource {
    /**
     * Retrieves the current transaction pool
     */
    retrieve(options?: RequestOptions): APIPromise<TxnPoolRetrieveResponse>;
}
export type TxnPoolRetrieveResponse = Array<TransactionAPI.Transaction>;
export declare namespace TxnPool {
    export { type TxnPoolRetrieveResponse as TxnPoolRetrieveResponse };
}
//# sourceMappingURL=txn-pool.d.ts.map