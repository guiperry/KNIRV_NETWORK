/**
 * Common error types for KNIRV SDK
 */
export declare class KNIRVError extends Error {
    code?: string;
    details?: any;
    constructor(message: string, code?: string, details?: any);
}
export declare class KNIRVNetworkError extends KNIRVError {
    constructor(message: string, details?: any);
}
export declare class KNIRVTransactionError extends KNIRVError {
    constructor(message: string, details?: any);
}
export declare class KNIRVWalletError extends KNIRVError {
    constructor(message: string, details?: any);
}
export declare class KNIRVSkillError extends KNIRVError {
    constructor(message: string, details?: any);
}
export declare class KNIRVResourceError extends KNIRVError {
    constructor(message: string, details?: any);
}
//# sourceMappingURL=errors.d.ts.map