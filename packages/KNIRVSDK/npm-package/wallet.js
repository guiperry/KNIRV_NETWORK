/**
 * The KNIRV wallet extension's message protocol, shared by the extension, the
 * pages that talk to `window.knirv` and anything else that reads its replies.
 * The request kinds, outcomes, codes and messages are generated from
 * core/data/wallet_protocol.json (also served by the core as `wallet.protocol`);
 * this file adds the types around them. No logic lives here.
 */
import * as protocol from "./wallet/protocol.generated.js";
// Each name is both the const object (for `WalletResponseFailureType.NO_ACCOUNT`) and the union of its values.
export const WalletResponseExecuteType = protocol.WalletResponseExecuteType;
export const WalletResponseSuccessType = protocol.WalletResponseSuccessType;
export const WalletResponseFailureType = protocol.WalletResponseFailureType;
export const WalletResponseRejectType = protocol.WalletResponseRejectType;
/** Code, message and type for every {@link WalletResponseType}. */
export const WalletMessageInfo = protocol.walletMessages;
/** The methods `window.knirv.request` understands. */
export const KnirvMethod = {
    RequestAccounts: "knirv_requestAccounts",
    Accounts: "knirv_accounts",
    GetBalance: "knirv_getBalance",
    SendTransaction: "knirv_sendTransaction",
    SignTransaction: "knirv_signTransaction",
    SignMessage: "knirv_signMessage",
    ChainId: "knirv_chainId",
    SwitchChain: "knirv_switchChain",
};
/** Error codes of `window.knirv.request`, following the EIP-1193 numbering. */
export const KnirvProviderErrorCode = {
    UserRejected: 4001,
    Unauthorized: 4100,
    UnsupportedMethod: 4200,
    Disconnected: 4900,
    UnrecognizedChain: 4902,
    InvalidParams: -32602,
    Internal: -32603,
};
export class KnirvProviderError extends Error {
    code;
    data;
    constructor(code, message, data) {
        super(message);
        this.code = code;
        this.data = data;
        this.name = "KnirvProviderError";
    }
}
