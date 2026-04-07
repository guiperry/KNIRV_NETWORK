import type { FinalRequestOptions } from './request-options';
import { type KnirvchainTransactionSDK } from '../client';
export type APIResponseProps = {
    response: Response;
    options: FinalRequestOptions;
    controller: AbortController;
    requestLogID: string;
    retryOfRequestLogID: string | undefined;
    startTime: number;
};
export declare function defaultParseResponse<T>(client: KnirvchainTransactionSDK, props: APIResponseProps): Promise<T>;
//# sourceMappingURL=parse.d.ts.map