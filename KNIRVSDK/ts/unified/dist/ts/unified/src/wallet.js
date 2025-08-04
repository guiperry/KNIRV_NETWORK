/**
 * Wallet compatibility layer for KNIRV SDK
 *
 * This module provides wallet functionality and maintains compatibility
 * with existing wallet implementations while integrating KNIRV services.
 */
export var WalletResponseStatus;
(function (WalletResponseStatus) {
    WalletResponseStatus["SUCCESS"] = "success";
    WalletResponseStatus["FAILURE"] = "failure";
    WalletResponseStatus["REJECT"] = "reject";
})(WalletResponseStatus || (WalletResponseStatus = {}));
export var WalletResponseType;
(function (WalletResponseType) {
    WalletResponseType["ESTABLISH"] = "establish";
    WalletResponseType["ACCOUNT"] = "account";
    WalletResponseType["NETWORK"] = "network";
    WalletResponseType["SIGN"] = "sign";
    WalletResponseType["TRANSACTION"] = "transaction";
})(WalletResponseType || (WalletResponseType = {}));
export var WalletResponseExecuteType;
(function (WalletResponseExecuteType) {
    WalletResponseExecuteType["ADD_ESTABLISH"] = "ADD_ESTABLISH";
    WalletResponseExecuteType["GET_ACCOUNT"] = "GET_ACCOUNT";
    WalletResponseExecuteType["ADD_NETWORK"] = "ADD_NETWORK";
    WalletResponseExecuteType["SWITCH_NETWORK"] = "SWITCH_NETWORK";
    WalletResponseExecuteType["DO_CONTRACT"] = "DO_CONTRACT";
    WalletResponseExecuteType["SIGN_TX"] = "SIGN_TX";
})(WalletResponseExecuteType || (WalletResponseExecuteType = {}));
export var WalletResponseFailureType;
(function (WalletResponseFailureType) {
    WalletResponseFailureType["NETWORK_TIMEOUT"] = "NETWORK_TIMEOUT";
    WalletResponseFailureType["UNAPPROVED_CHAIN"] = "UNAPPROVED_CHAIN";
    WalletResponseFailureType["UNAPPROVED_HOST"] = "UNAPPROVED_HOST";
    WalletResponseFailureType["LOCKED_ACCOUNT"] = "LOCKED_ACCOUNT";
    WalletResponseFailureType["INVALID_FORMAT"] = "INVALID_FORMAT";
    WalletResponseFailureType["INVALID_TRANSACTION"] = "INVALID_TRANSACTION";
    WalletResponseFailureType["UNEXPECTED_ERROR"] = "UNEXPECTED_ERROR";
})(WalletResponseFailureType || (WalletResponseFailureType = {}));
export var WalletResponseRejectType;
(function (WalletResponseRejectType) {
    WalletResponseRejectType["ESTABLISH_REJECTED"] = "ESTABLISH_REJECTED";
    WalletResponseRejectType["SIGN_REJECTED"] = "SIGN_REJECTED";
    WalletResponseRejectType["TRANSACTION_REJECTED"] = "TRANSACTION_REJECTED";
})(WalletResponseRejectType || (WalletResponseRejectType = {}));
export var WalletResponseSuccessType;
(function (WalletResponseSuccessType) {
    WalletResponseSuccessType["ESTABLISH_SUCCESS"] = "ESTABLISH_SUCCESS";
    WalletResponseSuccessType["SIGN_SUCCESS"] = "SIGN_SUCCESS";
    WalletResponseSuccessType["TRANSACTION_SUCCESS"] = "TRANSACTION_SUCCESS";
})(WalletResponseSuccessType || (WalletResponseSuccessType = {}));
// Legacy AdenaWallet class for backward compatibility
export class AdenaWallet {
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
// Re-export blockchain functionality from @gnolang/tm2-js-client
export { generateHDPath, TransactionEndpoint, Tx, Wallet as Tm2Wallet, TxSignature, } from '@gnolang/tm2-js-client';
// Re-export ledger functionality
export { LedgerConnector } from '@cosmjs/ledger-amino';
