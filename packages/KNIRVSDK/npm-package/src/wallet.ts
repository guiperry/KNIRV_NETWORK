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
export type WalletResponseExecuteType = (typeof protocol.WalletResponseExecuteType)[keyof typeof protocol.WalletResponseExecuteType];
export type WalletResponseSuccessType = (typeof protocol.WalletResponseSuccessType)[keyof typeof protocol.WalletResponseSuccessType];
export type WalletResponseFailureType = (typeof protocol.WalletResponseFailureType)[keyof typeof protocol.WalletResponseFailureType];
export type WalletResponseRejectType = (typeof protocol.WalletResponseRejectType)[keyof typeof protocol.WalletResponseRejectType];

/** Every key of the message table: a request kind or an outcome. */
export type WalletResponseType =
  | WalletResponseExecuteType
  | WalletResponseSuccessType
  | WalletResponseFailureType
  | WalletResponseRejectType;

/** Where a message is in its life: sent, answered, or ended well or badly. */
export type WalletResponseStatus = "request" | "response" | "common" | "success" | "failure";

export interface WalletMessage {
  readonly code: number;
  readonly message: string;
  /** The message's own key. */
  readonly type: WalletResponseType;
}

/** Code, message and type for every {@link WalletResponseType}. */
export const WalletMessageInfo: Readonly<Record<WalletResponseType, WalletMessage>> = protocol.walletMessages as Record<WalletResponseType, WalletMessage>;

/** What a page receives from the wallet. */
export interface WalletResponse<T = unknown> {
  status: WalletResponseStatus;
  code: number;
  type: WalletResponseType;
  message: string;
  data?: T;
}

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
} as const;
export type KnirvMethod = (typeof KnirvMethod)[keyof typeof KnirvMethod];

/** The events `window.knirv.on` emits. */
export type KnirvProviderEvent = "accountsChanged" | "chainChanged" | "connect" | "disconnect";

/** Error codes of `window.knirv.request`, following the EIP-1193 numbering. */
export const KnirvProviderErrorCode = {
  UserRejected: 4001,
  Unauthorized: 4100,
  UnsupportedMethod: 4200,
  Disconnected: 4900,
  UnrecognizedChain: 4902,
  InvalidParams: -32602,
  Internal: -32603,
} as const;

export class KnirvProviderError extends Error {
  constructor(
    readonly code: number,
    message: string,
    readonly data?: unknown,
  ) {
    super(message);
    this.name = "KnirvProviderError";
  }
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

  // The extension's own request-per-method API, kept for existing pages.
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
