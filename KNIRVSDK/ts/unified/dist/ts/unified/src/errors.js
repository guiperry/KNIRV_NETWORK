/**
 * Common error types for KNIRV SDK
 */
// Base KNIRV error class
export class KNIRVError extends Error {
    constructor(message, code, details) {
        super(message);
        this.code = code;
        this.details = details;
        this.name = 'KNIRVError';
    }
}
// Network-related errors
export class KNIRVNetworkError extends KNIRVError {
    constructor(message, details) {
        super(message, 'NETWORK_ERROR', details);
        this.name = 'KNIRVNetworkError';
    }
}
// Transaction-related errors
export class KNIRVTransactionError extends KNIRVError {
    constructor(message, details) {
        super(message, 'TRANSACTION_ERROR', details);
        this.name = 'KNIRVTransactionError';
    }
}
// Wallet-related errors
export class KNIRVWalletError extends KNIRVError {
    constructor(message, details) {
        super(message, 'WALLET_ERROR', details);
        this.name = 'KNIRVWalletError';
    }
}
// Skill invocation errors
export class KNIRVSkillError extends KNIRVError {
    constructor(message, details) {
        super(message, 'SKILL_ERROR', details);
        this.name = 'KNIRVSkillError';
    }
}
// Resource resolution errors
export class KNIRVResourceError extends KNIRVError {
    constructor(message, details) {
        super(message, 'RESOURCE_ERROR', details);
        this.name = 'KNIRVResourceError';
    }
}
// Additional error types can be added here as needed
