/**
 * The KNIRV wallet extension's message protocol, shared by the extension, the
 * pages that talk to `window.knirv` and anything else that reads its replies.
 * The request kinds, outcomes, codes and messages are generated from
 * core/data/wallet_protocol.json (also served by the core as `wallet.protocol`);
 * this file adds the types around them. No logic lives here.
 */
import * as protocol from "./wallet/protocol.generated.js";
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
export type WalletResponseExecuteType = (typeof protocol.WalletResponseExecuteType)[keyof typeof protocol.WalletResponseExecuteType];
export type WalletResponseSuccessType = (typeof protocol.WalletResponseSuccessType)[keyof typeof protocol.WalletResponseSuccessType];
export type WalletResponseFailureType = (typeof protocol.WalletResponseFailureType)[keyof typeof protocol.WalletResponseFailureType];
export type WalletResponseRejectType = (typeof protocol.WalletResponseRejectType)[keyof typeof protocol.WalletResponseRejectType];
/** Every key of the message table: a request kind or an outcome. */
export type WalletResponseType = WalletResponseExecuteType | WalletResponseSuccessType | WalletResponseFailureType | WalletResponseRejectType;
/** Where a message is in its life: sent, answered, or ended well or badly. */
export type WalletResponseStatus = "request" | "response" | "common" | "success" | "failure";
export interface WalletMessage {
    readonly code: number;
    readonly message: string;
    /** The message's own key. */
    readonly type: WalletResponseType;
}
/** Code, message and type for every {@link WalletResponseType}. */
export declare const WalletMessageInfo: Readonly<Record<WalletResponseType, WalletMessage>>;
/** What a page receives from the wallet. */
export interface WalletResponse<T = unknown> {
    status: WalletResponseStatus;
    code: number;
    type: WalletResponseType;
    message: string;
    data?: T;
}
/** The methods `window.knirv.request` understands. */
export declare const KnirvMethod: {
    readonly RequestAccounts: "knirv_requestAccounts";
    readonly Accounts: "knirv_accounts";
    readonly GetBalance: "knirv_getBalance";
    readonly SendTransaction: "knirv_sendTransaction";
    readonly SignTransaction: "knirv_signTransaction";
    readonly SignMessage: "knirv_signMessage";
    readonly ChainId: "knirv_chainId";
    readonly SwitchChain: "knirv_switchChain";
};
export type KnirvMethod = (typeof KnirvMethod)[keyof typeof KnirvMethod];
/** The events `window.knirv.on` emits. */
export type KnirvProviderEvent = "accountsChanged" | "chainChanged" | "connect" | "disconnect";
/** Error codes of `window.knirv.request`, following the EIP-1193 numbering. */
export declare const KnirvProviderErrorCode: {
    readonly UserRejected: 4001;
    readonly Unauthorized: 4100;
    readonly UnsupportedMethod: 4200;
    readonly Disconnected: 4900;
    readonly UnrecognizedChain: 4902;
    readonly InvalidParams: -32602;
    readonly Internal: -32603;
};
export declare class KnirvProviderError extends Error {
    readonly code: number;
    readonly data?: unknown | undefined;
    constructor(code: number, message: string, data?: unknown | undefined);
}
export interface KnirvRequestArguments {
    method: KnirvMethod | (string & {});
    params?: unknown;
}
/** The provider the extension injects as `window.knirv`. */
export interface KnirvWalletProvider {
    readonly isKnirvWallet: true;
    readonly version: string;
    /** Rejects with a {@link KnirvProviderError}. */
    request(args: KnirvRequestArguments): Promise<unknown>;
    on(event: KnirvProviderEvent, listener: (payload: unknown) => void): KnirvWalletProvider;
    removeListener(event: KnirvProviderEvent, listener: (payload: unknown) => void): KnirvWalletProvider;
    AddEstablish(name: string): Promise<WalletResponse>;
    GetAccount(): Promise<WalletResponse>;
    GetNetwork(): Promise<WalletResponse>;
    DoContract(params: unknown): Promise<WalletResponse>;
    Sign(params: unknown): Promise<WalletResponse>;
    SignTx(params: unknown): Promise<WalletResponse>;
    AddNetwork(chain: unknown): Promise<WalletResponse>;
    SwitchNetwork(chainId: string): Promise<WalletResponse>;
    /** Subscribes to `changedAccount` / `changedNetwork`; false for any other event name. */
    On(eventName: string, callback: (message: string) => void): boolean;
}
