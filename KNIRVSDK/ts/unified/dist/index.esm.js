export { Wallet as Tm2Wallet, TransactionEndpoint, Tx, TxSignature, generateHDPath } from '@gnolang/tm2-js-client';
export { LedgerConnector } from '@cosmjs/ledger-amino';

/**
 * Unified KNIRV Client - Provides access to all KNIRV Network services
 */
/**
 * Unified KNIRV Client providing access to all KNIRV Network services
 */
class KNIRVClient {
    constructor(config = {}) {
        // Initialize gateway client (temporarily disabled)
        // this.gateway = new KNIRVGatewayClient(config.gateway);
        // Initialize transaction client (temporarily disabled)
        // this.transaction = new KnirvchainTransactionSDK(config.transaction);
        // Initialize transmission client (temporarily disabled)
        // this.transmission = new KnirvClient(config.transmission || {
        //   bootstrapPeers: [],
        //   enableDHT: true,
        //   port: 0
        // });
        // Initialize wallet (placeholder - would need proper implementation)
        this.wallet = new (class {
            async getAccount() {
                throw new Error('Not implemented');
            }
            async addNetwork(params) {
                throw new Error('Not implemented');
            }
            async switchNetwork(chainId) {
                throw new Error('Not implemented');
            }
            async signTransaction(transaction) {
                throw new Error('Not implemented');
            }
            async doContract(params) {
                throw new Error('Not implemented');
            }
        })();
    }
    /**
     * Create a client optimized for gateway operations (temporarily disabled)
     */
    // static createGatewayClient(config?: KNIRVClientConfig['gateway']) {
    //   return new KNIRVGatewayClient(config);
    // }
    /**
     * Create a client optimized for transaction operations (temporarily disabled)
     */
    // static createTransactionClient(config?: KNIRVClientConfig['transaction']) {
    //   return new KnirvchainTransactionSDK(config);
    // }
    /**
     * Create a client optimized for transmission operations (temporarily disabled)
     */
    // static createTransmissionClient(config?: KNIRVClientConfig['transmission']) {
    //   return new KnirvClient(config || {
    //     bootstrapPeers: [],
    //     enableDHT: true,
    //     port: 0
    //   });
    // }
    /**
     * Health check for all services
     */
    async healthCheck() {
        const results = {
            gateway: false,
            transaction: false,
            transmission: false,
        };
        try {
            // await this.gateway.health.checkGatewayHealth(); // Temporarily disabled
            results.gateway = true;
        }
        catch (error) {
            console.warn('Gateway health check failed:', error);
        }
        // Add transaction and transmission health checks when available
        return results;
    }
}

/**
 * Wallet compatibility layer for KNIRV SDK
 *
 * This module provides wallet functionality and maintains compatibility
 * with existing wallet implementations while integrating KNIRV services.
 */
