export declare const WalletResponseExecuteType: {
    readonly ADD_ESTABLISH: "ADD_ESTABLISH";
    readonly GET_ACCOUNT: "GET_ACCOUNT";
    readonly GET_NETWORK: "GET_NETWORK";
    readonly ADD_NETWORK: "ADD_NETWORK";
    readonly SWITCH_NETWORK: "SWITCH_NETWORK";
    readonly DO_CONTRACT: "DO_CONTRACT";
    readonly SIGN_TX: "SIGN_TX";
    readonly SIGN_AMINO: "SIGN_AMINO";
};
export declare const WalletResponseSuccessType: {
    readonly CONNECTION_SUCCESS: "CONNECTION_SUCCESS";
    readonly GET_ACCOUNT_SUCCESS: "GET_ACCOUNT_SUCCESS";
    readonly GET_NETWORK_SUCCESS: "GET_NETWORK_SUCCESS";
    readonly ADD_NETWORK_SUCCESS: "ADD_NETWORK_SUCCESS";
    readonly SWITCH_NETWORK_SUCCESS: "SWITCH_NETWORK_SUCCESS";
    readonly SIGN_SUCCESS: "SIGN_SUCCESS";
    readonly TRANSACTION_SUCCESS: "TRANSACTION_SUCCESS";
};
export declare const WalletResponseFailureType: {
    readonly NETWORK_TIMEOUT: "NETWORK_TIMEOUT";
    readonly WALLET_LOCKED: "WALLET_LOCKED";
    readonly NO_ACCOUNT: "NO_ACCOUNT";
    readonly ACCOUNT_MISMATCH: "ACCOUNT_MISMATCH";
    readonly ALREADY_CONNECTED: "ALREADY_CONNECTED";
    readonly NOT_CONNECTED: "NOT_CONNECTED";
    readonly UNADDED_NETWORK: "UNADDED_NETWORK";
    readonly NETWORK_ALREADY_EXISTS: "NETWORK_ALREADY_EXISTS";
    readonly REDUNDANT_CHANGE_REQUEST: "REDUNDANT_CHANGE_REQUEST";
    readonly UNRESOLVED_TRANSACTION_EXISTS: "UNRESOLVED_TRANSACTION_EXISTS";
    readonly UNSUPPORTED_TYPE: "UNSUPPORTED_TYPE";
    readonly INVALID_FORMAT: "INVALID_FORMAT";
    readonly SIGN_FAILED: "SIGN_FAILED";
    readonly TRANSACTION_FAILED: "TRANSACTION_FAILED";
    readonly UNEXPECTED_ERROR: "UNEXPECTED_ERROR";
};
export declare const WalletResponseRejectType: {
    readonly CONNECTION_REJECTED: "CONNECTION_REJECTED";
    readonly SIGN_REJECTED: "SIGN_REJECTED";
    readonly TRANSACTION_REJECTED: "TRANSACTION_REJECTED";
    readonly ADD_NETWORK_REJECTED: "ADD_NETWORK_REJECTED";
    readonly SWITCH_NETWORK_REJECTED: "SWITCH_NETWORK_REJECTED";
};
export declare const walletMessages: {
    readonly ADD_ESTABLISH: {
        readonly code: 1001;
        readonly message: "Request to connect this site to the wallet";
        readonly type: "ADD_ESTABLISH";
    };
    readonly GET_ACCOUNT: {
        readonly code: 1002;
        readonly message: "Request the connected account";
        readonly type: "GET_ACCOUNT";
    };
    readonly GET_NETWORK: {
        readonly code: 1003;
        readonly message: "Request the current network";
        readonly type: "GET_NETWORK";
    };
    readonly ADD_NETWORK: {
        readonly code: 1004;
        readonly message: "Request to add a network";
        readonly type: "ADD_NETWORK";
    };
    readonly SWITCH_NETWORK: {
        readonly code: 1005;
        readonly message: "Request to switch network";
        readonly type: "SWITCH_NETWORK";
    };
    readonly DO_CONTRACT: {
        readonly code: 1006;
        readonly message: "Request to sign and broadcast a transaction";
        readonly type: "DO_CONTRACT";
    };
    readonly SIGN_TX: {
        readonly code: 1007;
        readonly message: "Request to sign a transaction";
        readonly type: "SIGN_TX";
    };
    readonly SIGN_AMINO: {
        readonly code: 1008;
        readonly message: "Request to sign with Amino (disabled: use SIGN_TX, which signs with SIGN_MODE_DIRECT)";
        readonly type: "SIGN_AMINO";
    };
    readonly CONNECTION_SUCCESS: {
        readonly code: 2000;
        readonly message: "The site is connected to the wallet";
        readonly type: "CONNECTION_SUCCESS";
    };
    readonly GET_ACCOUNT_SUCCESS: {
        readonly code: 2001;
        readonly message: "The account was returned";
        readonly type: "GET_ACCOUNT_SUCCESS";
    };
    readonly GET_NETWORK_SUCCESS: {
        readonly code: 2002;
        readonly message: "The network was returned";
        readonly type: "GET_NETWORK_SUCCESS";
    };
    readonly ADD_NETWORK_SUCCESS: {
        readonly code: 2003;
        readonly message: "The network was added";
        readonly type: "ADD_NETWORK_SUCCESS";
    };
    readonly SWITCH_NETWORK_SUCCESS: {
        readonly code: 2004;
        readonly message: "The network was switched";
        readonly type: "SWITCH_NETWORK_SUCCESS";
    };
    readonly SIGN_SUCCESS: {
        readonly code: 2005;
        readonly message: "The transaction was signed";
        readonly type: "SIGN_SUCCESS";
    };
    readonly TRANSACTION_SUCCESS: {
        readonly code: 2006;
        readonly message: "The transaction was sent";
        readonly type: "TRANSACTION_SUCCESS";
    };
    readonly NETWORK_TIMEOUT: {
        readonly code: 3000;
        readonly message: "The network did not respond in time";
        readonly type: "NETWORK_TIMEOUT";
    };
    readonly WALLET_LOCKED: {
        readonly code: 3001;
        readonly message: "The wallet is locked";
        readonly type: "WALLET_LOCKED";
    };
    readonly NO_ACCOUNT: {
        readonly code: 3002;
        readonly message: "The wallet has no account";
        readonly type: "NO_ACCOUNT";
    };
    readonly ACCOUNT_MISMATCH: {
        readonly code: 3003;
        readonly message: "The account does not match the request";
        readonly type: "ACCOUNT_MISMATCH";
    };
    readonly ALREADY_CONNECTED: {
        readonly code: 3004;
        readonly message: "This site is already connected";
        readonly type: "ALREADY_CONNECTED";
    };
    readonly NOT_CONNECTED: {
        readonly code: 3005;
        readonly message: "This site is not connected to the wallet";
        readonly type: "NOT_CONNECTED";
    };
    readonly UNADDED_NETWORK: {
        readonly code: 3006;
        readonly message: "The network has not been added to the wallet";
        readonly type: "UNADDED_NETWORK";
    };
    readonly NETWORK_ALREADY_EXISTS: {
        readonly code: 3007;
        readonly message: "The network already exists";
        readonly type: "NETWORK_ALREADY_EXISTS";
    };
    readonly REDUNDANT_CHANGE_REQUEST: {
        readonly code: 3008;
        readonly message: "The requested change is already in effect";
        readonly type: "REDUNDANT_CHANGE_REQUEST";
    };
    readonly UNRESOLVED_TRANSACTION_EXISTS: {
        readonly code: 3009;
        readonly message: "A transaction request is still waiting for approval";
        readonly type: "UNRESOLVED_TRANSACTION_EXISTS";
    };
    readonly UNSUPPORTED_TYPE: {
        readonly code: 3010;
        readonly message: "The request type is not supported";
        readonly type: "UNSUPPORTED_TYPE";
    };
    readonly INVALID_FORMAT: {
        readonly code: 3011;
        readonly message: "The request is not in a valid format";
        readonly type: "INVALID_FORMAT";
    };
    readonly SIGN_FAILED: {
        readonly code: 3012;
        readonly message: "Signing failed";
        readonly type: "SIGN_FAILED";
    };
    readonly TRANSACTION_FAILED: {
        readonly code: 3013;
        readonly message: "The transaction failed";
        readonly type: "TRANSACTION_FAILED";
    };
    readonly UNEXPECTED_ERROR: {
        readonly code: 3014;
        readonly message: "An unexpected error occurred";
        readonly type: "UNEXPECTED_ERROR";
    };
    readonly CONNECTION_REJECTED: {
        readonly code: 4000;
        readonly message: "The user rejected the connection";
        readonly type: "CONNECTION_REJECTED";
    };
    readonly SIGN_REJECTED: {
        readonly code: 4001;
        readonly message: "The user rejected signing";
        readonly type: "SIGN_REJECTED";
    };
    readonly TRANSACTION_REJECTED: {
        readonly code: 4002;
        readonly message: "The user rejected the transaction";
        readonly type: "TRANSACTION_REJECTED";
    };
    readonly ADD_NETWORK_REJECTED: {
        readonly code: 4003;
        readonly message: "The user rejected adding the network";
        readonly type: "ADD_NETWORK_REJECTED";
    };
    readonly SWITCH_NETWORK_REJECTED: {
        readonly code: 4004;
        readonly message: "The user rejected switching network";
        readonly type: "SWITCH_NETWORK_REJECTED";
    };
};
