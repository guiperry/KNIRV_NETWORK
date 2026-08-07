/** Controller-custodied wallet client. Private user keys never enter the SDK process. */
import type { DirectSignRequest, KNIRVFee } from './signing';
export interface WalletResponse<T = unknown> {
    code: number;
    status: WalletResponseStatus;
    type: WalletResponseType;
    message?: string;
    data?: T;
}
export declare enum WalletResponseStatus {
    SUCCESS = "success",
    FAILURE = "failure",
    REJECT = "reject"
}
export declare enum WalletResponseType {
    ESTABLISH = "establish",
    ACCOUNT = "account",
    NETWORK = "network",
    SIGN = "sign",
    TRANSACTION = "transaction"
}
export declare enum WalletResponseExecuteType {
    ADD_ESTABLISH = "ADD_ESTABLISH",
    GET_ACCOUNT = "GET_ACCOUNT",
    ADD_NETWORK = "ADD_NETWORK",
    SWITCH_NETWORK = "SWITCH_NETWORK",
    DO_CONTRACT = "DO_CONTRACT",
    SIGN_TX = "SIGN_TX"
}
export declare enum WalletResponseFailureType {
    NETWORK_TIMEOUT = "NETWORK_TIMEOUT",
    UNAPPROVED_CHAIN = "UNAPPROVED_CHAIN",
    UNAPPROVED_HOST = "UNAPPROVED_HOST",
    LOCKED_ACCOUNT = "LOCKED_ACCOUNT",
    INVALID_FORMAT = "INVALID_FORMAT",
    INVALID_TRANSACTION = "INVALID_TRANSACTION",
    UNEXPECTED_ERROR = "UNEXPECTED_ERROR"
}
export declare enum WalletResponseRejectType {
    ESTABLISH_REJECTED = "ESTABLISH_REJECTED",
    SIGN_REJECTED = "SIGN_REJECTED",
    TRANSACTION_REJECTED = "TRANSACTION_REJECTED"
}
export declare enum WalletResponseSuccessType {
    ESTABLISH_SUCCESS = "ESTABLISH_SUCCESS",
    SIGN_SUCCESS = "SIGN_SUCCESS",
    TRANSACTION_SUCCESS = "TRANSACTION_SUCCESS"
}
export interface SkillInvocationParams {
    user_id: string;
    skill_id: string;
    sender: string;
    amount: string;
    account_number: string | number | bigint;
    sequence: string | number | bigint;
    fee?: KNIRVFee;
    metadata?: Record<string, unknown>;
}
export interface KNIRVWalletConfig {
    provider?: string;
    chainId?: string;
    enableGasless?: boolean;
    apiKey?: string;
    approvalTimeoutMs?: number;
}
export declare class KNIRVWallet {
    private readonly gateways;
    private readonly chainId;
    private readonly apiKey?;
    private readonly approvalTimeoutMs;
    constructor(config?: KNIRVWalletConfig);
    private request;
    private approvedOperation;
    getAccount(): Promise<WalletResponse>;
    addNetwork(params: unknown): Promise<WalletResponse>;
    switchNetwork(chainId: string): Promise<WalletResponse>;
    signTransaction(transaction: DirectSignRequest): Promise<WalletResponse>;
    signMessage(envelope: unknown): Promise<WalletResponse>;
    doContract(transaction: DirectSignRequest): Promise<WalletResponse>;
    invokeSkill(params: SkillInvocationParams): Promise<WalletResponse>;
    resolveKnirvURI(uri: string): Promise<WalletResponse>;
    broadcastToNetwork(data: unknown): Promise<WalletResponse>;
}
export { KNIRVWallet as AdenaWallet };
export { generateHDPath, Provider, TransactionEndpoint, Tx, Wallet as Tm2Wallet, BroadcastTxCommitResult, BroadcastTxSyncResult, TxSignature, } from '@gnolang/tm2-js-client';
export { LedgerConnector } from '@cosmjs/ledger-amino';
//# sourceMappingURL=wallet.d.ts.map