var WalletResponseStatus;
(function (WalletResponseStatus) {
    WalletResponseStatus["SUCCESS"] = "success";
    WalletResponseStatus["FAILURE"] = "failure";
    WalletResponseStatus["REJECT"] = "reject";
})(WalletResponseStatus || (WalletResponseStatus = {}));
var WalletResponseType;
(function (WalletResponseType) {
    WalletResponseType["ESTABLISH"] = "establish";
    WalletResponseType["ACCOUNT"] = "account";
    WalletResponseType["NETWORK"] = "network";
    WalletResponseType["SIGN"] = "sign";
    WalletResponseType["TRANSACTION"] = "transaction";
})(WalletResponseType || (WalletResponseType = {}));
var WalletResponseExecuteType;
(function (WalletResponseExecuteType) {
    WalletResponseExecuteType["ADD_ESTABLISH"] = "ADD_ESTABLISH";
    WalletResponseExecuteType["GET_ACCOUNT"] = "GET_ACCOUNT";
    WalletResponseExecuteType["ADD_NETWORK"] = "ADD_NETWORK";
    WalletResponseExecuteType["SWITCH_NETWORK"] = "SWITCH_NETWORK";
    WalletResponseExecuteType["DO_CONTRACT"] = "DO_CONTRACT";
    WalletResponseExecuteType["SIGN_TX"] = "SIGN_TX";
})(WalletResponseExecuteType || (WalletResponseExecuteType = {}));
var WalletResponseFailureType;
(function (WalletResponseFailureType) {
    WalletResponseFailureType["NETWORK_TIMEOUT"] = "NETWORK_TIMEOUT";
    WalletResponseFailureType["UNAPPROVED_CHAIN"] = "UNAPPROVED_CHAIN";
    WalletResponseFailureType["UNAPPROVED_HOST"] = "UNAPPROVED_HOST";
    WalletResponseFailureType["LOCKED_ACCOUNT"] = "LOCKED_ACCOUNT";
    WalletResponseFailureType["INVALID_FORMAT"] = "INVALID_FORMAT";
    WalletResponseFailureType["INVALID_TRANSACTION"] = "INVALID_TRANSACTION";
    WalletResponseFailureType["UNEXPECTED_ERROR"] = "UNEXPECTED_ERROR";
})(WalletResponseFailureType || (WalletResponseFailureType = {}));
var WalletResponseRejectType;
(function (WalletResponseRejectType) {
    WalletResponseRejectType["ESTABLISH_REJECTED"] = "ESTABLISH_REJECTED";
    WalletResponseRejectType["SIGN_REJECTED"] = "SIGN_REJECTED";
    WalletResponseRejectType["TRANSACTION_REJECTED"] = "TRANSACTION_REJECTED";
})(WalletResponseRejectType || (WalletResponseRejectType = {}));
var WalletResponseSuccessType;
(function (WalletResponseSuccessType) {
    WalletResponseSuccessType["ESTABLISH_SUCCESS"] = "ESTABLISH_SUCCESS";
    WalletResponseSuccessType["SIGN_SUCCESS"] = "SIGN_SUCCESS";
    WalletResponseSuccessType["TRANSACTION_SUCCESS"] = "TRANSACTION_SUCCESS";
})(WalletResponseSuccessType || (WalletResponseSuccessType = {}));
// Legacy AdenaWallet class for backward compatibility
class AdenaWallet {
    constructor(config) {
        this.config = config;
    }
    async getAccount() {
        // Implementation would integrate with KNIRV transaction API
        throw new Error('Not implemented - integrate with KNIRV transaction API');
    }
    async addNetwork(params) {
        // Implementation would integrate with KNIRV gateway API
        throw new Error('Not implemented - integrate with KNIRV gateway API');
    }
    async switchNetwork(chainId) {
        // Implementation would integrate with KNIRV gateway API
        throw new Error('Not implemented - integrate with KNIRV gateway API');
    }
    async signTransaction(transaction) {
        // Implementation would integrate with KNIRV transaction API
        throw new Error('Not implemented - integrate with KNIRV transaction API');
    }
    async doContract(params) {
        // Implementation would integrate with KNIRV transaction API
        throw new Error('Not implemented - integrate with KNIRV transaction API');
    }
    async invokeSkill(params) {
        // Implementation would integrate with KNIRV gateway economics API
        throw new Error('Not implemented - integrate with KNIRV gateway economics API');
    }
    async resolveKnirvURI(uri) {
        // Implementation would integrate with KNIRV transmission API
        throw new Error('Not implemented - integrate with KNIRV transmission API');
    }
    async broadcastToNetwork(data) {
        // Implementation would integrate with KNIRV transmission API
        throw new Error('Not implemented - integrate with KNIRV transmission API');
    }
}

/**
 * Common error types for KNIRV SDK
 */
// Base KNIRV error class
class KNIRVError extends Error {
    constructor(message, code, details) {
        super(message);
        this.code = code;
        this.details = details;
        this.name = 'KNIRVError';
    }
}
// Network-related errors
class KNIRVNetworkError extends KNIRVError {
    constructor(message, details) {
        super(message, 'NETWORK_ERROR', details);
        this.name = 'KNIRVNetworkError';
    }
}
// Transaction-related errors
class KNIRVTransactionError extends KNIRVError {
    constructor(message, details) {
        super(message, 'TRANSACTION_ERROR', details);
        this.name = 'KNIRVTransactionError';
    }
}
// Wallet-related errors
class KNIRVWalletError extends KNIRVError {
    constructor(message, details) {
        super(message, 'WALLET_ERROR', details);
        this.name = 'KNIRVWalletError';
    }
}
// Skill invocation errors
class KNIRVSkillError extends KNIRVError {
    constructor(message, details) {
        super(message, 'SKILL_ERROR', details);
        this.name = 'KNIRVSkillError';
    }
}
// Resource resolution errors
class KNIRVResourceError extends KNIRVError {
    constructor(message, details) {
        super(message, 'RESOURCE_ERROR', details);
        this.name = 'KNIRVResourceError';
    }
}
// Additional error types can be added here as needed

/**
 * KNIRV SDK - Unified TypeScript/JavaScript SDK for KNIRV Network
 *
 * This package provides wallet functionality compatible with existing wallet implementations
 * and re-exports essential blockchain functionality.
 */
// Re-export the main client
// Version information
const VERSION = '1.0.0';

export { AdenaWallet, KNIRVClient, KNIRVError, KNIRVNetworkError, KNIRVResourceError, KNIRVSkillError, KNIRVTransactionError, KNIRVWalletError, VERSION, WalletResponseExecuteType, WalletResponseFailureType, WalletResponseRejectType, WalletResponseStatus, WalletResponseSuccessType, WalletResponseType };
//# sourceMappingURL=index.esm.js.map
